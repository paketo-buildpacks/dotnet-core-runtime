package runtime

import (
	"fmt"
	"os"

	"github.com/cloudfoundry/libcfbuildpack/buildpackplan"
	. "github.com/onsi/gomega"

	"path/filepath"
	"testing"

	"github.com/cloudfoundry/libcfbuildpack/test"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

func TestUnitDotnet(t *testing.T) {
	spec.Run(t, "Detect", testDotnet, spec.Report(report.Terminal{}))
}

func testDotnet(t *testing.T, when spec.G, it spec.S) {
	var (
		factory                  *test.BuildFactory
		stubDotnetRuntimeFixture = filepath.Join("testdata", "stub-dotnet-runtime.tar.xz")
	)

	it.Before(func() {
		RegisterTestingT(t)
		factory = test.NewBuildFactory(t)
		factory.AddDependencyWithVersion(DotnetRuntime, "2.2.5", stubDotnetRuntimeFixture)
	})

	when("runtime.NewContributor", func() {
		when("when there is no buildpack.yml", func() {
			it("returns true if a build plan exists and matching version is found", func() {
				factory.AddPlan(buildpackplan.Plan{Name: DotnetRuntime, Version: "2.2.5"})

				_, willContribute, err := NewContributor(factory.Build)
				Expect(err).NotTo(HaveOccurred())
				Expect(willContribute).To(BeTrue())
			})

			it("returns true if a build plan exists and no matching version is found", func() {
				factory.AddPlan(buildpackplan.Plan{Name: DotnetRuntime, Version: "1.0.0"})

				_, willContribute, err := NewContributor(factory.Build)
				Expect(err).To(HaveOccurred())
				Expect(willContribute).To(BeFalse())
			})
		})

		when("when there is a buildpack.yml", func() {
			it.Before(func() {
				factory.AddPlan(buildpackplan.Plan{Name: DotnetRuntime, Version: "2.1.0"})
				factory.AddDependencyWithVersion(DotnetRuntime, "2.1.5", stubDotnetRuntimeFixture)
				factory.AddDependencyWithVersion(DotnetRuntime, "2.2.2", stubDotnetRuntimeFixture)
			})

			it("that has a version range it returns that highest patch for that range", func() {
				test.WriteFile(t, filepath.Join(factory.Build.Application.Root, "buildpack.yml"), fmt.Sprintf("dotnet-framework:\n  version: %s", "2.2.*"))
				defer os.RemoveAll(filepath.Join(factory.Build.Application.Root, "buildpack.yml"))

				contributor, willContribute, err := NewContributor(factory.Build)
				Expect(err).NotTo(HaveOccurred())
				Expect(willContribute).To(BeTrue())
				Expect(contributor.runtimeLayer.Dependency.Version.String()).To(Equal("2.2.5"))
			})
			it("that has an exact version it only uses that exact version ", func() {
				test.WriteFile(t, filepath.Join(factory.Build.Application.Root, "buildpack.yml"), fmt.Sprintf("dotnet-framework:\n  version: %s", "2.2.2"))
				defer os.RemoveAll(filepath.Join(factory.Build.Application.Root, "buildpack.yml"))

				contributor, willContribute, err := NewContributor(factory.Build)
				Expect(err).NotTo(HaveOccurred())
				Expect(willContribute).To(BeTrue())
				Expect(contributor.runtimeLayer.Dependency.Version.String()).To(Equal("2.2.2"))

			})
		})

		it("returns false if a build plan does not exist", func() {
			contributor, willContribute, err := NewContributor(factory.Build)
			Expect(err).NotTo(HaveOccurred())
			Expect(willContribute).To(BeFalse())
			Expect(contributor).To(Equal(Contributor{}))
		})
	})

	when("Contribute", func() {
		it("writes default env vars, installs the runtime dependency", func() {
			factory.AddPlan(buildpackplan.Plan{Name: DotnetRuntime, Version: "2.2.5"})

			dotnetRuntimeContributor, _, err := NewContributor(factory.Build)
			Expect(err).NotTo(HaveOccurred())

			Expect(dotnetRuntimeContributor.Contribute()).To(Succeed())

			layer := factory.Build.Layers.Layer(DotnetRuntime)
			Expect(filepath.Join(layer.Root, "stub-dir", "stub.txt")).To(BeARegularFile())
			Expect(layer).To(test.HaveOverrideSharedEnvironment("DOTNET_ROOT", filepath.Join(layer.Root)))
			Expect(layer).To(test.HaveOverrideBuildEnvironment("RUNTIME_VERSION", "2.2.5"))
		})

		it("contributes dotnet runtime to the build layer when included in the build plan", func() {
			factory.AddPlan(buildpackplan.Plan{
				Name:    DotnetRuntime,
				Version: "2.2.5",
				Metadata: buildpackplan.Metadata{
					"build": true,
				},
			})

			dotnetRuntimeContributor, _, err := NewContributor(factory.Build)
			Expect(err).NotTo(HaveOccurred())

			Expect(dotnetRuntimeContributor.Contribute()).To(Succeed())

			layer := factory.Build.Layers.Layer(DotnetRuntime)
			Expect(layer).To(test.HaveLayerMetadata(true, false, false))
		})

		it("contributes dotnet runtime to the launch layer when included in the build plan", func() {
			factory.AddPlan(buildpackplan.Plan{
				Name:    DotnetRuntime,
				Version: "2.2.5",
				Metadata: buildpackplan.Metadata{
					"launch": true,
				},
			})

			dotnetRuntimeContributor, _, err := NewContributor(factory.Build)
			Expect(err).NotTo(HaveOccurred())

			Expect(dotnetRuntimeContributor.Contribute()).To(Succeed())

			layer := factory.Build.Layers.Layer(DotnetRuntime)
			Expect(layer).To(test.HaveLayerMetadata(false, false, true))
		})
	})
}
