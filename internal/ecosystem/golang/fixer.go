package golang

import (
	"archive/zip"
	"bytes"
	"cli/internal/common"
	"cli/internal/ecosystem/shared"
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
	tmpGoModPath        string
}

func (f *fixer) saveGoModFile() error {
	origModPath := filepath.Join(f.projectDir, "go.mod")
	tmpGoModPath := filepath.Join(f.workdir, "go.mod")
	err := common.CopyFile(origModPath, tmpGoModPath)
	if err != nil {
		return err
	}

	f.tmpGoModPath = tmpGoModPath
	return nil
}

// Run `go mod vendor` to create a vendor directory with all dependencies
// do nothing if it exists
func (f *fixer) Prepare() error {
	// create workdir
	if err := os.MkdirAll(f.workdir, 0755); err != nil {
		slog.Error("failed creating workdir", "workdir", f.workdir, "err", err)
		return err
	}

	err := f.saveGoModFile()
	if err != nil {
		slog.Error("failed copying go.mod file", "err", err)
		return err
	}

	err = PrepareVendorDir(f.projectDir)
	if err != nil {
		slog.Error("failed preparing vendor dir", "err", err)
	}

	return err
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
	success := true

	goModPath := filepath.Join(f.projectDir, goModFilename)
	err := common.CopyFile(f.tmpGoModPath, goModPath)
	if err != nil {
		slog.Error("failed rolling back go.mod file", "err", err) // Try and rollback the other changes
		success = false
	}

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
				success = false
			}

			if err := common.Move(tmp, orig); err != nil {
				slog.Error("failed renaming tmp to original version dir", "tmp", tmp, "orig", orig)
				success = false
			}
		}
	}

	return success
}

// Remove workdir
func (f *fixer) Cleanup() bool {
	if err := os.RemoveAll(f.workdir); err != nil {
		slog.Error("failed removing tmp dir", "dir", f.workdir, "err", err)
		return false
	}

	return true
}

func newFixer(projectDir string, workdir string, vendorDirPath string, vendorAlreadyExists bool) shared.DependencyFixer {
	return &fixer{
		projectDir:          projectDir,
		workdir:             workdir,
		rollback:            make(map[string]string, 100),
		vendorDir:           vendorDirPath,
		vendorAlreadyExists: vendorAlreadyExists,
	}
}
