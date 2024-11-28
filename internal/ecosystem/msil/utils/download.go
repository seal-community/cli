package utils

import (
	"cli/internal/api"
	"cli/internal/common"
	"cli/internal/ecosystem/shared"
	"fmt"
	"log/slog"
)

func DownloadNugetPackage(s api.ArtifactServer, name string, version string) ([]byte, string, error) {
	defer common.ExecutionTimer().Log()

	packageName := fmt.Sprintf("%s.%s.nupkg", name, version)
	libraryData, err := shared.DownloadFile(s, fmt.Sprintf("v3-flatcontainer/%s/%s/%s", name, version, packageName))
	if err != nil {
		slog.Error("failed getting nuget package data", "err", err, "name", name, "version", version)
		return nil, "", err
	}

	return libraryData, packageName, nil
}
