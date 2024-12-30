package dotnet

import (
	"cli/internal/actions"
	"cli/internal/api"
	"cli/internal/common"
	"cli/internal/config"
	"cli/internal/ecosystem/mappings"
	"cli/internal/ecosystem/msil/utils"
	"cli/internal/ecosystem/shared"
	"log/slog"
	"sort"
	"strings"
)

const dotnetExeName = "dotnet"
const dotnetManagerName = "dotnet"

const ProjectAssetsFileName = "project.assets.json"

const DotnetRestoreError = "Failed loading project assets, please run 'dotnet restore --force' to regenerate it"

// Ordered by priority
var dotnetSuffixIndicators = []string{".csproj"}

const versionFlag = "--version"

type DotnetMetadata struct {
	version      string
	packagesPath string // location of installed packages; as of now the global packages under the user folder
}

type DotnetPackageManager struct {
	Config           *config.Config
	targetDir        string
	dotnetTargetFile string
	metadata         *DotnetMetadata
}

func NewDotnetManager(config *config.Config, targetDir string, targetFile string) *DotnetPackageManager {
	metadata := getDotnetMetadata(targetDir)
	m := &DotnetPackageManager{Config: config, targetDir: targetDir, metadata: metadata, dotnetTargetFile: targetFile}

	return m
}

func (m *DotnetPackageManager) Name() string {
	return dotnetManagerName
}

func (m *DotnetPackageManager) Class() actions.ManagerClass {
	return actions.ManifestManager
}

func getDotnetMetadata(targetDir string) *DotnetMetadata {
	result, err := common.RunCmdWithArgs(targetDir, dotnetExeName, versionFlag)
	if err != nil {
		slog.Error("failed running dotnet version", "err", err)
		return nil
	}

	if result.Code != 0 {
		slog.Error("running dotnet version returned non-zero", "result", result)
		return nil
	}

	metadata := &DotnetMetadata{}
	metadata.version = strings.TrimSuffix(result.Stdout, "\n")
	metadata.packagesPath = utils.GetGlobalPackagesCachePath()

	return metadata
}

func (m *DotnetPackageManager) GetVersion() string {
	if m.metadata != nil {
		return m.metadata.version
	}

	return ""
}

func (m *DotnetPackageManager) IsVersionSupported(version string) bool {
	return true
}

func (m *DotnetPackageManager) ListDependencies(be api.Backend) (common.DependencyMap, error) {
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

func (m *DotnetPackageManager) GetProjectName() string {
	return "" // We might want to return the PackageId from the .csproj in the future
}

func (m *DotnetPackageManager) GetFixer(workdir string) shared.DependencyFixer {
	return newFixer(workdir, m.dotnetTargetFile, m.targetDir, m.metadata.packagesPath)
}

func IsDotnetIndicatorFile(path string) bool {
	for _, suffixIndicator := range dotnetSuffixIndicators {
		if strings.HasSuffix(path, suffixIndicator) {
			return true
		}
	}

	return false
}

func FindDotnetIndicatorFile(path string) (string, error) {
	// This function seraches all the files, which isn't ideal
	for _, suffixIndicator := range dotnetSuffixIndicators {
		files, err := common.FindPathsWithSuffix(path, suffixIndicator)
		if err != nil {
			return "", err
		}

		if len(files) == 0 {
			slog.Debug("did not find any indicator candidates", "suffix", suffixIndicator)
			continue
		}

		// sorting them because Walk returns DFS results and we prefer the top level files
		sort.Slice(files, func(i, j int) bool {
			return len(files[i]) < len(files[j])
		})

		slog.Info("found dotnet indicator files", "files", files, "indicator", suffixIndicator)
		return files[0], nil
	}

	slog.Debug("no file found with dotnet suffix", "path", path)
	return "", nil
}

// runs: dotnet list package --include-transitive --format json
func listPackages(targetDir string) (*common.ProcessResult, bool) {
	listResult, err := common.RunCmdWithArgs(targetDir, dotnetExeName,
		"list", "package",
		"--include-transitive",
		"--format", "json",
	)

	if err != nil {
		return nil, false
	}

	return listResult, true
}

func (m *DotnetPackageManager) GetEcosystem() string {
	return mappings.DotnetEcosystem
}

func (m *DotnetPackageManager) GetScanTargets() []string {
	return []string{m.dotnetTargetFile}
}

func (m *DotnetPackageManager) DownloadPackage(server api.ArtifactServer, descriptor shared.DependencyDescriptor) ([]byte, string, error) {
	return utils.DownloadNugetPackage(server, descriptor.AvailableFix.Library.NormalizedName, descriptor.AvailableFix.Version)
}

func (m *DotnetPackageManager) HandleFixes(fixes []shared.DependencyDescriptor) error {
	if m.Config.UseSealedNames {
		slog.Warn("using sealed names in dotnet is not supported yet")
	}

	return nil
}

func (m *DotnetPackageManager) NormalizePackageName(name string) string {
	return utils.NormalizeName(name)
}

func (m *DotnetPackageManager) SilencePackages(silenceArray []api.SilenceRule, allDependencies common.DependencyMap) (map[string][]string, error) {
	slog.Warn("Silencing packages is not support for dotnet")
	return nil, nil
}

func (m *DotnetPackageManager) ConsolidateVulnerabilities(vulnerablePackages *[]api.PackageVersion, allDependencies common.DependencyMap) (*[]api.PackageVersion, error) {
	return vulnerablePackages, nil
}
