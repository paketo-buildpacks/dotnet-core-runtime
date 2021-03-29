package dotnetcoreruntime

import (
	"fmt"
	"path/filepath"
	"regexp"
	"time"

	"github.com/Masterminds/semver"
	"github.com/paketo-buildpacks/packit"
	"github.com/paketo-buildpacks/packit/chronos"
	"github.com/paketo-buildpacks/packit/postal"
)

//go:generate faux --interface EntryResolver --output fakes/entry_resolver.go
type EntryResolver interface {
	Resolve(string, []packit.BuildpackPlanEntry, []interface{}) (packit.BuildpackPlanEntry, []packit.BuildpackPlanEntry)
	MergeLayerTypes(string, []packit.BuildpackPlanEntry) (launch, build bool)
}

//go:generate faux --interface DependencyManager --output fakes/dependency_manager.go
type DependencyManager interface {
	Install(dependency postal.Dependency, cnbPath, layerPath string) error
}

//go:generate faux --interface BuildPlanRefinery --output fakes/build_plan_refinery.go
type BuildPlanRefinery interface {
	BillOfMaterial(dependency postal.Dependency) packit.BuildpackPlan
}

//go:generate faux --interface DotnetSymlinker --output fakes/dotnet_symlinker.go
type DotnetSymlinker interface {
	Link(workingDir, layerPath string) (Err error)
}

//go:generate faux --interface VersionResolver --output fakes/version_resolver.go
type VersionResolver interface {
	Resolve(path string, entry packit.BuildpackPlanEntry, stack string) (postal.Dependency, error)
}

func Build(
	entries EntryResolver,
	dependencies DependencyManager,
	planRefinery BuildPlanRefinery,
	dotnetSymlinker DotnetSymlinker,
	versionResolver VersionResolver,
	logger LogEmitter,
	clock chronos.Clock,
) packit.BuildFunc {
	return func(context packit.BuildContext) (packit.BuildResult, error) {
		logger.Title("%s %s", context.BuildpackInfo.Name, context.BuildpackInfo.Version)
		logger.Process("Resolving Dotnet Core Runtime version")

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
			logger.Subprocess("WARNING: Setting the .NET Framework version through buildpack.yml will be deprecated soon in Dotnet Core Runtime Buildpack v%s.", nextMajorVersion.String())
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

		bom := planRefinery.BillOfMaterial(dependency)

		cachedDependencySHA, ok := dotnetCoreRuntimeLayer.Metadata["dependency-sha"]
		if ok && cachedDependencySHA == dependency.SHA256 {
			logger.Process(fmt.Sprintf("Reusing cached layer %s", dotnetCoreRuntimeLayer.Path))
			logger.Break()

			err = dotnetSymlinker.Link(context.WorkingDir, dotnetCoreRuntimeLayer.Path)
			if err != nil {
				return packit.BuildResult{}, err
			}

			return packit.BuildResult{
				Plan:   bom,
				Layers: []packit.Layer{dotnetCoreRuntimeLayer},
			}, nil

		}

		logger.Process("Executing build process")

		dotnetCoreRuntimeLayer, err = dotnetCoreRuntimeLayer.Reset()
		if err != nil {
			return packit.BuildResult{}, err
		}

		dotnetCoreRuntimeLayer.Launch, dotnetCoreRuntimeLayer.Build = entries.MergeLayerTypes("dotnet-runtime", context.Plan.Entries)
		dotnetCoreRuntimeLayer.Cache = dotnetCoreRuntimeLayer.Build

		logger.Subprocess("Installing Dotnet Core Runtime %s", dependency.Version)
		duration, err := clock.Measure(func() error {
			return dependencies.Install(dependency, context.CNBPath, dotnetCoreRuntimeLayer.Path)
		})
		if err != nil {
			return packit.BuildResult{}, err
		}

		logger.Action("Completed in %s", duration.Round(time.Millisecond))
		logger.Break()

		dotnetCoreRuntimeLayer.Metadata = map[string]interface{}{
			"dependency-sha": dependency.SHA256,
			"built_at":       clock.Now().Format(time.RFC3339Nano),
		}

		err = dotnetSymlinker.Link(context.WorkingDir, dotnetCoreRuntimeLayer.Path)
		if err != nil {
			return packit.BuildResult{}, err
		}

		// Set DOTNET_ROOT to the symlink directory in the working directory, instead of setting it to  the layer path itself.
		logger.Process("Configuring environment for build and launch")
		dotnetCoreRuntimeLayer.SharedEnv.Override("DOTNET_ROOT", filepath.Join(context.WorkingDir, ".dotnet_root"))
		logger.Environment(dotnetCoreRuntimeLayer.SharedEnv)

		logger.Process("Configuring environment for build")
		dotnetCoreRuntimeLayer.BuildEnv.Override("RUNTIME_VERSION", dependency.Version)
		logger.Environment(dotnetCoreRuntimeLayer.BuildEnv)

		return packit.BuildResult{
			Plan:   bom,
			Layers: []packit.Layer{dotnetCoreRuntimeLayer},
		}, nil
	}
}
