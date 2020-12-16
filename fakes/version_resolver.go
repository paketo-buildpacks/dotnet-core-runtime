package fakes

import (
	"sync"

	dotnetcoreruntime "github.com/paketo-buildpacks/dotnet-core-runtime"
	"github.com/paketo-buildpacks/packit"
	"github.com/paketo-buildpacks/packit/postal"
)

type VersionResolver struct {
	ResolveCall struct {
		sync.Mutex
		CallCount int
		Receives  struct {
			Path   string
			Entry  packit.BuildpackPlanEntry
			Stack  string
			Logger dotnetcoreruntime.LogEmitter
		}
		Returns struct {
			Dependency postal.Dependency
			Error      error
		}
		Stub func(string, packit.BuildpackPlanEntry, string, dotnetcoreruntime.LogEmitter) (postal.Dependency, error)
	}
}

func (f *VersionResolver) Resolve(param1 string, param2 packit.BuildpackPlanEntry, param3 string, param4 dotnetcoreruntime.LogEmitter) (postal.Dependency, error) {
	f.ResolveCall.Lock()
	defer f.ResolveCall.Unlock()
	f.ResolveCall.CallCount++
	f.ResolveCall.Receives.Path = param1
	f.ResolveCall.Receives.Entry = param2
	f.ResolveCall.Receives.Stack = param3
	f.ResolveCall.Receives.Logger = param4
	if f.ResolveCall.Stub != nil {
		return f.ResolveCall.Stub(param1, param2, param3, param4)
	}
	return f.ResolveCall.Returns.Dependency, f.ResolveCall.Returns.Error
}
