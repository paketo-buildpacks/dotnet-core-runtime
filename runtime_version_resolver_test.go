package dotnetcoreruntime_test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	dotnetcoreruntime "github.com/paketo-buildpacks/dotnet-core-runtime"
	"github.com/paketo-buildpacks/packit/postal"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testRuntimeVersionResolver(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		cnbDir          string
		versionResolver dotnetcoreruntime.RuntimeVersionResolver
	)

	it.Before(func() {
		var err error

		versionResolver = dotnetcoreruntime.NewRuntimeVersionResolver()
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
	})

	it.After(func() {
		Expect(os.RemoveAll(cnbDir)).To(Succeed())
	})

	context("when the buildpack.toml has the exact version", func() {
		it("returns a dependency with that version", func() {
			dependency, err := versionResolver.Resolve(filepath.Join(cnbDir, "buildpack.toml"), "dotnet-runtime", "1.2.3", "some-stack")
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

	context("when the buildpack.toml only has a major minor version match", func() {
		it("returns a compatible version", func() {
			dependency, err := versionResolver.Resolve(filepath.Join(cnbDir, "buildpack.toml"), "dotnet-runtime", "1.2.0", "some-stack")
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

	context("when the buildpack.toml only has a major version match", func() {
		it("returns a compatible version", func() {
			dependency, err := versionResolver.Resolve(filepath.Join(cnbDir, "buildpack.toml"), "dotnet-runtime", "1.1.7", "some-stack")
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

	context("when the buildpack.toml does not have a version match", func() {
		it("returns an error", func() {
			_, err := versionResolver.Resolve(filepath.Join(cnbDir, "buildpack.toml"), "dotnet-runtime", "2.0.0", "some-stack")
			Expect(err).To(MatchError(ContainSubstring("failed to satisfy \"dotnet-runtime\" dependency for stack \"some-stack\" with version constraint \"2.0.0\": no compatible versions. Supported versions are: [1.2.3, 1.2.4]")))
		})
	})

	context("when the buildpack.toml does not have a dependency with a matching ID", func() {
		it("returns an error", func() {
			_, err := versionResolver.Resolve(filepath.Join(cnbDir, "buildpack.toml"), "random-ID", "1.2.3", "some-stack")
			Expect(err).To(MatchError(ContainSubstring("failed to satisfy \"random-ID\" dependency for stack \"some-stack\" with version constraint \"1.2.3\": no compatible versions. Supported versions are: []")))
		})
	})

	context("when the buildpack.toml does not have a dependency with a matching stack", func() {
		it("returns an error", func() {
			_, err := versionResolver.Resolve(filepath.Join(cnbDir, "buildpack.toml"), "dotnet-runtime", "1.2.3", "random-stack")
			Expect(err).To(MatchError(ContainSubstring("failed to satisfy \"dotnet-runtime\" dependency for stack \"random-stack\" with version constraint \"1.2.3\": no compatible versions. Supported versions are: []")))
		})
	})

	context("failure cases", func() {
		context("when the buildpack.toml cannot be parsed", func() {
			it.Before(func() {
				Expect(ioutil.WriteFile(filepath.Join(cnbDir, "buildpack.toml"), []byte(`%%%`), 0600)).To(Succeed())
			})

			it("returns an error", func() {
				_, err := versionResolver.Resolve(filepath.Join(cnbDir, "buildpack.toml"), "dotnet-runtime", "1.2.3", "some-stack")
				Expect(err).To(MatchError(ContainSubstring("bare keys cannot contain '%'")))
			})
		})

		context("when the version is not semver compatible", func() {
			it("returns an error", func() {
				_, err := versionResolver.Resolve(filepath.Join(cnbDir, "buildpack.toml"), "dotnet-runtime", "invalid-version", "some-stack")
				Expect(err).To(MatchError(ContainSubstring("Invalid Semantic Version")))
			})
		})

		context("when a buildpack.toml version is not semver compatible", func() {
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
			})

			it("returns an error", func() {
				_, err := versionResolver.Resolve(filepath.Join(cnbDir, "buildpack.toml"), "dotnet-runtime", "1.2.0", "some-stack")
				Expect(err).To(MatchError(ContainSubstring("Invalid Semantic Version")))
			})
		})
	})

}
