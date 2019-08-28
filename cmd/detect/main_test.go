package main

import (
	"errors"
	"fmt"
	"github.com/BurntSushi/toml"
	"github.com/buildpack/libbuildpack/buildplan"
	"github.com/cloudfoundry/dotnet-core-runtime-cnb/runtime"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/cloudfoundry/libcfbuildpack/detect"
	"github.com/cloudfoundry/libcfbuildpack/test"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"
)

func TestUnitDetect(t *testing.T) {
	spec.Run(t, "Detect", testDetect, spec.Report(report.Terminal{}))
}

func testDetect(t *testing.T, when spec.G, it spec.S) {
	var factory *test.DetectFactory

	it.Before(func() {
		RegisterTestingT(t)
		factory = test.NewDetectFactory(t)
		fakeBuildpackToml := `
[[dependencies]]
id = "dotnet-runtime"
name = "Dotnet Runtime"
stacks = ["org.testeroni"]
uri = "some-uri"
version = "2.2.5"
`
		_, err := toml.Decode(fakeBuildpackToml, &factory.Detect.Buildpack.Metadata)
		Expect(err).ToNot(HaveOccurred())
		factory.Detect.Stack = "org.testeroni"
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

		it("passes when there is a valid runtimeconfig.json where the specified minor is provided", func() {
			runtimeConfigJSONPath := filepath.Join(factory.Detect.Application.Root, "appName.runtimeconfig.json")
			Expect(ioutil.WriteFile(runtimeConfigJSONPath, []byte(`
{
  "runtimeOptions": {
    "tfm": "netcoreapp2.2",
    "framework": {
      "name": "Microsoft.NETCore.App",
      "version": "2.2.0"
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

		it("passes when there is a valid runtimeconfig.json where the specified major is provided", func() {
			runtimeConfigJSONPath := filepath.Join(factory.Detect.Application.Root, "appName.runtimeconfig.json")
			Expect(ioutil.WriteFile(runtimeConfigJSONPath, []byte(`
{
  "runtimeOptions": {
    "tfm": "netcoreapp2.2",
    "framework": {
      "name": "Microsoft.NETCore.App",
      "version": "2.1.0"
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

		it("passes when there is a valid runtimeconfig.json where there are no valid roll forward versions available", func() {
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
			Expect(err).To(HaveOccurred())
			Expect(code).To(Equal(detect.FailStatusCode))
		})

		it("passes when there is a valid runtimeconfig.json where there are no runtime options meaning the app is a self contained deployment", func() {
			runtimeConfigJSONPath := filepath.Join(factory.Detect.Application.Root, "appName.runtimeconfig.json")
			Expect(ioutil.WriteFile(runtimeConfigJSONPath, []byte(`
{
 "runtimeOptions": {}
}
`), os.ModePerm)).To(Succeed())
			code, err := runDetect(factory.Detect)
			Expect(err).ToNot(HaveOccurred())
			Expect(code).To(Equal(detect.PassStatusCode))
			Expect(factory.Plans.Plan).To(Equal(buildplan.Plan{
				Provides: []buildplan.Provided{{Name: runtime.DotnetRuntime}},
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

		when("there is both a buildpack.yml and a runtimeconfig.json", func() {
			it("should error out when the major version of both the buildpack.yml and runtimeconfig.json don't match", func() {
				runtimeConfigJSONPath := filepath.Join(factory.Detect.Application.Root, "appName.runtimeconfig.json")
				Expect(ioutil.WriteFile(runtimeConfigJSONPath, []byte(`
{
 "runtimeOptions": {
   "tfm": "netcoreapp3.2",
   "framework": {
     "name": "Microsoft.NETCore.App",
     "version": "2.1.0"
   }
 }
}
`), os.ModePerm)).To(Succeed())
				test.WriteFile(t, filepath.Join(factory.Detect.Application.Root, "buildpack.yml"), fmt.Sprintf("dotnet-runtime:\n  version: %s", "3.1.2"))

				errorMessage := errors.New("major versions of runtimes do not match between buildpack.yml and runtimeconfig.json")
				code, err := runDetect(factory.Detect)
				Expect(err).To(HaveOccurred())
				Expect(code).To(Equal(detect.FailStatusCode))
				Expect(err).To(Equal(errorMessage))
			})

			it("should error out when the minor version of the buildpack.yml is less than than the minor version of the runtimeconfig.json", func() {
				runtimeConfigJSONPath := filepath.Join(factory.Detect.Application.Root, "appName.runtimeconfig.json")
				Expect(ioutil.WriteFile(runtimeConfigJSONPath, []byte(`
{
 "runtimeOptions": {
   "tfm": "netcoreapp3.2",
   "framework": {
     "name": "Microsoft.NETCore.App",
     "version": "2.2.0"
   }
 }
}
`), os.ModePerm)).To(Succeed())
				test.WriteFile(t, filepath.Join(factory.Detect.Application.Root, "buildpack.yml"), fmt.Sprintf("dotnet-runtime:\n  version: %s", "2.1.0"))

				errorMessage := errors.New("the minor version of the runtimeconfig.json is greater than the minor version of the buildpack.yml")
				code, err := runDetect(factory.Detect)
				Expect(err).To(HaveOccurred())
				Expect(code).To(Equal(detect.FailStatusCode))
				Expect(err).To(Equal(errorMessage))
			})

			it("roll forward the version found in the buildpack.yml", func() {
				runtimeConfigJSONPath := filepath.Join(factory.Detect.Application.Root, "appName.runtimeconfig.json")
				Expect(ioutil.WriteFile(runtimeConfigJSONPath, []byte(`
{
 "runtimeOptions": {
   "tfm": "netcoreapp3.2",
   "framework": {
     "name": "Microsoft.NETCore.App",
     "version": "2.1.0"
   }
 }
}
`), os.ModePerm)).To(Succeed())
				test.WriteFile(t, filepath.Join(factory.Detect.Application.Root, "buildpack.yml"), fmt.Sprintf("dotnet-runtime:\n  version: %s", "2.2.0"))

				fakeBuildpackToml := `
[[dependencies]]
id = "dotnet-runtime"
name = "Dotnet Runtime"
stacks = ["org.testeroni"]
uri = "some-uri"
version = "2.2.5"

[[dependencies]]
id = "dotnet-runtime"
name = "Dotnet Runtime"
stacks = ["org.testeroni"]
uri = "some-uri"
version = "2.1.4"
`
				_, err := toml.Decode(fakeBuildpackToml, &factory.Detect.Buildpack.Metadata)
				Expect(err).ToNot(HaveOccurred())
				factory.Detect.Stack = "org.testeroni"

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
