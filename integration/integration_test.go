package integration

import (
	"path/filepath"
	"testing"

	"github.com/cloudfoundry/dagger"

	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	. "github.com/onsi/gomega"
)

var (
	bp string
)

func TestIntegration(t *testing.T) {
	RegisterTestingT(t)
	root, err := dagger.FindBPRoot()
	Expect(err).ToNot(HaveOccurred())
	bp, err = dagger.PackageBuildpack(root)
	Expect(err).NotTo(HaveOccurred())
	defer func() {
		dagger.DeleteBuildpack(bp)
	}()

	spec.Run(t, "Integration", testIntegration, spec.Report(report.Terminal{}))
}

func testIntegration(t *testing.T, _ spec.G, it spec.S) {
	var (
		Expect func(interface{}, ...interface{}) Assertion
		app *dagger.App
		err error
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
		app, err = dagger.PackBuild(filepath.Join("testdata", "simple_app"), bp)
		Expect(err).ToNot(HaveOccurred())
		app.Memory = "128m"
		Expect(app.StartWithCommand("./source_code")).To(Succeed())

		body, _, err := app.HTTPGet("/")
		Expect(err).NotTo(HaveOccurred())
		Expect(body).To(ContainSubstring("Hello world!"))
	})

	it("runs a simple framework-dependent deployment with a framework-dependent executable that has a buildpack.yml in it", func() {
		app, err = dagger.PackBuild(filepath.Join("testdata", "simple_app_with_buildpack_yml"), bp)
		Expect(err).ToNot(HaveOccurred())
		app.Memory = "128m"
		Expect(app.StartWithCommand("./source_code")).To(Succeed())

		Expect(app.BuildLogs()).To(ContainSubstring("dotnet-runtime.2.1"))

		body, _, err := app.HTTPGet("/")
		Expect(err).NotTo(HaveOccurred())
		Expect(body).To(ContainSubstring("Hello world!"))
	})
}
