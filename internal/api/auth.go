package api

import (
	"cli/internal/common"
	"fmt"
	"log/slog"
)

func BuildBasicAuthHeader(token string) StringPair {
	return StringPair{"Authorization", fmt.Sprintf("Basic %s", token)}
}

func (s Server) CheckAuthenticationValid() error {
	defer common.ExecutionTimer().Log()
	_, statusCode, err := SendSealRequest[any](
		s.Client,
		"GET",
		AuthURL,
		nil,
		[]StringPair{BuildBasicAuthHeader(s.AuthToken)},
		nil,
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
