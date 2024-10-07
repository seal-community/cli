package nuget

import (
	"cli/internal/api"
	"cli/internal/common"
	"cli/internal/config"
	"cli/internal/ecosystem/dotnet/utils"
	"cli/internal/ecosystem/mappings"
	"cli/internal/ecosystem/shared"
	"log/slog"
	"sort"
	"strings"
)

const dotnetExeName = "dotnet"

const NugetManagerName = "nuget"

const ProjectAssetsFileName = "project.assets.json"

const DotnetRestoreError = "Failed loading project assets, please run 'dotnet restore --force' to regenerate it"

// Ordered by priority
var nugetSuffixIndicators = []string{".csproj"}

const versionFlag = "--version"

type NugetMetadata struct {
	version      string
	packagesPath string
}

type NugetPackageManager struct {
	Config          *config.Config
	targetDir       string
	nugetTargetFile string
	metadata        *NugetMetadata
}

func NewNugetManager(config *config.Config, targetDir string, targetFile string) *NugetPackageManager {
	metadata := getNugetMetadata(targetDir)
	m := &NugetPackageManager{Config: config, targetDir: targetDir, metadata: metadata, nugetTargetFile: targetFile}

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

func (m *NugetPackageManager) GetVersion() string {
	if m.metadata != nil {
		return m.metadata.version
	}

	return ""
}

func (m *NugetPackageManager) IsVersionSupported(version string) bool {
	return true
}

func (m *NugetPackageManager) ListDependencies() (common.DependencyMap, error) {
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

func (m *NugetPackageManager) GetProjectName() string {
	return "" // We might want to return the PackageId from the .csproj in the future
}

func (m *NugetPackageManager) GetFixer(workdir string) shared.DependencyFixer {
	return utils.NewFixer(m.targetDir, workdir)
}

func IsNugetIndicatorFile(path string) bool {
	for _, suffixIndicator := range nugetSuffixIndicators {
		if strings.HasSuffix(path, suffixIndicator) {
			return true
		}
	}

	return false
}

func FindNugetIndicatorFile(path string) (string, error) {
	// This function seraches all the files, which isn't ideal
	for _, suffixIndicator := range nugetSuffixIndicators {
		files, err := common.FindPathsWithSuffix(path, suffixIndicator)
		if err == nil && len(files) > 0 {
			// sorting them because Walk returns DFS results and we prefer the top level files
			sort.Slice(files, func(i, j int) bool {
				return len(files[i]) < len(files[j])
			})

			slog.Info("found nuget indicator files", "files", files, "indicator", suffixIndicator)
			return files[0], nil
		}
	}

	slog.Debug("no file found with dotnet suffix", "path", path)
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
	return mappings.DotnetEcosystem
}

func (m *NugetPackageManager) GetScanTargets() []string {
	return []string{m.nugetTargetFile}
}

func (m *NugetPackageManager) DownloadPackage(server api.ArtifactServer, descriptor shared.DependnecyDescriptor) ([]byte, error) {
	return DownloadNugetPackage(server, descriptor.AvailableFix.Library.Name, descriptor.AvailableFix.Version)
}

func (m *NugetPackageManager) HandleFixes(fixes []shared.DependnecyDescriptor) error {
	if m.Config.UseSealedNames {
		slog.Warn("using sealed names in nuget is not supported yet")
	}
	return handleFixes(m.targetDir, fixes)
}

func handleFixes(projectDir string, fixes []shared.DependnecyDescriptor) error {
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

// Nuget package names are case-insensitive as stated here:
// https://learn.microsoft.com/en-us/nuget/consume-packages/finding-and-choosing-packages
func (m *NugetPackageManager) NormalizePackageName(name string) string {
	return strings.ToLower(name)
}
