package composer

import (
	"cli/internal/api"
	"cli/internal/common"
	"cli/internal/ecosystem/shared"
	"fmt"
	"log/slog"
	"strings"
)

func buildUri(name string, version string) string {
	name = normalizePackageName(name)
	version = normalizePackageVersion(version)
	artifact_file_name := fmt.Sprintf("%s-%s.zip", strings.Replace(name, "/", "-", 1), version)
	// should be /download/vendor/package/1.1.1+sp1/vendor-package-1.1.1+sp1.zip
	return fmt.Sprintf("/download/%s/%s/%s", name, version, artifact_file_name)
}

func downloadPackage(s api.ArtifactServer, name string, version string) ([]byte, string, error) {
	defer common.ExecutionTimer().Log()

	uri := buildUri(name, version)

	libraryData, err := shared.DownloadFile(s, uri)
	if err != nil {
		slog.Error("failed sending request for composer package data", "err", err, "name", name, "version", version)
		return nil, "", err
	}

	return libraryData, common.FileNameFromUri(uri), nil
}
