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

const ProjectAssetsFileName = "project.assets.json"

const DotnetRestoreError = "Failed loading project assets, please run 'dotnet restore --force' to regenerate it"

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

func FindNugetIndicatorFile(path string) (bool, error) {
	for _, suffixIndicator := range nugetSuffixIndicators {
		files, err := common.FindPathsWithSuffix(path, suffixIndicator)
		if err == nil && len(files) > 0 {
			slog.Info("found nuget indicator files", "files", files, "indicator", suffixIndicator)
			return true, nil
		}
	}
	slog.Debug("no file found with dotnet suffix", "path", path)
	return false, nil
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
	return DownloadNugetPackage(server, pkg.Library.Name, pkg.RecommendedLibraryVersionString)
}

func (m *NugetPackageManager) HandleFixes(projectDir string, fixes shared.FixMap) error {
	return handleFixes(projectDir, fixes)
}

func handleFixes(projectDir string, fixes shared.FixMap) error {
	slog.Info("updating project.assets.json with fixes", "count", len(fixes))
	assetsPaths, err := common.FindPathsWithSuffix(projectDir, ProjectAssetsFileName)
	for _, assetsPath := range assetsPaths {
		if err != nil {
			slog.Error("failed getting project.assets.json path", "err", err)
			return common.NewPrintableError(DotnetRestoreError)
		}
		assets := common.JsonLoad(assetsPath)
		if assets == nil {
			slog.Error("failed loading project.assets.json in", "dir", assetsPath)
			return common.NewPrintableError(DotnetRestoreError)
		}

		if err := UpdateProjectAssetsfile(assets, fixes); err != nil {
			slog.Error("failed updating project.assets.json", "err", err)
			return common.FallbackPrintableMsg(err, "failed updating project.assets.json")
		}

		if err := common.JsonSave(assets, assetsPath); err != nil {
			slog.Error("failed saving updated project.assets.json", "err", err, "path", assetsPath)
			return common.FallbackPrintableMsg(err, "failed saving new project.assets.json")
		}
	}

	return nil
}
