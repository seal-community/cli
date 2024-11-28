package golang

import (
	"cli/internal/api"
	"cli/internal/common"
	"cli/internal/ecosystem/shared"
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

func DownloadPackage(s api.ArtifactServer, name string, version string) ([]byte, string, error) {
	defer common.ExecutionTimer().Log()

	uri := buildUri(name, version)
	filename := common.FileNameFromUri(uri)

	libraryData, err := shared.DownloadFile(s, uri)

	if err != nil {
		slog.Error("failed sending request for golang package data", "err", err, "name", name, "version", version)
		return nil, filename, err
	}

	return libraryData, filename, nil
}
