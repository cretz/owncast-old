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

type patchableFile struct {
	path            string
	fileBytes       []byte
	fileMode        os.FileMode
	certIndex       int
	existingCertLen int
}

type PatchableLib interface {
	Path() string
	Patch(replacementRootCADERBytes []byte) error
}

type UnpatchableLib interface {
	Path() string
	OrigPath() string
	Unpatch() error
}

func FindPatchableLib(startDir string, existingRootCADERBytes []byte) (PatchableLib, error) {
	return findPatchableFile(startDir, existingRootCADERBytes, false)
}

func FindUnpatchableLib(startDir string, existingRootCADERBytes []byte) (UnpatchableLib, error) {
	return findPatchableFile(startDir, existingRootCADERBytes, true)
}

func findPatchableFile(startDir string, existingRootCADERBytes []byte, backup bool) (file *patchableFile, err error) {
	fileSuffix := sharedLibSuffix
	if backup {
		fileSuffix += ".bak"
	}

	err = filepath.Walk(startDir, func(path string, info os.FileInfo, err error) error {
		if !strings.HasSuffix(info.Name(), fileSuffix) {
			return nil
		} else if info.IsDir() {
			return nil
		} else if fileBytes, err := ioutil.ReadFile(path); err != nil {
			return err
		} else if certIndex := bytes.Index(fileBytes, existingRootCADERBytes); certIndex != -1 {
			file = &patchableFile{
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
	} else if file == nil {
		return nil, fmt.Errorf("Unable to find shared lib with cert")
	}
	return
}

func (p *patchableFile) Path() string     { return p.path }
func (p *patchableFile) OrigPath() string { return strings.TrimSuffix(p.path, ".bak") }

func (p *patchableFile) Patch(replacementRootCADERBytes []byte) error {
	if p.fileBytes == nil {
		return fmt.Errorf("Patch cannot be run a second time")
	}
	if p.existingCertLen != len(replacementRootCADERBytes) {
		return fmt.Errorf("Replacement byte size %v != existing byte size %v",
			len(replacementRootCADERBytes), p.existingCertLen)
	}
	// First make backup
	if err := ioutil.WriteFile(p.path+".bak", p.fileBytes, p.fileMode); err != nil {
		return fmt.Errorf("Unable to make backup: %v", err)
	}
	// Now replace the bytes and write
	for i, b := range replacementRootCADERBytes {
		p.fileBytes[p.certIndex+i] = b
	}
	err := ioutil.WriteFile(p.path, p.fileBytes, p.fileMode)
	p.fileBytes = nil
	if err != nil {
		return fmt.Errorf("Failed patching file %v", p.path)
	}
	return nil
}

func (p *patchableFile) Unpatch() error {
	// Put bytes back
	if err := ioutil.WriteFile(p.OrigPath(), p.fileBytes, p.fileMode); err != nil {
		return fmt.Errorf("Unable to overwrite existing file from backup: %v", err)
	}
	// Delete backup
	if err := os.Remove(p.path); err != nil {
		return fmt.Errorf("Unpatched, but unable to remove backup: %v", err)
	}
	return nil
}

var sharedLibSuffix string

func init() {
	switch runtime.GOOS {
	case "windows":
		sharedLibSuffix = ".dll"
	case "darwin":
		sharedLibSuffix = "Framework"
	// TODO: need to test others
	default:
		panic(fmt.Errorf("OS not supported yet: %v", runtime.GOOS))
	}
}
