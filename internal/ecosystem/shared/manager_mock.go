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
	ProjectName      string
	Fixer            DependencyFixer
	ScanTargets      []string
}

func (m *FakePackageManager) Name() string {
	return m.ManagerName
}

func (m *FakePackageManager) GetEcosystem() string {
	return m.Ecosystem
}

func (m *FakePackageManager) GetVersion() string {
	return m.Version
}

func (m *FakePackageManager) IsVersionSupported(version string) bool {
	return m.VersionSupported
}

func (m *FakePackageManager) GetProjectName() string {
	return m.ProjectName
}

func (m *FakePackageManager) GetFixer(workdir string) DependencyFixer {
	return m.Fixer
}

func (m *FakePackageManager) GetScanTargets() []string {
	return m.ScanTargets
}

func (m *FakePackageManager) ListDependencies() (common.DependencyMap, error) {
	return nil, *new(error)
}

func (m *FakePackageManager) DownloadPackage(server api.ArtifactServer, descriptor DependencyDescriptor) ([]byte, string, error) {
	return nil, "", nil
}

func (m *FakePackageManager) HandleFixes(fixes []DependencyDescriptor) error {
	return nil
}

func (m *FakePackageManager) NormalizePackageName(name string) string {
	return name
}

func (m *FakePackageManager) SilencePackages(silenceArray []string, allDependencies common.DependencyMap) (map[string][]string, error) {
	return nil, nil
}
