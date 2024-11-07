package yum

import (
	"cli/internal/common"
	"cli/internal/ecosystem/shared"
	"fmt"
	"log/slog"
	"os"
	"path"
	"path/filepath"
)

func (m *YumPackageManager) Prepare() error {
	return nil
}

func buildRpmName(name, version, arch string) string {
	return fmt.Sprintf("%s-%s.%s.rpm", name, version, arch)
}

// Fix will write the package data to the workdir
// Later, the manager will install all packages in one yum transaction
// Otherwise, we need to deal with package obsoletes and conflicts, which does not give any more control
func (m *YumPackageManager) Fix(entry shared.DependencyDescriptor, dep *common.Dependency, packageData []byte) (bool, error) {
	packageName := buildRpmName(dep.Name, dep.Version, dep.Arch)
	packagePath := path.Join(m.workDir, packageName)
	file, err := common.CreateFile(packagePath)
	if err != nil {
		slog.Error("failed creating rpm file", "path", packagePath, "err", err)
		return false, err
	}
	defer file.Close()

	_, err = file.Write(packageData)
	if err != nil {
		slog.Error("failed writing rpm file", "path", packagePath, "err", err)
		return false, err
	}

	packagePath, err = filepath.Abs(packagePath)
	if err != nil {
		slog.Error("failed rpm getting abs path", "path", packagePath, "err", err)
		return false, err
	}

	// append the package path to the list of packages to install
	// so that the manager can install them all in one transaction
	m.installPaths = append(m.installPaths, packagePath)

	return true, nil
}

// Fix does not change anything, so there's no rollback
// yum itself will fail if something goes wrong
func (m *YumPackageManager) Rollback() bool {
	return true
}

func (m *YumPackageManager) Cleanup() bool {
	if err := os.RemoveAll(m.workDir); err != nil {
		slog.Error("failed removing tmp dir", "dir", m.workDir)
	}

	return true
}
