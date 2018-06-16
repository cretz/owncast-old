package chrome

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

type PatchableLib struct {
	path            string
	fileBytes       []byte
	fileMode        os.FileMode
	certIndex       int
	existingCertLen int
}

func FindPatchableLib(startDir string, existingRootCADERBytes []byte) (lib *PatchableLib, err error) {
	err = filepath.Walk(startDir, func(path string, info os.FileInfo, err error) error {
		if !strings.HasSuffix(info.Name(), sharedLibSuffix) {
			return nil
		} else if fileBytes, err := ioutil.ReadFile(path); err != nil {
			return err
		} else if certIndex := bytes.Index(fileBytes, existingRootCADERBytes); certIndex != -1 {
			lib = &PatchableLib{
				path:            path,
				fileBytes:       fileBytes,
				fileMode:        info.Mode(),
				certIndex:       certIndex,
				existingCertLen: len(existingRootCADERBytes),
			}
			return filepath.SkipDir
		}
		return nil
	})
	if err != nil && err != filepath.SkipDir {
		return nil, err
	} else if lib == nil {
		return nil, fmt.Errorf("Unable to find shared lib with cert")
	}
	return
}

func (p *PatchableLib) Patch(replacementRootCADERBytes []byte) error {
	if p.fileBytes == nil {
		return fmt.Errorf("Patch cannot be run a second time")
	}
	if p.existingCertLen != len(replacementRootCADERBytes) {
		return fmt.Errorf("Replacement not the same length as what it is replacing")
	}
	// First make backup
	if err := ioutil.WriteFile(p.path+".bak", p.fileBytes, 0777); err != nil {
		return fmt.Errorf("Unable to make backup: %v", err)
	}
	// Now replace the bytes and write
	for i, b := range replacementRootCADERBytes {
		p.fileBytes[p.certIndex+i] = b
	}
	err := ioutil.WriteFile(p.path, p.fileBytes, 0777)
	p.fileBytes = nil
	if err != nil {
		return fmt.Errorf("Failed patching file %v", p.path)
	}
	return nil
}

var sharedLibSuffix string

func init() {
	switch runtime.GOOS {
	case "windows":
		sharedLibSuffix = ".dll"
	// TODO: need to test others
	default:
		panic(fmt.Errorf("OS not supported yet: %v", runtime.GOOS))
	}
}
