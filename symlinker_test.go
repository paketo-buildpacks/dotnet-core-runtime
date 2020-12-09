package dotnetcoreruntime_test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	dotnetcoreruntime "github.com/paketo-buildpacks/dotnet-core-runtime"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testSymlinker(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		symlinker  dotnetcoreruntime.Symlinker
		workingDir string
		layerPath  string
	)

	it.Before(func() {
		var err error
		workingDir, err = ioutil.TempDir("", "working-dir")
		Expect(err).NotTo(HaveOccurred())

		layerPath, err = ioutil.TempDir("", "layer-path")
		Expect(err).NotTo(HaveOccurred())

		symlinker = dotnetcoreruntime.NewSymlinker()
	})

	it.After(func() {
		Expect(os.RemoveAll(workingDir)).To(Succeed())
		Expect(os.RemoveAll(layerPath)).To(Succeed())
	})

	context("Link", func() {
		it("creates a .dotnet_root dir in workspace with symlink to layerpath", func() {
			err := symlinker.Link(workingDir, layerPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(filepath.Join(workingDir, ".dotnet_root", "shared")).To(BeADirectory())

			fi, err := os.Lstat(filepath.Join(workingDir, ".dotnet_root", "shared", "Microsoft.NETCore.App"))
			Expect(err).NotTo(HaveOccurred())
			Expect(fi.Mode() & os.ModeSymlink).ToNot(BeZero())

			fi, err = os.Lstat(filepath.Join(workingDir, ".dotnet_root", "host"))
			Expect(err).NotTo(HaveOccurred())
			Expect(fi.Mode() & os.ModeSymlink).ToNot(BeZero())

			link, err := os.Readlink(filepath.Join(workingDir, ".dotnet_root", "shared", "Microsoft.NETCore.App"))
			Expect(err).NotTo(HaveOccurred())
			Expect(link).To(Equal(filepath.Join(layerPath, "shared", "Microsoft.NETCore.App")))

			link, err = os.Readlink(filepath.Join(workingDir, ".dotnet_root", "host"))
			Expect(err).NotTo(HaveOccurred())
			Expect(link).To(Equal(filepath.Join(layerPath, "host")))
		})

		context("error cases", func() {
			context("when the '.dotnet_root/shared' dir can not be created", func() {
				it.Before(func() {
					Expect(os.Chmod(filepath.Join(workingDir), 0000)).To(Succeed())
				})
				it("errors", func() {
					err := symlinker.Link(workingDir, layerPath)
					Expect(err).To(HaveOccurred())
					Expect(err).To(MatchError(ContainSubstring("permission denied")))
				})
			})

			context("when the shared directory symlink can not be created", func() {
				it.Before(func() {
					Expect(os.MkdirAll(filepath.Join(workingDir, ".dotnet_root", "shared"), os.ModePerm)).To(Succeed())
					Expect(os.Chmod(filepath.Join(workingDir, ".dotnet_root", "shared"), 0000)).To(Succeed())
				})
				it("errors", func() {
					err := symlinker.Link(workingDir, layerPath)
					Expect(err).To(HaveOccurred())
					Expect(err).To(MatchError(ContainSubstring("permission denied")))
				})
			})

			context("when the host directory symlink can not be created", func() {
				it.Before(func() {
					Expect(os.MkdirAll(filepath.Join(workingDir, ".dotnet_root"), os.ModePerm)).To(Succeed())
					Expect(os.Chmod(filepath.Join(workingDir, ".dotnet_root"), 0000)).To(Succeed())
				})
				it("errors", func() {
					err := symlinker.Link(workingDir, layerPath)
					Expect(err).To(HaveOccurred())
					Expect(err).To(MatchError(ContainSubstring("permission denied")))
				})
			})
		})
	})
}
