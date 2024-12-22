package files

import (
	"cli/internal/common"
	"cli/internal/ecosystem/shared"
	"log/slog"
	"os"
	"path/filepath"
)

type fixer struct {
	rollback  map[string]string // map of backup paths to original paths
	targetDir string
	workDir   string
}

func newFixer(targetDir string, workdir string) shared.DependencyFixer {
	return &fixer{
		targetDir: targetDir,
		workDir:   workdir,
		rollback:  make(map[string]string, 100),
	}
}

func (f *fixer) Prepare() error {
	return nil
}

// copy a backup for the original artifact to the workdir (tmp location)
// override the artifact file in the target dir
// add to the rollback map
func (f *fixer) Fix(entry shared.DependencyDescriptor, dep *common.Dependency, packageData []byte, fileName string) (bool, string, error) {
	artifactPath := dep.DiskPath

	// this assumes a single targetDir. If it becomes a list in the future, will require adjustment.
	relArtifactPath, err := filepath.Rel(f.targetDir, artifactPath)
	if err != nil {
		slog.Error("failed getting rel path", "target", f.targetDir, "path", artifactPath)
		return false, "", err
	}
	workDirArtifactPath := filepath.Join(f.workDir, relArtifactPath)

	slog.Debug("copying artifact to workdir", "from", artifactPath, "to", workDirArtifactPath)

	workDirArtifactPathDir := filepath.Dir(workDirArtifactPath)

	err = os.MkdirAll(workDirArtifactPathDir, os.ModePerm)
	if err != nil {
		slog.Error("failed creating backup inner dir", "path", workDirArtifactPathDir)
		return false, "", err
	}

	if err := common.Move(artifactPath, workDirArtifactPath); err != nil {
		slog.Error("failed renaming artifact", "err", err, "from", artifactPath, "to", workDirArtifactPath)
		return false, "", err
	}

	f.rollback[workDirArtifactPath] = artifactPath

	slog.Debug("writing jar file", "path", artifactPath)
	if err = common.DumpBytes(artifactPath, packageData); err != nil {
		slog.Error("failed writing to jar file", "path", artifactPath, "err", err)
		return false, "", err
	}

	return true, dep.DiskPath, nil
}

func (f *fixer) Rollback() bool {
	// go over the rollback map and move the original versions saved in the tmp location
	// to the original location to undo the fix
	for workDirArtifactPath, artifactPath := range f.rollback {
		if err := common.Move(workDirArtifactPath, artifactPath); err != nil {
			slog.Error("failed moving artifact back", "err", err, "from", workDirArtifactPath, "to", artifactPath)
			return false
		}

	}

	return true
}

func (f *fixer) Cleanup() bool {
	// remove the work dir as we succeeded fixing and we don't need it anymore
	if err := os.RemoveAll(f.workDir); err != nil {
		slog.Error("failed removing tmp dir", "dir", f.workDir, "err", err)
		return false
	}

	return true
}
