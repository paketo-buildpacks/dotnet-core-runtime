package dotnetcoreruntime_test

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	dotnetcoreruntime "github.com/paketo-buildpacks/dotnet-core-runtime"
	"github.com/paketo-buildpacks/packit"
	"github.com/paketo-buildpacks/packit/postal"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testRuntimeVersionResolver(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		buffer          *bytes.Buffer
		logEmitter      dotnetcoreruntime.LogEmitter
		cnbDir          string
		versionResolver dotnetcoreruntime.RuntimeVersionResolver
		entry           packit.BuildpackPlanEntry
	)

	it.Before(func() {
		var err error

		buffer = bytes.NewBuffer(nil)
		logEmitter = dotnetcoreruntime.NewLogEmitter(buffer)

		versionResolver = dotnetcoreruntime.NewRuntimeVersionResolver(logEmitter)
		cnbDir, err = ioutil.TempDir("", "cnb")
		Expect(err).NotTo(HaveOccurred())

		err = ioutil.WriteFile(filepath.Join(cnbDir, "buildpack.toml"), []byte(`api = "0.2"
[buildpack]
  id = "org.some-org.some-buildpack"
  name = "Some Buildpack"
  version = "some-version"

[metadata]

  [[metadata.dependencies]]
    id = "dotnet-runtime"
    sha256 = "some-sha"
    stacks = ["some-stack"]
    uri = "some-uri"
		version = "1.2.3"

  [[metadata.dependencies]]
    id = "dotnet-runtime"
    sha256 = "some-sha"
    stacks = ["some-stack"]
    uri = "some-uri"
		version = "1.2.4"
`), 0600)
		Expect(err).NotTo(HaveOccurred())

		entry = packit.BuildpackPlanEntry{
			Name: "dotnet-runtime",
			Metadata: map[string]interface{}{
				"version-source": "runtimeconfig.json",
				"launch":         true,
			},
		}
	})

	it.After(func() {
		Expect(os.RemoveAll(cnbDir)).To(Succeed())
	})

	context("the buildpack.toml has the exact version", func() {
		it.Before(func() {
			entry.Metadata["version"] = "1.2.3"
		})
		it("returns a dependency with that version", func() {
			dependency, err := versionResolver.Resolve(filepath.Join(cnbDir, "buildpack.toml"), entry, "some-stack")
			Expect(err).NotTo(HaveOccurred())

			Expect(dependency).To(Equal(postal.Dependency{
				ID:      "dotnet-runtime",
				Version: "1.2.3",
				URI:     "some-uri",
				SHA256:  "some-sha",
				Stacks:  []string{"some-stack"},
			}))
		})
	})

	context("the buildpack.toml only has a major minor version match", func() {
		it.Before(func() {
			entry.Metadata["version"] = "1.2.0"
		})
		it("returns a compatible version", func() {
			dependency, err := versionResolver.Resolve(filepath.Join(cnbDir, "buildpack.toml"), entry, "some-stack")
			Expect(err).NotTo(HaveOccurred())

			Expect(dependency).To(Equal(postal.Dependency{
				ID:      "dotnet-runtime",
				Version: "1.2.4",
				URI:     "some-uri",
				SHA256:  "some-sha",
				Stacks:  []string{"some-stack"},
			}))
		})
	})

	context("the buildpack.toml only has a major version match", func() {
		it.Before(func() {
			entry.Metadata["version"] = "1.1.7"
		})
		it("returns a compatible version", func() {
			dependency, err := versionResolver.Resolve(filepath.Join(cnbDir, "buildpack.toml"), entry, "some-stack")
			Expect(err).NotTo(HaveOccurred())

			Expect(dependency).To(Equal(postal.Dependency{
				ID:      "dotnet-runtime",
				Version: "1.2.4",
				URI:     "some-uri",
				SHA256:  "some-sha",
				Stacks:  []string{"some-stack"},
			}))
		})
	})

	context("the buildpack.toml does not have a version match", func() {
		context("the requested version is a major version higher", func() {
			it.Before(func() {
				entry.Metadata["version"] = "2.0.0"
			})
			it("returns an error", func() {
				_, err := versionResolver.Resolve(filepath.Join(cnbDir, "buildpack.toml"), entry, "some-stack")
				Expect(err).To(MatchError(ContainSubstring("failed to satisfy \"dotnet-runtime\" dependency for stack \"some-stack\" with version constraint \"2.0.0\": no compatible versions. Supported versions are: [1.2.3, 1.2.4]")))
			})
		})

		context("the requested version is a minor version higher", func() {
			it.Before(func() {
				entry.Metadata["version"] = "1.3.0"
			})
			it("returns an error", func() {
				_, err := versionResolver.Resolve(filepath.Join(cnbDir, "buildpack.toml"), entry, "some-stack")
				Expect(err).To(MatchError(ContainSubstring("failed to satisfy \"dotnet-runtime\" dependency for stack \"some-stack\" with version constraint \"1.3.0\": no compatible versions. Supported versions are: [1.2.3, 1.2.4]")))
			})
		})

		context("the requested version is a patch version higher", func() {
			it.Before(func() {
				entry.Metadata["version"] = "1.2.5"
			})
			it("returns an error", func() {
				_, err := versionResolver.Resolve(filepath.Join(cnbDir, "buildpack.toml"), entry, "some-stack")
				Expect(err).To(MatchError(ContainSubstring("failed to satisfy \"dotnet-runtime\" dependency for stack \"some-stack\" with version constraint \"1.2.5\": no compatible versions. Supported versions are: [1.2.3, 1.2.4]")))
			})
		})
	})

	context("the buildpack.toml does not have a dependency with a matching ID", func() {
		it.Before(func() {
			entry.Metadata["version"] = "1.2.3"
			entry.Name = "random-ID"
		})
		it("returns an error", func() {
			_, err := versionResolver.Resolve(filepath.Join(cnbDir, "buildpack.toml"), entry, "some-stack")
			Expect(err).To(MatchError(ContainSubstring("failed to satisfy \"random-ID\" dependency for stack \"some-stack\" with version constraint \"1.2.3\": no compatible versions. Supported versions are: []")))
		})
	})

	context("the buildpack.toml does not have a dependency with a matching stack", func() {
		it.Before(func() {
			entry.Metadata["version"] = "1.2.3"
		})
		it("returns an error", func() {
			_, err := versionResolver.Resolve(filepath.Join(cnbDir, "buildpack.toml"), entry, "random-stack")
			Expect(err).To(MatchError(ContainSubstring("failed to satisfy \"dotnet-runtime\" dependency for stack \"random-stack\" with version constraint \"1.2.3\": no compatible versions. Supported versions are: []")))
		})
	})

	context("the version is not a valid semver version", func() {
		it.Before(func() {
			entry.Metadata["version"] = "1.2.*"
		})
		it("attempts to turn the given versions into the only constraint", func() {
			dependency, err := versionResolver.Resolve(filepath.Join(cnbDir, "buildpack.toml"), entry, "some-stack")
			Expect(err).NotTo(HaveOccurred())

			Expect(dependency).To(Equal(postal.Dependency{
				ID:      "dotnet-runtime",
				Version: "1.2.4",
				URI:     "some-uri",
				SHA256:  "some-sha",
				Stacks:  []string{"some-stack"},
			}))
		})
	})

	context("the version is empty", func() {
		it("returns the latest version", func() {
			dependency, err := versionResolver.Resolve(filepath.Join(cnbDir, "buildpack.toml"), entry, "some-stack")
			Expect(err).NotTo(HaveOccurred())

			Expect(dependency).To(Equal(postal.Dependency{
				ID:      "dotnet-runtime",
				Version: "1.2.4",
				URI:     "some-uri",
				SHA256:  "some-sha",
				Stacks:  []string{"some-stack"},
			}))
		})
	})

	context("the version source is empty", func() {
		it.Before(func() {
			delete(entry.Metadata, "version-source")
			entry.Metadata["version"] = "1.2.2"
		})
		it("returns the latest version", func() {
			dependency, err := versionResolver.Resolve(filepath.Join(cnbDir, "buildpack.toml"), entry, "some-stack")
			Expect(err).NotTo(HaveOccurred())

			Expect(dependency).To(Equal(postal.Dependency{
				ID:      "dotnet-runtime",
				Version: "1.2.4",
				URI:     "some-uri",
				SHA256:  "some-sha",
				Stacks:  []string{"some-stack"},
			}))
		})
	})

	context("the version is default", func() {
		it.Before(func() {
			entry.Metadata["version"] = "default"
		})
		it("returns the latest version", func() {
			dependency, err := versionResolver.Resolve(filepath.Join(cnbDir, "buildpack.toml"), entry, "some-stack")
			Expect(err).NotTo(HaveOccurred())

			Expect(dependency).To(Equal(postal.Dependency{
				ID:      "dotnet-runtime",
				Version: "1.2.4",
				URI:     "some-uri",
				SHA256:  "some-sha",
				Stacks:  []string{"some-stack"},
			}))
		})
	})

	context("the version source is buildpack.yml", func() {
		it.Before(func() {
			entry.Metadata["version-source"] = "buildpack.yml"
		})

		context("the buildpack.toml only has a major minor version match", func() {
			it.Before(func() {
				entry.Metadata["version"] = "1.2.0"
			})
			it("returns an error", func() {
				_, err := versionResolver.Resolve(filepath.Join(cnbDir, "buildpack.toml"), entry, "some-stack")
				Expect(err).To(MatchError(ContainSubstring("failed to satisfy \"dotnet-runtime\" dependency for stack \"some-stack\" with version constraint \"1.2.0\": no compatible versions. Supported versions are: [1.2.3, 1.2.4]")))
			})
		})

		context("the version contains a `*`", func() {
			it.Before(func() {
				entry.Metadata["version"] = "1.2.*"
			})
			it("attempts to turn the given versions into the only constraint", func() {
				dependency, err := versionResolver.Resolve(filepath.Join(cnbDir, "buildpack.toml"), entry, "some-stack")
				Expect(err).NotTo(HaveOccurred())

				Expect(dependency).To(Equal(postal.Dependency{
					ID:      "dotnet-runtime",
					Version: "1.2.4",
					URI:     "some-uri",
					SHA256:  "some-sha",
					Stacks:  []string{"some-stack"},
				}))
			})
		})
	})

	context("failure cases", func() {
		context("the buildpack.toml cannot be parsed", func() {
			it.Before(func() {
				Expect(ioutil.WriteFile(filepath.Join(cnbDir, "buildpack.toml"), []byte(`%%%`), 0600)).To(Succeed())
				entry.Metadata["version"] = "1.2.3"
			})

			it("returns an error", func() {
				_, err := versionResolver.Resolve(filepath.Join(cnbDir, "buildpack.toml"), entry, "some-stack")
				Expect(err).To(MatchError(ContainSubstring("bare keys cannot contain '%'")))
			})
		})

		context("the version is not semver compatible", func() {
			it.Before(func() {
				entry.Metadata["version"] = "invalid-version"
			})
			it("returns an error", func() {
				_, err := versionResolver.Resolve(filepath.Join(cnbDir, "buildpack.toml"), entry, "some-stack")
				Expect(err).To(MatchError(ContainSubstring("improper constraint")))
			})
		})

		context("a buildpack.toml version is not semver compatible", func() {
			it.Before(func() {
				err := ioutil.WriteFile(filepath.Join(cnbDir, "buildpack.toml"), []byte(`api = "0.2"
[buildpack]
id = "org.some-org.some-buildpack"
name = "Some Buildpack"
version = "some-version"

[metadata]

[[metadata.dependencies]]
id = "dotnet-runtime"
sha256 = "some-sha"
stacks = ["some-stack"]
uri = "some-uri"
version = "invalid-version"

[[metadata.dependencies]]
id = "dotnet-runtime"
sha256 = "some-sha"
stacks = ["some-stack"]
uri = "some-uri"
version = "1.2.4"
`), 0600)
				Expect(err).NotTo(HaveOccurred())
				entry.Metadata["version"] = "1.2.0"
			})

			it("returns an error", func() {
				_, err := versionResolver.Resolve(filepath.Join(cnbDir, "buildpack.toml"), entry, "some-stack")
				Expect(err).To(MatchError(ContainSubstring("Invalid Semantic Version")))
			})
		})
	})

}
