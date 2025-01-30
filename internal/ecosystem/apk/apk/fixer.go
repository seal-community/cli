package apk

import (
	"cli/internal/common"
	"cli/internal/ecosystem/shared"
	"fmt"
	"log/slog"
	"os"
	"path"
	"path/filepath"
)

func (m *APKPackageManager) Prepare() error {
	return nil
}

func buildApkName(name, version string) string {
	return fmt.Sprintf("%s-%s.apk", name, version)
}

// Fix will write the package data to the workdir
// Later, the manager will install all packages in one apk transaction
// Otherwise, we need to deal with package obsoletes and conflicts, which does not give any more control
func (m *APKPackageManager) Fix(entry shared.DependencyDescriptor, dep *common.Dependency, packageData []byte, fileName string) (bool, string, error) {
	packageName := buildApkName(dep.Name, dep.Version)
	packagePath := path.Join(m.workDir, packageName)
	err := common.DumpBytes(packagePath, packageData)
	if err != nil {
		slog.Error("failed writing apk file", "path", packagePath, "err", err)
		return false, "", err
	}

	packagePath, err = filepath.Abs(packagePath)
	if err != nil {
		slog.Error("failed apk getting abs path", "path", packagePath, "err", err)
		return false, "", err
	}

	// append the package path to the list of packages to install
	// so that the manager can install them all in one transaction
	m.installPaths = append(m.installPaths, packagePath)

	return true, "", nil // diskpath is empty for apk
}

// Fix does not change anything, so there's no rollback
// apk itself will fail if something goes wrong
func (m *APKPackageManager) Rollback() bool {
	return true
}

func (m *APKPackageManager) Cleanup() bool {
	if err := os.RemoveAll(m.workDir); err != nil {
		slog.Error("failed removing tmp dir", "dir", m.workDir)
		return false
	}

	return true
}
