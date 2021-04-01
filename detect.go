package dotnetcoreruntime

import (
	"os"
	"path/filepath"

	"github.com/paketo-buildpacks/packit"
)

//go:generate faux --interface VersionParser --output fakes/version_parser.go
type VersionParser interface {
	ParseVersion(path string) (version string, err error)
}

func Detect(buildpackYMLParser VersionParser) packit.DetectFunc {
	return func(context packit.DetectContext) (packit.DetectResult, error) {
		var requirements []packit.BuildPlanRequirement

		// check if BP_DOTNET_FRAMEWORK_VERSION is set
		if version, ok := os.LookupEnv("BP_DOTNET_FRAMEWORK_VERSION"); ok {
			requirements = append(requirements, packit.BuildPlanRequirement{
				Name: "dotnet-runtime",
				Metadata: map[string]interface{}{
					"version-source": "BP_DOTNET_FRAMEWORK_VERSION",
					"version":        version,
				},
			})

		}

		// check if the version is set in the buildpack.yml
		version, err := buildpackYMLParser.ParseVersion(filepath.Join(context.WorkingDir, "buildpack.yml"))
		if err != nil {
			return packit.DetectResult{}, err
		}

		if version != "" {
			requirements = append(requirements, packit.BuildPlanRequirement{
				Name: "dotnet-runtime",
				Metadata: map[string]interface{}{
					"version-source": "buildpack.yml",
					"version":        version,
				},
			})
		}

		return packit.DetectResult{
			Plan: packit.BuildPlan{
				Provides: []packit.BuildPlanProvision{
					{Name: "dotnet-runtime"},
				},
				Requires: requirements,
			},
		}, nil
	}
}
