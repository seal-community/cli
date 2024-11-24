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
	Config    *config.Config
	version   string
	targetDir string
}

func NewPnpmManager(config *config.Config, targetDir string) *PnpmPackageManager {
	return &PnpmPackageManager{Config: config, targetDir: targetDir}
}

func (m *PnpmPackageManager) Name() string {
	return PnpmManager
}

func (m *PnpmPackageManager) GetProjectName() string {
	return utils.GetProjectName(m.targetDir)
}

func (m *PnpmPackageManager) GetVersion() string {
	if m.version == "" {
		m.version, _ = getPnpmVersion(m.targetDir)
	}

	return m.version
}

func (m *PnpmPackageManager) IsVersionSupported(version string) bool {
	return true
}

func (m *PnpmPackageManager) ListDependencies() (common.DependencyMap, error) {
	result, ok := listPnpmPackages(m.targetDir, m.version, m.Config.Pnpm.ProdOnlyDeps)
	if !ok {
		slog.Error("failed running package manager in the current dir", "name", m.Name())
		return nil, shared.ManagerProcessFailed
	}

	parser := &pnpmDependencyParser{config: m.Config, normalizer: m}
	dependencyMap, err := parser.Parse(result.Stdout, m.targetDir)
	if err != nil {
		slog.Error("failed parsing package manager output", "err", err, "code", result.Code, "stderr", result.Stderr)
		slog.Debug("manager output", "stdout", result.Stdout) // useful for debugging its output
		return nil, shared.FailedParsingManagerOutput
	}

	return dependencyMap, nil
}

func (m *PnpmPackageManager) GetFixer(workdir string) shared.DependencyFixer {
	return utils.NewFixer(m.targetDir, workdir)
}

func IsPnpmIndicatorFile(path string) bool {
	return strings.HasSuffix(path, pnpmLockFileName)
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
	return []string{filepath.Join(m.targetDir, utils.PackageJsonFile)}
}

func (m *PnpmPackageManager) DownloadPackage(server api.ArtifactServer, descriptor shared.DependencyDescriptor) ([]byte, string, error) {
	return utils.DownloadNPMPackage(server, descriptor.AvailableFix.Library.Name, descriptor.AvailableFix.Version)
}

func (m *PnpmPackageManager) HandleFixes(fixes []shared.DependencyDescriptor) error {
	if m.Config.UseSealedNames {
		slog.Warn("using sealed names in pnpm is not supported yet")
	}
	return nil
}

func (m *PnpmPackageManager) NormalizePackageName(name string) string {
	return name
}

func (m *PnpmPackageManager) SilencePackages(silenceArray []string, allDependencies common.DependencyMap) (map[string][]string, error) {
	slog.Warn("Silencing packages is not support for pnpm")
	return nil, nil
}
