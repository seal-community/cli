package dotnet

import (
	"cli/internal/config"
	"cli/internal/ecosystem/dotnet/nuget"
	"cli/internal/ecosystem/shared"
	"fmt"
	"log/slog"
)

func GetPackageManager(config *config.Config, targetDir string, targetFile string) (shared.PackageManager, error) {
	slog.Debug("checking provided target for dotnet indicator", "file", targetFile, "dir", targetDir)

	if targetFile != "" {
		if nuget.IsNugetIndicatorFile(targetFile) {
			slog.Debug("nuget manager supports target", "target-file", targetFile, "target-dir", targetDir)
			return nuget.NewNugetManager(config, targetDir, targetFile), nil
		}

		return nil, fmt.Errorf("not a nuget file indicator: %s", targetFile)
	}

	slog.Debug("looking for nuget target in project dir", "dir", targetDir)
	indicator, err := nuget.FindNugetIndicatorFile(targetDir)
	if err != nil || indicator == "" {
		return nil, fmt.Errorf("failed detecting nuget directory %w", err)
	}

	slog.Debug("nuget manager supports target", "target-file", targetFile, "target-dir", targetDir)
	return nuget.NewNugetManager(config, targetDir, indicator), nil
}
