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
	return fmt.Sprintf("pool/main/s/seal/%s_%s_%s.deb", name, noEpochVersion, arch)
}

func DownloadDebPackage(s api.ArtifactServer, name string, version string, arch string) ([]byte, string, error) {
	uri := buildUri(name, version, arch)
	filename := common.FileNameFromUri(uri)
	libraryData, err := shared.DownloadFile(s, uri)

	if err != nil {
		slog.Error("failed sending request for debian package data", "err", err, "name", name, "version", version, "arch", arch)
		return nil, "", err
	}

	return libraryData, filename, nil
}
