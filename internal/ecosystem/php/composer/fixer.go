package composer

import (
	"archive/zip"
	"bytes"
	"cli/internal/common"
	"cli/internal/ecosystem/shared"
	"log/slog"
	"os"
	"path/filepath"
)

type fixer struct {
	rollback  map[string]string
	vendorDir string
	workdir   string
}

func NewFixer(projectDir string, workdir string) shared.DependencyFixer {
	return &fixer{
		vendorDir: filepath.Join(projectDir, composerModulesDirName),
		workdir:   workdir,
		rollback:  make(map[string]string, 100),
	}
}

func (f *fixer) Prepare() error {
	return nil
}

// Fix a dependency by replacing the code in the vendor directory
// Store a copy of the original dependency in the workdir for rollback
func (f *fixer) Fix(entry shared.DependencyDescriptor, dep *common.Dependency, packageData []byte) (bool, error) {
	origDepDirPath := dep.DiskPath
	tmpDepDirPath := filepath.Join(f.workdir, dep.Name)

	slog.Debug("backing up dependency dir", "orig", origDepDirPath, "tmp", tmpDepDirPath)
	tmpParentDir := filepath.Dir(tmpDepDirPath)
	if err := os.MkdirAll(tmpParentDir, 0755); err != nil {
		slog.Error("failed creating tmp dir", "dir", tmpDepDirPath, "err", err)
		return false, err
	}

	if err := common.Move(origDepDirPath, tmpDepDirPath); err != nil {
		slog.Error("failed moving original version dir to tmp", "orig", origDepDirPath, "tmp", tmpDepDirPath, "err", err)
		return false, common.NewPrintableError("failed backing up the original version for: %s", origDepDirPath)
	}

	f.rollback[origDepDirPath] = tmpDepDirPath

	r, err := zip.NewReader(bytes.NewReader(packageData), int64(len(packageData)))
	if err != nil {
		slog.Error("failed reading package", "err", err, "availableFix", entry.AvailableFix.Id(), "packageDataLen", len(packageData))
		return false, err
	}

	for _, file := range r.File {
		common.Trace("extracting file", "file", file.Name)
		if err = common.UnzipFile(file, f.vendorDir); err != nil {
			slog.Error("failed extracting file", "file", file.Name, "err", err)
			return false, err
		}
	}

	return true, nil
}

func rollbackDependency(from string, to string) error {
	slog.Debug("rolling back", "from", from, "to", to)
	if err := os.RemoveAll(to); err != nil {
		slog.Error("failed removing original version dir", "dir", to, "err", err)
	}

	if err := common.Move(from, to); err != nil {
		slog.Error("failed rollback", "err", err, "from", from, "to", to)
		return err
	}

	return nil
}

func (f *fixer) Rollback() bool {
	finishedOk := true
	for orig, tmpName := range f.rollback {
		if err := rollbackDependency(tmpName, orig); err != nil {
			slog.Error("failed rollback", "err", err, "from", tmpName, "to", orig)
			finishedOk = false
		}
	}

	return finishedOk
}

func (f *fixer) Cleanup() bool {
	finishedOk := true
	for orig, tmpName := range f.rollback {
		if err := os.RemoveAll(tmpName); err != nil {
			slog.Error("failed removing tmp dir", "orig", orig, "tmp", tmpName)
			finishedOk = false
		}
	}

	return finishedOk
}
