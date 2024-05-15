package shared

import (
	"cli/internal/api"
	"cli/internal/common"
)

type DependencyFixer interface {
	Prepare() error
	Fix(entry DependnecyDescriptor, dep *common.Dependency, packageData []byte) (bool, error)
	Rollback() bool
	Cleanup() bool
}

type ResultParser interface {
	Parse(lsOutput string, projectDir string) (common.DependencyMap, error)
}

type OverriddenMethod string

const (
	NotOverridden        OverriddenMethod = "" // default
	OverriddenFromLocal  OverriddenMethod = "local"
	OverriddenFromRemote OverriddenMethod = "remote"
)

type DependnecyDescriptor struct {
	VulnerablePackage *api.PackageVersion
	AvailableFix      *api.PackageVersion
	Locations         map[string]common.Dependency
	FixedLocations    []string // matching map keys to Locations
	OverrideMethod    OverriddenMethod
}

func (d DependnecyDescriptor) IsOverridden() bool {
	return d.OverrideMethod != NotOverridden
}

type PackageDownload struct {
	Entry DependnecyDescriptor
	Data  []byte
}

type PackageManager interface {
	Name() string
	GetVersion(targetDir string) string
	ListDependencies(targetDir string) (*common.ProcessResult, bool)
	GetParser() ResultParser
	GetProjectName(dir string) string // empty string means failure
	GetFixer(projectDir string, workdir string) DependencyFixer
	GetEcosystem() string
	GetScanTargets() []string
	DownloadPackage(server api.Server, descriptor DependnecyDescriptor) ([]byte, error)
	HandleFixes(projectDir string, fixes []DependnecyDescriptor) error
}
