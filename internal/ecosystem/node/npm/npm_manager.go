package npm

import (
	"cli/internal/common"
	"cli/internal/config"
	"cli/internal/ecosystem/node/utils"
	"cli/internal/ecosystem/shared"
	"log/slog"
	"path/filepath"
	"strconv"
	"strings"
)

const npmExeName = "npm"

const NpmManagerName = "npm"

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
		slog.Warn("package.json does not exist", "path", packageFile)
		return false, nil
	}
	return true, nil
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

func listPackages(targetDir string, npmVersion string, prodOnly bool) (*common.ProcessResult, bool) {
	/*
		runs npm command at target dir to list npm packages.
		there is possible additional text that is printed to stderr like version upgrade and warnings that are ignored

		using:
			`ll`: 			show the versions as well as paths
			`--json`:		prints verbose json tree for all dependencies
			`--all`:		show transitive dependencies as well

		according to https://docs.npmjs.com/cli/v6/commands/npm-ls?v=true:
			- `parseable` and using `ll` are supported in all listed version
			- `--all` was required since version 7.x
			- `--json` supported since 6.x, maybe earlier

		--prod works since npm 6.x, but shows warning to replace with --omit=dev on versions newer than 7.x
	*/

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
			slog.Error("failed converting semver major to int", "err", err, "version", npmVersion)
			slog.Warn("using old flag due to error") // it is still supported as of version 10 of npm
		}

		args = append(args, prodOnlyFlag)
	}

	result, err := common.RunCmdWithArgs(targetDir, npmExeName, args...)
	return result, err == nil
}

func (m *NpmPackageManager) GetEcosystem() string {
	return shared.NodeEcosystem
}

func (m *NpmPackageManager) GetScanTargets() []string {
  return []string{utils.PackageJsonFile}
}
