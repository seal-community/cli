package maven

import (
	"cli/internal/common"
	"cli/internal/ecosystem/java/utils"
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

// creates a 'private' repo in the project's folder and uses symlinks to all the files in the original
func prepareCacheDir(newCacheDir string, oldCacheDir string) error {
	defer common.ExecutionTimer().Log()

	err := os.RemoveAll(newCacheDir)
	if err != nil {
		slog.Error("failed cleaning new cache dir", "dir", newCacheDir, "err", err)
		return common.NewPrintableError("failed cleaning new cache dir")
	}

	// Copy the old cache to the new cache to save time
	if oldCacheDir != "" {
		if err := os.Mkdir(newCacheDir, os.ModePerm); err != nil {
			slog.Error("failed making cache dir", "err", err, "path", newCacheDir)
			return err
		}

		if err := utils.CreateRecursiveLinkTree(oldCacheDir, newCacheDir); err != nil {
			slog.Error("failed making copy of m2 tree dir", "err", err, "from", oldCacheDir, "to", newCacheDir)
			return err
		}
	}

	return nil
}

func newFixer(projectDir string, workdir string, sealCacheDir string) shared.DependencyFixer {
	return &fixer{
		projectDir: projectDir,
		workdir:    workdir,
		rollback:   make(map[string]string, 100),
		cacheDir:   sealCacheDir,
	}
}

func (f *fixer) Prepare() error {
	currCacheDir := utils.GetCacheDir(f.projectDir)
	if currCacheDir == "" {
		slog.Warn("failed getting maven cache dir")
		return common.NewPrintableError("failed getting maven cache dir")
	}

	if currCacheDir == f.cacheDir {
		slog.Info("skipping cache dir setup, already set")
		return nil
	}

	slog.Info("preparing maven cache dir", "dir", f.cacheDir)
	err := prepareCacheDir(f.cacheDir, currCacheDir)

	return err
}

// copy a backup for the original artifact to the workdir (tmp location)
// override the artifact file in the cache dir (will set it as the cache in HandleFixes)
// add to the rollback map
func (f *fixer) Fix(entry shared.DependencyDescriptor, dep *common.Dependency, packageData []byte, fileName string) (bool, string, error) {
	_, artifactId, err := utils.SplitJavaPackageName(dep.NormalizedName)
	if err != nil {
		slog.Error("failed getting package name for dep", "err", err, "path", dep.Name)
		return false, "", err
	}

	origVersionDirPath := utils.GetJavaPackagePath(f.cacheDir, dep.Name, dep.Version)
	if origVersionDirPath == "" {
		return false, "", fmt.Errorf("failed getting package path in the cache")
	}

	tmpVersionDirPath := utils.GetJavaPackagePath(f.workdir, dep.Name, dep.Version)
	if tmpVersionDirPath == "" {
		return false, "", fmt.Errorf("failed getting temp storage path for package")
	}

	resolvedOriginDir, err := filepath.EvalSymlinks(origVersionDirPath)
	if err != nil {
		slog.Error("failed resolving symlink", "err", err, "path", origVersionDirPath)
		return false, "", err
	}

	if filepath.Clean(origVersionDirPath) != resolvedOriginDir {
		slog.Error("original artifact is symlink", "path", origVersionDirPath, "resolved", resolvedOriginDir)
		return false, "", common.NewPrintableError("cannot fix %s:%s - located behind a symbolic link", dep.Name, dep.Version)
	}

	artifactFileName := utils.GetPackageFileName(artifactId, dep.Version)
	bkupPath := filepath.Join(tmpVersionDirPath, artifactFileName)
	artifactPath := filepath.Join(origVersionDirPath, artifactFileName)

	if err := os.MkdirAll(tmpVersionDirPath, 0755); err != nil {
		slog.Error("failed making backup dirs", "err", err, "path", tmpVersionDirPath)
		return false, "", err
	}

	if err := common.Move(artifactPath, bkupPath); err != nil {
		slog.Error("failed renaming artifact", "err", err, "from", artifactPath, "to", bkupPath)
		return false, "", err
	}

	f.rollback[artifactPath] = bkupPath

	slog.Info("writing to jar file", "path", artifactPath)

	jarFile, err := os.OpenFile(artifactPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		slog.Error("failed creating file", "err", err, "path", jarFile)
		return false, "", fmt.Errorf("failed creating jar file: %w", err)
	}
	defer jarFile.Close()

	_, err = jarFile.Write(packageData)
	if err != nil {
		slog.Error("failed fixing package", "err", err, "path", jarFile)
		return false, "", fmt.Errorf("failed writing to jar file: %w", err)
	}

	return true, dep.DiskPath, nil
}

func (f *fixer) Rollback() bool {
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

func (f *fixer) Cleanup() bool {
	// remove the tmp dir (workdir) as we succeeded fixing and we don't need it anymore
	if err := os.RemoveAll(f.workdir); err != nil {
		slog.Error("failed removing tmp dir", "dir", f.workdir, "err", err)
		return false
	}
	return true
}
