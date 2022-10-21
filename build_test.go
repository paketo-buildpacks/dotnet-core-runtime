package dotnetcoreruntime_test

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	dotnetcoreruntime "github.com/paketo-buildpacks/dotnet-core-runtime"
	"github.com/paketo-buildpacks/dotnet-core-runtime/fakes"
	"github.com/paketo-buildpacks/packit/v2"
	"github.com/paketo-buildpacks/packit/v2/chronos"
	"github.com/paketo-buildpacks/packit/v2/sbom"
	"github.com/paketo-buildpacks/packit/v2/scribe"

	//nolint Ignore SA1019, informed usage of deprecated package
	"github.com/paketo-buildpacks/packit/v2/paketosbom"
	"github.com/paketo-buildpacks/packit/v2/postal"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testBuild(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		layersDir  string
		workingDir string
		cnbDir     string
		buffer     *bytes.Buffer

		entryResolver     *fakes.EntryResolver
		dependencyManager *fakes.DependencyManager
		dotnetSymlinker   *fakes.DotnetSymlinker
		versionResolver   *fakes.VersionResolver
		sbomGenerator     *fakes.SBOMGenerator

		buildContext packit.BuildContext
		build        packit.BuildFunc
	)

	it.Before(func() {
		layersDir = t.TempDir()
		cnbDir = t.TempDir()
		workingDir = t.TempDir()

		entryResolver = &fakes.EntryResolver{}
		entryResolver.ResolveCall.Returns.BuildpackPlanEntry = packit.BuildpackPlanEntry{
			Name: "dotnet-runtime",
			Metadata: map[string]interface{}{
				"version-source": "BP_DOTNET_FRAMEWORK_VERSION",
				"version":        "2.5.x",
				"launch":         true,
			},
		}

		entryResolver.MergeLayerTypesCall.Returns.Launch = true

		dependencyManager = &fakes.DependencyManager{}
		dependencyManager.GenerateBillOfMaterialsCall.Returns.BOMEntrySlice = []packit.BOMEntry{
			{
				Name: "dotnet-runtime",
				Metadata: paketosbom.BOMMetadata{
					Version: "dotnet-runtime-dep-version",
					Checksum: paketosbom.BOMChecksum{
						Algorithm: paketosbom.SHA256,
						Hash:      "dotnet-runtime-dep-sha",
					},
					URI: "dotnet-runtime-dep-uri",
				},
			},
		}

		dotnetSymlinker = &fakes.DotnetSymlinker{}

		versionResolver = &fakes.VersionResolver{}
		versionResolver.ResolveCall.Returns.Dependency = postal.Dependency{
			ID:      "dotnet-runtime",
			Version: "2.5.x",
			Name:    ".NET Core Runtime",
			SHA256:  "some-sha", //nolint:staticcheck
		}

		sbomGenerator = &fakes.SBOMGenerator{}
		sbomGenerator.GenerateFromDependencyCall.Returns.SBOM = sbom.SBOM{}

		buffer = bytes.NewBuffer(nil)
		logEmitter := scribe.NewEmitter(buffer)

		buildContext = packit.BuildContext{
			WorkingDir: workingDir,
			CNBPath:    cnbDir,
			Stack:      "some-stack",
			BuildpackInfo: packit.BuildpackInfo{
				Name:        "Some Buildpack",
				Version:     "some-version",
				SBOMFormats: []string{sbom.CycloneDXFormat, sbom.SPDXFormat},
			},
			Platform: packit.Platform{Path: "platform"},
			Plan: packit.BuildpackPlan{
				Entries: []packit.BuildpackPlanEntry{
					{
						Name: "dotnet-runtime",
						Metadata: map[string]interface{}{
							"version-source": "BP_DOTNET_FRAMEWORK_VERSION",
							"version":        "2.5.x",
							"launch":         true,
						},
					},
				},
			},
			Layers: packit.Layers{Path: layersDir},
		}

		build = dotnetcoreruntime.Build(entryResolver, dependencyManager, dotnetSymlinker, versionResolver, sbomGenerator, logEmitter, chronos.DefaultClock)
	})

	it("returns a result that installs the dotnet runtime libraries", func() {
		result, err := build(buildContext)
		Expect(err).NotTo(HaveOccurred())

		Expect(result.Layers).To(HaveLen(1))
		layer := result.Layers[0]

		Expect(layer.Name).To(Equal("dotnet-core-runtime"))
		Expect(layer.Path).To(Equal(filepath.Join(layersDir, "dotnet-core-runtime")))
		Expect(layer.LaunchEnv).To(Equal(packit.Environment{
			"DOTNET_ROOT.override": filepath.Join(workingDir, ".dotnet_root"),
		}))
		Expect(layer.BuildEnv).To(Equal(packit.Environment{
			"RUNTIME_VERSION.override": "2.5.x",
		}))
		Expect(layer.Metadata).To(Equal(map[string]interface{}{
			"dependency-sha": "some-sha",
		}))

		Expect(layer.Build).To(BeFalse())
		Expect(layer.Launch).To(BeTrue())
		Expect(layer.Cache).To(BeFalse())

		Expect(layer.SBOM.Formats()).To(Equal([]packit.SBOMFormat{
			{
				Extension: sbom.Format(sbom.CycloneDXFormat).Extension(),
				Content:   sbom.NewFormattedReader(sbom.SBOM{}, sbom.CycloneDXFormat),
			},
			{
				Extension: sbom.Format(sbom.SPDXFormat).Extension(),
				Content:   sbom.NewFormattedReader(sbom.SBOM{}, sbom.SPDXFormat),
			},
		}))

		Expect(result.Launch.BOM).To(HaveLen(1))
		launchBOMEntry := result.Launch.BOM[0]
		Expect(launchBOMEntry.Name).To(Equal("dotnet-runtime"))
		Expect(launchBOMEntry.Metadata).To(Equal(paketosbom.BOMMetadata{
			Version: "dotnet-runtime-dep-version",
			Checksum: paketosbom.BOMChecksum{
				Algorithm: paketosbom.SHA256,
				Hash:      "dotnet-runtime-dep-sha",
			},
			URI: "dotnet-runtime-dep-uri",
		}))

		Expect(entryResolver.ResolveCall.Receives.BuildpackPlanEntrySlice).To(Equal([]packit.BuildpackPlanEntry{
			{
				Name: "dotnet-runtime",
				Metadata: map[string]interface{}{
					"version-source": "BP_DOTNET_FRAMEWORK_VERSION",
					"version":        "2.5.x",
					"launch":         true,
				},
			},
		}))

		Expect(dependencyManager.GenerateBillOfMaterialsCall.Receives.Dependencies).To(Equal([]postal.Dependency{
			{
				ID:      "dotnet-runtime",
				Version: "2.5.x",
				Name:    ".NET Core Runtime",
				SHA256:  "some-sha", //nolint:staticcheck
			},
		}))

		Expect(versionResolver.ResolveCall.Receives.Path).To(Equal(filepath.Join(cnbDir, "buildpack.toml")))
		Expect(versionResolver.ResolveCall.Receives.Entry).To(Equal(entryResolver.ResolveCall.Returns.BuildpackPlanEntry))
		Expect(versionResolver.ResolveCall.Receives.Stack).To(Equal("some-stack"))

		Expect(dependencyManager.DeliverCall.Receives.Dependency).To(Equal(postal.Dependency{
			ID:      "dotnet-runtime",
			Version: "2.5.x",
			Name:    ".NET Core Runtime",
			SHA256:  "some-sha", //nolint:staticcheck
		}))
		Expect(dependencyManager.DeliverCall.Receives.CnbPath).To(Equal(cnbDir))
		Expect(dependencyManager.DeliverCall.Receives.LayerPath).To(Equal(filepath.Join(layersDir, "dotnet-core-runtime")))
		Expect(dependencyManager.DeliverCall.Receives.PlatformPath).To(Equal("platform"))

		Expect(dotnetSymlinker.LinkCall.CallCount).To(Equal(1))
		Expect(dotnetSymlinker.LinkCall.Receives.WorkingDir).To(Equal(workingDir))
		Expect(dotnetSymlinker.LinkCall.Receives.LayerPath).To(Equal(filepath.Join(layersDir, "dotnet-core-runtime")))

		Expect(sbomGenerator.GenerateFromDependencyCall.Receives.Dependency).To(Equal(postal.Dependency{
			ID:      "dotnet-runtime",
			Version: "2.5.x",
			Name:    ".NET Core Runtime",
			SHA256:  "some-sha", //nolint:staticcheck
		}))
		Expect(sbomGenerator.GenerateFromDependencyCall.Receives.Dir).To(Equal(filepath.Join(layersDir, "dotnet-core-runtime")))

		Expect(buffer.String()).To(ContainSubstring("Some Buildpack some-version"))
		Expect(buffer.String()).To(ContainSubstring("Resolving .NET Core Runtime version"))
		Expect(buffer.String()).To(ContainSubstring("Selected .NET Core Runtime version (using BP_DOTNET_FRAMEWORK_VERSION): "))
		Expect(buffer.String()).To(ContainSubstring("Executing build process"))
		Expect(buffer.String()).To(ContainSubstring("Configuring build environment"))
		Expect(buffer.String()).To(ContainSubstring("Configuring launch environment"))
	})

	context("when there is a dependency cache match", func() {
		it.Before(func() {
			entryResolver.MergeLayerTypesCall.Returns.Build = true
			entryResolver.MergeLayerTypesCall.Returns.Launch = false

			err := os.WriteFile(filepath.Join(layersDir, "dotnet-core-runtime.toml"), []byte("[metadata]\ndependency-sha = \"some-sha\"\n"), 0600)
			Expect(err).NotTo(HaveOccurred())
		})

		it("returns a result that installs the dotnet runtime libraries", func() {
			_, err := build(buildContext)
			Expect(err).NotTo(HaveOccurred())

			Expect(entryResolver.ResolveCall.Receives.BuildpackPlanEntrySlice).To(Equal([]packit.BuildpackPlanEntry{
				{
					Name: "dotnet-runtime",
					Metadata: map[string]interface{}{
						"version-source": "BP_DOTNET_FRAMEWORK_VERSION",
						"version":        "2.5.x",
						"launch":         true,
					},
				},
			}))

			Expect(dependencyManager.GenerateBillOfMaterialsCall.Receives.Dependencies).To(Equal([]postal.Dependency{
				{
					ID:      "dotnet-runtime",
					Version: "2.5.x",
					Name:    ".NET Core Runtime",
					SHA256:  "some-sha", //nolint:staticcheck
				},
			}))

			Expect(versionResolver.ResolveCall.Receives.Path).To(Equal(filepath.Join(cnbDir, "buildpack.toml")))
			Expect(versionResolver.ResolveCall.Receives.Entry).To(Equal(entryResolver.ResolveCall.Returns.BuildpackPlanEntry))
			Expect(versionResolver.ResolveCall.Receives.Stack).To(Equal("some-stack"))

			Expect(dependencyManager.DeliverCall.CallCount).To(Equal(0))

			Expect(dotnetSymlinker.LinkCall.CallCount).To(Equal(1))
			Expect(dotnetSymlinker.LinkCall.Receives.WorkingDir).To(Equal(workingDir))
			Expect(dotnetSymlinker.LinkCall.Receives.LayerPath).To(Equal(filepath.Join(layersDir, "dotnet-core-runtime")))

			Expect(buffer.String()).To(ContainSubstring("Some Buildpack some-version"))
			Expect(buffer.String()).To(ContainSubstring("Resolving .NET Core Runtime version"))
			Expect(buffer.String()).To(ContainSubstring("Selected .NET Core Runtime version (using BP_DOTNET_FRAMEWORK_VERSION): "))
			Expect(buffer.String()).To(ContainSubstring("Reusing cached layer"))
			Expect(buffer.String()).NotTo(ContainSubstring("Executing build process"))
		})
	})

	context("when version-source of the selected entry is buildpack.yml", func() {
		it.Before(func() {
			buildContext.BuildpackInfo.Version = "0.1.2"
			buildContext.Plan.Entries[0].Metadata["version-source"] = "buildpack.yml"

			entryResolver.ResolveCall.Returns.BuildpackPlanEntry = packit.BuildpackPlanEntry{
				Name: "dotnet-runtime",
				Metadata: map[string]interface{}{
					"version-source": "buildpack.yml",
					"version":        "2.5.x",
					"launch":         true,
				},
			}
		})

		it("prints a deprecation warning", func() {
			_, err := build(buildContext)
			Expect(err).NotTo(HaveOccurred())

			Expect(buffer.String()).To(ContainSubstring("Some Buildpack 0.1.2"))
			Expect(buffer.String()).To(ContainSubstring("Resolving .NET Core Runtime version"))
			Expect(buffer.String()).To(ContainSubstring("Selected .NET Core Runtime version (using buildpack.yml): "))
			// v1.0.0 because that's the next major after input version v0.1.2
			Expect(buffer.String()).To(ContainSubstring("WARNING: Setting the .NET Framework version through buildpack.yml will be deprecated soon in .NET Core Runtime Buildpack v1.0.0."))
			Expect(buffer.String()).To(ContainSubstring("Please specify the version through the $BP_DOTNET_FRAMEWORK_VERSION environment variable instead. See docs for more information."))
			Expect(buffer.String()).To(ContainSubstring("Executing build process"))
			Expect(buffer.String()).To(ContainSubstring("Configuring build environment"))
			Expect(buffer.String()).To(ContainSubstring("Configuring launch environment"))
		})
	})

	context("failure cases", func() {
		context("when a dependency cannot be resolved", func() {
			it.Before(func() {
				versionResolver.ResolveCall.Returns.Error = errors.New("failed to resolve version")
			})

			it("returns an error", func() {
				_, err := build(buildContext)
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
				_, err := build(buildContext)
				Expect(err).To(MatchError(ContainSubstring("permission denied")))
			})
		})

		context("when the layer directory cannot be removed", func() {
			var layerDir string
			it.Before(func() {
				layerDir = filepath.Join(layersDir, "dotnet-core-runtime")
				Expect(os.MkdirAll(filepath.Join(layerDir, "dotnet-core-runtime"), os.ModePerm)).To(Succeed())
				Expect(os.Chmod(layerDir, 0500)).To(Succeed())
			})

			it.After(func() {
				Expect(os.Chmod(layerDir, os.ModePerm)).To(Succeed())
				Expect(os.RemoveAll(layerDir)).To(Succeed())
			})

			it("returns an error", func() {
				_, err := build(buildContext)
				Expect(err).To(MatchError(ContainSubstring("permission denied")))
			})
		})

		context("when the executable errors", func() {
			it.Before(func() {
				dependencyManager.DeliverCall.Returns.Error = errors.New("some-error")
			})

			it("returns an error", func() {
				_, err := build(buildContext)
				Expect(err).To(MatchError(ContainSubstring("some-error")))
			})
		})

		context("when generating the SBOM returns an error", func() {
			it.Before(func() {
				sbomGenerator.GenerateFromDependencyCall.Returns.Error = errors.New("failed to generate SBOM")
			})

			it("returns an error", func() {
				_, err := build(buildContext)
				Expect(err).To(MatchError(ContainSubstring("failed to generate SBOM")))
			})
		})

		context("when formatting the SBOM returns an error", func() {
			it.Before(func() {
				buildContext.BuildpackInfo.SBOMFormats = []string{"random-format"}
			})

			it("returns an error", func() {
				_, err := build(buildContext)
				Expect(err).To(MatchError("unsupported SBOM format: 'random-format'"))
			})
		})
	})
}
