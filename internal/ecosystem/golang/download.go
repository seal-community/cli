package golang

import (
	"cli/internal/api"
	"cli/internal/common"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
)

// case-encode per https://go.dev/ref/mod#module-proxy
func caseEncode(name string) string {
	re := regexp.MustCompile(`[A-Z]`)
	encoded := re.ReplaceAllStringFunc(name, func(s string) string {
		return fmt.Sprintf("!%s", strings.ToLower(s))
	})

	return encoded
}

func buildUri(name string, version string) string {
	return fmt.Sprintf("%s/@v/v%s.zip", caseEncode(name), caseEncode(version))
}

func DownloadPackage(s api.ArtifactServer, name string, version string) ([]byte, error) {
	defer common.ExecutionTimer().Log()

	uri := buildUri(name, version)

	libraryData, statusCode, err := s.Get(
		uri,
		nil, nil,
	)

	if err != nil {
		slog.Error("failed sending request for golang package data", "err", err, "name", name, "version", version)
		return nil, err
	}

	if statusCode != 200 {
		slog.Error("bad response code for golang package payload", "err", err, "status", statusCode)
		return nil, fmt.Errorf("bad status code golang package data; status: %d", statusCode)
	}

	if len(libraryData) == 0 {
		slog.Error("no payload content from server")
		return nil, fmt.Errorf("no package content")
	}

	return libraryData, nil
}
