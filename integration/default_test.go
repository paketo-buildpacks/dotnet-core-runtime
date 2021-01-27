package integration_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/paketo-buildpacks/occam"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
	. "github.com/paketo-buildpacks/occam/matchers"
)

func testDefault(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect     = NewWithT(t).Expect
		Eventually = NewWithT(t).Eventually
		pack       occam.Pack
		docker     occam.Docker
	)

	it.Before(func() {
		pack = occam.NewPack()
		docker = occam.NewDocker()
	})

	context("when building a container with dotnet-runtime", func() {
		var (
			image     occam.Image
			container occam.Container
			name      string
			source    string
			err       error
		)

		it.Before(func() {
			name, err = occam.RandomName()
			Expect(err).NotTo(HaveOccurred())
		})

		it.After(func() {
			Expect(docker.Container.Remove.Execute(container.ID)).To(Succeed())
			Expect(docker.Image.Remove.Execute(image.ID)).To(Succeed())
			Expect(docker.Volume.Remove.Execute(occam.CacheVolumeNames(name))).To(Succeed())
			Expect(os.RemoveAll(source)).To(Succeed())
		})

		it("installs the dotnet runtime into a layer", func() {
			source, err = occam.Source(filepath.Join("testdata", "default"))
			Expect(err).NotTo(HaveOccurred())

			var logs fmt.Stringer
			image, logs, err = pack.WithNoColor().Build.
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
				"      <unknown> -> \"*\"",
				"",
				MatchRegexp(`    Selected dotnet-runtime version \(using <unknown>\): \d+\.\d+\.\d+`),
				"",
				"  Executing build process",
				MatchRegexp(`    Installing Dotnet Core Runtime \d+\.\d+\.\d+`),
				MatchRegexp(`      Completed in ([0-9]*(\.[0-9]*)?[a-z]+)+`),
				"",
				"  Configuring environment for build and launch",
				`    DOTNET_ROOT -> "/workspace/.dotnet_root"`,
				"",
				"  Configuring environment for build",
				MatchRegexp(`    RUNTIME_VERSION -> "\d+\.\d+\.\d+"`),
			))

			container, err = docker.Container.Run.
				WithCommand("ls -al $DOTNET_ROOT && ls -al $DOTNET_ROOT/shared").
				Execute(image.ID)
			Expect(err).NotTo(HaveOccurred())

			Eventually(func() string {
				cLogs, err := docker.Container.Logs.Execute(container.ID)
				Expect(err).NotTo(HaveOccurred())
				return cLogs.String()
			}).Should(
				And(
					MatchRegexp(fmt.Sprintf(`lrwxrwxrwx \d+ cnb cnb   \d+ .* host -> \/layers\/%s\/dotnet-core-runtime\/host`, strings.ReplaceAll(settings.BuildpackInfo.Buildpack.ID, "/", "_"))),
					MatchRegexp(`drwxr-xr-x \d+ cnb cnb \d+ .* shared`),
					MatchRegexp(fmt.Sprintf(`lrwxrwxrwx \d+ cnb cnb   \d+ .* Microsoft.NETCore.App -> \/layers\/%s\/dotnet-core-runtime\/shared\/Microsoft.NETCore.App`, strings.ReplaceAll(settings.BuildpackInfo.Buildpack.ID, "/", "_"))),
				),
			)
		})
	})

	context("when the app contains a buildpack.yml that specifies an unsupported version ", func() {
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

		it("fails to build instead of rolling the version", func() {
			source, err = occam.Source(filepath.Join("testdata", "with_buildpack_yml"))
			Expect(err).NotTo(HaveOccurred())

			var logs fmt.Stringer
			_, logs, err = pack.WithNoColor().Build.
				WithPullPolicy("never").
				WithBuildpacks(
					settings.Buildpacks.DotnetCoreRuntime.Online,
					settings.Buildpacks.BuildPlan.Online,
				).
				Execute(name, source)
			Expect(err).To(HaveOccurred(), logs.String())

			Expect(logs).To(ContainLines(
				MatchRegexp(fmt.Sprintf(`%s \d+\.\d+\.\d+`, settings.BuildpackInfo.Buildpack.Name)),
				"  Resolving Dotnet Core Runtime version",
				"    Candidate version sources (in priority order):",
				"      buildpack.yml -> \"2.0.0\"",
				"      <unknown>     -> \"*\"",
				"",
			))
			Expect(logs).To(ContainLines(MatchRegexp(`failed to satisfy "dotnet-runtime" dependency for stack "io.buildpacks.stacks.bionic" with version constraint "2.0.0": no compatible versions. Supported versions are: \[(\d+\.\d+\.\d+(, )?)*\]`)))
		})
	})
}
