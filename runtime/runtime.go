package runtime

import (
	"fmt"
	"github.com/Masterminds/semver"
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

type BuildpackYAML struct {
	Config struct {
		Version string `yaml:"version""`
	} `yaml:"dotnet-runtime"`
}

func NewContributor(context build.Build) (Contributor, bool, error) {
	plan, wantDependency, err := context.Plans.GetShallowMerged(DotnetRuntime)
	if err != nil {
		return Contributor{}, false, err
	}
	if !wantDependency {
		return Contributor{}, false, nil
	}

	version := plan.Version

	if plan.Version != "" {
		var compatibleVersion bool
		rollForwardVersion := plan.Version

		buildpackYAML, err := LoadBuildpackYAML(context.Application.Root)
		if err != nil {
			return Contributor{}, false, err
		}

		if buildpackYAML != (BuildpackYAML{}) {
			err := checkIfVersionsAreValid(rollForwardVersion, buildpackYAML.Config.Version)
			if err != nil {
				return Contributor{}, false, err
			}
			rollForwardVersion = buildpackYAML.Config.Version
		}

		version, compatibleVersion, err = rollForward(rollForwardVersion, context)
		if err != nil {
			return Contributor{}, false, err
		}

		if !compatibleVersion {
			return Contributor{}, false, fmt.Errorf("no version of the dotnet-runtime was compatible with what was specified in the runtimeconfig.json of the application")
		}
	}

	dep, err := context.Buildpack.RuntimeDependency(DotnetRuntime, version, context.Stack)
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

		if err := layer.OverrideSharedEnv("DOTNET_ROOT", filepath.Join(layer.Root)); err != nil {
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

func checkIfVersionsAreValid(versionRuntimeConfig, versionBuildpackYAML string) error {
	runtimeVersion, err := semver.NewVersion(versionRuntimeConfig)
	if err != nil {
		return err
	}

	buildpackYAMLVersion, err := semver.NewVersion(versionBuildpackYAML)
	if err != nil {
		return err
	}

	if runtimeVersion.Major() != buildpackYAMLVersion.Major(){
		return fmt.Errorf("major versions of runtimes do not match between buildpack.yml and runtimeconfig.json")
	}

	if buildpackYAMLVersion.Minor() < runtimeVersion.Minor() {
		return fmt.Errorf("the minor version of the runtimeconfig.json is greater than the minor version of the buildpack.yml")
	}

	return nil
}

func rollForward(version string, context build.Build) (string, bool, error) {
	splitVersion, err := semver.NewVersion(version)
	if err != nil {
		return "", false, err
	}
	anyPatch := fmt.Sprintf("%d.%d.*", splitVersion.Major(), splitVersion.Minor())
	anyMinor := fmt.Sprintf("%d.*.*", splitVersion.Major())

	versions := []string{version, anyPatch, anyMinor}

	deps, err := context.Buildpack.Dependencies()
	if err != nil {
		return "", false, err
	}

	for _, versionConstraint := range versions {
		highestVersion, err := deps.Best(DotnetRuntime, versionConstraint, context.Stack)
		if err == nil {
			return highestVersion.Version.String(), true, nil
		}
	}

	return "", false, fmt.Errorf("no compatible versions found")
}

func LoadBuildpackYAML(appRoot string) (BuildpackYAML, error) {
	var err error
	buildpackYAML := BuildpackYAML{}
	bpYamlPath := filepath.Join(appRoot, "buildpack.yml")

	if exists, err := helper.FileExists(bpYamlPath); err != nil {
		return BuildpackYAML{}, err
	} else if exists {
		err = helper.ReadBuildpackYaml(bpYamlPath, &buildpackYAML)
	}
	return buildpackYAML, err
}
