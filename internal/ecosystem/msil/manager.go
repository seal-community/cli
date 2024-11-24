package msil

import (
	"cli/internal/config"
	"cli/internal/ecosystem/msil/dotnet"
	"cli/internal/ecosystem/msil/nuget"
	"cli/internal/ecosystem/msil/utils"
	"cli/internal/ecosystem/shared"
	"fmt"
	"log/slog"
)

func GetPackageManager(config *config.Config, targetDir string, targetFile string) (shared.PackageManager, error) {
	slog.Debug("checking provided target for dotnet indicator", "file", targetFile, "dir", targetDir)
	if targetFile == "" {
		var err error
		slog.Info("looking for msil target in project dir", "dir", targetDir)
		targetFile, err = dotnet.FindDotnetIndicatorFile(targetDir)
		if err != nil || targetFile == "" {
			return nil, fmt.Errorf("failed detecting dotnet directory %w", err)
		}
	} else {
		if !dotnet.IsDotnetIndicatorFile(targetFile) {
			slog.Error("not a msil indicator file", "target", targetFile)
			return nil, fmt.Errorf("not a msil file indicator: %s", targetFile)
		}
	}

	format, err := utils.DetectProjectFormat(targetFile)
	if err != nil || format == utils.FormatUnknown {
		slog.Error("failed detecting format for project", "err", err, "format", format)
		return nil, fmt.Errorf("failed checking project format")
	}

	slog.Info("detected project fromat", "format", format)
	if format < utils.FormatSupportedByDotnet {
		slog.Info("detect nuget supported format - creating nuget manager", "target-file", targetFile, "target-dir", targetDir, "format", format)
		if format != utils.FormatLegacyPackagesConfig {
			slog.Error("unsupported legacy format", "format", format)
			return nil, fmt.Errorf("unsupported legacy project")
		}

		return nuget.NewNugetManager(config, targetDir, targetFile, format, "", "")
	}

	slog.Info("detect dotnet supported format - creating dotnet manager", "target-file", targetFile, "target-dir", targetDir, "format", format)
	return dotnet.NewDotnetManager(config, targetDir, targetFile), nil
}
