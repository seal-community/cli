package dpkg

import (
	"cli/internal/actions"
	"cli/internal/api"
	"cli/internal/common"
	"cli/internal/config"
	"cli/internal/ecosystem/deb/utils"
	"cli/internal/ecosystem/mappings"
	"cli/internal/ecosystem/shared"
	"errors"
	"fmt"
	"log/slog"
	"os"
)

const dpkgExeName = "dpkg"
const dpkgQueryExeName = "dpkg-query"
const dpkgQueryFormat = "${Package} ${Version} ${Architecture} ${Status}\n"
const debianStatusFilePath = "/var/lib/dpkg/status"
const debianInfoFilesDirPath = "/var/lib/dpkg/info"

const DpkgManagerName = "DPKG"

type DpkgPackageManager struct {
	Config       *config.Config
	targetDir    string
	workDir      string // The directory where the fixer will run
	installPaths []string
}

func NewDpkgManager(config *config.Config, targetDir string) *DpkgPackageManager {
	m := &DpkgPackageManager{Config: config, targetDir: targetDir, installPaths: make([]string, 0)}
	return m
}

func (m *DpkgPackageManager) Name() string {
	return DpkgManagerName
}

func (m *DpkgPackageManager) Class() actions.ManagerClass {
	return actions.OsManager
}

func (m *DpkgPackageManager) GetVersion() string {
	versionOutput, err := common.RunCmdWithArgs(m.targetDir, dpkgExeName, "--version")
	if err != nil || versionOutput.Code != 0 {
		slog.Error("failed running dpkg version", "err", err)
		return ""
	}

	version := utils.ParseDpkgVersion(versionOutput.Stdout)
	slog.Debug("got dpkg version", "version", version)

	return version
}

// dpkg is always supported on debian based OS and we support all versions of dpkg
func (m *DpkgPackageManager) IsVersionSupported(version string) bool {
	return true
}

func (m *DpkgPackageManager) ListDependencies(be api.Backend) (common.DependencyMap, error) {
	// -W: show information on all packages, -f: format the output as specified in dpkgQueryFormat
	listOutput, err := common.RunCmdWithArgs(m.targetDir, dpkgQueryExeName, "-W", "-f", dpkgQueryFormat)
	if err != nil || listOutput.Code != 0 {
		slog.Error("failed running dpkg-query show", "err", err)
		return nil, err
	}

	deps, err := utils.ParseDpkgQueryInstalled(listOutput.Stdout)
	if err != nil {
		slog.Error("failed parsing dpkg-query show output", "err", err)
		return nil, err
	}

	return deps, nil
}

func (m *DpkgPackageManager) GetProjectName() string {

	// When running in OS mode, user must provide the project
	// So, we return an empty string
	return ""
}

func (m *DpkgPackageManager) GetFixer(workdir string) shared.DependencyFixer {
	// In Debian, the fixer logic is very limited, and requires passing information back to the manager for HandleFixes
	// So, we create a single object that implements both PackageManager and DependencyFixer interfaces
	// This way, the manager can pass installPaths around easily
	m.workDir = workdir
	return m
}

func (m *DpkgPackageManager) GetEcosystem() string {
	return mappings.DebEcosystem
}

func (m *DpkgPackageManager) GetScanTargets() []string {
	return []string{"dpkg"} // We use dpkg as the target to indicate the source of the scan
}

func (m *DpkgPackageManager) DownloadPackage(server api.ArtifactServer, descriptor shared.DependencyDescriptor) ([]byte, string, error) {
	arch := descriptor.Locations[""].Arch // Debian packages have no location, so the map includes a single empty string key

	if arch == "" {
		slog.Error("failed to extract arch from installed package", "name", descriptor.VulnerablePackage.Library.Name)
		return nil, "", fmt.Errorf("failed to extract arch for installed package")
	}

	return utils.DownloadDebPackage(server, descriptor.AvailableFix.Library.Name, descriptor.AvailableFix.Version, arch)
}

// Installs all the sealed libraries in one dpkg transaction
// In case any of the sealed libraries cause conflicts, dpkg will fail the whole transaction
func (m *DpkgPackageManager) HandleFixes(fixes []shared.DependencyDescriptor) error {
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

	if m.Config.UseSealedNames {
		for _, fix := range fixes {
			err := utils.RenameFix(
				fix,
				debianStatusFilePath,
				debianInfoFilesDirPath,
			)
			if err != nil {
				slog.Error("failed renaming package", "err", err)
				return err
			}
		}
	}

	return nil
}

func (m *DpkgPackageManager) NormalizePackageName(name string) string {
	// dpkg does not require normalization
	return name
}

func (m *DpkgPackageManager) SilencePackages(silenceArray []api.SilenceRule, allDependencies common.DependencyMap) (map[string][]string, error) {
	silencedPackages := make(map[string][]string)
	for _, rule := range silenceArray {
		packageId, silencedPaths, err := utils.SilencePackage(
			rule,
			allDependencies,
			debianStatusFilePath,
			debianInfoFilesDirPath,
		)
		if err != nil {
			// USE_SEALED_NAMES and --silence are both flows that rename a package.
			// If used together, they could collide.
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

func (m *DpkgPackageManager) ConsolidateVulnerabilities(vulnerablePackages *[]api.PackageVersion, allDependencies common.DependencyMap) (*[]api.PackageVersion, error) {
	return vulnerablePackages, nil
}
