package dotnetcoreruntime

import (
	"io"
	"time"

	"github.com/paketo-buildpacks/packit"
	"github.com/paketo-buildpacks/packit/postal"
	"github.com/paketo-buildpacks/packit/scribe"
)

type LogEmitter struct {
	// Emitter is embedded and therefore delegates all of its functions to the
	// LogEmitter.
	scribe.Emitter
}

func NewLogEmitter(output io.Writer) LogEmitter {
	return LogEmitter{
		Emitter: scribe.NewEmitter(output),
	}
}

func (e LogEmitter) SelectedDependency(entry packit.BuildpackPlanEntry, dependency postal.Dependency, now time.Time) {
	dependency.Name = dependency.ID
	e.Emitter.SelectedDependency(entry, dependency, now)
}

func (l LogEmitter) Environment(env packit.Environment) {
	l.Subprocess("%s", scribe.NewFormattedMapFromEnvironment(env))
	l.Break()
}
