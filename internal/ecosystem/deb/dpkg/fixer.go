package dpkg

import (
	"cli/internal/common"
	"cli/internal/ecosystem/deb/utils"
	"cli/internal/ecosystem/shared"
	"log/slog"
	"os"
	"path"
	"path/filepath"
)

func (m *DpkgPackageManager) Prepare() error {
	return nil
}

// Fix will write the package data to the workdir
// Later, the manager will install all packages in one dpkg transaction
// Otherwise, we need to deal with package obsoletes and conflicts, which does not give any more control
func (m *DpkgPackageManager) Fix(entry shared.DependencyDescriptor, dep *common.Dependency, packageData []byte, fileName string) (bool, string, error) {
	packageName := utils.BuildDebName(dep.Name, dep.Version, dep.Arch)
	packagePath := path.Join(m.workDir, packageName)
	packagePath, err := filepath.Abs(packagePath)
	if err != nil {
		slog.Error("failed getting deb abs path", "path", packagePath, "err", err)
		return false, "", err
	}

	err = common.DumpBytes(packagePath, packageData)
	if err != nil {
		slog.Error("failed writing deb file", "path", packagePath, "err", err)
		return false, "", err
	}

	// append the package path to the list of packages to install
	// so that the manager can install them all in one transaction
	m.installPaths = append(m.installPaths, packagePath)

	return true, "", nil // diskpath is empty for dpkg
}

// Fix does not change anything, so there's no rollback
// dpkg itself will fail if something goes wrong
func (m *DpkgPackageManager) Rollback() bool {
	return true
}

func (m *DpkgPackageManager) Cleanup() bool {
	if err := os.RemoveAll(m.workDir); err != nil {
		slog.Error("failed removing tmp dir", "dir", m.workDir)
		return false
	}

	return true
}
