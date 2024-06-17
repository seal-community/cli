//go:build mock
// +build mock

// this will only be bundled when running unit-tests

package shared

import (
	"cli/internal/api"
	"cli/internal/common"
)

type FakePackageManager struct {
	ManagerName      string
	Ecosystem        string
	Version          string
	VersionSupported bool
	ProjetName       string
	Fixer            DependencyFixer
	ScanTargets      []string
}

func (m *FakePackageManager) Name() string {
	return m.ManagerName
}

func (m *FakePackageManager) GetEcosystem() string {
	return m.Ecosystem
}

func (m *FakePackageManager) GetVersion(targetDir string) string {
	return m.Version
}

func (m *FakePackageManager) IsVersionSupported(version string) bool {
	return m.VersionSupported
}

func (m *FakePackageManager) GetProjectName(projectDir string) string {
	return m.ProjetName
}

func (m *FakePackageManager) GetFixer(projectDir string, workdir string) DependencyFixer {
	return m.Fixer
}

func (m *FakePackageManager) GetScanTargets() []string {
	return m.ScanTargets
}

func (m *FakePackageManager) ListDependencies(targetDir string) (common.DependencyMap, error) {
	return nil, *new(error)
}

func (m *FakePackageManager) DownloadPackage(server api.Server, descriptor DependnecyDescriptor) ([]byte, error) {
	return nil, nil
}

func (m *FakePackageManager) HandleFixes(projectDir string, fixes []DependnecyDescriptor) error {
	return nil
}

func (m *FakePackageManager) NormalizePackageName(name string) string {
	return name
}
