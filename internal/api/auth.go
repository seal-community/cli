package api

import (
	"cli/internal/common"
	"fmt"
	"log/slog"
)

func (s Server) CheckAuthenticationValid() error {
	defer common.ExecutionTimer().Log()
	_, statusCode, err := SendRequest[any](
		s.Client,
		"GET",
		"https://authorization.sealsecurity.io/",
		nil,
		[]StringPair{{"Authorization", fmt.Sprintf("Basic %s", s.AuthToken)}},
		[]StringPair{},
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
