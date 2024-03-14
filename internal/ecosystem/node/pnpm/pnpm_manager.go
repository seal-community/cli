package pnpm

import (
	"cli/internal/api"
	"cli/internal/common"
	"cli/internal/config"
	"cli/internal/ecosystem/mappings"
	"cli/internal/ecosystem/node/utils"
	"cli/internal/ecosystem/shared"
	"log/slog"
	"path/filepath"
	"strings"
)

const pnpmBaseCommand = "pnpm"
const pnpmLockFileName = "pnpm-lock.yaml"

type PnpmPackageManager struct {
	Config  *config.Config
	version string
}

func NewPnpmManager(config *config.Config) *PnpmPackageManager {
	return &PnpmPackageManager{Config: config}
}

func (m *PnpmPackageManager) Name() string {
	return PnpmManager
}

func (m *PnpmPackageManager) GetProjectName(projectDir string) string {
	return utils.GetProjectName(projectDir)
}

func (m *PnpmPackageManager) GetVersion(targetDir string) string {
	if m.version == "" {
		m.version, _ = getPnpmVersion(targetDir)
	}

	return m.version
}

func (m *PnpmPackageManager) ListDependencies(targetDir string) (*common.ProcessResult, bool) {
	return listPnpmPackages(targetDir, m.version, m.Config.Pnpm.ProdOnlyDeps)
}

func (m *PnpmPackageManager) GetParser() shared.ResultParser {
	return &pnpmDependencyParser{config: m.Config}
}

func (m *PnpmPackageManager) GetFixer(projectDir string, workdir string) shared.DependencyFixer {
	return utils.NewFixer(projectDir, workdir)
}

func IsPnpmProjectDir(path string) bool {
	// initial check to see if the target path has pnpm files
	// could additionally check .pnpm folder in node modules among others

	lockFile := filepath.Join(path, pnpmLockFileName)
	exists, _ := common.PathExists(lockFile)
	if exists {
		slog.Info("found pnpm lock file", "path", lockFile)
		return true
	}

	return false
}

func getPnpmVersion(targetDir string) (string, bool) {
	result, err := common.RunCmdWithArgs(targetDir, pnpmBaseCommand, "-v")
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

// runs pnpm to list dependencies in parseable format
//
//	`ls`:
//	`--json`:		        json output
//	`--depth Infinity`:		show all transitive dependencies
//	`--long`				shows version as well as path (`ll` was not always supported)
//
// testsed commandline flags on:
//   - 3.8.1
//   - 4.14.4
//   - 5.18.10
//   - 6.35.1
//   - 7.33.7
//   - 8.15.3
func listPnpmPackages(targetDir string, npmVersion string, prodOnly bool) (*common.ProcessResult, bool) {
	args := []string{"ls", "--depth", "Infinity", "--parseable", "--long"}
	if prodOnly {
		slog.Info("will ignore dev dependencies")
		prodOnlyFlag := "--prod" // available from at least version 7: https://pnpm.io/7.x/cli/install
		args = append(args, prodOnlyFlag)
	}

	result, err := common.RunCmdWithArgs(targetDir, pnpmBaseCommand, args...)
	return result, err == nil
}

func (m *PnpmPackageManager) GetEcosystem() string {
	return mappings.NodeEcosystem
}

func (m *PnpmPackageManager) GetScanTargets() []string {
	return []string{utils.PackageJsonFile}
}

func (m *PnpmPackageManager) DownloadPackage(server api.Server, pkg api.PackageVersion) ([]byte, error) {
	return utils.DownloadNPMPackage(server, pkg.Library.Name, pkg.RecommendedLibraryVersionString)
}

func (m *PnpmPackageManager) HandleFixes(projectDir string, fixes shared.FixMap) error {
	return nil
}
