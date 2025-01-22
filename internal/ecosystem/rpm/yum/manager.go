package yum

import (
	"cli/internal/actions"
	"cli/internal/api"
	"cli/internal/common"
	"cli/internal/config"
	"cli/internal/ecosystem/mappings"
	"cli/internal/ecosystem/rpm/utils"
	"cli/internal/ecosystem/shared"
	"errors"
	"fmt"
	"log/slog"
	"os"
)

const yumExeName = "yum"

const YumManagerName = "yum"

type YumPackageManager struct {
	Config       *config.Config
	targetDir    string
	workDir      string
	installPaths []string
}

func NewYumManager(config *config.Config, targetDir string) *YumPackageManager {
	m := &YumPackageManager{Config: config, targetDir: targetDir, installPaths: make([]string, 0)}
	return m
}

func (m *YumPackageManager) Name() string {
	return YumManagerName
}

func (m *YumPackageManager) Class() actions.ManagerClass {
	return actions.OsManager
}

func (m *YumPackageManager) GetVersion() string {
	versionOutput, err := common.RunCmdWithArgs(m.targetDir, yumExeName, "--version")
	if err != nil {
		slog.Error("failed running yum version", "err", err)
		return ""
	}

	if versionOutput.Code != 0 {
		slog.Error("running yum version returned non-zero", "result", versionOutput, "exitcode", versionOutput.Code)
		return ""
	}

	version := parseYumVersion(versionOutput.Stdout)
	slog.Debug("got yum version", "version", version)

	return version
}

// Yum is always supported on centos and rhel and we support all versions of yum
func (m *YumPackageManager) IsVersionSupported(version string) bool {
	return true
}

func (m *YumPackageManager) ListDependencies(be api.Backend) (common.DependencyMap, error) {
	listOutput, err := common.RunCmdWithArgs(m.targetDir, yumExeName, "list", "installed")
	if err != nil {
		slog.Error("failed running yum list installed", "err", err)
		return nil, err
	}

	if listOutput.Code != 0 {
		slog.Error("running yum list installed returned non-zero", "result", listOutput, "exitcode", listOutput.Code)
		return nil, fmt.Errorf("failed running yum list installed")
	}

	deps, err := parseYumListInstalled(listOutput.Stdout, m)
	if err != nil {
		return nil, err
	}

	return deps, nil
}

func (m *YumPackageManager) GetProjectName() string {

	// When running in OS mode, user must provide the project
	// So, we return an empty string
	return ""
}

func (m *YumPackageManager) GetFixer(workdir string) shared.DependencyFixer {
	// In RPM, the fixer logic is very limited, and requires passing information back to the manager for HandleFixes
	// So, we create a single object that implements both PackageManager and DependencyFixer interfaces
	// This way, the manager can pass installPaths around easily
	m.workDir = workdir
	return m
}

func (m *YumPackageManager) GetEcosystem() string {
	return mappings.RpmEcosystem
}

func (m *YumPackageManager) GetScanTargets() []string {
	return []string{"yum"} // We use yum as the target to indicate the source of the scan
}

func (m *YumPackageManager) DownloadPackage(server api.ArtifactServer, descriptor shared.DependencyDescriptor) ([]byte, string, error) {
	arch := descriptor.Locations[""].Arch // RPM packages have no location, so the map includes a single empty string key

	if arch == "" {
		slog.Error("failed to find arch for package", "name", descriptor.VulnerablePackage.Library.Name)
		return nil, "", fmt.Errorf("failed to find arch for package")
	}

	return utils.DownloadRpmPackage(server, descriptor.AvailableFix.Library.Name, descriptor.AvailableFix.Version, arch)
}

// Installs all the sealed libraries in one yum transaction
// In case any of the sealed libraries cause conflicts, yum will fail the whole transaction
func (m *YumPackageManager) HandleFixes(fixes []shared.DependencyDescriptor) error {
	if len(m.installPaths) == 0 {
		slog.Debug("no libraries to install via yum")
		return nil
	}

	if os.Geteuid() != 0 {
		slog.Error("non-root user trying to fix OS packages", "user", os.Getenv("USER"), "euid", os.Geteuid())
		return common.NewPrintableError("You must be root to fix OS packages")
	}

	// --disable-repo=* is used to prevent yum from trying to fetch packages other than the ones we are installing
	installArgs := append([]string{"localinstall", "-y", "--disablerepo=*"}, m.installPaths...)
	installOutput, err := common.RunCmdWithArgs(m.targetDir, yumExeName, installArgs...)
	if err != nil {
		slog.Error("failed running yum install", "err", err)
		return err
	}

	if installOutput.Code != 0 {
		slog.Error("running yum install returned non-zero", "result", installOutput, "exitcode", installOutput.Code, "stderr", installOutput.Stderr)
		return fmt.Errorf("failed running yum install")
	}

	return nil
}

func (m *YumPackageManager) NormalizePackageName(name string) string {
	return name
}

func (m *YumPackageManager) SilencePackages(silenceArray []api.SilenceRule, allDependencies common.DependencyMap) (map[string][]string, error) {
	silencedPackages := make(map[string][]string)
	for _, rule := range silenceArray {
		packageId, silencedPaths, err := utils.SilencePackage(rule, allDependencies)
		if err != nil {
			var e *utils.PackageNotFoundError
			if errors.As(err, &e) {
				slog.Warn("failed to silence package, it might have already been renamed", "err", err, "package", rule.Library)
				continue
			}

			slog.Error("failed to silence package", "rule", rule, "err", err)
			return silencedPackages, err
		}
		silencedPackages[packageId] = silencedPaths
	}

	return silencedPackages, nil
}

func (m *YumPackageManager) ConsolidateVulnerabilities(vulnerablePackages *[]api.PackageVersion, allDependencies common.DependencyMap) (*[]api.PackageVersion, error) {
	return vulnerablePackages, nil
}
