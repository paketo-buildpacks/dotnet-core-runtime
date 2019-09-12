package runtime

import (
	"fmt"
	"github.com/cloudfoundry/libcfbuildpack/buildpackplan"
	. "github.com/onsi/gomega"
	"os"

	"github.com/cloudfoundry/libcfbuildpack/test"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
	"path/filepath"
	"testing"
)

func TestUnitDotnet(t *testing.T) {
	spec.Run(t, "Detect", testDotnet, spec.Report(report.Terminal{}))
}

func testDotnet(t *testing.T, when spec.G, it spec.S) {
	var (
		factory     *test.BuildFactory
		stubDotnetRuntimeFixture = filepath.Join("testdata", "stub-dotnet-runtime.tar.xz")

	)

	it.Before(func() {
		RegisterTestingT(t)
		factory = test.NewBuildFactory(t)
		factory.AddDependencyWithVersion(DotnetRuntime, "2.2.5", stubDotnetRuntimeFixture)
	})

	when("runtime.NewContributor", func() {
		when("when there is no buildpack.yml", func () {
			it("returns true if a build plan exists and matching version is found", func() {
				factory.AddPlan(buildpackplan.Plan{Name: DotnetRuntime, Version: "2.2.5"})

				contributor, willContribute, err := NewContributor(factory.Build)
				Expect(err).NotTo(HaveOccurred())
				Expect(willContribute).To(BeTrue())
				Expect(contributor.runtimeLayer.Dependency.Version.String()).To(Equal("2.2.5"))
			})

			it("returns true if a build plan exists and matching minor is found", func() {
				factory.AddPlan(buildpackplan.Plan{Name: DotnetRuntime, Version: "2.2.0"})

				contributor, willContribute, err := NewContributor(factory.Build)
				Expect(err).NotTo(HaveOccurred())
				Expect(willContribute).To(BeTrue())
				Expect(contributor.runtimeLayer.Dependency.Version.String()).To(Equal("2.2.5"))
			})

			it("returns true if a build plan exists and matching major is found", func() {
				factory.AddPlan(buildpackplan.Plan{Name: DotnetRuntime, Version: "2.1.0"})

				contributor, willContribute, err := NewContributor(factory.Build)
				Expect(err).NotTo(HaveOccurred())
				Expect(willContribute).To(BeTrue())
				Expect(contributor.runtimeLayer.Dependency.Version.String()).To(Equal("2.2.5"))
			})

			it("returns true if a build plan exists and no valid roll forward version is found", func() {
				factory.AddPlan(buildpackplan.Plan{Name: DotnetRuntime, Version: "1.0.0"})

				contributor, willContribute, err := NewContributor(factory.Build)
				Expect(err).To(HaveOccurred())
				Expect(willContribute).To(BeFalse())
				Expect(contributor).To(Equal(Contributor{}))
			})
		})

		when("when there is a buildpack.yml", func () {
			it("returns roll forward from buildpack.yml", func() {
				factory.AddPlan(buildpackplan.Plan{Name: DotnetRuntime, Version: "2.0.0"})
				factory.AddDependencyWithVersion(DotnetRuntime, "2.1.0", stubDotnetRuntimeFixture)
				test.WriteFile(t, filepath.Join(factory.Build.Application.Root, "buildpack.yml"), fmt.Sprintf("dotnet-runtime:\n  version: %s", "2.2.0"))
				defer os.RemoveAll(filepath.Join(factory.Build.Application.Root, "buildpack.yml"))

				contributor, willContribute, err := NewContributor(factory.Build)
				Expect(err).NotTo(HaveOccurred())
				Expect(willContribute).To(BeTrue())
				Expect(contributor.runtimeLayer.Dependency.Version.String()).To(Equal("2.2.5"))
			})

			it("returns false if plan version and buildpack.yml version have different majors", func() {
				factory.AddPlan(buildpackplan.Plan{Name: DotnetRuntime, Version: "2.0.0"})
				test.WriteFile(t, filepath.Join(factory.Build.Application.Root, "buildpack.yml"), fmt.Sprintf("dotnet-runtime:\n  version: %s", "3.2.0"))
				defer os.RemoveAll(filepath.Join(factory.Build.Application.Root, "buildpack.yml"))

				contributor, willContribute, err := NewContributor(factory.Build)
				Expect(err).To(HaveOccurred())
				Expect(willContribute).To(BeFalse())
				Expect(contributor).To(Equal(Contributor{}))

			})

			it("returns false if plan version minor is greater than and buildpack.yml version minor", func() {
				factory.AddPlan(buildpackplan.Plan{Name: DotnetRuntime, Version: "2.3.0"})
				test.WriteFile(t, filepath.Join(factory.Build.Application.Root, "buildpack.yml"), fmt.Sprintf("dotnet-runtime:\n  version: %s", "2.2.0"))
				defer os.RemoveAll(filepath.Join(factory.Build.Application.Root, "buildpack.yml"))

				contributor, willContribute, err := NewContributor(factory.Build)
				Expect(err).To(HaveOccurred())
				Expect(willContribute).To(BeFalse())
				Expect(contributor).To(Equal(Contributor{}))

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
		})

		it("uses the default version when a version is not requested", func() {
			factory.AddDependencyWithVersion(DotnetRuntime, "0.9", filepath.Join("testdata", "stub-dotnet-runtime-default.tar.xz"))
			factory.SetDefaultVersion(DotnetRuntime, "0.9")
			factory.AddPlan(buildpackplan.Plan{Name: DotnetRuntime})

			dotnetRuntimeContributor, _, err := NewContributor(factory.Build)
			Expect(err).NotTo(HaveOccurred())

			Expect(dotnetRuntimeContributor.Contribute()).To(Succeed())
			layer := factory.Build.Layers.Layer(DotnetRuntime)
			Expect(layer).To(test.HaveLayerVersion("0.9"))
		})

		it("contributes dotnet runtime to the build layer when included in the build plan", func() {
			factory.AddPlan(buildpackplan.Plan{
				Name: DotnetRuntime,
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
				Name: DotnetRuntime,
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

		it("returns an error when unsupported version of dotnet runtime is included in the build plan", func() {
			factory.AddPlan(buildpackplan.Plan{
				Name:    DotnetRuntime,
				Version: "9000.0.0",
				Metadata: buildpackplan.Metadata{
					"launch": true,
				},
			})

			_, shouldContribute, err := NewContributor(factory.Build)
			Expect(err).To(HaveOccurred())
			Expect(shouldContribute).To(BeFalse())
		})

	})
}
