package utils

import (
	"cli/internal/api"
	"cli/internal/common"
	"cli/internal/ecosystem/shared"
	"fmt"
	"log/slog"
)

func buildUri(name string, version string, arch string) string {
	_, noEpochVersion := common.GetNoEpochVersion(version)
	return fmt.Sprintf("seal/seal/%s/%s-%s.apk", arch, name, noEpochVersion)
}

func DownloadAPKPackage(s api.ArtifactServer, name string, version string, arch string) ([]byte, string, error) {
	defer common.ExecutionTimer().Log()

	uri := buildUri(name, version, arch)
	filename := common.FileNameFromUri(uri)
	libraryData, err := shared.DownloadFile(s, uri)

	if err != nil {
		slog.Error("failed sending request for apk package data", "err", err, "name", name, "version", version, "arch", arch)
		return nil, "", err
	}

	return libraryData, filename, nil
}
