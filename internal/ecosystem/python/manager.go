package python

import (
	"cli/internal/config"
	"cli/internal/ecosystem/python/pip"
	"cli/internal/ecosystem/shared"
	"fmt"
)

func GetPackageManager(config *config.Config, targetDir string) (shared.PackageManager, error) {
	pythonFile, err := pip.GetPythonIndicatorFile(targetDir)
	if err != nil || pythonFile == "" {
		return nil, fmt.Errorf("failed detecting python directory %w", err)
	}

	return pip.NewPipManager(config, pythonFile, targetDir), nil
}
