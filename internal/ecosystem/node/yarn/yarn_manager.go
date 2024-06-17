package yarn

import (
	"cli/internal/api"
	"cli/internal/common"
	"cli/internal/config"
	"cli/internal/ecosystem/mappings"
	"cli/internal/ecosystem/node/npm"
	"cli/internal/ecosystem/node/utils"
	"cli/internal/ecosystem/shared"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
)

const yarnLockFileName = "yarn.lock"
const YarnManagerName = "yarn"
const yarnBaseCommand = "yarn"

type YarnPackageManager struct {
	Config     *config.Config
	version    string
	npmManager npm.NpmPackageManager // currently using npm for parsing installed content for yarn and wrapping it
}

func IsYarnProjectDir(path string) bool {
	// initial check to see if the target path has yarn lock file
	lockFile := filepath.Join(path, yarnLockFileName)
	exists, _ := common.PathExists(lockFile)
	if exists {
		slog.Info("found yarn lock file", "path", lockFile)
		return true
	}

	return false
}

func IsYarnIndicatorFile(path string) bool {
	return strings.HasSuffix(path, yarnLockFileName)
}

func NewYarnManager(config *config.Config) *YarnPackageManager {
	y := &YarnPackageManager{Config: config, npmManager: npm.NpmPackageManager{Config: config}}
	return y
}

func (m *YarnPackageManager) Name() string {
	return YarnManagerName
}

func (m *YarnPackageManager) GetProjectName(projectDir string) string {
	return utils.GetProjectName(projectDir)
}

func (m *YarnPackageManager) GetFixer(projectDir string, workdir string) shared.DependencyFixer {
	return m.npmManager.GetFixer(projectDir, workdir)
}

func (m *YarnPackageManager) GetVersion(targetDir string) string {
	if m.version == "" {
		m.version, _ = getYarnVersion(targetDir)
	}

	npmVersion := m.npmManager.GetVersion(targetDir)

	return fmt.Sprintf("%s (npm %s)", m.version, npmVersion) // specifying both versions for metadata until we return a map here
}

func (m *YarnPackageManager) IsVersionSupported(version string) bool {
	return true
}

func (m *YarnPackageManager) ListDependencies(targetDir string) (common.DependencyMap, error) {
	dependencyMap, err := m.npmManager.ListDependencies(targetDir)
	if err != nil {
		slog.Error("failed running package manager in the current dir", "name", m.Name())
		return nil, shared.ManagerProcessFailed
	}

	return dependencyMap, nil
}

func getYarnVersion(targetDir string) (string, bool) {
	result, err := common.RunCmdWithArgs(targetDir, yarnBaseCommand, "-v")
	if err != nil {
		return "", false
	}

	// version command should not fail
	if result.Code != 0 {
		return "", false
	}

	version := strings.TrimSuffix(result.Stdout, "\n") // if it contains a new line
	return version, true
}

func (m *YarnPackageManager) GetEcosystem() string {
	return mappings.NodeEcosystem
}

func (m *YarnPackageManager) GetScanTargets() []string {
	return m.npmManager.GetScanTargets()
}

func (m *YarnPackageManager) DownloadPackage(server api.Server, descriptor shared.DependnecyDescriptor) ([]byte, error) {
	return utils.DownloadNPMPackage(server, descriptor.AvailableFix.Library.Name, descriptor.AvailableFix.Version)
}

func (m *YarnPackageManager) HandleFixes(projectDir string, fixes []shared.DependnecyDescriptor) error {
	return nil
}

// yarn is case sensitive
func (m *YarnPackageManager) NormalizePackageName(name string) string {
	return name
}
