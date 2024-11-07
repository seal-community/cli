package shared

import (
	"cli/internal/api"
	"cli/internal/common"
)

type DependencyFixer interface {
	Prepare() error
	Fix(entry DependencyDescriptor, dep *common.Dependency, packageData []byte) (bool, error)
	Rollback() bool
	Cleanup() bool
}

type OverriddenMethod string

const (
	NotOverridden        OverriddenMethod = "" // default
	OverriddenFromLocal  OverriddenMethod = "local"
	OverriddenFromRemote OverriddenMethod = "remote"
)

type DependencyDescriptor struct {
	VulnerablePackage *api.PackageVersion
	AvailableFix      *api.PackageVersion
	Locations         map[string]common.Dependency
	FixedLocations    []string // matching map keys to Locations
	OverrideMethod    OverriddenMethod
}

func (d DependencyDescriptor) IsOverridden() bool {
	return d.OverrideMethod != NotOverridden
}

type PackageDownload struct {
	Entry DependencyDescriptor
	Data  []byte
}

type Normalizer interface {
	NormalizePackageName(name string) string
}

type PackageManager interface {
	Name() string
	GetVersion() string
	IsVersionSupported(version string) bool
	ListDependencies() (common.DependencyMap, error)
	GetProjectName() string // empty string means failure
	GetFixer(workdir string) DependencyFixer
	GetEcosystem() string
	GetScanTargets() []string
	DownloadPackage(server api.ArtifactServer, descriptor DependencyDescriptor) ([]byte, error)
	HandleFixes(fixes []DependencyDescriptor) error
	NormalizePackageName(name string) string
	// Silences the given packages (silenceArray) in the given dependencies map.
	// returns a map of all the silenced package ids to a list of the paths they were silenced in
	SilencePackages(silenceArray []string, allDependencies common.DependencyMap) (map[string][]string, error)
}
