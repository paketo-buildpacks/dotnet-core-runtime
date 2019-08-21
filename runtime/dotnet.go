package runtime

import (
	"github.com/cloudfoundry/libcfbuildpack/build"
	"github.com/cloudfoundry/libcfbuildpack/buildpackplan"
	"github.com/cloudfoundry/libcfbuildpack/helper"
	"github.com/cloudfoundry/libcfbuildpack/layers"
	"github.com/cloudfoundry/libcfbuildpack/logger"
	"path/filepath"
)

const DotnetRuntime = "dotnet-runtime"

type Contributor struct {
	context      build.Build
	plan         buildpackplan.Plan
	runtimeLayer layers.DependencyLayer
	logger       logger.Logger
}

func NewContributor(context build.Build) (Contributor, bool, error) {
	plan, wantDependency, err := context.Plans.GetShallowMerged(DotnetRuntime)
	if err != nil {
		return Contributor{}, false, err
	}
	if !wantDependency {
		return Contributor{}, false, nil
	}

	dep, err := context.Buildpack.RuntimeDependency(DotnetRuntime, plan.Version, context.Stack)
	if err != nil {
		return Contributor{}, false, err
	}

	return Contributor{
		context:      context,
		plan:         plan,
		runtimeLayer: context.Layers.DependencyLayer(dep),
		logger:       context.Logger,
	}, true, nil
}

func (c Contributor) Contribute() error {

	return c.runtimeLayer.Contribute(func(artifact string, layer layers.DependencyLayer) error {
		layer.Logger.Body("Expanding to %s", layer.Root)

		if err := helper.ExtractTarXz(artifact, layer.Root, 0); err != nil {
			return err
		}

		if err := layer.OverrideSharedEnv("DOTNET_ROOT", filepath.Join(c.runtimeLayer.Root)); err != nil {
			return err
		}

		return nil
	}, getFlags(c.plan.Metadata)...)
}

func getFlags(metadata buildpackplan.Metadata) []layers.Flag{
	flagsArray := []layers.Flag{}
	flagValueMap := map[string]layers.Flag {"build": layers.Build, "launch": layers.Launch, "cache": layers.Cache}
	for _, flagName := range []string{"build", "launch", "cache"} {
		flagPresent, _ := metadata[flagName].(bool)
		if flagPresent {
			flagsArray = append(flagsArray, flagValueMap[flagName])
		}
	}
	return flagsArray
}
