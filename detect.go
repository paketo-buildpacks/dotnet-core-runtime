package dotnetcoreruntime

import (
	"os"
	"path/filepath"

	"github.com/paketo-buildpacks/packit/v2"
	"github.com/paketo-buildpacks/packit/v2/planning"
)

//go:generate faux --interface VersionParser --output fakes/version_parser.go
type VersionParser interface {
	ParseVersion(path string) (version string, err error)
}

func Detect(buildpackYMLParser VersionParser) packit.DetectFuncOf[planning.Metadata] {
	return func(context packit.DetectContext) (packit.DetectResultOf[planning.Metadata], error) {
		var requirements []packit.BuildPlanRequirementOf[planning.Metadata]

		// check if BP_DOTNET_FRAMEWORK_VERSION is set
		if version, ok := os.LookupEnv("BP_DOTNET_FRAMEWORK_VERSION"); ok {
			requirements = append(requirements, packit.BuildPlanRequirementOf[planning.Metadata]{
				Name: "dotnet-runtime",
				Metadata: planning.Metadata{
					VersionSource: "BP_DOTNET_FRAMEWORK_VERSION",
					Version:       version,
				},
			})

		}

		// check if the version is set in the buildpack.yml
		version, err := buildpackYMLParser.ParseVersion(filepath.Join(context.WorkingDir, "buildpack.yml"))
		if err != nil {
			return packit.DetectResultOf[planning.Metadata]{}, err
		}

		if version != "" {
			requirements = append(requirements, packit.BuildPlanRequirementOf[planning.Metadata]{
				Name: "dotnet-runtime",
				Metadata: planning.Metadata{
					VersionSource: "buildpack.yml",
					Version:       version,
				},
			})
		}

		return packit.DetectResultOf[planning.Metadata]{
			Plan: packit.BuildPlanOf[planning.Metadata]{
				Provides: []packit.BuildPlanProvision{
					{Name: "dotnet-runtime"},
				},
				Requires: requirements,
			},
		}, nil
	}
}
