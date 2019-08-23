package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/cloudfoundry/dotnet-core-runtime-cnb/runtime"
	"github.com/cloudfoundry/libcfbuildpack/helper"
	"github.com/gravityblast/go-jsmin"
	"github.com/pkg/errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/buildpack/libbuildpack/buildplan"
	"github.com/cloudfoundry/libcfbuildpack/detect"
)

type BuildpackYAML struct {
	Config struct{
		Version string `yaml:"version""`
	} `yaml:"dotnet-runtime"`
}

type ConfigJSON struct {
	RuntimeOptions struct {
		Framework struct {
			Name    string `json:"name"`
			Version string `json:"version"`
		} `json:"framework"`
		ApplyPatches *bool `json:"applyPatches"`
	} `json:"runtimeOptions"`
}

func main() {
	context, err := detect.DefaultDetect()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "failed to create a default detection context: %s", err)
		os.Exit(100)
	}

	code, err := runDetect(context)
	if err != nil {
		context.Logger.Info(err.Error())
	}

	os.Exit(code)
}


func runDetect(context detect.Detect) (int, error) {
	plan := buildplan.Plan{
		Provides: []buildplan.Provided{{Name: runtime.DotnetRuntime}}}

	runtimeConfig, err := createRuntimeConfig(context.Application.Root)
	if err != nil {
		return context.Fail(), err
	}

	buildpackYAML, err := LoadBuildpackYAML(context.Application.Root)
	if err != nil {
		return context.Fail(), err
	}

	//Is FDD
	if hasRuntimeOptions(runtimeConfig) {
		rollForwardVersion := runtimeConfig.RuntimeOptions.Framework.Version

		if buildpackYAML != (BuildpackYAML{}) {
			err := checkIfVersionsAreValid(runtimeConfig.RuntimeOptions.Framework.Version, buildpackYAML.Config.Version)
			if err != nil {
				return context.Fail(), err
			}
			rollForwardVersion = buildpackYAML.Config.Version
		}

		version, compatibleVersion, err := rollForward(rollForwardVersion, context)
		if err != nil {
			return context.Fail(), err
		}

		if !compatibleVersion {
			return context.Fail(), fmt.Errorf("no version of the dotnet-runtime was compatible with what was specified in the runtimeconfig.json of the application")
		}

		plan.Requires = []buildplan.Required{{
			Name:     runtime.DotnetRuntime,
			Version:  version,
			Metadata: buildplan.Metadata{"launch": true},
		}}
	}

	return context.Pass(plan)
}

func hasRuntimeOptions(runtimeConfig ConfigJSON) bool {
	return runtimeConfig.RuntimeOptions.Framework.Name != ""
}

func checkIfVersionsAreValid(versionRuntimeConfig, versionBuildpackYAML string) error{
	splitVersionRuntimeConfig := strings.Split(versionRuntimeConfig, ".")
	splitVersionBuildpackYAML := strings.Split(versionBuildpackYAML, ".")

	if splitVersionBuildpackYAML[0] != splitVersionRuntimeConfig[0] {
		return fmt.Errorf("major versions of runtimes do not match between buildpack.yml and runtimeconfig.json")
	}

	minorBPYAML, err := strconv.Atoi(splitVersionBuildpackYAML[1])
	if err != nil{
		return err
	}

	minorRuntimeConfig, err := strconv.Atoi(splitVersionRuntimeConfig[1])
	if err != nil{
		return err
	}

	if minorBPYAML < minorRuntimeConfig{
		return fmt.Errorf("the minor version of the runtimeconfig.json is greater than the minor version of the buildpack.yml")
	}

	return nil
}

func createRuntimeConfig(appRoot string) (ConfigJSON, error){
	path, err := runtimeConfigPath(appRoot)
	if err != nil {
		return ConfigJSON{}, err
	}

	runtimeJSON := ConfigJSON{}

	if path != "" {
		runtimeJSON, err = parseRuntimeConfig(path)
		if err != nil {
			return ConfigJSON{}, err
		}
	}

	return runtimeJSON, nil
}

func runtimeConfigPath(appRoot string) (string, error) {
	if configFiles, err := filepath.Glob(filepath.Join(appRoot, "*.runtimeconfig.json")); err != nil {
		return "", err
	} else if len(configFiles) == 1 {
		return configFiles[0], nil
	} else if len(configFiles) > 1 {
		return "", fmt.Errorf("multiple *.runtimeconfig.json files present")
	}
	return "", nil
}

func parseRuntimeConfig(runtimeConfigPath string) (ConfigJSON, error) {
	obj := ConfigJSON{}

	buf, err := sanitizeJsonConfig(runtimeConfigPath)
	if err != nil {
		return obj, err
	}

	if err := json.Unmarshal(buf, &obj); err != nil {
		return obj, errors.Wrap(err, "unable to parse runtime config")
	}

	return obj, nil
}

func sanitizeJsonConfig(runtimeConfigPath string) ([]byte, error) {
	input, err := os.Open(runtimeConfigPath)
	if err != nil {
		return nil, err
	}
	defer input.Close()

	output := &bytes.Buffer{}

	if err := jsmin.Min(input, output); err != nil {
		return nil, err
	}

	return output.Bytes(), nil
}


func rollForward(version string, context detect.Detect) (string, bool, error){
	splitVersion := strings.Split(version, ".")
	anyPatch := fmt.Sprintf("%s.%s.*", splitVersion[0], splitVersion[1])
	anyMinor := fmt.Sprintf("%s.*.*", splitVersion[0])

	versions := []string {version, anyPatch, anyMinor}

	deps, err := context.Buildpack.Dependencies()
	if err != nil {
		return "", false, err
	}

	for _, versionConstraint := range versions {
		highestVersion, err := deps.Best(runtime.DotnetRuntime, versionConstraint, context.Stack)
		if err == nil {
			return highestVersion.Version.Original(), true, nil
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
