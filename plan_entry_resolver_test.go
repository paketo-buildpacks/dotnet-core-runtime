package dotnetcoreruntime_test

import (
	"bytes"
	"testing"

	dotnetcoreruntime "github.com/paketo-buildpacks/dotnet-core-runtime"
	"github.com/paketo-buildpacks/packit"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testPlanEntryResolver(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		buffer   *bytes.Buffer
		resolver dotnetcoreruntime.PlanEntryResolver
	)

	it.Before(func() {
		buffer = bytes.NewBuffer(nil)
		resolver = dotnetcoreruntime.NewPlanEntryResolver(dotnetcoreruntime.NewLogEmitter(buffer))
	})

	context("when a buildpack.yml entry is included", func() {
		it("resolves the best plan entry", func() {
			entry := resolver.Resolve([]packit.BuildpackPlanEntry{
				{
					Name: "dotnet-core-runtime",
					Metadata: map[string]interface{}{
						"version": "other-version",
					},
				},
				{
					Name: "dotnet-core-runtime",
					Metadata: map[string]interface{}{
						"version-source": "buildpack.yml",
						"version":        "buildpack-yml-version",
					},
				},
			})
			Expect(entry).To(Equal(packit.BuildpackPlanEntry{
				Name: "dotnet-core-runtime",
				Metadata: map[string]interface{}{
					"version-source": "buildpack.yml",
					"version":        "buildpack-yml-version",
				},
			}))

			Expect(buffer.String()).To(ContainSubstring("    Candidate version sources (in priority order):"))
			Expect(buffer.String()).To(ContainSubstring("      buildpack.yml -> \"buildpack-yml-version\""))
			Expect(buffer.String()).To(ContainSubstring("      <unknown>     -> \"other-version\""))
		})
	})

	context("when a buildpack.yml and *sproj are both included", func() {
		it("resolves the best plan entry", func() {
			entry := resolver.Resolve([]packit.BuildpackPlanEntry{
				{
					Name: "dotnet-core-runtime",
					Metadata: map[string]interface{}{
						"version": "other-version",
					},
				},
				{
					Name: "dotnet-core-runtime",
					Metadata: map[string]interface{}{
						"version-source": "buildpack.yml",
						"version":        "buildpack-yml-version",
					},
				},
				{
					Name: "dotnet-core-runtime",
					Metadata: map[string]interface{}{
						"version-source": "*sproj",
						"version":        "*sproj-version",
					},
				},
			})
			Expect(entry).To(Equal(packit.BuildpackPlanEntry{
				Name: "dotnet-core-runtime",
				Metadata: map[string]interface{}{
					"version-source": "buildpack.yml",
					"version":        "buildpack-yml-version",
				},
			}))

			Expect(buffer.String()).To(ContainSubstring("    Candidate version sources (in priority order):"))
			Expect(buffer.String()).To(ContainSubstring("      buildpack.yml -> \"buildpack-yml-version\""))
			Expect(buffer.String()).To(ContainSubstring("      *sproj        -> \"*sproj-version\""))
			Expect(buffer.String()).To(ContainSubstring("      <unknown>     -> \"other-version\""))
		})
	})

	context("when a buildpack.yml and project file are both included", func() {
		it("resolves the best plan entry", func() {
			entry := resolver.Resolve([]packit.BuildpackPlanEntry{
				{
					Name: "dotnet-core-runtime",
					Metadata: map[string]interface{}{
						"version": "other-version",
					},
				},
				{
					Name: "dotnet-core-runtime",
					Metadata: map[string]interface{}{
						"version-source": "buildpack.yml",
						"version":        "buildpack-yml-version",
					},
				},
				{
					Name: "dotnet-core-runtime",
					Metadata: map[string]interface{}{
						"version-source": "project file",
						"version":        "project-file-version",
					},
				},
			})
			Expect(entry).To(Equal(packit.BuildpackPlanEntry{
				Name: "dotnet-core-runtime",
				Metadata: map[string]interface{}{
					"version-source": "buildpack.yml",
					"version":        "buildpack-yml-version",
				},
			}))

			Expect(buffer.String()).To(ContainSubstring("    Candidate version sources (in priority order):"))
			Expect(buffer.String()).To(ContainSubstring("      buildpack.yml -> \"buildpack-yml-version\""))
			Expect(buffer.String()).To(ContainSubstring("      project file  -> \"project-file-version\""))
			Expect(buffer.String()).To(ContainSubstring("      <unknown>     -> \"other-version\""))
		})
	})

	context("when a buildpack.yml and runtimeconfig.json are both included", func() {
		it("resolves the best plan entry", func() {
			entry := resolver.Resolve([]packit.BuildpackPlanEntry{
				{
					Name: "dotnet-core-runtime",
					Metadata: map[string]interface{}{
						"version": "other-version",
					},
				},
				{
					Name: "dotnet-core-runtime",
					Metadata: map[string]interface{}{
						"version-source": "buildpack.yml",
						"version":        "buildpack-yml-version",
					},
				},
				{
					Name: "dotnet-core-runtime",
					Metadata: map[string]interface{}{
						"version-source": "runtimeconfig.json",
						"version":        "runtimeconfig-version",
					},
				},
			})
			Expect(entry).To(Equal(packit.BuildpackPlanEntry{
				Name: "dotnet-core-runtime",
				Metadata: map[string]interface{}{
					"version-source": "buildpack.yml",
					"version":        "buildpack-yml-version",
				},
			}))

			Expect(buffer.String()).To(ContainSubstring("    Candidate version sources (in priority order):"))
			Expect(buffer.String()).To(ContainSubstring("      buildpack.yml      -> \"buildpack-yml-version\""))
			Expect(buffer.String()).To(ContainSubstring("      runtimeconfig.json -> \"runtimeconfig-version\""))
			Expect(buffer.String()).To(ContainSubstring("      <unknown>          -> \"other-version\""))
		})
	})

	context("when a project file and unknown are both included", func() {
		it("resolves the best plan entry", func() {
			entry := resolver.Resolve([]packit.BuildpackPlanEntry{
				{
					Name: "dotnet-core-runtime",
					Metadata: map[string]interface{}{
						"version":        "other-version",
						"version-source": "unknown source",
					},
				},
				{
					Name: "dotnet-core-runtime",
					Metadata: map[string]interface{}{
						"version-source": "project file",
						"version":        "project-file-version",
					},
				},
			})
			Expect(entry).To(Equal(packit.BuildpackPlanEntry{
				Name: "dotnet-core-runtime",
				Metadata: map[string]interface{}{
					"version-source": "project file",
					"version":        "project-file-version",
				},
			}))

			Expect(buffer.String()).To(ContainSubstring("    Candidate version sources (in priority order):"))
			Expect(buffer.String()).To(ContainSubstring("      project file   -> \"project-file-version\""))
			Expect(buffer.String()).To(ContainSubstring("      unknown source -> \"other-version\""))
		})
	})

	context("when entry flags differ", func() {
		context("OR's them together on best plan entry", func() {
			it("has all flags", func() {
				entry := resolver.Resolve([]packit.BuildpackPlanEntry{
					{
						Name: "dotnet-core-runtime",
						Metadata: map[string]interface{}{
							"version-source": "buildpack.yml",
							"version":        "buildpack-yml-version",
						},
					},
					{
						Name: "dotnet-core-runtime",
						Metadata: map[string]interface{}{
							"build": true,
						},
					},
				})
				Expect(entry).To(Equal(packit.BuildpackPlanEntry{
					Name: "dotnet-core-runtime",
					Metadata: map[string]interface{}{
						"version-source": "buildpack.yml",
						"version":        "buildpack-yml-version",
						"build":          true,
					},
				}))
			})
		})
	})

	context("when an unknown source entry is included", func() {
		it("resolves the best plan entry", func() {
			entry := resolver.Resolve([]packit.BuildpackPlanEntry{
				{
					Name: "dotnet-core-runtime",
					Metadata: map[string]interface{}{
						"version": "other-version",
					},
				},
			})
			Expect(entry).To(Equal(packit.BuildpackPlanEntry{
				Name: "dotnet-core-runtime",
				Metadata: map[string]interface{}{
					"version": "other-version",
				},
			}))
		})
	})
}
