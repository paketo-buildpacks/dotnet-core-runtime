package integration_test

import (
	"fmt"
	"io/ioutil"
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
		Expect                = NewWithT(t).Expect
		pack                  occam.Pack
		docker                occam.Docker
		runtimeConfigTemplate string
	)

	it.Before(func() {
		pack = occam.NewPack()
		docker = occam.NewDocker()

		runtimeConfigTemplate = `
{
  "runtimeOptions": {
    "tfm": "netcoreapp2.1",
    "framework": {
      "name": "Microsoft.NETCore.App",
			"version": "%s"
    }
  }
}`
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
				testVersion := "2.0.0"
				source, err = occam.Source(filepath.Join("testdata", "rollforward"))
				ioutil.WriteFile(filepath.Join(source, "app.runtimeconfig.json"), []byte(fmt.Sprintf(runtimeConfigTemplate, testVersion)), 0600)
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
					"      runtimeconfig.json -> \"2.0.0\"",
					"",
					"    No exact version match found; attempting version roll-forward",
					"",
					MatchRegexp(`    Selected dotnet-runtime version \(using runtimeconfig.json\): \d+\.\d+\.\d+`),
					"",
					"  Executing build process",
					MatchRegexp(`    Installing Dotnet Core Runtime \d+\.\d+\.\d+`),
					MatchRegexp(`      Completed in ([0-9]*(\.[0-9]*)?[a-z]+)+`),
					"",
					"  Configuring environment",
					`    DOTNET_ROOT -> "/workspace/.dotnet_root"`,
					"",
					MatchRegexp(`    RUNTIME_VERSION -> "\d+\.\d+\.\d+"`),
				))
			})
		})
	})

}