package python

import (
	"cli/internal/config"
	"cli/internal/ecosystem/python/pip"
	"cli/internal/ecosystem/shared"
	"fmt"
	"log/slog"
)

func GetPackageManager(config *config.Config, targetDir string, targetFile string) (shared.PackageManager, error) {
	if targetFile != "" {
		slog.Debug("checking package manager for target file")
		if pip.IsPythonIndicatorFile(targetFile) {
			return pip.NewPipManager(config, targetFile, targetDir), nil
		}

		return nil, fmt.Errorf("not a python file indicator")
	}

	slog.Debug("checking package manager for target dir")
	pythonFile, err := pip.GetPythonIndicatorFile(targetDir)
	if err != nil || pythonFile == "" {
		return nil, fmt.Errorf("failed detecting python directory %w", err)
	}

	return pip.NewPipManager(config, pythonFile, targetDir), nil
}
