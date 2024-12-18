package dpkgless

import (
	"bytes"
	"cli/internal/common"
	"cli/internal/ecosystem/shared"
	"log/slog"
	"os"
	"path/filepath"
)

type fixer struct {
	workDir    string
	backupPath string
}

func NewFixer(workDir string) shared.DependencyFixer {
	return &fixer{
		workDir:    workDir,
		backupPath: filepath.Join(workDir, ".seal.backup"),
	}
}

func (f *fixer) Prepare() error {
	if os.Geteuid() != 0 {
		slog.Error("non-root user trying to fix OS packages", "user", os.Getenv("USER"), "euid", os.Geteuid())
		return common.NewPrintableError("You must be root to fix OS packages")
	}

	return nil
}

// fix uninstalls the package and then installs the new package
// it will save the uninstalled files in a backup directory so they can be restored
// if anything goes wrong
func (f *fixer) Fix(entry shared.DependencyDescriptor, dep *common.Dependency, packageData []byte, fileName string) (bool, string, error) {
	err := UninstallDebPackage(dep.Name, f.backupPath)
	if err != nil {
		slog.Error("failed to uninstall package", "name", dep.Name, "err", err)
		return false, "", common.NewPrintableError("Failed to uninstall package %s", dep.Name)
	}

	packageReader := bytes.NewReader(packageData)

	err = InstallDebPackage(packageReader)
	if err != nil {
		slog.Error("failed to install package", "path", dep.Name, "err", err)
		return false, "", common.NewPrintableError("Failed to install package %s", dep.Name)
	}
	return true, "", nil // diskpath is empty for dpkg
}

func (f *fixer) Rollback() bool {
	// go over the files in backupPath and move them back to their original location
	err := filepath.Walk(f.backupPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			slog.Error("failed to walk backup path", "path", path, "err", err)
			return err
		}

		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(f.backupPath, path)
		if err != nil {
			slog.Error("failed to get relative path", "path", path, "backupPath", f.backupPath, "err", err)
			return err
		}

		slog.Debug("restoring file", "src", path)
		originalPath := filepath.Join("/", relPath)
		err = common.Move(path, originalPath)
		if err != nil {
			slog.Error("failed to move file", "src", path, "dst", originalPath, "err", err)
			return err
		}

		return nil
	})

	if err != nil {
		slog.Error("failed to walk backup path", "path", f.backupPath, "err", err)
		return false
	}

	return true
}

func (f *fixer) Cleanup() bool {
	slog.Debug("Cleaning up work dir", "dir", f.workDir)
	if err := os.RemoveAll(f.workDir); err != nil {
		slog.Error("failed removing tmp dir", "dir", f.workDir)
		return false
	}

	return true
}
