package nuget

import (
	"cli/internal/api"
	"cli/internal/common"
	"cli/internal/config"
	"cli/internal/ecosystem/dotnet/utils"
	"cli/internal/ecosystem/mappings"
	"cli/internal/ecosystem/shared"
	"log/slog"
	"strings"
)

const dotnetExeName = "dotnet"

const NugetManagerName = "nuget"

// Ordered by priority
var nugetSuffixIndicators = []string{".csproj", ".sln"}

const versionFlag = "--version"

type NugetMetadata struct {
	version      string
	packagesPath string
}

type NugetPackageManager struct {
	Config          *config.Config
	workDir         string
	nugetTargetFile string
	metadata        *NugetMetadata
}

func NewNugetManager(config *config.Config, targetDir string) *NugetPackageManager {
	m := &NugetPackageManager{Config: config, workDir: targetDir}
	m.metadata = getNugetMetadata(targetDir)

	return m
}

func (m *NugetPackageManager) Name() string {
	return NugetManagerName
}

func getNugetMetadata(targetDir string) *NugetMetadata {
	result, err := common.RunCmdWithArgs(targetDir, dotnetExeName, versionFlag)
	if err != nil {
		slog.Error("failed running nuget version", "err", err)
		return nil
	}
	if result.Code != 0 {
		slog.Error("running nuget version returned non-zero", "result", result)
		return nil
	}

	metadata := &NugetMetadata{}
	metadata.version = strings.TrimSuffix(result.Stdout, "\n")
	metadata.packagesPath = utils.GetGlobalPackagesCachePath()

	return metadata
}

func (m *NugetPackageManager) GetVersion(targetDir string) string {
	if m.metadata != nil {
		return m.metadata.version
	}

	return ""
}

func (m *NugetPackageManager) ListDependencies(targetDir string) (*common.ProcessResult, bool) {
	return listPackages(targetDir)
}

func (m *NugetPackageManager) GetParser() shared.ResultParser {
	return &dependencyParser{config: m.Config}
}

func (m *NugetPackageManager) GetProjectName(projectDir string) string {
	return "" // We might want to return the PackageId from the .csproj in the future
}

func (m *NugetPackageManager) GetFixer(projectDir string, workdir string) shared.DependencyFixer {
	return utils.NewFixer(projectDir, workdir)
}

func GetNugetIndicatorFile(path string) (string, error) {
	for _, suffixIndicator := range nugetSuffixIndicators {
		file, err := common.FindFileWithSuffix(path, suffixIndicator)
		if err == nil {
			slog.Info("found python indicator file", "file", file, "indicator", suffixIndicator)
			return file, nil
		}
	}
	slog.Debug("no file found with dotnet endings", "path", path)
	return "", nil
}

// runs: dotnet list package --include-transitive --format json
func listPackages(targetDir string) (*common.ProcessResult, bool) {
	args := []string{"list", "package", "--include-transitive", "--format", "json"}
	listResult, err := common.RunCmdWithArgs(targetDir, dotnetExeName, args...)
	if err != nil {
		return nil, false
	}

	return listResult, true
}

func (m *NugetPackageManager) GetEcosystem() string {
	return mappings.NugetManager
}

func (m *NugetPackageManager) GetScanTargets() []string {
	return []string{m.nugetTargetFile}
}

func (m *NugetPackageManager) DownloadPackage(server api.Server, pkg api.PackageVersion) ([]byte, error) {
	return []byte{}, nil // We don't need to download packages for nuget
}

func (m *NugetPackageManager) HandleFixes(projectDir string, fixes shared.FixMap) error {
	return common.NewPrintableError("We don't support fixing nuget packages yet.")
}
