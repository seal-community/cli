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

func (m *YarnPackageManager) ListDependencies(targetDir string) (*common.ProcessResult, bool) {
	return m.npmManager.ListDependencies(targetDir)
}

func (m *YarnPackageManager) GetParser() shared.ResultParser {
	return m.npmManager.GetParser()
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

func (m *YarnPackageManager) DownloadPackage(server api.Server, pkg api.PackageVersion) ([]byte, error) {
	return utils.DownloadNPMPackage(server, pkg.Library.Name, pkg.RecommendedLibraryVersionString)
}

func (m *YarnPackageManager) HandleFixes(projectDir string, fixes shared.FixMap) error {
	return nil
}
