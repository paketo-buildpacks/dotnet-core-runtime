package fakes

import (
	"sync"

	"github.com/paketo-buildpacks/packit"
	"github.com/paketo-buildpacks/packit/postal"
)

type DependencyManager struct {
	GenerateBillOfMaterialsCall struct {
		mutex     sync.Mutex
		CallCount int
		Receives  struct {
			Dependencies []postal.Dependency
		}
		Returns struct {
			BOMEntrySlice []packit.BOMEntry
		}
		Stub func(...postal.Dependency) []packit.BOMEntry
	}
	InstallCall struct {
		mutex     sync.Mutex
		CallCount int
		Receives  struct {
			Dependency postal.Dependency
			CnbPath    string
			LayerPath  string
		}
		Returns struct {
			Error error
		}
		Stub func(postal.Dependency, string, string) error
	}
}

func (f *DependencyManager) GenerateBillOfMaterials(param1 ...postal.Dependency) []packit.BOMEntry {
	f.GenerateBillOfMaterialsCall.mutex.Lock()
	defer f.GenerateBillOfMaterialsCall.mutex.Unlock()
	f.GenerateBillOfMaterialsCall.CallCount++
	f.GenerateBillOfMaterialsCall.Receives.Dependencies = param1
	if f.GenerateBillOfMaterialsCall.Stub != nil {
		return f.GenerateBillOfMaterialsCall.Stub(param1...)
	}
	return f.GenerateBillOfMaterialsCall.Returns.BOMEntrySlice
}
func (f *DependencyManager) Install(param1 postal.Dependency, param2 string, param3 string) error {
	f.InstallCall.mutex.Lock()
	defer f.InstallCall.mutex.Unlock()
	f.InstallCall.CallCount++
	f.InstallCall.Receives.Dependency = param1
	f.InstallCall.Receives.CnbPath = param2
	f.InstallCall.Receives.LayerPath = param3
	if f.InstallCall.Stub != nil {
		return f.InstallCall.Stub(param1, param2, param3)
	}
	return f.InstallCall.Returns.Error
}
