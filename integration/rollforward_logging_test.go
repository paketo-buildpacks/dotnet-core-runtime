package integration_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/paketo-buildpacks/occam"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
	. "github.com/paketo-buildpacks/occam/matchers"
)

func testRollForwardLogging(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect
		pack   occam.Pack
		docker occam.Docker
	)

	it.Before(func() {
		pack = occam.NewPack()
		docker = occam.NewDocker()
	})

	context("the buildpack is run with pack build and rolls forward runtime version", func() {
		var (
			name   string
			source string
			err    error
		)

		it.Before(func() {
			name, err = occam.RandomName()
			Expect(err).NotTo(HaveOccurred())
		})

		it.After(func() {
			Expect(docker.Volume.Remove.Execute(occam.CacheVolumeNames(name))).To(Succeed())
			Expect(os.RemoveAll(source)).To(Succeed())
		})

		context("when version requested does not have an exact match", func() {
			it.Before(func() {
				source, err = occam.Source(filepath.Join("testdata", "rollforward"))
			})
			it("logs useful information about rolling forward the version", func() {
				Expect(err).NotTo(HaveOccurred())

				var logs fmt.Stringer
				_, logs, err = pack.WithNoColor().Build.
					WithPullPolicy("never").
					WithBuildpacks(
						settings.Buildpacks.DotnetCoreRuntime.Online,
						settings.Buildpacks.BuildPlan.Online,
					).
					Execute(name, source)
				Expect(err).NotTo(HaveOccurred(), logs.String())

				Expect(logs).To(ContainLines(
					MatchRegexp(fmt.Sprintf(`%s \d+\.\d+\.\d+`, settings.BuildpackInfo.Buildpack.Name)),
					"  Resolving Dotnet Core Runtime version",
					"    Candidate version sources (in priority order):",
					"      runtimeconfig.json -> \"3.0.0\"",
					"",
					"    No exact version match found; attempting version roll-forward",
					"",
					MatchRegexp(`    Selected Dotnet Core Runtime version \(using runtimeconfig.json\): \d+\.\d+\.\d+`),
					"",
					"  Executing build process",
					MatchRegexp(`    Installing Dotnet Core Runtime \d+\.\d+\.\d+`),
					MatchRegexp(`      Completed in ([0-9]*(\.[0-9]*)?[a-z]+)+`),
					"",
					"  Configuring build environment",
					`    DOTNET_ROOT     -> "/workspace/.dotnet_root"`,
					MatchRegexp(`    RUNTIME_VERSION -> "\d+\.\d+\.\d+"`),
					"",
					"  Configuring launch environment",
					`    DOTNET_ROOT -> "/workspace/.dotnet_root"`,
				))
			})
		})

		context("when version requested in a runtimeconfig.json has an exact match", func() {
			var availableVersion string
			it.Before(func() {
				source, err = occam.Source(filepath.Join("testdata", "rollforward"))
				availableVersion = settings.BuildpackInfo.Metadata.Dependencies[0].Version
				err = os.WriteFile(filepath.Join(source, "plan.toml"), []byte(fmt.Sprintf(`[[requires]]
			name = "dotnet-runtime"

				[requires.metadata]
					launch = true
					version-source = "runtimeconfig.json"
					version = "%s"
			`, availableVersion)), os.ModePerm)
				Expect(err).NotTo(HaveOccurred())
			})
			it("does not log about rolling forward the version", func() {
				Expect(err).NotTo(HaveOccurred())

				var logs fmt.Stringer
				_, logs, err = pack.WithNoColor().Build.
					WithPullPolicy("never").
					WithBuildpacks(
						settings.Buildpacks.DotnetCoreRuntime.Online,
						settings.Buildpacks.BuildPlan.Online,
					).
					Execute(name, source)
				Expect(err).NotTo(HaveOccurred(), logs.String())

				Expect(logs).To(ContainLines(
					MatchRegexp(fmt.Sprintf(`    Selected Dotnet Core Runtime version \(using runtimeconfig.json\): %s`, availableVersion)),
				))
				Expect(logs).NotTo(ContainSubstring("No exact version match found; attempting version roll-forward"))
			})
		})
	})

}
