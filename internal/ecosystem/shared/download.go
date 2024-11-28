package shared

import (
	"cli/internal/api"
	"cli/internal/common"
	"fmt"
	"log/slog"
)

func DownloadFile(s api.ArtifactServer, uri string) ([]byte, error) {
	defer common.ExecutionTimer().Log()

	fileData, statusCode, err := s.Get(uri, nil, nil)

	if err != nil {
		slog.Error("failed sending request for file data using", "err", err, "uri", uri)
		return nil, err
	}

	if statusCode != 200 {
		slog.Error("bad response code for file payload", "err", err, "status", statusCode, "uri", uri)
		return nil, fmt.Errorf("bad status code for file data; status: %d", statusCode)
	}

	if len(fileData) == 0 {
		slog.Error("no payload content from server")
		return nil, fmt.Errorf("no file content")
	}

	return fileData, nil
}
