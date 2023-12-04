package pnpm

import (
	"cli/internal/common"
	"cli/internal/config"
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

func listPnpmPackages(targetDir string, npmVersion string, prodOnly bool) (*common.ProcessResult, bool) {
	/*
		runs pnpm to list dependencies in json format

		using (testsed on version 5.18.10, 8.x):
			`ll`: 			        using ll to have "from" field, which is used as the package name (unknown if cannot be trusted)
			`--json`:		        json output
			`--depth Infinity`:		show all transitive dependencies
	*/

	args := []string{"ll", "--depth", "Infinity", "--json"}
	if prodOnly {
		slog.Info("will ignore dev dependencies")
		prodOnlyFlag := "--prod" // available from at least version 7: https://pnpm.io/7.x/cli/install
		args = append(args, prodOnlyFlag)
	}

	result, err := common.RunCmdWithArgs(targetDir, pnpmBaseCommand, args...)
	return result, err == nil
}
