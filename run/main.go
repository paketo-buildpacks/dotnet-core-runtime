package main

import (
	"os"

	dotnetcoreruntime "github.com/paketo-buildpacks/dotnet-core-runtime"
	"github.com/paketo-buildpacks/packit/v2"
	"github.com/paketo-buildpacks/packit/v2/cargo"
	"github.com/paketo-buildpacks/packit/v2/chronos"
	"github.com/paketo-buildpacks/packit/v2/draft"
	"github.com/paketo-buildpacks/packit/v2/postal"
)

func main() {
	bpYMLParser := dotnetcoreruntime.NewBuildpackYMLParser()
	logEmitter := dotnetcoreruntime.NewLogEmitter(os.Stdout)
	entryResolver := draft.NewPlanner()
	dependencyManager := postal.NewService(cargo.NewTransport())
	symlinker := dotnetcoreruntime.NewSymlinker()
	runtimeVersionResolver := dotnetcoreruntime.NewRuntimeVersionResolver(logEmitter)

	packit.Run(
		dotnetcoreruntime.Detect(bpYMLParser),
		dotnetcoreruntime.Build(
			entryResolver,
			dependencyManager,
			symlinker,
			runtimeVersionResolver,
			logEmitter,
			chronos.DefaultClock,
		),
	)
}
