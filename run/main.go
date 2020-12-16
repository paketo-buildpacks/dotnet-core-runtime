package main

import (
	"os"

	dotnetcoreruntime "github.com/paketo-buildpacks/dotnet-core-runtime"
	"github.com/paketo-buildpacks/packit"
	"github.com/paketo-buildpacks/packit/cargo"
	"github.com/paketo-buildpacks/packit/chronos"
	"github.com/paketo-buildpacks/packit/postal"
)

func main() {
	bpYMLParser := dotnetcoreruntime.NewBuildpackYMLParser()
	logEmitter := dotnetcoreruntime.NewLogEmitter(os.Stdout)
	entryResolver := dotnetcoreruntime.NewPlanEntryResolver(logEmitter)
	dependencyManager := postal.NewService(cargo.NewTransport())
	planRefinery := dotnetcoreruntime.NewPlanRefinery()
	symlinker := dotnetcoreruntime.NewSymlinker()
	runtimeVersionResolver := dotnetcoreruntime.NewRuntimeVersionResolver(logEmitter)

	packit.Run(
		dotnetcoreruntime.Detect(bpYMLParser),
		dotnetcoreruntime.Build(
			entryResolver,
			dependencyManager,
			planRefinery,
			symlinker,
			runtimeVersionResolver,
			logEmitter,
			chronos.DefaultClock,
		),
	)
}
