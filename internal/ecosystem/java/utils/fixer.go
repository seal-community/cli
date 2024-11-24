package utils

import (
	"cli/internal/common"
	"cli/internal/ecosystem/shared"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/otiai10/copy"
)

type fixer struct {
	rollback   map[string]string // original version path -> tmp-location
	projectDir string
	workdir    string
	cacheDir   string
}

func copyTreeLinks(root string, targetRoot string) error {
	defer common.ExecutionTimer().Log()
	err := filepath.WalkDir(root, func(path string, de fs.DirEntry, err error) error {
		if err != nil {
			slog.Error("failed walkdir", "root", root, "path", path, "err", err)
			return err
		}

		if root == path {
			// is the root of the entire tree
			return nil
		}

		info, err := de.Info()
		if err != nil {
			slog.Error("failed getting info", "entry", de)
			return err
		}

		ft := de.Type()
		rel, err := filepath.Rel(root, path)
		if err != nil {
			slog.Error("failed getting rel path", "root", root, "path", path)
			return err
		}

		target := filepath.Join(targetRoot, rel)

		if info.Mode()&os.ModeSymlink != 0 {
			slog.Warn("found symlink - copyin as is", "path", path, "target", target)

			opts := copy.Options{
				PreserveTimes: true,
				PreserveOwner: true,
				OnSymlink: func(src string) copy.SymlinkAction {
					return copy.Shallow
				}}

			if err := copy.Copy(path, target, opts); err != nil {
				slog.Error("failed copying rel path", "target", target, "path", path)
				return err
			}

			return nil
		}

		if ft.IsDir() {
			if err := os.Mkdir(target, os.ModePerm); err != nil {
				slog.Error("failed making dir in target", "target", target)
				return err
			}

			return nil
		}

		if ft.IsRegular() {
			// file - link it
			if err := os.Symlink(path, target); err != nil {
				slog.Error("failed making symlink to file in target", "path", path, "target", target)
				return err
			}

			return nil
		}

		slog.Warn("unsupported dir entry type", "entry", de, "file-mode", ft)

		return nil
	})

	return err
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

		if err := copyTreeLinks(oldCacheDir, newCacheDir); err != nil {
			slog.Error("failed making copy of m2 tree dir", "err", err, "from", oldCacheDir, "to", newCacheDir)
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
	_, artifactId, err := SplitJavaPackageName(dep.NormalizedName)
	if err != nil {
		slog.Error("failed getting package name for dep", "err", err, "path", dep.Name)
		return false, "", err
	}

	origVersionDirPath := GetJavaPackagePath(f.cacheDir, dep.Name, dep.Version)
	if origVersionDirPath == "" {
		return false, "", fmt.Errorf("failed getting package path in the cache")
	}

	tmpVersionDirPath := GetJavaPackagePath(f.workdir, dep.Name, dep.Version)
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

	artifactFileName := GetPackageFileName(artifactId, dep.Version)
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
