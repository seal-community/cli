package utils

import (
	"cli/internal/api"
	"cli/internal/common"
	"fmt"
	"log/slog"
)

func DownloadNugetPackage(s api.ArtifactServer, name string, version string) ([]byte, string, error) {
	defer common.ExecutionTimer().Log()

	packageName := fmt.Sprintf("%s.%s.nupkg", name, version)
	libraryData, statusCode, err := s.Get(
		fmt.Sprintf("v3-flatcontainer/%s/%s/%s", name, version, packageName),
		nil,
		nil,
	)

	if err != nil {
		slog.Error("failed sending request for nuget package data", "err", err, "name", name, "version", version)
		return nil, "", err
	}

	if statusCode != 200 {
		slog.Error("bad response code for nuget package payload", "err", err, "status", statusCode)
		return nil, "", fmt.Errorf("bad status code for nuget package data; status: %d", statusCode)
	}

	if len(libraryData) == 0 {
		slog.Error("no payload content from server")
		return nil, "", fmt.Errorf("no package content")
	}

	return libraryData, packageName, nil
}
