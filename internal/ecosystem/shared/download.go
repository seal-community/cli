package shared

import (
	"cli/internal/api"
	"cli/internal/common"
	"fmt"
	"log/slog"
)

func DownloadFile(s api.ArtifactServer, uri string) ([]byte, error) {
	defer common.ExecutionTimer().Log()

	libraryData, statusCode, err := s.Get(uri, nil, nil)

	if err != nil {
		slog.Error("failed sending request for library data using", "err", err, "uri", uri)
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
