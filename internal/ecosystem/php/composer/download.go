package composer

import (
	"cli/internal/api"
	"cli/internal/common"
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

func downloadPackage(s api.ArtifactServer, name string, version string) ([]byte, error) {
	defer common.ExecutionTimer().Log()

	uri := buildUri(name, version)

	libraryData, statusCode, err := s.Get(uri, nil, nil)
	if err != nil {
		slog.Error("failed sending request for composer package data", "err", err, "name", name, "version", version)
		return nil, err
	}

	if statusCode != 200 {
		slog.Error("bad response code for composer package payload", "err", err, "status", statusCode)
		return nil, fmt.Errorf("bad status code composer package data; status: %d", statusCode)
	}

	if len(libraryData) == 0 {
		slog.Error("no payload content from server")
		return nil, fmt.Errorf("no package content")
	}

	return libraryData, nil
}
