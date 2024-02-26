package pip

import (
	"cli/internal/api"
	"cli/internal/common"
	"cli/internal/config"
	"cli/internal/ecosystem/mappings"
	"cli/internal/ecosystem/python/utils"
	"cli/internal/ecosystem/shared"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
)

const pipExeName = "pip"

const PipManagerName = "pip"

// Ordered by priority
var pythonIndicators = []string{"poetry.lock", "pipfile.lock", "requirements.txt", "pyproject.toml", "pipfile"}

const versionFlag = "--version"
const pipResultSeparator = "~-~-~-~"

type PipPackageManager struct {
	Config           *config.Config
	version          string
	workDir          string
	compatibleTags   []string
	pythonTargetFile string
}

func NewPipManager(config *config.Config, pythonFile string) *PipPackageManager {
	return &PipPackageManager{Config: config, pythonTargetFile: pythonFile}
}

func (m *PipPackageManager) Name() string {
	return PipManagerName
}

func (m *PipPackageManager) GetVersion(targetDir string) string {
	if m.version == "" {
		m.version, _ = getPipVersion(targetDir)
	}

	return m.version
}

func (m *PipPackageManager) ListDependencies(targetDir string) (*common.ProcessResult, bool) {
	m.workDir = targetDir // Store the workdir for later use
	return listPackages(targetDir)
}

func (m *PipPackageManager) GetParser() shared.ResultParser {
	return &dependencyParser{config: m.Config}
}

func (m *PipPackageManager) GetProjectName(projectDir string) string {
	return ""
}

func (m *PipPackageManager) GetFixer(projectDir string, workdir string) shared.DependencyFixer {
	return utils.NewFixer(projectDir, workdir)
}

func GetPythonIndicatorFile(path string) (string, error) {
	// Assumes pythonIndicators are ordered by priority
	for _, file := range pythonIndicators {
		packageFile := filepath.Join(path, file)
		exists, err := common.PathExists(packageFile)
		if err != nil {
			slog.Error("failed checking file exists", "file", file, "err", err)
			continue
		}

		if exists {
			slog.Info("found python indicator file", "file", file, "path", packageFile)
			return file, nil
		}
	}

	slog.Debug("no python indicator file found")
	return "", nil
}

func getPipVersion(targetDir string) (string, bool) {
	result, err := common.RunCmdWithArgs(targetDir, pipExeName, versionFlag)
	if err != nil {
		return "", false
	}

	// version command should not fail
	if result.Code != 0 {
		return "", false
	}

	versionWithSuffix := strings.TrimPrefix(result.Stdout, "pip ") // it contains a new line
	spaceIndex := strings.Index(versionWithSuffix, " ")
	version := versionWithSuffix[:spaceIndex]
	return version, true
}

// runs pip command twice at target dir to gather all information needed for scanning and fixing.
// 1. `pip list --format json` to get the list of dependencies in json format.
// 2. `pip --version` to get the path to site-packages.
func listPackages(targetDir string) (*common.ProcessResult, bool) {
	args := []string{"list", "--format", "json"}
	listResult, err := common.RunCmdWithArgs(targetDir, pipExeName, args...)
	if err != nil {
		return nil, false
	}
	args = []string{versionFlag}
	versionResult, err := common.RunCmdWithArgs(targetDir, pipExeName, args...)
	if err != nil {
		return nil, false
	}

	if listResult.Code != 0 {
		return listResult, false
	}
	if versionResult.Code != 0 {
		return versionResult, false
	}

	combinedOutput := fmt.Sprintf("%s%s%s", versionResult.Stdout, pipResultSeparator, listResult.Stdout)

	result := &common.ProcessResult{
		Stdout: combinedOutput,
		Stderr: "",
		Code:   0,
	}
	return result, true
}

func (m *PipPackageManager) GetEcosystem() string {
	return mappings.PythonEcosystem
}

func (m *PipPackageManager) GetScanTargets() []string {
	return []string{m.pythonTargetFile}
}

// Extract compatible tags from pip debug output, see tests for example.
func parseCompatibleTags(debugOutput string) ([]string, error) {
	tags := make([]string, 0)
	tagsIndex := strings.Index(debugOutput, "Compatible tags:")
	if tagsIndex == -1 {
		slog.Error("failed finding compatible tags", "result", debugOutput)
		return nil, fmt.Errorf("failed finding compatible tags")
	}

	tagsStr := debugOutput[tagsIndex:]
	tagLines := strings.Split(tagsStr, "\n")
	for _, line := range tagLines {
		if strings.HasPrefix(line, "Compatible tags:") {
			continue
		}
		strippedTag := strings.TrimSpace(line)
		if strippedTag == "" {
			continue
		}
		tags = append(tags, strippedTag)
	}

	return tags, nil
}

func (m *PipPackageManager) getCompatibleTags() ([]string, error) {
	if m.compatibleTags != nil {
		return m.compatibleTags, nil
	}

	result, err := common.RunCmdWithArgs(m.workDir, pipExeName, "debug", "--verbose")
	if err != nil {
		return nil, err
	}
	if result.Code != 0 {
		return nil, fmt.Errorf("failed running pip debug")
	}

	tags, err := parseCompatibleTags(result.Stdout)
	if err != nil {
		return nil, err
	}

	m.compatibleTags = tags
	return tags, nil
}

func (m *PipPackageManager) DownloadPackage(server api.Server, name string, version string) ([]byte, error) {
	compatibleTags, err := m.getCompatibleTags()
	if err != nil {
		return nil, err
	}
	return utils.DownloadPythonPackage(server, name, version, compatibleTags)
}
