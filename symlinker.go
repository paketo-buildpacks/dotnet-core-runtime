package dotnetcoreruntime

import (
	"os"
	"path/filepath"
)

type Symlinker struct{}

func NewSymlinker() Symlinker {
	return Symlinker{}
}

func (s Symlinker) Link(layerPath, dotnetRoot string) error {
	// Grab all of the files from the layer path
	runtimeFiles, err := filepath.Glob(filepath.Join(layerPath, "*"))
	if err != nil {
		return err
	}

	// Create symlinks for each file
	for _, file := range runtimeFiles {
		err = os.Symlink(file, filepath.Join(dotnetRoot, filepath.Base(file)))
		if err != nil {
			return err
		}
	}
	return nil
}
