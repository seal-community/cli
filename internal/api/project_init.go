package api

import (
	"log/slog"
)

type ProjectInitRequest struct {
	Name string `json:"name"`
	Tag  string `json:"tag"`
}

type ProjectDescriptor struct {
	ProjectInitRequest
	New bool `json:"is_new"`
}

func (s Server) InitializeProject(tag string, name string) (*ProjectDescriptor, error) {
	if s.AuthToken == "" {
		slog.Error("missing auth token for querying remote config")
		return nil, MissingTokenForApiRequest
	}

	headers := []StringPair{BuildBasicAuthHeader(s.AuthToken)}

	data, statusCode, err := sendSealApiRequest[ProjectInitRequest, ProjectDescriptor](
		s.Client,
		"POST",
		"/authenticated/v1/project",
		&ProjectInitRequest{Tag: tag, Name: name},
		headers,
		nil,
	)

	if statusCode != 200 {
		slog.Error("server returned bad status code for query", "status", statusCode, "err", err)
		return nil, BadServerResponseCode
	}

	if err != nil {
		slog.Error("http error", "err", err, "status", statusCode)
		return nil, err
	}

	return data, nil
}
