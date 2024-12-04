package dotnet

import (
	"cli/internal/common"
	"cli/internal/ecosystem/msil/utils"
	"cli/internal/ecosystem/shared"
	"fmt"
	"log/slog"
	"path/filepath"
)

type fixer struct {
	workdir     string // used to place nupkgs, treated as Source for dotnet add command (does not need to be structured like global-packages)
	projectFile string
	targetDir   string
	packagesDir string
}

func (*fixer) Cleanup() bool {
	return true // No cleanup needed
}

func (f *fixer) Prepare() error {
	return nil
}

// equivalent of running:
// >   dotnet add {project-file} package {package-id} --version {version} --source {folder-with-nupkgs}
// ref: https://learn.microsoft.com/en-us/dotnet/core/tools/dotnet-add-package
func (f *fixer) dotnetAdd(library string, version string) error {
	// running from solution dir just in case

	res, err := common.RunCmdWithArgs(f.targetDir,
		dotnetExeName,
		"add", f.projectFile,
		"package", library,
		"--version", version,
		"--source", f.workdir,
	)

	if err != nil {
		slog.Error("error while running dotnet add", "err", err)
		return err
	}

	if res.Code != 0 {
		slog.Error("failed running dotnet add", "err", err, "library", library, "version", version)
		return fmt.Errorf("dotnet error code failure %d", res.Code)
	}

	return nil
}

// same folder as the one nuget cli uses, but for dotnet it uses a newer format:
//   - folders for dependencies are using lower case
//   - versions are handled as a sub-folder
//
// e.g.:
// {cache}/packages/newtonsoft.json/12.0.0
func formatCachePackagePath(packagesDir string, library string, version string) string {
	return filepath.Join(packagesDir, utils.NormalizeName(library), utils.NormalizeName(version))
}

// saves the package file into `<workdir>/`
// dotnet lets us specify a directory with .nupkg files as source, without extracting them to the packages format
// instead of editing the project we just add our fixed version, which should override existing dependency
func (f *fixer) Fix(entry shared.DependencyDescriptor, dep *common.Dependency, packageData []byte, fileName string) (bool, string, error) {
	if fileName == "" {
		// should not happen
		slog.Error("empty artifact name", "id", entry.AvailableFix.Id())
		return false, "", fmt.Errorf("downloaded artifact name is empty")
	}

	pkgPath := filepath.Join(f.workdir, fileName)
	if err := common.DumpBytes(pkgPath, packageData); err != nil {
		slog.Error("failed dumping fixed data dotnet package", "err", err, "size", len(packageData), "path", pkgPath, "id", entry.AvailableFix.Id())
		return false, "", err
	}

	fixedLibName := entry.AvailableFix.Library.Name
	fixedVersion := entry.AvailableFix.Version

	if err := f.dotnetAdd(fixedLibName, fixedVersion); err != nil {
		slog.Error("failed adding package to source", "path", pkgPath, "id", entry.AvailableFix.Id())
		return false, "", err
	}

	// return the path to the new package, as it is not replacing the original, but sits next to it
	fixedPath := formatCachePackagePath(f.packagesDir, fixedLibName, fixedVersion)
	if exists, err := common.DirExists(fixedPath); !exists || err != nil {
		// should not happen, but maybe if environment variables set / NuGet config
		// only warn for now
		slog.Warn("could not find fixed package path in packages dir", "err", err, "exists", exists, "path", fixedPath, "packages-dir", f.packagesDir)
	}

	slog.Info("returning new fixed location", "path", fixedPath)
	return true, fixedPath, nil
}

func (*fixer) Rollback() bool {
	return true // No need to rollback files, we dont modify the source version
}

// packagesDir is the cache where all the packages are stored
func newFixer(workdir string, projectFile string, targetDir string, packagesDir string) shared.DependencyFixer {
	return &fixer{
		workdir:     workdir,
		projectFile: projectFile,
		targetDir:   targetDir,
		packagesDir: packagesDir,
	}
}
