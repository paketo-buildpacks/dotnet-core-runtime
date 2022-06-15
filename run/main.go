package main

import (
	"os"

	dotnetcoreruntime "github.com/paketo-buildpacks/dotnet-core-runtime"
	"github.com/paketo-buildpacks/packit/v2"
	"github.com/paketo-buildpacks/packit/v2/cargo"
	"github.com/paketo-buildpacks/packit/v2/chronos"
	"github.com/paketo-buildpacks/packit/v2/draft"
	"github.com/paketo-buildpacks/packit/v2/postal"
	"github.com/paketo-buildpacks/packit/v2/sbom"
	"github.com/paketo-buildpacks/packit/v2/scribe"
)

type Generator struct{}

func (f Generator) GenerateFromDependency(dependency postal.Dependency, path string) (sbom.SBOM, error) {
	return sbom.GenerateFromDependency(dependency, path)
}

func main() {
	bpYMLParser := dotnetcoreruntime.NewBuildpackYMLParser()
	logEmitter := scribe.NewEmitter(os.Stdout)
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
			Generator{},
			logEmitter,
			chronos.DefaultClock,
		),
	)
}
