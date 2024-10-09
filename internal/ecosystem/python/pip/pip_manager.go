package pip

import (
	"cli/internal/api"
	"cli/internal/common"
	"cli/internal/config"
	"cli/internal/ecosystem/mappings"
	"cli/internal/ecosystem/python/utils"
	"cli/internal/ecosystem/shared"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

const pipExeName = "pip"

const PipManagerName = "pip"

// Ordered by priority
var pythonIndicators = []string{"poetry.lock", "pipfile.lock", "Pipfile.lock", "requirements.txt", "pyproject.toml", "pipfile", "Pipfile"}

const versionFlag = "--version"
const pipResultSeparator = "~-~-~-~"
const whlFilename = "WHEEL"

type pipMetadata struct {
	version          string
	sitePackagesPath string
}

type PipPackageManager struct {
	Config           *config.Config
	targetDir        string
	compatibleTags   []string
	pythonTargetFile string
	metadata         *pipMetadata
}

func NewPipManager(config *config.Config, pythonFile string, targetDir string) *PipPackageManager {
	m := &PipPackageManager{Config: config, pythonTargetFile: pythonFile, targetDir: targetDir}
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
		slog.Error("running pip version returned non-zero", "result", result, "exitcode", result.Code)
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

func (m *PipPackageManager) GetVersion() string {
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

func (m *PipPackageManager) ListDependencies() (common.DependencyMap, error) {
	result, ok := listPackages(m.targetDir)
	if !ok {
		slog.Error("failed running package manager in the current dir", "name", m.Name())
		return nil, shared.ManagerProcessFailed
	}

	parser := &dependencyParser{config: m.Config, normalizer: m}
	dependencyMap, err := parser.Parse(result.Stdout, m.targetDir)
	if err != nil {
		slog.Error("failed parsing package manager output", "err", err, "code", result.Code, "stderr", result.Stderr)
		slog.Debug("manager output", "stdout", result.Stdout) // useful for debugging its output
		return nil, shared.FailedParsingManagerOutput
	}

	return dependencyMap, nil
}

func (m *PipPackageManager) GetProjectName() string {
	return utils.GetPyprojectProjectName(m.targetDir)
}

func (m *PipPackageManager) GetFixer(workdir string) shared.DependencyFixer {
	return utils.NewFixer(m.targetDir, workdir)
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

	result, err := common.RunCmdWithArgs(m.targetDir, pipExeName, "debug", "--verbose")
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
// Tries to get the tags of the existing installation + the compatible tags from the host in this order.
// If one fails, the other is used. If both faield, fail.
func (m *PipPackageManager) getCompatibleTags(name string, version string) ([]string, error) {
	slog.Info("getting package compatible tags", "name", name, "version", version)
	compatibleTags, err1 := m.getPackageCompatibleTags(name, version)
	pipCompatibleTags, err2 := m.getHostCompatibleTags()

	res := make([]string, 0)
	if err1 == nil {
		res = append(res, compatibleTags...)
	} else {
		slog.Warn("failed getting package compatible tags", "err", err1)
	}

	if err2 == nil {
		res = append(res, pipCompatibleTags...)
	} else {
		slog.Warn("failed getting host compatible tags", "err", err2)
	}

	if len(res) == 0 {
		return nil, errors.Join(err1, err2)
	}

	return res, nil
}

func (m *PipPackageManager) DownloadPackage(server api.ArtifactServer, descriptor shared.DependnecyDescriptor) ([]byte, error) {
	compatibleTags, err := m.getCompatibleTags(descriptor.VulnerablePackage.Library.Name, descriptor.VulnerablePackage.Version)
	if err != nil {
		return nil, err
	}

	return utils.DownloadPythonPackage(server, descriptor.AvailableFix.Library.Name, descriptor.AvailableFix.Version, compatibleTags, m.Config.Python.OnlyBinary)
}

func (m *PipPackageManager) HandleFixes(fixes []shared.DependnecyDescriptor) error {
	if m.Config.UseSealedNames {
		slog.Warn("using sealed names in pip is not supported yet")
	}
	return nil
}

// pip is case insensitive and doesn't distinguish between hyphens and underscores.
func (m *PipPackageManager) NormalizePackageName(name string) string {
	return strings.Replace(strings.ToLower(name), "_", "-", -1)
}

func (m *PipPackageManager) SilencePackages(silenceArray []string, allDependencies common.DependencyMap) ([]common.Dependency, error) {
	slog.Warn("Silencing packages is not support for pip")
	return nil, nil
}
