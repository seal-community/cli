package gradle

import (
	"cli/internal/common"
	"cli/internal/ecosystem/java/utils"
	"cli/internal/ecosystem/shared"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

func (f *GradlePackageManager) Prepare() error {
	if !f.useGradlew {
		slog.Error("gradle fixer only works with gradlew")
		return fmt.Errorf("gradle fixer only works with gradlew")
	}

	if f.privateHomeDir == f.originalHomeDir {
		slog.Info("gradle cache dir is the same as the original dir")
		return nil
	}

	defer common.ExecutionTimer().Log()
	slog.Info("cleaning old seal cache dir", "dir", f.privateHomeDir)
	if err := os.RemoveAll(f.privateHomeDir); err != nil {
		slog.Error("failed cleaning dir", "dir", f.privateHomeDir, "err", err)
		return common.NewPrintableError("failed cleaning new cache dir")
	}

	slog.Info("preparing gradle cache dir", "dir", f.privateHomeDir)
	if err := os.Mkdir(f.privateHomeDir, os.ModePerm); err != nil {
		slog.Error("failed making cache dir", "dir", f.privateHomeDir, "err", err)
		return err
	}

	err := utils.CreateRecursiveLinkTree(f.originalHomeDir, f.privateHomeDir)
	if err != nil {
		slog.Error("failed making copy of gradle cache tree dir", "err", err, "from", f.originalHomeDir, "to", f.privateHomeDir)
	}

	return err
}

func (f *GradlePackageManager) backupFixedFile(artifactPath string, backupPath string) error {
	backupDir := filepath.Dir(backupPath)
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		slog.Error("failed making backup dirs", "err", err, "path", backupDir)
		return err
	}

	if err := common.Move(artifactPath, backupPath); err != nil {
		slog.Error("failed renaming artifact", "err", err, "from", artifactPath, "to", backupPath)
		return err
	}

	return nil
}

// copy a backup for the original artifact to the workdir (tmp location)
// override the artifact file in the cache dir (will set it as the cache in HandleFixes)
// add to the rollback map
func (f *GradlePackageManager) Fix(entry shared.DependencyDescriptor, dep *common.Dependency, packageData []byte, fileName string) (bool, string, error) {
	if !strings.HasPrefix(dep.DiskPath, f.originalHomeDir) {
		slog.Error("artifact path is not in the gradle cache dir", "path", dep.DiskPath)
		return false, "", fmt.Errorf("artifact path is not in the gradle cache dir")
	}

	artifactPath := strings.Replace(dep.DiskPath, f.originalHomeDir, f.privateHomeDir, 1)

	backupPath := strings.Replace(dep.DiskPath, f.originalHomeDir, f.workdir, 1)
	if err := f.backupFixedFile(artifactPath, backupPath); err != nil {
		slog.Error("failed backing up original version", "err", err, "path", artifactPath)
		return false, "", err
	}
	f.rollback[artifactPath] = backupPath

	slog.Info("writing to jar file", "path", artifactPath)
	if err := common.DumpBytes(artifactPath, packageData); err != nil {
		slog.Error("failed writing to jar file", "path", artifactPath, "err", err)
		return false, "", err
	}

	return true, dep.DiskPath, nil
}

func (f *GradlePackageManager) Rollback() bool {
	// go over the rollback map and move the original versions saved in the tmp location
	// to the original location to undo the fix
	for orig, tmp := range f.rollback {
		if err := os.RemoveAll(orig); err != nil {
			slog.Error("failed removing original version dir", "dir", orig)
		}

		if err := common.Move(tmp, orig); err != nil {
			slog.Error("failed renaming tmp to original version dir", "tmp", tmp, "orig", orig)
		}
	}
	return true
}

func (f *GradlePackageManager) Cleanup() bool {
	// remove the tmp dir (workdir) as we succeeded fixing and we don't need it anymore
	if err := os.RemoveAll(f.workdir); err != nil {
		slog.Error("failed removing tmp dir", "dir", f.workdir, "err", err)
		return false
	}
	return true
}
