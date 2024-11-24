package golang

import (
	"archive/zip"
	"bytes"
	"cli/internal/common"
	"cli/internal/ecosystem/shared"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

// Fixing a dependency in go is done using the vendor folder, based on https://go.dev/ref/mod#vendoring
// First, we run `go mod vendor` to create a vendor directory with all dependencies
// Then, we replace the code of the dependency in the vendor directory with the sealed package
// We finish with a sealed vendor folder, so that following runs of `go build` will use the sealed packages
// If we fail, we remove the vendor folder entirely, so there's no effect to our changes
//
// If the vendor folder already exists, we will seal the code there, without running `go mod vendor` again
// In such case, on failure, we rollback the vendor folder to its original state
//
// We use this method to contain the sealing to be affect only on the folder that's being built.

const vendorDir = "vendor"

type fixer struct {
	rollback            map[string]string // original-dependency-path -> tmp-location
	projectDir          string
	workdir             string
	vendorAlreadyExists bool
	vendorDir           string
}

// Run `go mod vendor` to create a vendor directory with all dependencies
// do nothing if it exists
func (f *fixer) Prepare() error {
	// create workdir
	if err := os.MkdirAll(f.workdir, 0755); err != nil {
		slog.Error("failed creating workdir", "workdir", f.workdir, "err", err)
		return err
	}

	f.vendorDir = filepath.Join(f.projectDir, vendorDir)
	exists, err := common.DirExists(f.vendorDir)
	if err != nil {
		slog.Error("failed checking if vendor directory exists", "err", err)
		return err
	}

	if exists {
		slog.Info("vendor directory already exists, will not create", "vendorDir", f.vendorDir)
		f.vendorAlreadyExists = true
		return nil
	}

	slog.Info("running go mod vendor", "vendorDir", f.vendorDir)
	pr, err := common.RunCmdWithArgs(f.projectDir, "go", "mod", "vendor")
	if err != nil {
		slog.Error("failed running go mod vendor", "err", err)
		return err
	}
	if pr.Code != 0 {
		slog.Error("running go mod vendor returned non-zero", "result", pr)
		return fmt.Errorf("running go mod vendor returned non-zero")
	}

	return nil
}

// files in zip include the module's version, but should appear without it in the vendor folder
// e.g. github.com/Masterminds/goutils@v1.1.0-sp1/wordutils.go -> github.com/Masterminds/goutils/wordutils.go
func removeVersionPath(path, version string) string {
	return strings.Replace(path, "@v"+version, "", -1)

}

// Fix a dependency by replacing the code in the vendor directory
// Store a copy of the original dependency in the workdir for rollback
func (f *fixer) Fix(entry shared.DependencyDescriptor, dep *common.Dependency, packageData []byte, fileName string) (bool, string, error) {
	origDepDirPath := filepath.Join(f.vendorDir, dep.Name)
	tmpDepDirPath := filepath.Join(f.workdir, dep.Name)

	slog.Debug("backing up dependency dir", "orig", origDepDirPath, "tmp", tmpDepDirPath)
	tmpParentDir := filepath.Dir(tmpDepDirPath)
	if err := os.MkdirAll(tmpParentDir, 0755); err != nil {
		slog.Error("failed creating tmp dir", "dir", tmpDepDirPath, "err", err)
		return false, "", err
	}

	if err := common.Move(origDepDirPath, tmpDepDirPath); err != nil {
		slog.Error("failed moving original version dir to tmp", "orig", origDepDirPath, "tmp", tmpDepDirPath, "err", err)
		return false, "", common.NewPrintableError("failed backing up the original version for: %s", origDepDirPath)
	}

	f.rollback[origDepDirPath] = tmpDepDirPath

	r, err := zip.NewReader(bytes.NewReader(packageData), int64(len(packageData)))
	if err != nil {
		slog.Error("failed reading package", "err", err, "availableFix", entry.AvailableFix.Id(), "packageDataLen", len(packageData))
		return false, "", err
	}

	for _, file := range r.File {
		file.Name = removeVersionPath(file.Name, entry.AvailableFix.Version)

		common.Trace("extracting file", "file", file.Name)
		err = common.UnzipFile(file, f.vendorDir)
		if err != nil {
			return false, "", err
		}
	}

	return true, dep.DiskPath, nil
}

// Rollback the changes made to the vendor directory
// If it already existed before the fix, rollback each dependency to previous state
// Otherwise, remove it entirely
func (f *fixer) Rollback() bool {
	if !f.vendorAlreadyExists {
		// remove `vendor` folder entirely
		slog.Info("rollback, removing vendor directory", "vendorDir", f.vendorDir)
		err := os.RemoveAll(f.vendorDir)
		if err != nil {
			slog.Error("failed removing vendor directory", "err", err)
			return false
		}
	} else {
		// need to rollback each dependency to previous state
		for orig, tmp := range f.rollback {
			if err := os.RemoveAll(orig); err != nil {
				slog.Error("failed removing original version dir", "dir", orig)
			}

			if err := common.Move(tmp, orig); err != nil {
				slog.Error("failed renaming tmp to original version dir", "tmp", tmp, "orig", orig)
			}
		}
	}

	return true
}

// Remove workdir
func (f *fixer) Cleanup() bool {
	if err := os.RemoveAll(f.workdir); err != nil {
		slog.Error("failed removing tmp dir", "dir", f.workdir, "err", err)
		return false
	}

	return true
}

func NewFixer(projectDir string, workdir string) shared.DependencyFixer {
	return &fixer{
		projectDir: projectDir,
		workdir:    workdir,
		rollback:   make(map[string]string, 100),
	}
}
