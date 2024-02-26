package python

import (
	"cli/internal/config"
	"cli/internal/ecosystem/python/pip"
	"cli/internal/ecosystem/shared"
	"fmt"
)

func GetPackageManager(config *config.Config, targetDir string) (shared.PackageManager, error) {
	isPipDir, err := pip.IsPipProjectDir(targetDir)
	if err != nil {
		return nil, fmt.Errorf("failed detecting pip directory %w", err)
	}
	if !isPipDir {
		return nil, fmt.Errorf("failed detecting pip directory")
	}

	return pip.NewPipManager(config), nil
}
