package msil

import (
	"cli/internal/config"
	"cli/internal/ecosystem/msil/dotnet"
	"cli/internal/ecosystem/shared"
	"fmt"
	"log/slog"
)

func GetPackageManager(config *config.Config, targetDir string, targetFile string) (shared.PackageManager, error) {
	slog.Debug("checking provided target for dotnet indicator", "file", targetFile, "dir", targetDir)

	if targetFile != "" {
		if dotnet.IsDotnetIndicatorFile(targetFile) {
			slog.Debug("nuget manager supports target", "target-file", targetFile, "target-dir", targetDir)
			return dotnet.NewDotnetManager(config, targetDir, targetFile), nil
		}

		return nil, fmt.Errorf("not a nuget file indicator: %s", targetFile)
	}

	slog.Debug("looking for nuget target in project dir", "dir", targetDir)
	indicator, err := dotnet.FindDotnetIndicatorFile(targetDir)
	if err != nil || indicator == "" {
		return nil, fmt.Errorf("failed detecting nuget directory %w", err)
	}

	slog.Debug("nuget manager supports target", "target-file", targetFile, "target-dir", targetDir)
	return dotnet.NewDotnetManager(config, targetDir, indicator), nil
}
