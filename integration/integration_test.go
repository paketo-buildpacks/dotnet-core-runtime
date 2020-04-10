package integration

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/Masterminds/semver"
	"github.com/cloudfoundry/dagger"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	. "github.com/onsi/gomega"
)

var (
	runtimeURI, builder string
	bpList              []string
)

const testBuildpack = "test-buildpack"

func BeforeSuite() {
	root, err := dagger.FindBPRoot()
	Expect(err).ToNot(HaveOccurred())
	runtimeURI, err = dagger.PackageBuildpack(root)
	Expect(err).NotTo(HaveOccurred())

	config, err := dagger.ParseConfig("config.json")
	Expect(err).NotTo(HaveOccurred())

	builder = config.Builder

	for _, bp := range config.BuildpackOrder[builder] {
		var bpURI string
		if bp == testBuildpack {
			bpList = append(bpList, runtimeURI)
			continue
		}
		bpURI, err = dagger.GetLatestBuildpack(bp)
		Expect(err).NotTo(HaveOccurred())
		bpList = append(bpList, bpURI)
	}
}

func AfterSuite() {
	for _, bp := range bpList {
		Expect(dagger.DeleteBuildpack(bp)).To(Succeed())
	}
}
func TestIntegration(t *testing.T) {
	RegisterTestingT(t)
	BeforeSuite()
	spec.Run(t, "Integration", testIntegration, spec.Report(report.Terminal{}))
	AfterSuite()
}

func testIntegration(t *testing.T, _ spec.G, it spec.S) {
	var (
		Expect func(interface{}, ...interface{}) Assertion
		app    *dagger.App
		err    error
	)
	it.Before(func() {
		Expect = NewWithT(t).Expect
	})
	it.After(func() {
		if app != nil {
			app.Destroy()
		}
	})

	it("runs a simple framework-dependent deployment with a framework-dependent executable", func() {
		app, err = dagger.NewPack(
			filepath.Join("testdata", "simple_app"),
			dagger.RandomImage(),
			dagger.SetBuildpacks(bpList...),
			dagger.SetBuilder(builder),
		).Build()
		Expect(err).ToNot(HaveOccurred())
		app.Memory = "128m"

		if builder == "bionic" {
			app.SetHealthCheck("stat /workspace", "2s", "15s")
		}

		Expect(app.StartWithCommand("./source_code")).To(Succeed())

		body, _, err := app.HTTPGet("/")
		Expect(err).NotTo(HaveOccurred())
		Expect(body).To(ContainSubstring("Hello world!"))
	})

	it("runs a simple framework-dependent deployment with a framework-dependent executable that has a buildpack.yml in it", func() {
		majorMinor := "2.1"
		version, err := GetLowestRuntimeVersionInMajorMinor(majorMinor, filepath.Join("..", "buildpack.toml"))
		Expect(err).ToNot(HaveOccurred())
		bpYml := fmt.Sprintf(`---
dotnet-framework:
  version: "%s"
`, version)

		bpYmlPath := filepath.Join("testdata", "simple_app_with_buildpack_yml", "buildpack.yml")
		Expect(ioutil.WriteFile(bpYmlPath, []byte(bpYml), 0644)).To(Succeed())

		app, err = dagger.NewPack(
			filepath.Join("testdata", "simple_app_with_buildpack_yml"),
			dagger.RandomImage(),
			dagger.SetBuildpacks(bpList...),
			dagger.SetBuilder(builder),
		).Build()
		Expect(err).ToNot(HaveOccurred())
		app.Memory = "128m"

		if builder == "bionic" {
			app.SetHealthCheck("stat /workspace", "2s", "15s")
		}

		Expect(app.StartWithCommand("./source_code")).To(Succeed())

		Expect(app.BuildLogs()).To(ContainSubstring(fmt.Sprintf("dotnet-runtime.%s", version)))

		body, _, err := app.HTTPGet("/")
		Expect(err).NotTo(HaveOccurred())
		Expect(body).To(ContainSubstring("Hello world!"))
	})
}

func GetLowestRuntimeVersionInMajorMinor(majorMinor, bpTomlPath string) (string, error) {
	type buildpackTomlVersion struct {
		Metadata struct {
			Dependencies []struct {
				Version string `toml:"version"`
			} `toml:"dependencies"`
		} `toml:"metadata"`
	}

	bpToml := buildpackTomlVersion{}
	output, err := ioutil.ReadFile(filepath.Join(bpTomlPath))
	if err != nil {
		return "", err
	}

	majorMinorConstraint, err := semver.NewConstraint(fmt.Sprintf("%s.*", majorMinor))
	if err != nil {
		return "", err
	}

	lowestVersion, err := semver.NewVersion(fmt.Sprintf("%s.9999", majorMinor))
	if err != nil {
		return "", err
	}

	_, err = toml.Decode(string(output), &bpToml)
	if err != nil {
		return "", err
	}

	for _, dep := range bpToml.Metadata.Dependencies {
		depVersion, err := semver.NewVersion(dep.Version)
		if err != nil {
			return "", err
		}
		if majorMinorConstraint.Check(depVersion) {
			if depVersion.LessThan(lowestVersion) {
				lowestVersion = depVersion
			}
		}
	}

	return lowestVersion.String(), nil
}
