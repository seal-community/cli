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
	"os"
	"path/filepath"
	"strings"
)

const pipExeName = "pip"

const PipManagerName = "pip"

// Ordered by priority
var pythonIndicators = []string{"poetry.lock", "pipfile.lock", "requirements.txt", "pyproject.toml", "pipfile"}

const versionFlag = "--version"
const pipResultSeparator = "~-~-~-~"
const whlFilename = "WHEEL"

type pipMetadata struct {
	version          string
	sitePackagesPath string
}

type PipPackageManager struct {
	Config           *config.Config
	workDir          string
	compatibleTags   []string
	pythonTargetFile string
	metadata         *pipMetadata
}

func NewPipManager(config *config.Config, pythonFile string, targetDir string) *PipPackageManager {
	m := &PipPackageManager{Config: config, pythonTargetFile: pythonFile, workDir: targetDir}
	m.metadata = getPipMetadata(targetDir)

	return m
}

func (m *PipPackageManager) Name() string {
	return PipManagerName
}

func getPipMetadata(targetDir string) *pipMetadata {
	result, err := common.RunCmdWithArgs(targetDir, pipExeName, versionFlag)
	if err != nil {
		slog.Error("failed running pip version", "err", err)
		return nil
	}
	if result.Code != 0 {
		slog.Error("running pip version returned non-zero", "result", result)
		return nil
	}

	metadata := &pipMetadata{}
	metadata.version, metadata.sitePackagesPath, err = utils.GetMetadata(result.Stdout)
	if err != nil {
		slog.Error("failed getting metadata", "err", err)
		metadata.version = ""
		metadata.sitePackagesPath = ""
	}

	return metadata
}

func (m *PipPackageManager) GetVersion(targetDir string) string {
	if m.metadata != nil {
		return m.metadata.version
	}

	return ""
}

func (m *PipPackageManager) getSitePackages() string {
	if m.metadata != nil {
		return m.metadata.sitePackagesPath
	}

	return ""
}

func (m *PipPackageManager) IsVersionSupported(version string) bool {
	return true
}

func (m *PipPackageManager) ListDependencies(targetDir string) (*common.ProcessResult, bool) {
	return listPackages(targetDir)
}

func (m *PipPackageManager) GetParser() shared.ResultParser {
	return &dependencyParser{config: m.Config}
}

func (m *PipPackageManager) GetProjectName(projectDir string) string {
	return utils.GetProjectName(projectDir)
}

func (m *PipPackageManager) GetFixer(projectDir string, workdir string) shared.DependencyFixer {
	return utils.NewFixer(projectDir, workdir)
}

func IsPythonIndicatorFile(path string) bool {
	for _, f := range pythonIndicators {
		if strings.HasSuffix(path, f) {
			return true
		}
	}

	return false
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

// runs pip command twice at target dir to gather all information needed for scanning and fixing.
// 1. `pip list --format json` to get the list of dependencies in json format.
// 2. `pip --version` to get the path to site-packages.
func listPackages(targetDir string) (*common.ProcessResult, bool) {
	args := []string{"list", "--format", "json"}
	listResult, err := common.RunCmdWithArgs(targetDir, pipExeName, args...)
	if err != nil {
		return nil, false
	}
	versionResult, err := common.RunCmdWithArgs(targetDir, pipExeName, versionFlag)
	if err != nil {
		slog.Error("failed running pip version", "err", err)
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

func (m *PipPackageManager) getHostCompatibleTags() ([]string, error) {
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

func parseWheelTags(wheel string) []string {
	tags := make([]string, 0)

	lines := strings.Split(wheel, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "Tag:") {
			tag := strings.TrimPrefix(line, "Tag: ")
			tag = strings.TrimSpace(tag)
			tags = append(tags, tag)
		}
	}

	return tags
}

func (m *PipPackageManager) getPackageCompatibleTags(name string, version string) ([]string, error) {
	distInfo := utils.DistInfoPath(name, version)

	sitePackagesPath := m.getSitePackages()
	if sitePackagesPath == "" {
		return nil, fmt.Errorf("site packages path not found")
	}

	whlPath := filepath.Join(sitePackagesPath, distInfo, whlFilename)
	if exists, err := common.PathExists(whlPath); err != nil || !exists {
		return nil, fmt.Errorf("whl file not found")
	}

	whl, err := os.ReadFile(whlPath)
	if err != nil {
		slog.Error("failed reading whl file", "err", err, "path", whlPath)
		return nil, err
	}

	tags := parseWheelTags(string(whl))
	slog.Debug("parsed wheel tags", "name", name, "version", version, "tags", tags)

	return tags, nil
}

// Finds the compatible tags to use when choosing a .whl file to download.
// Takes the tags of the existing installation if available.
// If not, takes the tags of the host based on pip debug.
func (m *PipPackageManager) getCompatibleTags(name string, version string) ([]string, error) {
	slog.Info("getting package compatible tags", "name", name, "version", version)
	compatibleTags, err := m.getPackageCompatibleTags(name, version)
	if err == nil {
		return compatibleTags, nil
	}

	// In the rare case we failed parsing tags, don't fail the whole process.
	slog.Warn("failed getting package compatible tags, using host compatible tags", "err", err)

	return m.getHostCompatibleTags()
}

func (m *PipPackageManager) DownloadPackage(server api.Server, descriptor shared.DependnecyDescriptor) ([]byte, error) {
	compatibleTags, err := m.getCompatibleTags(descriptor.VulnerablePackage.Library.Name, descriptor.VulnerablePackage.Version)
	if err != nil {
		return nil, err
	}

	return utils.DownloadPythonPackage(server, descriptor.AvailableFix.Library.Name, descriptor.AvailableFix.Version, compatibleTags)
}

func (m *PipPackageManager) HandleFixes(projectDir string, fixes []shared.DependnecyDescriptor) error {
	return nil
}
