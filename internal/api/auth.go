package api

import (
	"cli/internal/common"
	"fmt"
	"log/slog"
)

func (s Server) CheckAuthenticationValid() error {
	defer common.ExecutionTimer().Log()
	_, statusCode, err := sendRequest[any](
		s.client,
		"GET",
		"https://authorization.sealsecurity.io/",
		nil,
		HeaderPair{"Authorization", fmt.Sprintf("Basic %s", s.AuthToken)},
	)

	if err != nil {
		slog.Error("failed sending request", "err", err)
		return err
	}

	if statusCode < 200 || statusCode >= 300 {
		slog.Error("server returned bad status code for authentication test", "status", statusCode)
		return common.NewPrintableError("authentication failed with error %d", statusCode)
	}

	return nil
}
