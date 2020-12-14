package dotnetcoreruntime_test

import (
	"bytes"
	"errors"
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

		layersDir  string
		workingDir string
		cnbDir     string
		clock      chronos.Clock
		timeStamp  time.Time
		buffer     *bytes.Buffer

		entryResolver     *fakes.EntryResolver
		dependencyManager *fakes.DependencyManager
		planRefinery      *fakes.BuildPlanRefinery
		dotnetSymlinker   *fakes.DotnetSymlinker
		versionResolver   *fakes.VersionResolver

		build packit.BuildFunc
	)

	it.Before(func() {
		var err error
		layersDir, err = ioutil.TempDir("", "layers")
		Expect(err).NotTo(HaveOccurred())

		cnbDir, err = ioutil.TempDir("", "cnb")
		Expect(err).NotTo(HaveOccurred())

		workingDir, err = ioutil.TempDir("", "working-dir")
		Expect(err).NotTo(HaveOccurred())

		entryResolver = &fakes.EntryResolver{}
		entryResolver.ResolveCall.Returns.BuildpackPlanEntry = packit.BuildpackPlanEntry{
			Name: "dotnet-runtime",
			Metadata: map[string]interface{}{
				"version-source": "buildpack.yml",
				"version":        "2.5.x",
				"launch":         true,
			},
		}

		dependencyManager = &fakes.DependencyManager{}

		planRefinery = &fakes.BuildPlanRefinery{}

		planRefinery.BillOfMaterialCall.Returns.BuildpackPlan = packit.BuildpackPlan{
			Entries: []packit.BuildpackPlanEntry{
				{
					Name: "dotnet-runtime",
					Metadata: map[string]interface{}{
						"version-source": "buildpack.yml",
						"version":        "2.5.x",
						"launch":         true,
					},
				},
			},
		}

		dotnetSymlinker = &fakes.DotnetSymlinker{}

		versionResolver = &fakes.VersionResolver{}
		versionResolver.ResolveCall.Returns.Dependency = postal.Dependency{
			ID:      "dotnet-runtime",
			Version: "2.5.x",
			Name:    "Dotnet Core Runtime",
			SHA256:  "some-sha",
		}

		buffer = bytes.NewBuffer(nil)
		logEmitter := dotnetcoreruntime.NewLogEmitter(buffer)

		timeStamp = time.Now()
		clock = chronos.NewClock(func() time.Time {
			return timeStamp
		})

		build = dotnetcoreruntime.Build(entryResolver, dependencyManager, planRefinery, dotnetSymlinker, versionResolver, logEmitter, clock)
	})

	it.After(func() {
		Expect(os.RemoveAll(layersDir)).To(Succeed())
		Expect(os.RemoveAll(cnbDir)).To(Succeed())
		Expect(os.RemoveAll(workingDir)).To(Succeed())
	})

	it("returns a result that installs the dotnet runtime libraries", func() {
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
						Name: "dotnet-runtime",
						Metadata: map[string]interface{}{
							"version-source": "buildpack.yml",
							"version":        "2.5.x",
							"launch":         true,
						},
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
						Name: "dotnet-runtime",
						Metadata: map[string]interface{}{
							"version-source": "buildpack.yml",
							"version":        "2.5.x",
							"launch":         true,
						},
					},
				},
			},
			Layers: []packit.Layer{
				{
					Name: "dotnet-core-runtime",
					Path: filepath.Join(layersDir, "dotnet-core-runtime"),
					SharedEnv: packit.Environment{
						"DOTNET_ROOT.override": filepath.Join(workingDir, ".dotnet_root"),
					},
					BuildEnv: packit.Environment{
						"RUNTIME_VERSION.override": "2.5.x",
					},
					LaunchEnv: packit.Environment{},
					Build:     false,
					Launch:    true,
					Cache:     false,
					Metadata: map[string]interface{}{
						"dependency-sha": "some-sha",
						"built_at":       timeStamp.Format(time.RFC3339Nano),
					},
				},
			},
		}))

		Expect(entryResolver.ResolveCall.Receives.BuildpackPlanEntrySlice).To(Equal([]packit.BuildpackPlanEntry{
			{
				Name: "dotnet-runtime",
				Metadata: map[string]interface{}{
					"version-source": "buildpack.yml",
					"version":        "2.5.x",
					"launch":         true,
				},
			},
		}))

		Expect(planRefinery.BillOfMaterialCall.CallCount).To(Equal(1))
		Expect(planRefinery.BillOfMaterialCall.Receives.Dependency).To(Equal(postal.Dependency{
			ID:      "dotnet-runtime",
			Version: "2.5.x",
			Name:    "Dotnet Core Runtime",
			SHA256:  "some-sha",
		}))

		Expect(versionResolver.ResolveCall.Receives.Path).To(Equal(filepath.Join(cnbDir, "buildpack.toml")))
		Expect(versionResolver.ResolveCall.Receives.Id).To(Equal("dotnet-runtime"))
		Expect(versionResolver.ResolveCall.Receives.Version).To(Equal("2.5.x"))
		Expect(versionResolver.ResolveCall.Receives.Stack).To(Equal("some-stack"))

		Expect(dependencyManager.InstallCall.Receives.Dependency).To(Equal(postal.Dependency{
			ID:      "dotnet-runtime",
			Version: "2.5.x",
			Name:    "Dotnet Core Runtime",
			SHA256:  "some-sha",
		}))
		Expect(dependencyManager.InstallCall.Receives.CnbPath).To(Equal(cnbDir))
		Expect(dependencyManager.InstallCall.Receives.LayerPath).To(Equal(filepath.Join(layersDir, "dotnet-core-runtime")))

		Expect(dotnetSymlinker.LinkCall.CallCount).To(Equal(1))
		Expect(dotnetSymlinker.LinkCall.Receives.WorkingDir).To(Equal(workingDir))
		Expect(dotnetSymlinker.LinkCall.Receives.LayerPath).To(Equal(filepath.Join(layersDir, "dotnet-core-runtime")))

		Expect(buffer.String()).To(ContainSubstring("Some Buildpack some-version"))
		Expect(buffer.String()).To(ContainSubstring("Resolving Dotnet Core Runtime version"))
		Expect(buffer.String()).To(ContainSubstring("Selected dotnet-runtime version (using buildpack.yml): "))
		Expect(buffer.String()).To(ContainSubstring("Executing build process"))
		Expect(buffer.String()).To(ContainSubstring("Configuring environment"))
	})

	context("when there is a dependency cache match", func() {
		it.Before(func() {
			err := ioutil.WriteFile(filepath.Join(layersDir, "dotnet-core-runtime.toml"), []byte("[metadata]\ndependency-sha = \"some-sha\"\n"), 0600)
			Expect(err).NotTo(HaveOccurred())
		})

		it("returns a result that installs the dotnet runtime libraries", func() {
			_, err := build(packit.BuildContext{
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
							Name: "dotnet-runtime",
							Metadata: map[string]interface{}{
								"version-source": "buildpack.yml",
								"version":        "2.5.x",
								"launch":         true,
							},
						},
					},
				},
				Layers: packit.Layers{Path: layersDir},
			})
			Expect(err).NotTo(HaveOccurred())

			Expect(entryResolver.ResolveCall.Receives.BuildpackPlanEntrySlice).To(Equal([]packit.BuildpackPlanEntry{
				{
					Name: "dotnet-runtime",
					Metadata: map[string]interface{}{
						"version-source": "buildpack.yml",
						"version":        "2.5.x",
						"launch":         true,
					},
				},
			}))

			Expect(planRefinery.BillOfMaterialCall.CallCount).To(Equal(1))
			Expect(planRefinery.BillOfMaterialCall.Receives.Dependency).To(Equal(postal.Dependency{
				ID:      "dotnet-runtime",
				Version: "2.5.x",
				Name:    "Dotnet Core Runtime",
				SHA256:  "some-sha",
			}))

			Expect(versionResolver.ResolveCall.Receives.Path).To(Equal(filepath.Join(cnbDir, "buildpack.toml")))
			Expect(versionResolver.ResolveCall.Receives.Id).To(Equal("dotnet-runtime"))
			Expect(versionResolver.ResolveCall.Receives.Version).To(Equal("2.5.x"))
			Expect(versionResolver.ResolveCall.Receives.Stack).To(Equal("some-stack"))

			Expect(dependencyManager.InstallCall.CallCount).To(Equal(0))

			Expect(dotnetSymlinker.LinkCall.CallCount).To(Equal(1))
			Expect(dotnetSymlinker.LinkCall.Receives.WorkingDir).To(Equal(workingDir))
			Expect(dotnetSymlinker.LinkCall.Receives.LayerPath).To(Equal(filepath.Join(layersDir, "dotnet-core-runtime")))

			Expect(buffer.String()).To(ContainSubstring("Some Buildpack some-version"))
			Expect(buffer.String()).To(ContainSubstring("Resolving Dotnet Core Runtime version"))
			Expect(buffer.String()).To(ContainSubstring("Selected dotnet-runtime version (using buildpack.yml): "))
			Expect(buffer.String()).To(ContainSubstring("Reusing cached layer"))
			Expect(buffer.String()).NotTo(ContainSubstring("Executing build process"))
		})
	})

	context("failure cases", func() {
		context("when a dependency cannot be resolved", func() {
			it.Before(func() {
				versionResolver.ResolveCall.Returns.Error = errors.New("failed to resolve version")
			})

			it("returns an error", func() {
				_, err := build(packit.BuildContext{
					CNBPath: cnbDir,
					Plan: packit.BuildpackPlan{
						Entries: []packit.BuildpackPlanEntry{
							{
								Name: "dotnet-runtime",
								Metadata: map[string]interface{}{
									"version-source": "buildpack.yml",
									"version":        "2.5.x",
								},
							},
						},
					},
					Layers: packit.Layers{Path: layersDir},
				})
				Expect(err).To(MatchError("failed to resolve version"))
			})
		})

		context("when a dependency cannot be written to", func() {
			it.Before(func() {
				Expect(os.Chmod(layersDir, 0000)).To(Succeed())
			})

			it.After(func() {
				Expect(os.Chmod(layersDir, os.ModePerm)).To(Succeed())
			})

			it("returns an error", func() {
				_, err := build(packit.BuildContext{
					CNBPath: cnbDir,
					Plan: packit.BuildpackPlan{
						Entries: []packit.BuildpackPlanEntry{
							{
								Name: "dotnet-runtime",
								Metadata: map[string]interface{}{
									"version-source": "buildpack.yml",
									"version":        "2.5.x",
								},
							},
						},
					},
					Layers: packit.Layers{Path: layersDir},
				})
				Expect(err).To(MatchError(ContainSubstring("permission denied")))
			})
		})

		context("when the layer directory cannot be removed", func() {
			var layerDir string
			it.Before(func() {
				layerDir = filepath.Join(layersDir, "dotnet-core-runtime")
				Expect(os.MkdirAll(filepath.Join(layerDir, "dotnet-core-runtime"), os.ModePerm)).To(Succeed())
				Expect(os.Chmod(layerDir, 0000)).To(Succeed())
			})

			it.After(func() {
				Expect(os.Chmod(layerDir, os.ModePerm)).To(Succeed())
				Expect(os.RemoveAll(layerDir)).To(Succeed())
			})

			it("returns an error", func() {
				_, err := build(packit.BuildContext{
					CNBPath: cnbDir,
					Plan: packit.BuildpackPlan{
						Entries: []packit.BuildpackPlanEntry{
							{
								Name: "dotnet-core-runtime",
								Metadata: map[string]interface{}{
									"version-source": "buildpack.yml",
									"version":        "2.5.x",
								},
							},
						},
					},
					Layers: packit.Layers{Path: layersDir},
				})
				Expect(err).To(MatchError(ContainSubstring("permission denied")))
			})
		})

		context("when the executable errors", func() {
			it.Before(func() {
				dependencyManager.InstallCall.Returns.Error = errors.New("some-error")
			})

			it("returns an error", func() {
				_, err := build(packit.BuildContext{
					CNBPath: cnbDir,
					Plan: packit.BuildpackPlan{
						Entries: []packit.BuildpackPlanEntry{
							{
								Name: "dotnet-core-runtime",
								Metadata: map[string]interface{}{
									"version-source": "buildpack.yml",
									"version":        "2.5.x",
								},
							},
						},
					},
					Layers: packit.Layers{Path: layersDir},
				})
				Expect(err).To(MatchError(ContainSubstring("some-error")))
			})
		})
	})
}
