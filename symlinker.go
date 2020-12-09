package dotnetcoreruntime

import (
	"os"
	"path/filepath"
)

type Symlinker struct{}

func NewSymlinker() Symlinker {
	return Symlinker{}
}

func (s Symlinker) Link(workingDir, layerPath string) error {
	err := os.MkdirAll(filepath.Join(workingDir, ".dotnet_root", "shared"), os.ModePerm)
	if err != nil {
		return err
	}

	err = os.Symlink(filepath.Join(layerPath, "shared", "Microsoft.NETCore.App"), filepath.Join(workingDir, ".dotnet_root", "shared", "Microsoft.NETCore.App"))
	if err != nil {
		return err
	}

	err = os.Symlink(filepath.Join(layerPath, "host"), filepath.Join(workingDir, ".dotnet_root", "host"))
	if err != nil {
		return err
	}

	return nil
}
