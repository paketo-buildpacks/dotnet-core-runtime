package runtime

import (
	"github.com/cloudfoundry/libcfbuildpack/buildpackplan"
	. "github.com/onsi/gomega"

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
		factory.AddDependency(DotnetRuntime, stubDotnetRuntimeFixture)

	})

	when("runtime.NewContributor", func() {
		it("returns true if a build plan exists", func() {
			factory.AddPlan(buildpackplan.Plan{Name: DotnetRuntime})

			_, willContribute, err := NewContributor(factory.Build)
			Expect(err).NotTo(HaveOccurred())
			Expect(willContribute).To(BeTrue())
		})

		it("returns false if a build plan does not exist", func() {
			_, willContribute, err := NewContributor(factory.Build)
			Expect(err).NotTo(HaveOccurred())
			Expect(willContribute).To(BeFalse())
		})
	})

	when("Contribute", func() {
		it("writes default env vars, installs the runtime dependency", func() {
			factory.AddPlan(buildpackplan.Plan{Name: DotnetRuntime})

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
