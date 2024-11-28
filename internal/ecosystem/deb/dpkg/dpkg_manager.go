package dpkg

import (
	"cli/internal/api"
	"cli/internal/common"
	"cli/internal/config"
	"cli/internal/ecosystem/deb/utils"
	"cli/internal/ecosystem/mappings"
	"cli/internal/ecosystem/shared"
	"fmt"
	"log/slog"
	"os"
)

const dpkgExeName = "dpkg"
const dpkgQueryExeName = "dpkg-query"
const dpkgQueryFormat = "${Package} ${Version} ${Architecture} ${Status}\n"

const DPKGManagerName = "DPKG"

type DPKGPackageManager struct {
	Config       *config.Config
	targetDir    string
	workDir      string // The directory where the fixer will run
	installPaths []string
}

func NewDPKGManager(config *config.Config, targetDir string) *DPKGPackageManager {
	m := &DPKGPackageManager{Config: config, targetDir: targetDir, installPaths: make([]string, 0)}
	return m
}

func (m *DPKGPackageManager) Name() string {
	return DPKGManagerName
}

func (m *DPKGPackageManager) GetVersion() string {
	versionOutput, err := common.RunCmdWithArgs(m.targetDir, dpkgExeName, "--version")
	if err != nil || versionOutput.Code != 0 {
		slog.Error("failed running dpkg version", "err", err)
		return ""
	}

	version := parseDPKGVersion(versionOutput.Stdout)
	slog.Debug("got dpkg version", "version", version)

	return version
}

// dpkg is always supported on debian based OS and we support all versions of dpkg
func (m *DPKGPackageManager) IsVersionSupported(version string) bool {
	return true
}

func (m *DPKGPackageManager) ListDependencies() (common.DependencyMap, error) {
	// -W: show information on all packages, -f: format the output as specified in dpkgQueryFormat
	listOutput, err := common.RunCmdWithArgs(m.targetDir, dpkgQueryExeName, "-W", "-f", dpkgQueryFormat)
	if err != nil || listOutput.Code != 0 {
		slog.Error("failed running dpkg-query show", "err", err)
		return nil, err
	}

	deps, err := parseDPKGQueryInstalled(listOutput.Stdout)
	if err != nil {
		slog.Error("failed parsing dpkg-query show output", "err", err)
		return nil, err
	}

	return deps, nil
}

func (m *DPKGPackageManager) GetProjectName() string {

	// When running in OS mode, user must provide the project
	// So, we return an empty string
	return ""
}

func (m *DPKGPackageManager) GetFixer(workdir string) shared.DependencyFixer {
	// In Debian, the fixer logic is very limited, and requires passing information back to the manager for HandleFixes
	// So, we create a single object that implements both PackageManager and DependencyFixer interfaces
	// This way, the manager can pass installPaths around easily
	m.workDir = workdir
	return m
}

func (m *DPKGPackageManager) GetEcosystem() string {
	return mappings.DebEcosystem
}

func (m *DPKGPackageManager) GetScanTargets() []string {
	return []string{"dpkg"} // We use dpkg as the target to indicate the source of the scan
}

func (m *DPKGPackageManager) DownloadPackage(server api.ArtifactServer, descriptor shared.DependencyDescriptor) ([]byte, string, error) {
	arch := descriptor.Locations[""].Arch // Debian packages have no location, so the map includes a single empty string key

	if arch == "" {
		slog.Error("failed to extract arch from installed package", "name", descriptor.VulnerablePackage.Library.Name)
		return nil, "", fmt.Errorf("failed to extract arch for installed package")
	}

	return utils.DownloadDebPackage(server, descriptor.AvailableFix.Library.Name, descriptor.AvailableFix.Version, arch)
}

// Installs all the sealed libraries in one dpkg transaction
// In case any of the sealed libraries cause conflicts, dpkg will fail the whole transaction
func (m *DPKGPackageManager) HandleFixes(fixes []shared.DependencyDescriptor) error {
	if len(m.installPaths) == 0 {
		slog.Debug("no libraries to install via dpkg")
		return nil
	}

	if os.Geteuid() != 0 {
		slog.Error("non-root user trying to fix OS packages", "user", os.Getenv("USER"), "euid", os.Geteuid())
		return common.NewPrintableError("You must be root to fix OS packages")
	}

	installArgs := append([]string{"--install"}, m.installPaths...)
	installOutput, err := common.RunCmdWithArgs(m.targetDir, dpkgExeName, installArgs...)
	if err != nil {
		slog.Error("failed running dpkg -i", "err", err)
		return err
	}

	if installOutput.Code != 0 {
		slog.Error("running dpkg install returned non-zero", "result", installOutput, "exitcode", installOutput.Code, "stderr", installOutput.Stderr)
		return fmt.Errorf("failed running dpkg install")
	}

	return nil
}

func (m *DPKGPackageManager) NormalizePackageName(name string) string {
	// dpkg does not require normalization
	return name
}

func (m *DPKGPackageManager) SilencePackages(silenceArray []string, allDependencies common.DependencyMap) (map[string][]string, error) {
	slog.Warn("Silencing packages is not support for dpkg")
	return nil, nil
}

func (m *DPKGPackageManager) ConsolidateVulnerabilities(vulnerablePackages *[]api.PackageVersion, allDependencies common.DependencyMap) (*[]api.PackageVersion, error) {
	return vulnerablePackages, nil
}
