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
		layerPath  string
		dotnetRoot string
	)

	it.Before(func() {
		var err error

		layerPath, err = ioutil.TempDir("", "layer-path")
		Expect(err).NotTo(HaveOccurred())

		err = ioutil.WriteFile(filepath.Join(layerPath, "testFile"), nil, 0644)
		Expect(err).NotTo(HaveOccurred())

		dotnetRoot, err = ioutil.TempDir("", ".dotnet_root")
		Expect(err).NotTo(HaveOccurred())

		symlinker = dotnetcoreruntime.NewSymlinker()
	})

	it.After(func() {
		Expect(os.RemoveAll(layerPath)).To(Succeed())
		Expect(os.RemoveAll(dotnetRoot)).To(Succeed())
	})

	context("Link", func() {
		it("creates a symlink from the layerpath to the .dotnet_root", func() {
			err := symlinker.Link(layerPath, dotnetRoot)
			Expect(err).NotTo(HaveOccurred())

			fi, err := os.Lstat(filepath.Join(dotnetRoot, "testFile"))
			Expect(err).NotTo(HaveOccurred())
			Expect(fi.Mode() & os.ModeSymlink).ToNot(BeZero())

			link, err := os.Readlink(filepath.Join(dotnetRoot, "testFile"))
			Expect(err).NotTo(HaveOccurred())
			Expect(link).To(Equal(filepath.Join(layerPath, "testFile")))
		})

		context("error cases", func() {
			context("when the symlink can not be created", func() {
				it.Before(func() {
					Expect(os.Chmod(filepath.Join(dotnetRoot), 0000)).To(Succeed())
				})
				it("errors", func() {
					err := symlinker.Link(layerPath, dotnetRoot)
					Expect(err).To(HaveOccurred())
					Expect(err).To(MatchError(ContainSubstring("permission denied")))
				})
			})
		})
	})
}
