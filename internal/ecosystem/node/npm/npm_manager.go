package npm

import (
	"cli/internal/api"
	"cli/internal/common"
	"cli/internal/config"
	"cli/internal/ecosystem/mappings"
	"cli/internal/ecosystem/node/utils"
	"cli/internal/ecosystem/shared"
	"log/slog"
	"path/filepath"
	"strconv"
	"strings"
)

const npmExeName = "npm"

const NpmManagerName = "npm"

const npmLockFileName = "package-lock.json"

type NpmPackageManager struct {
	Config  *config.Config
	version string
}

func NewNpmManager(config *config.Config) *NpmPackageManager {
	return &NpmPackageManager{Config: config}
}

func (m *NpmPackageManager) Name() string {
	return NpmManagerName
}

func (m *NpmPackageManager) GetVersion(targetDir string) string {
	if m.version == "" {
		m.version, _ = getNpmVersion(targetDir)
	}

	return m.version
}

func (m *NpmPackageManager) ListDependencies(targetDir string) (*common.ProcessResult, bool) {
	return listPackages(targetDir, m.GetVersion(targetDir), m.Config.Npm.ProdOnlyDeps)
}

func (m *NpmPackageManager) GetParser() shared.ResultParser {
	return &dependencyParser{config: m.Config}
}

func (m *NpmPackageManager) GetProjectName(projectDir string) string {
	return utils.GetProjectName(projectDir)
}

func (m *NpmPackageManager) GetFixer(projectDir string, workdir string) shared.DependencyFixer {
	return utils.NewFixer(projectDir, workdir)
}

func IsNpmProjectDir(path string) (bool, error) {
	// initial check to see if the target path is an npm project directory.
	// in the future we might want to check other dirs/files like .npmrc, node_modules, package-lock, package json, shrinkwrap, yarn
	packageFile := filepath.Join(path, utils.PackageJsonFile)
	exists, err := common.PathExists(packageFile)
	if err != nil {
		slog.Error("failed checking package.json exists", "err", err)
		return false, err
	}
	if !exists {
		slog.Info("package.json does not exist", "path", packageFile)
		return false, nil
	}
	return true, nil
}

func IsNpmIndicatorFile(path string) bool {
	return strings.HasSuffix(path, npmLockFileName)
}

func getNpmVersion(targetDir string) (string, bool) {
	result, err := common.RunCmdWithArgs(targetDir, npmExeName, "-v")
	if err != nil {
		return "", false
	}

	// version command should not fail
	if result.Code != 0 {
		return "", false
	}

	version := strings.TrimSuffix(result.Stdout, "\n") // it contains a new line
	return version, true
}

// runs npm command at target dir to list npm packages.
// there is possible additional text that is printed to stderr like version upgrade and warnings that are ignored

// using:
// 	`ll`: 			show the versions as well as paths
// 	`--json`:		prints verbose json tree for all dependencies
// 	`--all`:		show transitive dependencies as well

// according to https://docs.npmjs.com/cli/v6/commands/npm-ls?v=true:
// 	- `parseable` and using `ll` are supported in all listed version
// 	- `--all` was required since version 7.x
// 	- `--json` supported since 6.x, maybe earlier

// --prod works since npm 6.x, but shows warning to replace with --omit=dev on versions newer than 7.x
func listPackages(targetDir string, npmVersion string, prodOnly bool) (*common.ProcessResult, bool) {
	args := []string{"ll", "--json", "--all"}
	if prodOnly {
		slog.Info("will ignore dev dependencies")
		prodOnlyFlag := "--omit=dev"
		majorComponent := strings.Split(npmVersion, ".")[0]
		major, err := strconv.Atoi(majorComponent)
		if err == nil {
			if major < 8 {
				slog.Debug("using old flag for omitting dev deps")
				prodOnlyFlag = "--prod"
			}
		} else {
			// it is still supported as of version 10 of npm
			slog.Warn("failed converting semver major to int", "err", err, "version", npmVersion)
		}

		args = append(args, prodOnlyFlag)
	}

	result, err := common.RunCmdWithArgs(targetDir, npmExeName, args...)
	return result, err == nil
}

func (m *NpmPackageManager) GetEcosystem() string {
	return mappings.NodeEcosystem
}

func (m *NpmPackageManager) GetScanTargets() []string {
	return []string{utils.PackageJsonFile}
}

func (m *NpmPackageManager) DownloadPackage(server api.Server, pkg api.PackageVersion) ([]byte, error) {
	return utils.DownloadNPMPackage(server, pkg.Library.Name, pkg.RecommendedLibraryVersionString)
}

// according to config, update lock file with the seal prefix
func (m *NpmPackageManager) HandleFixes(projectDir string, fixes shared.FixMap) error {
	if !m.Config.Npm.UpdatePackageNames {
		slog.Debug("not updating package lock")
		return nil
	}

	slog.Info("updating npm package lock file with fixes", "count", len(fixes))
	lock := common.JsonLoad(filepath.Join(projectDir, npmLockFileName))
	if lock == nil {
		slog.Error("failed loading lockfile in", "dir", projectDir)
		return common.NewPrintableError("failed loading package-lock.json")
	}

	if err := UpdateLockfile(lock, fixes, projectDir); err != nil {
		slog.Error("failed updating lockfile", "err", err)
		return common.FallbackPrintableMsg(err, "failed updating package-lock.json")
	}

	if err := common.JsonSave(lock, filepath.Join(projectDir, npmLockFileName)); err != nil {
		slog.Error("failed saving updated lockfile", "err", err)
		return common.FallbackPrintableMsg(err, "failed saving new package-lock.json")
	}

	return nil
}
