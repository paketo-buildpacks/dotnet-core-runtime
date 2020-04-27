package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/buildpack/libbuildpack/buildplan"
	"github.com/cloudfoundry/libcfbuildpack/detect"
	"github.com/cloudfoundry/libcfbuildpack/test"
	"github.com/paketo-buildpacks/dotnet-core-runtime/runtime"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	. "github.com/onsi/gomega"
)

func TestUnitDetect(t *testing.T) {
	spec.Run(t, "Detect", testDetect, spec.Report(report.Terminal{}))
}

func testDetect(t *testing.T, when spec.G, it spec.S) {
	var factory *test.DetectFactory

	it.Before(func() {
		RegisterTestingT(t)
		factory = test.NewDetectFactory(t)
	})

	when("app has a FDE", func() {
		it.Before(func() {
			Expect(ioutil.WriteFile(filepath.Join(factory.Detect.Application.Root, "appName"), []byte(`fake exe`), os.ModePerm)).To(Succeed())
		})
		it("passes when there is a valid runtimeconfig.json where the specified version is provided", func() {
			runtimeConfigJSONPath := filepath.Join(factory.Detect.Application.Root, "appName.runtimeconfig.json")
			Expect(ioutil.WriteFile(runtimeConfigJSONPath, []byte(`
{
  "runtimeOptions": {
    "tfm": "netcoreapp2.2",
    "framework": {
      "name": "Microsoft.NETCore.App",
      "version": "2.2.5"
    }
  }
}
`), os.ModePerm)).To(Succeed())
			code, err := runDetect(factory.Detect)
			Expect(err).ToNot(HaveOccurred())
			Expect(code).To(Equal(detect.PassStatusCode))
			Expect(factory.Plans.Plan).To(Equal(buildplan.Plan{
				Provides: []buildplan.Provided{{Name: runtime.DotnetRuntime}},
				Requires: []buildplan.Required{{
					Name:     runtime.DotnetRuntime,
					Version:  "2.2.5",
					Metadata: buildplan.Metadata{"launch": true},
				}},
			}))
		})

		it("passes when there is a valid runtimeconfig.json where the only runtime options are for ASPNet", func() {
			runtimeConfigJSONPath := filepath.Join(factory.Detect.Application.Root, "appName.runtimeconfig.json")
			Expect(ioutil.WriteFile(runtimeConfigJSONPath, []byte(`
{
   "runtimeOptions": {
    "tfm": "netcoreapp2.2",
    "framework": {
      "name": "Microsoft.AspNetCore.App",
      "version": "1.1.0"
    }
  }
}
`), os.ModePerm)).To(Succeed())
			code, err := runDetect(factory.Detect)
			Expect(err).ToNot(HaveOccurred())
			Expect(code).To(Equal(detect.PassStatusCode))
			Expect(factory.Plans.Plan).To(Equal(buildplan.Plan{
				Provides: []buildplan.Provided{{Name: runtime.DotnetRuntime}},
			}))
		})

		it("passes when there is no valid runtimeconfig.json meaning that app is source based", func() {
			code, err := runDetect(factory.Detect)
			Expect(err).ToNot(HaveOccurred())
			Expect(code).To(Equal(detect.PassStatusCode))
			Expect(factory.Plans.Plan).To(Equal(buildplan.Plan{
				Provides: []buildplan.Provided{{Name: runtime.DotnetRuntime}},
			}))
		})
	})

	when("the app does not have a FDE", func() {
		it("passes when there is a valid runtimeconfig.json", func() {
			runtimeConfigJSONPath := filepath.Join(factory.Detect.Application.Root, "appName.runtimeconfig.json")
			Expect(ioutil.WriteFile(runtimeConfigJSONPath, []byte(`
{
   "runtimeOptions": {
    "tfm": "netcoreapp2.2",
    "framework": {
      "name": "Microsoft.NETCore.App",
      "version": "1.1.0"
    }
  }
}
`), os.ModePerm)).To(Succeed())
			code, err := runDetect(factory.Detect)
			Expect(err).ToNot(HaveOccurred())
			Expect(code).To(Equal(detect.PassStatusCode))
			Expect(factory.Plans.Plan).To(Equal(buildplan.Plan{
				Provides: []buildplan.Provided{{Name: runtime.DotnetRuntime}},
			}))
		})

	})
}
