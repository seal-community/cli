package dotnet

import (
	"cli/internal/config"
	"cli/internal/ecosystem/dotnet/nuget"
	"cli/internal/ecosystem/shared"
	"fmt"
)

func GetPackageManager(config *config.Config, targetDir string) (shared.PackageManager, error) {
	nugetFile, err := nuget.GetNugetIndicatorFile(targetDir)
	if err != nil || nugetFile == "" {
		return nil, fmt.Errorf("failed detecting nuget directory %w", err)
	}

	return nuget.NewNugetManager(config, targetDir), nil
}
