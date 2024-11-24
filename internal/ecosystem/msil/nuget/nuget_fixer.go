package nuget

import (
	"cli/internal/common"
	"cli/internal/ecosystem/shared"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
)

type packagesConfFixer struct {
	// abs paths
	packagesDirPath   string
	packageConfigPath string
	solutionDirPath   string

	// paths generated during prepare;
	rootWorkdir        string // '.seal/.nuget/' - stores tmp fixed nupkg files for running `nuget add``
	privateSourcesPath string // '.seal/.nuget/packages' - stores our 'local' cache of fixed packages for running `nuget update`
}

func newFixer(
	workdir string, // .seal
	packagesDirPath string,
	packageConfigPath string,
	solutionDirPath string,
) *packagesConfFixer {

	ngtDir := filepath.Join(workdir, ".nuget")
	sourcesDir := filepath.Join(ngtDir, "packages")
	return &packagesConfFixer{
		packagesDirPath:    packagesDirPath,
		packageConfigPath:  packageConfigPath,
		privateSourcesPath: sourcesDir,
		rootWorkdir:        ngtDir,
		solutionDirPath:    solutionDirPath,
	}
}

func (f *packagesConfFixer) Prepare() error {
	if err := os.MkdirAll(f.privateSourcesPath, 0755); err != nil {
		slog.Error("failed creating sources dir", "path", f.privateSourcesPath, "err", err)
		return err
	}

	return nil
}

// equivalent of running:
// >  nuget add {.nupkg} -Source {sources dir}
func (f *packagesConfFixer) nugetAdd(packagePath string) error {
	// running from solution dir just in case
	res, err := common.RunCmdWithArgs(f.solutionDirPath,
		nugetExeName,
		"add", packagePath,
		"-Source", f.privateSourcesPath,
	)

	if err != nil {
		slog.Error("failed running nuget add", "err", err)
		return err
	}

	if res.Code != 0 {
		slog.Error("bad status code running nuget add", "code", res.Code, "package-path", packagePath)
		return fmt.Errorf("nuget error code failure %d", res.Code)
	}

	return nil
}

// equivalent of running:
// >  nuget update {packages.config file} -Id {lib} -Version {version}  -Source {sources dir} -RepositoryPath {packages dir} -Safe -PreRelease
// ref: https://learn.microsoft.com/en-us/nuget/reference/cli-reference/cli-ref-update
func (f *packagesConfFixer) nugetUpdate(library string, version string) error {
	// running from solution dir just in case
	res, err := common.RunCmdWithArgs(f.solutionDirPath,
		nugetExeName,
		"update", f.packageConfigPath,
		"-Id", library,
		"-Version", version,
		"-Source", f.privateSourcesPath,
		"-RepositoryPath", f.packagesDirPath,
		"-Safe",       // will not update if major/minor are not the same
		"-PreRelease", // IMPORTANT: otherwise won't apply our SP due to the hyphen
	)

	if err != nil {
		slog.Error("failed running nuget update", "err", err)
		return err
	}

	if res.Code != 0 {
		slog.Error("failed running nuget update", "err", err, "library", library, "version", version)
		return fmt.Errorf("nuget error code failure %d", res.Code)
	}

	return nil
}

func formatPackagesFolderEntry(packagesDir string, library string, version string) string {
	// folders for dependencies are case-sensitive, and looks like so:
	// `Newtonsoft.Json.12.0.0`
	return filepath.Join(packagesDir, fmt.Sprintf("%s.%s", library, version))
}

// adds the fixes to local 'private' source; then updates from that source
func (f *packagesConfFixer) Fix(entry shared.DependencyDescriptor, dep *common.Dependency, packageData []byte, fileName string) (bool, string, error) {
	if fileName == "" {
		// should not happen
		slog.Error("empty artifact name", "id", entry.AvailableFix.Id())
		return false, "", fmt.Errorf("downloaded artifact name is empty")
	}

	pkgPath := filepath.Join(f.rootWorkdir, fileName)
	if err := common.DumpBytes(pkgPath, packageData); err != nil {
		slog.Error("failed dumping fixed data nuget package", "err", err, "size", len(packageData), "path", pkgPath, "id", entry.AvailableFix.Id())
		return false, "", err
	}

	if err := f.nugetAdd(pkgPath); err != nil {
		slog.Error("failed adding package to source", "path", pkgPath, "id", entry.AvailableFix.Id())
		return false, "", err
	}

	fixedLibName := entry.AvailableFix.Library.Name
	fixedVersion := entry.AvailableFix.Version
	if err := f.nugetUpdate(fixedLibName, fixedVersion); err != nil {
		slog.Error("failed installing package from source", "path", pkgPath, "id", entry.AvailableFix.Id())
		return false, "", err
	}

	// return the path to the new package, as it is not replacing the original, but sits next to it
	fixedPath := formatPackagesFolderEntry(f.packagesDirPath, fixedLibName, fixedVersion)
	slog.Info("returning new fixed location", "path", fixedPath)
	return true, fixedPath, nil
}

func (f *packagesConfFixer) Rollback() bool {
	slog.Warn("nuget rollback not possible, skipping")
	return true
}

func (f *packagesConfFixer) Cleanup() bool {
	if err := os.RemoveAll(f.rootWorkdir); err != nil {
		slog.Error("failed removing sources dir", "dir", f.rootWorkdir, "err", err)
		return false
	}

	return true
}
