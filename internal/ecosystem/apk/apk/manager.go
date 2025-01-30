package apk

import (
	"cli/internal/actions"
	"cli/internal/api"
	"cli/internal/common"
	"cli/internal/config"
	"cli/internal/ecosystem/apk/utils"
	"cli/internal/ecosystem/mappings"
	"cli/internal/ecosystem/shared"
	"fmt"
	"log/slog"
	"os"
)

const apkExeName = "apk"

const ApkManagerName = "apk"

type APKPackageManager struct {
	Config       *config.Config
	targetDir    string
	workDir      string
	installPaths []string
}

func NewAPKManager(config *config.Config, targetDir string) *APKPackageManager {
	m := &APKPackageManager{Config: config, targetDir: targetDir, installPaths: make([]string, 0)}
	return m
}

func (m *APKPackageManager) Name() string {
	return ApkManagerName
}

func (m *APKPackageManager) Class() actions.ManagerClass {
	return actions.OsManager
}

func (m *APKPackageManager) GetVersion() string {
	versionOutput, err := common.RunCmdWithArgs(m.targetDir, apkExeName, "--version")
	if err != nil {
		slog.Error("failed running apk version", "err", err)
		return ""
	}

	if versionOutput.Code != 0 {
		slog.Error("running apk version returned non-zero", "result", versionOutput, "exitcode", versionOutput.Code)
		return ""
	}

	version := parseAPKVersion(versionOutput.Stdout)
	slog.Debug("got apk version", "version", version)

	return version
}

func (m *APKPackageManager) IsVersionSupported(version string) bool {
	return true
}

func (m *APKPackageManager) ListDependencies(be api.Backend) (common.DependencyMap, error) {
	listOutput, err := common.RunCmdWithArgs(m.targetDir, apkExeName, "list", "--installed")
	if err != nil {
		slog.Error("failed running apk list installed", "err", err)
		return nil, err
	}

	if listOutput.Code != 0 {
		slog.Error("running apk list installed returned non-zero", "result", listOutput, "exitcode", listOutput.Code)
		return nil, fmt.Errorf("failed running apk list installed")
	}

	deps, err := parseAPKListInstalled(listOutput.Stdout)
	if err != nil {
		return nil, err
	}

	return deps, nil
}

func (m *APKPackageManager) GetProjectName() string {
	// When running in OS mode, user must provide the project
	// So, we return an empty string
	return ""
}

func (m *APKPackageManager) GetFixer(workdir string) shared.DependencyFixer {
	// In APK, the fixer logic is very limited, and requires passing information back to the manager for HandleFixes
	// So, we create a single object that implements both PackageManager and DependencyFixer interfaces
	// This way, the manager can pass installPaths around easily
	m.workDir = workdir
	return m
}

func (m *APKPackageManager) GetEcosystem() string {
	return mappings.ApkEcosystem
}

func (m *APKPackageManager) GetScanTargets() []string {
	return []string{"apk"} // We use apk as the target to indicate the source of the scan
}

func (m *APKPackageManager) DownloadPackage(server api.ArtifactServer, descriptor shared.DependencyDescriptor) ([]byte, string, error) {
	arch := descriptor.Locations[""].Arch // APK packages have no location, so the map includes a single empty string key

	if arch == "" {
		slog.Error("failed to find arch for package", "name", descriptor.VulnerablePackage.Library.Name)
		return nil, "", fmt.Errorf("failed to find arch for package")
	}

	return utils.DownloadAPKPackage(server, descriptor.AvailableFix.Library.Name, descriptor.AvailableFix.Version, arch)
}

// Installs all the sealed libraries in one apk transaction
// In case any of the sealed libraries cause conflicts, apk will fail the whole transaction
func (m *APKPackageManager) HandleFixes(fixes []shared.DependencyDescriptor) error {
	if len(m.installPaths) == 0 {
		slog.Debug("no libraries to install via apk")
		return nil
	}

	if os.Geteuid() != 0 {
		slog.Error("non-root user trying to fix OS packages", "user", os.Getenv("USER"), "euid", os.Geteuid())
		return common.NewPrintableError("You must be root to fix OS packages")
	}

	installArgs := append([]string{"add", "--allow-untrusted"}, m.installPaths...)
	installOutput, err := common.RunCmdWithArgs(m.targetDir, apkExeName, installArgs...)
	if err != nil {
		slog.Error("failed running apk install", "err", err)
		return err
	}

	if installOutput.Code != 0 {
		slog.Error("running apk install returned non-zero", "result", installOutput, "exitcode", installOutput.Code, "stderr", installOutput.Stderr)
		return fmt.Errorf("failed running apk install")
	}

	return nil
}

func (m *APKPackageManager) NormalizePackageName(name string) string {
	return name
}

func (m *APKPackageManager) SilencePackages(silenceArray []api.SilenceRule, allDependencies common.DependencyMap) (map[string][]string, error) {
	slog.Warn("Silencing packages is not support for apk")
	return nil, nil
}

func (m *APKPackageManager) ConsolidateVulnerabilities(vulnerablePackages *[]api.PackageVersion, allDependencies common.DependencyMap) (*[]api.PackageVersion, error) {
	return vulnerablePackages, nil
}
