package main

import (
	"fmt"
	"os"

	"github.com/cloudfoundry/dotnet-core-runtime-cnb/runtime"

	"github.com/buildpack/libbuildpack/buildplan"
	"github.com/cloudfoundry/libcfbuildpack/detect"
)

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

	runtimeConfig, err := runtime.NewRuntimeConfig(context.Application.Root)
	if err != nil {
		return context.Fail(), err
	}

	hasFDE, err := runtimeConfig.HasExecutable()
	if err != nil {
		return context.Fail(), err
	}

	//Is FDD that only relies on runtime
	if runtimeConfig.HasRuntimeDependency() {
		// Has an FDE
		if hasFDE {

			plan.Requires = []buildplan.Required{{
				Name:     runtime.DotnetRuntime,
				Version:  runtimeConfig.Version,
				Metadata: buildplan.Metadata{"launch": true},
			}}
		}
	}

	return context.Pass(plan)
}
