package dpkgless

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
)

const DpkglessManagerName = "DPKGLESS"
const dpkglessStatusFileDirPath = "/var/lib/dpkg/status.d"

// This manager mimics dpkg's actions in the filesystem manually
type DpkglessPackageManager struct {
	Config    *config.Config
	targetDir string
}

func NewDpkglessManager(config *config.Config, targetDir string) *DpkglessPackageManager {
	m := &DpkglessPackageManager{Config: config, targetDir: targetDir}
	return m
}

func (m *DpkglessPackageManager) Name() string {
	return DpkglessManagerName
}

func (m *DpkglessPackageManager) Class() actions.ManagerClass {
	return actions.OsManager
}

// no version for a filesystem search
func (m *DpkglessPackageManager) GetVersion() string {
	return "no-version"
}

// reading the installed packages from the filesystem is always supported
func (m *DpkglessPackageManager) IsVersionSupported(version string) bool {
	return true
}

func (m *DpkglessPackageManager) ListDependencies(be api.Backend) (common.DependencyMap, error) {
	filesystemOutput, err := ListPackagesFromFilesystem()
	if err != nil {
		slog.Error("failed listing dependencies from file system")
		return nil, err
	}

	deps, err := utils.ParseDpkgQueryInstalled(filesystemOutput)
	if err != nil {
		slog.Error("failed parsing dpkg-query show output", "err", err)
		return nil, err
	}

	return deps, nil
}

func (m *DpkglessPackageManager) GetProjectName() string {
	// When running in OS mode, user must provide the project
	// So, we return an empty string
	return ""
}

func (m *DpkglessPackageManager) GetFixer(workdir string) shared.DependencyFixer {
	return NewFixer(workdir)
}

func (m *DpkglessPackageManager) GetEcosystem() string {
	return mappings.DebEcosystem
}

func (m *DpkglessPackageManager) GetScanTargets() []string {
	return []string{"dpkg-distroless"} // We use dpkg-distroless as the target to indicate the source of the scan
}

func (m *DpkglessPackageManager) DownloadPackage(server api.ArtifactServer, descriptor shared.DependencyDescriptor) ([]byte, string, error) {
	arch := descriptor.Locations[""].Arch // Debian packages have no location, so the map includes a single empty string key

	if arch == "" {
		slog.Error("failed to extract arch from installed package", "name", descriptor.VulnerablePackage.Library.Name)
		return nil, "", fmt.Errorf("failed to extract arch for installed package")
	}

	return utils.DownloadDebPackage(server, descriptor.AvailableFix.Library.Name, descriptor.AvailableFix.Version, arch)
}

func (m *DpkglessPackageManager) HandleFixes(fixes []shared.DependencyDescriptor) error {
	if m.Config.UseSealedNames {
		for _, fix := range fixes {
			err := utils.RenameFix(
				fix,
				fmt.Sprintf("%s/%s", dpkglessStatusFileDirPath, fix.VulnerablePackage.Library.Name),
				dpkglessStatusFileDirPath,
			)
			if err != nil {
				slog.Error("failed renaming package", "err", err)
				return err
			}
		}
	}

	return nil
}

func (m *DpkglessPackageManager) NormalizePackageName(name string) string {
	// dpkg does not require normalization
	return name
}

func (m *DpkglessPackageManager) SilencePackages(silenceArray []api.SilenceRule, allDependencies common.DependencyMap) (map[string][]string, error) {
	silencedPackages := make(map[string][]string)
	for _, rule := range silenceArray {
		dependencyId, silencedPaths, err := utils.SilencePackage(
			rule,
			allDependencies,
			fmt.Sprintf("%s/%s", dpkglessStatusFileDirPath, rule.Library),
			dpkglessStatusFileDirPath,
		)
		if err != nil {
			// USE_SEALED_NAMES and --silence are both flows that rename a package.
			// If used together, they could collide.
			var e *utils.PackageNotFoundError
			if errors.As(err, &e) {
				slog.Warn("failed to silence package, it might have already been renamed", "err", err, "package", rule.Library)
				continue
			}
			slog.Error("failed to silence package", "err", err)
			return silencedPackages, err
		}
		silencedPackages[dependencyId] = silencedPaths
	}

	return silencedPackages, nil
}

func (m *DpkglessPackageManager) ConsolidateVulnerabilities(vulnerablePackages *[]api.PackageVersion, allDependencies common.DependencyMap) (*[]api.PackageVersion, error) {
	return vulnerablePackages, nil
}
