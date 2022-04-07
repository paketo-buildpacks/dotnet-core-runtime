package dotnetcoreruntime_test

import (
	"errors"
	"os"
	"testing"

	dotnetcoreruntime "github.com/paketo-buildpacks/dotnet-core-runtime"
	"github.com/paketo-buildpacks/dotnet-core-runtime/fakes"
	"github.com/paketo-buildpacks/packit/v2"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testDetect(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		buildpackYMLParser *fakes.VersionParser
		workingDir         string
		detect             packit.DetectFunc
	)

	it.Before(func() {
		var err error
		workingDir, err = os.MkdirTemp("", "working-dir")
		Expect(err).NotTo(HaveOccurred())

		buildpackYMLParser = &fakes.VersionParser{}
		detect = dotnetcoreruntime.Detect(buildpackYMLParser)
	})

	it.After(func() {
		Expect(os.RemoveAll(workingDir)).To(Succeed())
	})

	context("when there is no buildpack.yml and BP_DOTNET_FRAMEWORK_VERSION is unset", func() {
		it("provides dotnet core runtime", func() {
			result, err := detect(packit.DetectContext{
				WorkingDir: workingDir,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Plan).To(Equal(packit.BuildPlan{
				Provides: []packit.BuildPlanProvision{
					{
						Name: "dotnet-runtime",
					},
				},
			}))
		})
	})

	context("when BP_DOTNET_FRAMEWORK_VERSION is set", func() {
		it.Before(func() {
			Expect(os.Setenv("BP_DOTNET_FRAMEWORK_VERSION", "1.2.3")).To(Succeed())
		})

		it.After(func() {
			Expect(os.Unsetenv("BP_DOTNET_FRAMEWORK_VERSION")).To(Succeed())
		})

		it("provides and requires dotnet core runtime", func() {
			result, err := detect(packit.DetectContext{
				WorkingDir: workingDir,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Plan).To(Equal(packit.BuildPlan{
				Provides: []packit.BuildPlanProvision{
					{
						Name: "dotnet-runtime",
					},
				},
				Requires: []packit.BuildPlanRequirement{
					{
						Name: "dotnet-runtime",
						Metadata: map[string]interface{}{
							"version-source": "BP_DOTNET_FRAMEWORK_VERSION",
							"version":        "1.2.3",
						},
					},
				},
			}))
		})
	})

	context("when there is a buildpack.yml", func() {
		it.Before(func() {
			buildpackYMLParser.ParseVersionCall.Returns.Version = "1.2.3"
		})

		it("provides and requires dotnet core runtime", func() {
			result, err := detect(packit.DetectContext{
				WorkingDir: workingDir,
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Plan).To(Equal(packit.BuildPlan{
				Provides: []packit.BuildPlanProvision{
					{
						Name: "dotnet-runtime",
					},
				},
				Requires: []packit.BuildPlanRequirement{
					{
						Name: "dotnet-runtime",
						Metadata: map[string]interface{}{
							"version-source": "buildpack.yml",
							"version":        "1.2.3",
						},
					},
				},
			}))
		})
	})

	context("failure cases", func() {
		context("when the buildpack.yml parser fails", func() {
			it.Before(func() {
				buildpackYMLParser.ParseVersionCall.Returns.Err = errors.New("failed to parse buildpack.yml")
			})

			it("returns an error", func() {
				_, err := detect(packit.DetectContext{
					WorkingDir: "/working-dir",
				})
				Expect(err).To(MatchError("failed to parse buildpack.yml"))
			})
		})
	})
}
