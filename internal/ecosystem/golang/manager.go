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
	"strings"

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
	golangTargetFile string
	targetDir        string
	goMod            *modfile.File
}

func NewGolangManager(config *config.Config, targetFile string, targetDir string) *GolangPackageManager {
	return &GolangPackageManager{Config: config, golangTargetFile: targetFile, targetDir: targetDir}
}

func (m *GolangPackageManager) Name() string {
	return GolangManagerName
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

func (m *GolangPackageManager) ListDependencies() (common.DependencyMap, error) {
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
	return NewFixer(m.targetDir, workdir)
}

func (m *GolangPackageManager) GetEcosystem() string {
	return mappings.GolangEcosystem
}

func (m *GolangPackageManager) GetScanTargets() []string {
	return []string{m.golangTargetFile}
}

func (m *GolangPackageManager) DownloadPackage(server api.ArtifactServer, descriptor shared.DependnecyDescriptor) ([]byte, error) {
	return DownloadPackage(server, descriptor.AvailableFix.Library.Name, descriptor.AvailableFix.Version)
}

func (m *GolangPackageManager) HandleFixes(fixes []shared.DependnecyDescriptor) error {
	if m.Config.UseSealedNames {
		slog.Warn("using sealed names in golang is not supported yet")
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

func (m *GolangPackageManager) SilencePackages(silenceArray []string, allDependencies common.DependencyMap) ([]common.Dependency, error) {
	slog.Warn("Silencing packages is not support for golang")
	return nil, nil
}
