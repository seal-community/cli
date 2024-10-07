package php

import (
	"cli/internal/config"
	"cli/internal/ecosystem/php/composer"
	"cli/internal/ecosystem/shared"
	"fmt"
	"log/slog"
)

func GetPackageManager(config *config.Config, targetDir string, targetFile string) (shared.PackageManager, error) {
	slog.Debug("checking package manager for target dir")
	packageManager, err := composer.GetPackageManager(config, targetDir, targetFile)
	if err != nil || packageManager == nil {
		return nil, fmt.Errorf("failed detecting composer directory %w", err)
	}

	slog.Debug("php manager supports target", "target-file", targetFile, "target-dir", targetDir)
	return packageManager, nil
}
