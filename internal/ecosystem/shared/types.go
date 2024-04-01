package shared

import (
	"cli/internal/api"
	"cli/internal/common"
)

type PackageDownload struct {
	PackageVersion *api.PackageVersion
	Data           []byte
}

type DependencyFixer interface {
	Fix(dep *common.Dependency, packageDownload PackageDownload) (bool, error)
	Rollback() bool
	Cleanup() bool
}

type ResultParser interface {
	Parse(lsOutput string, projectDir string) (common.DependencyMap, error)
}

type FixedEntry struct {
	Package *api.PackageVersion
	Paths   map[string]bool
}

type FixMap map[string]*FixedEntry

type PackageManager interface {
	Name() string
	GetVersion(targetDir string) string
	ListDependencies(targetDir string) (*common.ProcessResult, bool)
	GetParser() ResultParser
	GetProjectName(dir string) string // empty string means failure
	GetFixer(projectDir string, workdir string) DependencyFixer
	GetEcosystem() string
	GetScanTargets() []string
	DownloadPackage(server api.Server, pkg api.PackageVersion) ([]byte, error)
	HandleFixes(projectDir string, fixes FixMap) error
}
