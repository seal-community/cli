package utils

import (
	"cli/internal/common"
	"cli/internal/ecosystem/shared"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
)

type fixer struct {
	rollback   map[string]string // original version path -> tmp-location
	projectDir string
	workdir    string
	cacheDir   string
}

func prepareCacheDir(newCacheDir string, oldCacheDir string) error {
	// Clean the new cache dir
	err := os.RemoveAll(newCacheDir)
	if err != nil {
		slog.Error("failed cleaning new cache dir", "dir", newCacheDir)
		return common.NewPrintableError("failed cleaning new cache dir")
	}

	// Copy the old cache to the new cache to save time
	if oldCacheDir != "" {
		err = common.CopyDir(oldCacheDir, newCacheDir)
		if err != nil {
			return err
		}
	}

	return nil
}

func NewFixer(projectDir string, workdir string, sealCacheDir string) shared.DependencyFixer {
	return &fixer{
		projectDir: projectDir,
		workdir:    workdir,
		rollback:   make(map[string]string, 100),
		cacheDir:   sealCacheDir,
	}
}

func (f *fixer) Prepare() error {
	currCacheDir := GetCacheDir(f.projectDir)
	if currCacheDir == "" {
		slog.Warn("failed getting maven cache dir")
		return common.NewPrintableError("failed getting maven cache dir")
	}
	if currCacheDir != f.cacheDir {
		slog.Info("preparing maven cache dir", "dir", f.cacheDir)
		err := prepareCacheDir(f.cacheDir, currCacheDir)
		if err != nil {
			return err
		}
	}
	return nil
}

func (f *fixer) Fix(entry shared.DependnecyDescriptor, dep *common.Dependency, packageData []byte) (bool, error) {
	// copy a backup for the original version to the workdir (tmp location)
	// override the jar file in the cache dir (will set it as the cache in HandleFixes)
	// add to the rollback map
	_, ArtifactName, err := SplitJavaPackageName(dep.Name)
	if err != nil {
		return false, err
	}

	origVersionDirPath := GetJavaPackagePath(f.cacheDir, dep.Name, dep.Version)
	if origVersionDirPath == "" {
		return false, fmt.Errorf("failed getting package path in the cache")
	}

	tmpVersionDirPath := GetJavaPackagePath(f.workdir, dep.Name, dep.Version)
	if tmpVersionDirPath == "" {
		return false, fmt.Errorf("failed getting temp storage path for package")
	}

	err = common.CopyDir(origVersionDirPath, tmpVersionDirPath)
	if err != nil {
		slog.Error("failed copying original version dir to tmp", "orig", origVersionDirPath, "tmp", tmpVersionDirPath)
		return false, common.NewPrintableError("failed backing up the original version for: %s", origVersionDirPath)
	}

	f.rollback[origVersionDirPath] = tmpVersionDirPath

	jarPath := filepath.Join(origVersionDirPath, GetPackageFileName(ArtifactName, dep.Version))
	slog.Info("writing to jar file", "path", jarPath)
	jarFile, err := os.OpenFile(jarPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return false, fmt.Errorf("failed creating jar file: %w", err)
	}
	defer jarFile.Close()

	_, err = jarFile.Write(packageData)
	if err != nil {
		return false, fmt.Errorf("failed writing to jar file: %w", err)
	}

	return true, nil
}

func (f *fixer) Rollback() bool {
	// go over the rollback map and move the original versions saved in the tmp location
	// to the original location to undo the fix
	for orig, tmp := range f.rollback {
		if err := os.RemoveAll(orig); err != nil {
			slog.Error("failed removing original version dir", "dir", orig)
		}

		if err := os.Rename(tmp, orig); err != nil {
			slog.Error("failed renaming tmp to original version dir", "tmp", tmp, "orig", orig)
		}
	}
	return true
}

func (f *fixer) Cleanup() bool {
	// remove the tmp dir (workdir) as we succeeded fixing and we don't need it anymore
	if err := os.RemoveAll(f.workdir); err != nil {
		slog.Error("failed removing tmp dir", "dir", f.workdir)
	}
	return true
}
