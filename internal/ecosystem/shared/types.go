package shared

import "cli/internal/common"

// backend package manager enum
const (
	NpmManager = "NPM"
)

const (
	NodeEcosystem = "node"
)

type DependencyFixer interface {
	Fix(dep *common.Dependency, payload []byte) (bool, error)
	Rollback() bool
	Cleanup() bool
}

type ResultParser interface {
	Parse(lsOutput string, projectDir string) (common.DependencyMap, error)
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
}
