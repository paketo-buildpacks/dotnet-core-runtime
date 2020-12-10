package fakes

import (
	"sync"

	"github.com/paketo-buildpacks/packit/postal"
)

type VersionResolver struct {
	ResolveCall struct {
		sync.Mutex
		CallCount int
		Receives  struct {
			Path    string
			Id      string
			Version string
			Stack   string
		}
		Returns struct {
			Dependency postal.Dependency
			Error      error
		}
		Stub func(string, string, string, string) (postal.Dependency, error)
	}
}

func (f *VersionResolver) Resolve(param1 string, param2 string, param3 string, param4 string) (postal.Dependency, error) {
	f.ResolveCall.Lock()
	defer f.ResolveCall.Unlock()
	f.ResolveCall.CallCount++
	f.ResolveCall.Receives.Path = param1
	f.ResolveCall.Receives.Id = param2
	f.ResolveCall.Receives.Version = param3
	f.ResolveCall.Receives.Stack = param4
	if f.ResolveCall.Stub != nil {
		return f.ResolveCall.Stub(param1, param2, param3, param4)
	}
	return f.ResolveCall.Returns.Dependency, f.ResolveCall.Returns.Error
}
