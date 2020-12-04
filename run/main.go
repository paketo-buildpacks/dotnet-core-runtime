package main

import (
	dotnetcoreruntime "github.com/paketo-buildpacks/dotnet-core-runtime"
	"github.com/paketo-buildpacks/packit"
)

func main() {
	bpYMLParser := dotnetcoreruntime.NewBuildpackYMLParser()
	packit.Run(dotnetcoreruntime.Detect(bpYMLParser), dotnetcoreruntime.Build())
}
