package shared

import (
	"cli/internal/actions"
	"cli/internal/api"
	"cli/internal/common"
)

type DependencyFixer interface {
	Prepare() error

	// fileName could be empty;
	// fixedPath should be the location where the SP is placed, empty means not fixed; could be the original DiskPath if done in-place
	Fix(entry DependencyDescriptor, dep *common.Dependency, packageData []byte, fileName string) (fixed bool, fixedPath string, err error)
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
	Entry            DependencyDescriptor
	Data             []byte
	ArtifactFileName string // name of the file downloaded, could be empty
}

type Normalizer interface {
	NormalizePackageName(name string) string
}

type PackageManager interface {
	Normalizer

	Name() string
	// Each manager has a type, which corresponds to the TargetType
	Class() actions.ManagerClass
	GetVersion() string
	IsVersionSupported(version string) bool
	ListDependencies() (common.DependencyMap, error)
	GetProjectName() string // empty string means failure
	GetFixer(workdir string) DependencyFixer
	GetEcosystem() string
	GetScanTargets() []string
	DownloadPackage(server api.ArtifactServer, descriptor DependencyDescriptor) ([]byte, string, error)
	HandleFixes(fixes []DependencyDescriptor) error
	// Silences the given packages (silenceArray) in the given dependencies map.
	// returns a map of all the silenced package ids to a list of the paths they were silenced in
	SilencePackages(silenceArray []api.SilenceRule, allDependencies common.DependencyMap) (map[string][]string, error)

	// Callback allowing the manager to consolidate and manipulate scan results
	ConsolidateVulnerabilities(vulnerablePackages *[]api.PackageVersion, allDependencies common.DependencyMap) (*[]api.PackageVersion, error)
}
