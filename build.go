package dotnetcoreruntime

import (
	"path/filepath"
	"time"

	"github.com/paketo-buildpacks/packit"
	"github.com/paketo-buildpacks/packit/chronos"
	"github.com/paketo-buildpacks/packit/postal"
)

//go:generate faux --interface EntryResolver --output fakes/entry_resolver.go
type EntryResolver interface {
	Resolve([]packit.BuildpackPlanEntry) packit.BuildpackPlanEntry
}

//go:generate faux --interface DependencyManager --output fakes/dependency_manager.go
type DependencyManager interface {
	Resolve(path, id, version, stack string) (postal.Dependency, error)
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

func Build(entries EntryResolver, dependencies DependencyManager, planRefinery BuildPlanRefinery, dotnetSymlinker DotnetSymlinker, logger LogEmitter, clock chronos.Clock) packit.BuildFunc {
	return func(context packit.BuildContext) (packit.BuildResult, error) {
		logger.Title("%s %s", context.BuildpackInfo.Name, context.BuildpackInfo.Version)
		logger.Process("Resolving Dotnet Core Runtime version")

		entry := entries.Resolve(context.Plan.Entries)
		version, _ := entry.Metadata["version"].(string)

		dependency, err := dependencies.Resolve(filepath.Join(context.CNBPath, "buildpack.toml"), entry.Name, version, context.Stack)
		if err != nil {
			return packit.BuildResult{}, err
		}

		logger.SelectedDependency(entry, dependency, clock.Now())

		dotnetCoreRuntimeLayer, err := context.Layers.Get("dotnet-core-runtime")
		if err != nil {
			return packit.BuildResult{}, err
		}

		bom := planRefinery.BillOfMaterial(postal.Dependency{
			ID:      dependency.ID,
			Name:    dependency.Name,
			SHA256:  dependency.SHA256,
			Stacks:  dependency.Stacks,
			URI:     dependency.URI,
			Version: dependency.Version,
		})

		logger.Process("Executing build process")

		err = dotnetCoreRuntimeLayer.Reset()
		if err != nil {
			return packit.BuildResult{}, err
		}

		dotnetCoreRuntimeLayer.Launch = entry.Metadata["launch"] == true
		dotnetCoreRuntimeLayer.Build = entry.Metadata["build"] == true
		dotnetCoreRuntimeLayer.Cache = entry.Metadata["build"] == true

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
		dotnetCoreRuntimeLayer.SharedEnv.Override("DOTNET_ROOT", filepath.Join(context.WorkingDir, ".dotnet_root"))
		logger.Environment(dotnetCoreRuntimeLayer.SharedEnv)

		dotnetCoreRuntimeLayer.BuildEnv.Override("RUNTIME_VERSION", version)
		logger.Environment(dotnetCoreRuntimeLayer.BuildEnv)

		return packit.BuildResult{
			Plan:   bom,
			Layers: []packit.Layer{dotnetCoreRuntimeLayer},
		}, nil
	}
}
