package dotnet

import (
	"cli/internal/config"
	"cli/internal/ecosystem/dotnet/nuget"
	"cli/internal/ecosystem/shared"
	"fmt"
)

func GetPackageManager(config *config.Config, targetDir string, targetFile string) (shared.PackageManager, error) {
	if targetFile != "" {
		if nuget.IsNugetIndicatorFile(targetFile) {
			return nuget.NewNugetManager(config, targetDir), nil
		}

		return nil, fmt.Errorf("not a nuget file indicator: %s", targetFile)
	}

	found, err := nuget.FindNugetIndicatorFile(targetDir)
	if err != nil || !found {
		return nil, fmt.Errorf("failed detecting nuget directory %w", err)
	}

	return nuget.NewNugetManager(config, targetDir), nil
}
