package utils

import (
	"cli/internal/common"
	"cli/internal/ecosystem/shared"
)

type fixer struct {
	rollback       map[string]string // original-dependency-path -> tmp-location
	projectDir     string
	workdir        string
}

// Cleanup implements shared.DependencyFixer.
func (*fixer) Cleanup() bool {
	return false // Going to get implemented with the fixing logic
}

// Fix implements shared.DependencyFixer.
func (*fixer) Fix(dep *common.Dependency, payload []byte) (bool, error) {
	return false, common.NewPrintableError("We don't support fixing nuget packages yet.") // Going to get implemented with all the fixing logic
}

func (*fixer) Rollback() bool {
	return false // Going to get implemented with all the fixing logic
}

func NewFixer(projectDir string, workdir string) shared.DependencyFixer {
	return &fixer{
		projectDir: projectDir,
		workdir:    workdir,
		rollback:   make(map[string]string, 100),
	}
}
