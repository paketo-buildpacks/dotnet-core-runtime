package dotnetcoreruntime_test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
	"time"

	dotnetcoreruntime "github.com/paketo-buildpacks/dotnet-core-runtime"
	"github.com/paketo-buildpacks/dotnet-core-runtime/fakes"
	"github.com/paketo-buildpacks/packit"
	"github.com/paketo-buildpacks/packit/chronos"
	"github.com/paketo-buildpacks/packit/postal"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testBuild(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		layersDir         string
		workingDir        string
		cnbDir            string
		entryResolver     *fakes.EntryResolver
		dependencyManager *fakes.DependencyManager
		clock             chronos.Clock
		timeStamp         time.Time
		planRefinery      *fakes.BuildPlanRefinery
		// timestamp  time.Time

		build packit.BuildFunc
	)

	it.Before(func() {
		var err error
		layersDir, err = ioutil.TempDir("", "layers")
		Expect(err).NotTo(HaveOccurred())

		cnbDir, err = ioutil.TempDir("", "cnb")
		Expect(err).NotTo(HaveOccurred())

		err = ioutil.WriteFile(filepath.Join(cnbDir, "buildpack.toml"), []byte(`api = "0.2"
[buildpack]
  id = "org.some-org.some-buildpack"
  name = "Some Buildpack"
  version = "some-version"

[metadata]
  [metadata.default-versions]
    mri = "2.5.x"

  [[metadata.dependencies]]
    id = "some-dep"
    name = "Some Dep"
    sha256 = "some-sha"
    stacks = ["some-stack"]
    uri = "some-uri"
    version = "some-dep-version"
`), 0600)
		Expect(err).NotTo(HaveOccurred())

		entryResolver = &fakes.EntryResolver{}
		entryResolver.ResolveCall.Returns.BuildpackPlanEntry = packit.BuildpackPlanEntry{
			Name: "dotnet-core-runtime",
			Metadata: map[string]interface{}{
				"version-source": "buildpack.yml",
				"version":        "1.2.3",
				"launch":         true,
				"build":          true,
			},
		}

		dependencyManager = &fakes.DependencyManager{}
		dependencyManager.ResolveCall.Returns.Dependency = postal.Dependency{ID: "dotnet-runtime", Name: "Dotnet Core Runtime"}

		planRefinery = &fakes.BuildPlanRefinery{}

		timeStamp = time.Now()
		clock = chronos.NewClock(func() time.Time {
			return timeStamp
		})

		planRefinery.BillOfMaterialCall.Returns.BuildpackPlan = packit.BuildpackPlan{
			Entries: []packit.BuildpackPlanEntry{
				{
					Name: "dotnet-core-runtime",
					Metadata: map[string]interface{}{
						"version-source": "buildpack.yml",
						"version":        "1.2.3",
						"launch":         true,
						"build":          true,
					},
				},
			},
		}

		// timestamp = time.Now()
		// clock := chronos.NewClock(func() time.Time {
		// 	return timestamp
		// })

		build = dotnetcoreruntime.Build(entryResolver, dependencyManager, planRefinery)
	})

	it.After(func() {
		Expect(os.RemoveAll(layersDir)).To(Succeed())
		Expect(os.RemoveAll(cnbDir)).To(Succeed())
		Expect(os.RemoveAll(workingDir)).To(Succeed())
	})

	it.Focus("returns a result that installs the dotnet runtime libraries", func() {
		result, err := build(packit.BuildContext{
			WorkingDir: workingDir,
			CNBPath:    cnbDir,
			Stack:      "some-stack",
			BuildpackInfo: packit.BuildpackInfo{
				Name:    "Some Buildpack",
				Version: "some-version",
			},
			Plan: packit.BuildpackPlan{
				Entries: []packit.BuildpackPlanEntry{
					{
						Name: "dotnet-core-runtime",
					},
				},
			},
			Layers: packit.Layers{Path: layersDir},
		})
		Expect(err).NotTo(HaveOccurred())

		Expect(result).To(Equal(packit.BuildResult{
			Plan: packit.BuildpackPlan{
				Entries: []packit.BuildpackPlanEntry{
					{
						Name: "dotnet-core-runtime",
						Metadata: map[string]interface{}{
							"name":   "dotnet-core-runtime-dependency-name",
							"sha256": "dotnet-core-runtime-dependency-sha",
							"stacks": []string{"some-stack"},
							"uri":    "dotnet-core-runtime-dependency-uri",
						},
					},
				},
			},
			Layers: []packit.Layer{
				{
					Name:      "dotnet-core-runtime",
					Path:      filepath.Join(layersDir, "dotnet-core-runtime"),
					SharedEnv: packit.Environment{},
					BuildEnv:  packit.Environment{},
					LaunchEnv: packit.Environment{},
					Build:     false,
					Launch:    false,
					Cache:     false,
					Metadata:  map[string]interface{}{
						// dotnetcoreruntime.DependencyCacheKey: "dotnet-core-runtime",
						// "built_at":                           timestamp.Format(time.RFC3339Nano),
					},
				},
			},
		}))
	})
}
