package golang

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
	"strings"

	"golang.org/x/mod/modfile"
)

const goModFilename = "go.mod"
const GolangManagerName = "golang"
const goExe = "go"

// require go 1.17.0 or higher
// otherwise, go.mod will include only direct dependencies
const MinimalSupportedVersion = "1.17.0"

type GolangPackageManager struct {
	Config              *config.Config
	golangTargetFile    string
	targetDir           string
	goMod               *modfile.File
	vendorDir           string
	vendorAlreadyExists bool
}

func NewGolangManager(config *config.Config, targetFile string, targetDir string) *GolangPackageManager {
	vendorDirPath := filepath.Join(targetDir, vendorDir)
	vendorAlreadyExists, err := isVendorDirExist(targetDir)
	if err != nil {
		slog.Error("failed checking vendor dir exists", "err", err)
		return nil
	}

	return &GolangPackageManager{
		Config:              config,
		golangTargetFile:    targetFile,
		targetDir:           targetDir,
		vendorDir:           vendorDirPath,
		vendorAlreadyExists: vendorAlreadyExists,
	}
}

func (m *GolangPackageManager) Name() string {
	return GolangManagerName
}

func (m *GolangPackageManager) Class() actions.ManagerClass {
	return actions.ManifestManager
}

func (m *GolangPackageManager) GetVersion() string {
	versionOutput, err := common.RunCmdWithArgs(m.targetDir, goExe, "version")
	if err != nil {
		slog.Error("failed running go version", "err", err)
		return ""
	}

	if versionOutput.Code != 0 {
		slog.Error("running go version returned non-zero", "result", versionOutput, "exitcode", versionOutput.Code)
		return ""
	}

	version := ParseGoVersion(versionOutput.Stdout)
	slog.Info("got go version", "version", version)

	return version
}

func (m *GolangPackageManager) IsVersionSupported(version string) bool {
	supported, _ := common.VersionAtLeast(version, MinimalSupportedVersion)
	return supported
}

func (m *GolangPackageManager) parseGoMod(targetDir string) error {
	if m.goMod != nil {
		return nil
	}

	goModPath := filepath.Join(targetDir, goModFilename)
	goMod, err := ParseGoModFile(goModPath)
	if err != nil {
		return err
	}

	m.goMod = goMod

	return nil
}

func (m *GolangPackageManager) ListDependencies(be api.Backend) (common.DependencyMap, error) {
	err := m.parseGoMod(m.targetDir)
	if err != nil {
		return nil, err
	}
	return BuildDependencyMap(m.goMod), nil
}

func (m *GolangPackageManager) GetProjectName() string {
	err := m.parseGoMod(m.targetDir)
	if err != nil {
		slog.Warn("failed parsing go.mod file", "err", err)
		return ""
	}

	return m.goMod.Module.Mod.Path
}

func (m *GolangPackageManager) GetFixer(workdir string) shared.DependencyFixer {
	return newFixer(m.targetDir, workdir, m.vendorDir, m.vendorAlreadyExists)
}

func (m *GolangPackageManager) GetEcosystem() string {
	return mappings.GolangEcosystem
}

func (m *GolangPackageManager) GetScanTargets() []string {
	return []string{m.golangTargetFile}
}

func (m *GolangPackageManager) DownloadPackage(server api.ArtifactServer, descriptor shared.DependencyDescriptor) ([]byte, string, error) {
	return DownloadPackage(server, descriptor.AvailableFix.Library.Name, descriptor.AvailableFix.Version)
}

func (m *GolangPackageManager) HandleFixes(fixes []shared.DependencyDescriptor) error {
	if !m.Config.UseSealedNames {
		return nil
	}

	slog.Info("using sealed names")
	for _, fix := range fixes {
		err := renamePackage(m.vendorDir, fix.VulnerablePackage.Library.Name, fix.VulnerablePackage.Version)
		if err != nil {
			slog.Error("failed renaming package", "package", fix.VulnerablePackage.Library.Name, "version", fix.VulnerablePackage.Version, "err", err)
			return err
		}
	}

	return nil
}

func (m *GolangPackageManager) NormalizePackageName(name string) string {
	return NormalizePackageName(name)
}

func IsGolangIndicatorFile(path string) bool {
	return strings.HasSuffix(path, goModFilename)
}

func GetPackageManager(config *config.Config, targetDir string, targetFile string) (shared.PackageManager, error) {
	slog.Debug("checking provided target for golang indicator", "file", targetFile, "dir", targetDir)

	if targetFile == "" {
		targetFile = filepath.Join(targetDir, goModFilename)
	} else {
		if !IsGolangIndicatorFile(targetFile) {
			return nil, fmt.Errorf("not a golang file indicator: %s", targetFile)
		}
	}

	slog.Debug("checking package manager for target file", "file", targetFile)
	exists, err := common.PathExists(targetFile)
	if err != nil {
		slog.Error("failed checking go.mod file exists", "err", err)
		return nil, fmt.Errorf("failed checking go.mod file")
	}

	if !exists {
		return nil, fmt.Errorf("not a golang file indicator")
	}

	slog.Debug("golang manager supports target", "target-file", targetFile, "target-dir", targetDir)
	return NewGolangManager(config, targetFile, targetDir), nil
}

func (m *GolangPackageManager) SilencePackages(silenceArray []api.SilenceRule, allDependencies common.DependencyMap) (map[string][]string, error) {
	exists, err := isVendorDirExist(m.vendorDir)
	if err != nil {
		slog.Error("failed checking vendor dir exists", "err", err)
		return nil, err
	}

	if !exists {
		err := PrepareVendorDir(m.targetDir) // prepare if was not done already when applied fixes
		if err != nil {
			slog.Error("failed preparing vendor dir", "err", err)
			return nil, err
		}
	}

	silenced := []api.SilenceRule{}
	for _, rule := range silenceArray {
		err = renamePackage(m.vendorDir, rule.Library, rule.Version)
		if err != nil {
			slog.Error("failed renaming package", "package", rule.Library, "version", rule.Version, "err", err)
			break
		}

		silenced = append(silenced, rule)
	}

	return api.GetSilencedMap(silenced, allDependencies, mappings.GolangManager), err
}

func (m *GolangPackageManager) ConsolidateVulnerabilities(vulnerablePackages *[]api.PackageVersion, allDependencies common.DependencyMap) (*[]api.PackageVersion, error) {
	return vulnerablePackages, nil
}
