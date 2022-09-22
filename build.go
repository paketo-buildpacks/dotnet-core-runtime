package dotnetcoreruntime

import (
	"fmt"
	"path/filepath"
	"regexp"
	"time"

	"github.com/Masterminds/semver"
	"github.com/paketo-buildpacks/packit/v2"
	"github.com/paketo-buildpacks/packit/v2/chronos"
	"github.com/paketo-buildpacks/packit/v2/postal"
	"github.com/paketo-buildpacks/packit/v2/sbom"
	"github.com/paketo-buildpacks/packit/v2/scribe"
)

//go:generate faux --interface EntryResolver --output fakes/entry_resolver.go
type EntryResolver interface {
	Resolve(string, []packit.BuildpackPlanEntry, []interface{}) (packit.BuildpackPlanEntry, []packit.BuildpackPlanEntry)
	MergeLayerTypes(string, []packit.BuildpackPlanEntry) (launch, build bool)
}

//go:generate faux --interface DependencyManager --output fakes/dependency_manager.go
type DependencyManager interface {
	Deliver(dependency postal.Dependency, cnbPath, layerPath, platformPath string) error
	GenerateBillOfMaterials(dependencies ...postal.Dependency) []packit.BOMEntry
}

//go:generate faux --interface DotnetSymlinker --output fakes/dotnet_symlinker.go
type DotnetSymlinker interface {
	Link(workingDir, layerPath string) (Err error)
}

//go:generate faux --interface VersionResolver --output fakes/version_resolver.go
type VersionResolver interface {
	Resolve(path string, entry packit.BuildpackPlanEntry, stack string) (postal.Dependency, error)
}

//go:generate faux --interface SBOMGenerator --output fakes/sbom_generator.go
type SBOMGenerator interface {
	GenerateFromDependency(dependency postal.Dependency, dir string) (sbom.SBOM, error)
}

func Build(
	entries EntryResolver,
	dependencies DependencyManager,
	dotnetSymlinker DotnetSymlinker,
	versionResolver VersionResolver,
	sbomGenerator SBOMGenerator,
	logger scribe.Emitter,
	clock chronos.Clock,
) packit.BuildFunc {
	return func(context packit.BuildContext) (packit.BuildResult, error) {
		logger.Title("%s %s", context.BuildpackInfo.Name, context.BuildpackInfo.Version)
		logger.Process("Resolving .NET Core Runtime version")

		priorities := []interface{}{
			"BP_DOTNET_FRAMEWORK_VERSION",
			"buildpack.yml",
			regexp.MustCompile(`.*\.(cs)|(fs)|(vb)proj`),
			"runtimeconfig.json",
		}
		entry, sortedEntries := entries.Resolve("dotnet-runtime", context.Plan.Entries, priorities)
		logger.Candidates(sortedEntries)

		source, _ := entry.Metadata["version-source"].(string)
		if source == "buildpack.yml" {
			nextMajorVersion := semver.MustParse(context.BuildpackInfo.Version).IncMajor()
			logger.Subprocess("WARNING: Setting the .NET Framework version through buildpack.yml will be deprecated soon in .NET Core Runtime Buildpack v%s.", nextMajorVersion.String())
			logger.Subprocess("Please specify the version through the $BP_DOTNET_FRAMEWORK_VERSION environment variable instead. See docs for more information.")
			logger.Break()
		}

		dependency, err := versionResolver.Resolve(filepath.Join(context.CNBPath, "buildpack.toml"), entry, context.Stack)
		if err != nil {
			return packit.BuildResult{}, err
		}

		logger.SelectedDependency(entry, dependency, clock.Now())

		dotnetCoreRuntimeLayer, err := context.Layers.Get("dotnet-core-runtime")
		if err != nil {
			return packit.BuildResult{}, err
		}

		bom := dependencies.GenerateBillOfMaterials(dependency)
		launch, build := entries.MergeLayerTypes("dotnet-runtime", context.Plan.Entries)

		var buildMetadata packit.BuildMetadata
		if build {
			buildMetadata.BOM = bom
		}

		var launchMetadata packit.LaunchMetadata
		if launch {
			launchMetadata.BOM = bom
		}

		cachedDependencySHA, ok := dotnetCoreRuntimeLayer.Metadata["dependency-sha"]
		if ok && cachedDependencySHA == dependency.SHA256 { //nolint:staticcheck
			logger.Process(fmt.Sprintf("Reusing cached layer %s", dotnetCoreRuntimeLayer.Path))
			logger.Break()

			err = dotnetSymlinker.Link(context.WorkingDir, dotnetCoreRuntimeLayer.Path)
			if err != nil {
				return packit.BuildResult{}, err
			}

			dotnetCoreRuntimeLayer.Launch, dotnetCoreRuntimeLayer.Build, dotnetCoreRuntimeLayer.Cache = launch, build, build

			return packit.BuildResult{
				Layers: []packit.Layer{dotnetCoreRuntimeLayer},
				Build:  buildMetadata,
				Launch: launchMetadata,
			}, nil

		}

		logger.Process("Executing build process")

		dotnetCoreRuntimeLayer, err = dotnetCoreRuntimeLayer.Reset()
		if err != nil {
			return packit.BuildResult{}, err
		}

		dotnetCoreRuntimeLayer.Launch, dotnetCoreRuntimeLayer.Build, dotnetCoreRuntimeLayer.Cache = launch, build, build

		logger.Subprocess("Installing .NET Core Runtime %s", dependency.Version)
		duration, err := clock.Measure(func() error {
			return dependencies.Deliver(dependency, context.CNBPath, dotnetCoreRuntimeLayer.Path, context.Platform.Path)
		})
		if err != nil {
			return packit.BuildResult{}, err
		}

		logger.Action("Completed in %s", duration.Round(time.Millisecond))
		logger.Break()

		dotnetCoreRuntimeLayer.Metadata = map[string]interface{}{
			"dependency-sha": dependency.SHA256, //nolint:staticcheck
		}

		err = dotnetSymlinker.Link(context.WorkingDir, dotnetCoreRuntimeLayer.Path)
		if err != nil {
			return packit.BuildResult{}, err
		}

		// Set DOTNET_ROOT to the symlink directory in the working directory, instead of setting it to  the layer path itself.
		dotnetCoreRuntimeLayer.LaunchEnv.Override("DOTNET_ROOT", filepath.Join(context.WorkingDir, ".dotnet_root"))

		dotnetCoreRuntimeLayer.BuildEnv.Override("RUNTIME_VERSION", dependency.Version)

		logger.EnvironmentVariables(dotnetCoreRuntimeLayer)

		logger.GeneratingSBOM(dotnetCoreRuntimeLayer.Path)
		var sbomContent sbom.SBOM
		duration, err = clock.Measure(func() error {
			sbomContent, err = sbomGenerator.GenerateFromDependency(dependency, dotnetCoreRuntimeLayer.Path)
			return err
		})
		if err != nil {
			return packit.BuildResult{}, err
		}

		logger.Action("Completed in %s", duration.Round(time.Millisecond))
		logger.Break()

		logger.FormattingSBOM(context.BuildpackInfo.SBOMFormats...)
		dotnetCoreRuntimeLayer.SBOM, err = sbomContent.InFormats(context.BuildpackInfo.SBOMFormats...)
		if err != nil {
			return packit.BuildResult{}, err
		}

		return packit.BuildResult{
			Layers: []packit.Layer{dotnetCoreRuntimeLayer},
			Build:  buildMetadata,
			Launch: launchMetadata,
		}, nil
	}
}
