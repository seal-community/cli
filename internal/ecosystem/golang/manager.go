package golang

import (
	"cli/internal/api"
	"cli/internal/common"
	"cli/internal/config"
	"cli/internal/ecosystem/mappings"
	"cli/internal/ecosystem/shared"
	"fmt"
	"log/slog"
	"path/filepath"

	"github.com/Masterminds/semver/v3"
	"golang.org/x/mod/modfile"
)

const goModFilename = "go.mod"
const GolangManagerName = "golang"
const goExe = "go"

// require go 1.17.0 or higher
// otherwise, go.mod will include only direct dependencies
const MinimalSupportedVersion = "1.17.0"

type GolangPackageManager struct {
	Config           *config.Config
	workDir          string
	golangTargetFile string
	goMod            *modfile.File
}

func NewGolangManager(config *config.Config, goModFile string, targetDir string) *GolangPackageManager {
	return &GolangPackageManager{Config: config, golangTargetFile: goModFile, workDir: targetDir}
}

func (m *GolangPackageManager) Name() string {
	return GolangManagerName
}

func (m *GolangPackageManager) GetVersion(targetDir string) string {
	versionOutput, err := common.RunCmdWithArgs(targetDir, goExe, "version")
	if err != nil {
		slog.Error("failed running go version", "err", err)
		return ""
	}
	if versionOutput.Code != 0 {
		slog.Error("running go version returned non-zero", "result", versionOutput)
		return ""
	}

	version := ParseGoVersion(versionOutput.Stdout)
	slog.Info("got go version", "version", version)

	return version
}

func (m *GolangPackageManager) IsVersionSupported(version string) bool {
	golangVersion, err := semver.NewVersion(version)
	if err != nil {
		slog.Error("failed parsing go version", "err", err)
		return false
	}

	minVer, err := semver.NewVersion(MinimalSupportedVersion)
	if err != nil {
		slog.Error("failed parsing min go version", "err", err)
		return false
	}

	return golangVersion.Compare(minVer) >= 0
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

func (m *GolangPackageManager) ListDependencies(targetDir string) (common.DependencyMap, error) {
	err := m.parseGoMod(targetDir)
	if err != nil {
		return nil, err
	}
	return BuildDependencyMap(m.goMod), nil
}

func (m *GolangPackageManager) GetProjectName(projectDir string) string {
	err := m.parseGoMod(projectDir)
	if err != nil {
		slog.Warn("failed parsing go.mod file", "err", err)
		return ""
	}

	normalized := common.NormalizeProjectName(m.goMod.Module.Mod.Path)
	return normalized
}

func (m *GolangPackageManager) GetFixer(projectDir string, workdir string) shared.DependencyFixer {
	return NewFixer(projectDir, workdir)
}

func (m *GolangPackageManager) GetEcosystem() string {
	return mappings.GolangEcosystem
}

func (m *GolangPackageManager) GetScanTargets() []string {
	return []string{m.golangTargetFile}
}

func (m *GolangPackageManager) DownloadPackage(server api.Server, descriptor shared.DependnecyDescriptor) ([]byte, error) {
	return DownloadPackage(server, descriptor.AvailableFix.Library.Name, descriptor.AvailableFix.Version)
}

func (m *GolangPackageManager) HandleFixes(projectDir string, fixes []shared.DependnecyDescriptor) error {
	return nil
}

func (m *GolangPackageManager) NormalizePackageName(name string) string {
	return NormalizePackageName(name)
}

func GetPackageManager(config *config.Config, targetDir string, targetFile string) (shared.PackageManager, error) {
	if targetFile == "" {
		targetFile = filepath.Join(targetDir, goModFilename)
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

	return NewGolangManager(config, targetFile, targetDir), nil
}
