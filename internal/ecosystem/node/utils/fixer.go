package utils

import (
	"bytes"
	"cli/internal/common"
	"cli/internal/ecosystem/shared"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

func getDepRollbackDir(projectDir string, tempDir string, depDir string) (string, error) {
	if projectDir[len(projectDir)-1] != os.PathSeparator {
		projectDir = projectDir + string(os.PathSeparator)
	}

	if !strings.HasPrefix(depDir, projectDir) {
		return "", fmt.Errorf("dep directory %s  not in project dir %s", depDir, projectDir)
	}

	suffix := strings.TrimPrefix(depDir, projectDir)
	target := filepath.Join(tempDir, suffix)

	return target, nil
}

type fixer struct {
	rollback   map[string]string // original-dependency-path -> tmp-location
	projectDir string
	workdir    string
}

func NewFixer(projectDir string, workdir string) shared.DependencyFixer {
	return &fixer{
		projectDir: projectDir,
		workdir:    workdir,
		rollback:   make(map[string]string, 100),
	}
}

func containsNestedNodeModules(path string) bool {
	nodeModules := filepath.Join(path, nodeModulesDirName)
	_, err := os.Stat(nodeModules)
	return !os.IsNotExist(err) // might FP if had other error
}

func moveNodeModules(src string, target string) error {
	srcModules := filepath.Join(src, nodeModulesDirName)
	targetModules := filepath.Join(target, nodeModulesDirName)
	if err := os.Rename(srcModules, targetModules); err != nil {
		slog.Error("failed moving node_modules dir", "src", srcModules, "target", targetModules)
		return err
	}

	return nil
}

func createDepRollbackDir(target string, dep *common.Dependency) error {

	if _, err := os.Stat(target); err == nil || !os.IsNotExist(err) {
		slog.Error("tmp name already exists or failed checking for it", "err", err)
		return err
	}
	slog.Debug("got new temp path for dep", "path", dep.DiskPath, "id", dep.Id(), "tmp-path", target)

	if err := os.MkdirAll(filepath.Dir(target), os.ModePerm); err != nil {
		slog.Error("failed making target", "err", err, "path", target)
		return common.NewPrintableError("failed settup up backup for package %s", dep.PrintableName())
	}

	return nil
}
func (f *fixer) Fix(dep *common.Dependency, packageDownload shared.PackageDownload) (bool, error) {

	if _, ok := f.rollback[dep.DiskPath]; ok {
		// will have issue in future with N branches that have been dedup'd so that they both point to the same
		// physical path and we will want to update one but not the other:
		//		A -> B -> C   : we want to patch since C's patch is breaking but not for B's usage
		//		D -> E -> C   : we don't want to patch
		// also this is touchy with current logic of rollback
		slog.Warn("dup already patched", "id", dep.Id(), "original-path", dep.DiskPath)
		return false, nil
	}

	tmpName, err := getDepRollbackDir(f.projectDir, f.workdir, dep.DiskPath)
	if err != nil {
		slog.Error("failed formatting temp dir for dep", "err", err, "original", dep.DiskPath, "project-dir", f.projectDir)
		return false, err
	}

	if err := createDepRollbackDir(tmpName, dep); err != nil {
		return false, err
	}

	slog.Debug("moving dep to seal tmp", "from", dep.DiskPath, "to", tmpName)
	if err := os.Rename(dep.DiskPath, tmpName); err != nil {
		slog.Error("failed moving original to temp dir", "err", err, "original", dep.DiskPath, "tmp-path", tmpName)
		return false, common.NewPrintableError("failed backing up package %s", dep.PrintableName())
	}

	f.rollback[dep.DiskPath] = tmpName

	// requires since it was moved
	err = os.Mkdir(dep.DiskPath, os.ModePerm)
	if err != nil {
		slog.Error("failed recreating original path for dep", "err", err, "original", dep.DiskPath)
		return false, common.NewPrintableError("failed creating package folder for %s", dep.PrintableName())
	}

	err = UntarNpmPackage(bytes.NewReader(packageDownload.Data), dep.DiskPath)
	if err != nil {
		slog.Error("failed untaring package", "err", err, "target", dep.DiskPath, "payloadLen", len(packageDownload.Data))
		return false, common.NewPrintableError("failed applying fix for package %s", dep.PrintableName())
	}

	if containsNestedNodeModules(tmpName) {
		// either dedupd or legacy mode, or brought node_modules in published package
		// e.g. $projectDir/node_modules/depA/node_modules/depB
		slog.Warn("moving nested node_modules dir back from seal dir", "path", dep.DiskPath)
		if containsNestedNodeModules(dep.DiskPath) {
			// we might encounter conflict if package published theirs with node modules
			slog.Warn("fixed package already contains node_modules directory", "path", dep.DiskPath)
			return false, nil
		}

		if err := moveNodeModules(tmpName, dep.DiskPath); err != nil {
			return false, err
		}
	}

	slog.Info("fixed package instance", "path", dep.DiskPath)
	return true, nil
}

func (f *fixer) rollbackDependecy(from string, to string) error {
	slog.Debug("rolling back", "from", from, "to", to)
	// since we keep the original node_modules folder after installing our fixed version
	// we need to 'spare' it from removal when rolling back
	if containsNestedNodeModules(from) {
		slog.Info("keeping node_modules while rolling back", "from", from, "to", to)
		if err := moveNodeModules(to, from); err != nil {
			return err
		}
	}

	_ = os.RemoveAll(to) // delete the patched contents, ignore if doesn't exist, other failure will cause rename to fail as well
	if err := os.Rename(from, to); err != nil {
		slog.Error("failed rollback", "err", err, "from", from, "to", to)
		// greedy try to restore as much as possible
		return err
	}

	return nil
}

func (f *fixer) Rollback() bool {
	finishedOk := true
	for orig, tmpName := range f.rollback {
		if err := f.rollbackDependecy(tmpName, orig); err != nil {
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
