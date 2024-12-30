package composer

import (
	"cli/internal/actions"
	"cli/internal/api"
	"cli/internal/common"
	"cli/internal/config"
	"cli/internal/ecosystem/mappings"
	"cli/internal/ecosystem/shared"
	"fmt"
	"log/slog"
	"path/filepath"
)

const composerJsonFilename = "composer.json"
const composerLockFilename = "composer.lock"
const ComposerManagerName = "composer"
const composerExe = "composer"

const MinimalSupportedVersion = "2.0.0"

type ComposerPackageManager struct {
	Config             *config.Config
	composerTargetFile string
	targetDir          string
}

func NewComposerManager(config *config.Config, targetFile string, targetDir string) *ComposerPackageManager {
	return &ComposerPackageManager{Config: config, composerTargetFile: targetFile, targetDir: targetDir}
}

func (m *ComposerPackageManager) Name() string {
	return ComposerManagerName
}

func (m *ComposerPackageManager) Class() actions.ManagerClass {
	return actions.ManifestManager
}

func (m *ComposerPackageManager) GetVersion() string {
	versionOutput, err := common.RunCmdWithArgs(m.targetDir, composerExe,
		"--version",
		"--no-ansi", // strip color codes
	)
	if err != nil {
		slog.Error("failed running composer --version", "err", err)
		return ""
	}

	if versionOutput.Code != 0 {
		slog.Error("running composer version returned non-zero", "result", versionOutput, "exitcode", versionOutput.Code)
		return ""
	}

	version := parseComposerVersion(versionOutput.Stdout)
	slog.Info("got composer version", "version", version)

	return version
}

func (m *ComposerPackageManager) IsVersionSupported(version string) bool {
	supported, _ := common.VersionAtLeast(version, MinimalSupportedVersion)
	return supported
}

func (m *ComposerPackageManager) ListDependencies(be api.Backend) (common.DependencyMap, error) {
	DependencyTreeArgs := []string{"show", "--format", "json", "--locked"}
	if m.Config.Composer.ProdOnlyDeps {
		slog.Info("will ignore dev dependencies")
		DependencyTreeArgs = append(DependencyTreeArgs, "--no-dev")
	}
	result, err := common.RunCmdWithArgs(m.targetDir, composerExe, DependencyTreeArgs...)
	if err != nil {
		slog.Error("failed to get composer dependencies", "err", err)
		return nil, err
	}

	if result.Code != 0 {
		return nil, common.NewPrintableError("running `composer show` returned non-zero")
	}

	dependencies, err := ParseComposerDependencies(result.Stdout, m.targetDir)
	if err != nil {
		return nil, err
	}
	return dependencies, nil
}

func (m *ComposerPackageManager) GetProjectName() string {
	composerJsonPath := filepath.Join(m.targetDir, composerJsonFilename)
	composerJsonMapping := common.JsonLoad(composerJsonPath)
	if composerJsonMapping == nil {
		return ""
	}

	name, exists := composerJsonMapping.Get("name")
	if !exists {
		return ""
	}

	return normalizePackageName(name.(string))
}

func (m *ComposerPackageManager) GetFixer(workdir string) shared.DependencyFixer {
	return NewFixer(m.targetDir, workdir)
}

func (m *ComposerPackageManager) GetEcosystem() string {
	return mappings.PhpEcosystem
}

func (m *ComposerPackageManager) GetScanTargets() []string {
	return []string{m.composerTargetFile}
}

func (m *ComposerPackageManager) DownloadPackage(server api.ArtifactServer, descriptor shared.DependencyDescriptor) ([]byte, string, error) {
	return downloadPackage(server, descriptor.AvailableFix.Library.Name, descriptor.AvailableFix.Version)
}

func (m *ComposerPackageManager) HandleFixes(fixes []shared.DependencyDescriptor) error {
	for _, fix := range fixes {
		metadata := shared.SealPackageMetadata{SealedVersion: fix.AvailableFix.Version}
		metadataFilePath := getMetadataDepFile(m.targetDir, fix.VulnerablePackage.Library.Name)

		err := shared.SavePackageMetadata(metadata, metadataFilePath)
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *ComposerPackageManager) NormalizePackageName(name string) string {
	return normalizePackageName(name)
}

func IsComposerIndicatorFile(path string) bool {
	// composer.lock is always in lower case
	return filepath.Base(path) == composerLockFilename
}

func GetPackageManager(config *config.Config, targetDir string, targetFile string) (shared.PackageManager, error) {
	slog.Debug("checking provided target for composer indicator", "file", targetFile, "dir", targetDir)

	if targetFile == "" {
		targetFile = filepath.Join(targetDir, composerLockFilename)
	}

	if !IsComposerIndicatorFile(targetFile) {
		return nil, fmt.Errorf("not a composer file indicator: %s", targetFile)
	}

	slog.Debug("checking package manager for target file", "file", targetFile)
	exists, err := common.PathExists(targetFile)
	if err != nil {
		slog.Error("failed checking composer.lock file exists", "err", err)
		return nil, fmt.Errorf("failed checking composer.lock file")
	}

	if !exists {
		slog.Debug("no composer.lock file found", "path", targetFile)
		return nil, fmt.Errorf("not a composer file indicator")
	}

	targetDir = filepath.Dir(targetFile)
	slog.Debug("composer manager supports target", "target-file", targetFile, "target-dir", targetDir)
	return NewComposerManager(config, targetFile, targetDir), nil
}

func (m *ComposerPackageManager) SilencePackages(silenceArray []api.SilenceRule, allDependencies common.DependencyMap) (map[string][]string, error) {
	slog.Warn("Silencing packages is not support for composer")
	return nil, nil
}

func (m *ComposerPackageManager) ConsolidateVulnerabilities(vulnerablePackages *[]api.PackageVersion, allDependencies common.DependencyMap) (*[]api.PackageVersion, error) {
	return vulnerablePackages, nil
}
