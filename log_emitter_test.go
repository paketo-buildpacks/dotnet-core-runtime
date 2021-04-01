package dotnetcoreruntime_test

import (
	"bytes"
	"testing"

	dotnetcoreruntime "github.com/paketo-buildpacks/dotnet-core-runtime"
	"github.com/paketo-buildpacks/packit"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testLogEmitter(t *testing.T, context spec.G, it spec.S) {
	var (
		Expect = NewWithT(t).Expect

		buffer  *bytes.Buffer
		emitter dotnetcoreruntime.LogEmitter
	)

	it.Before(func() {
		buffer = bytes.NewBuffer(nil)
		emitter = dotnetcoreruntime.NewLogEmitter(buffer)
	})

	context("Environment", func() {
		it("prints details about the environment", func() {
			emitter.Environment(packit.Environment{
				"GEM_PATH.override": "/some/path",
			})

			Expect(buffer.String()).To(ContainSubstring("    GEM_PATH -> \"/some/path\""))
		})
	})
}
