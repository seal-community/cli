package api

import (
	"cli/internal/common"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/url"
)

// to allow backend to find the project we inject as part of the url.
// we use a specific separator between the requested url and the project
// which looks like so:
//
//					       base
//	    /------------------------------------\/--------\-/----\
//		https://example.com/seal.../.../jfrog/{project}/-/{uri}
const projectUriSeparator = "-"

type JFrogArtifactServer struct {
	client  http.Client
	project string

	baseUrl string

	// using bearer
	authHeader StringPair
}

func NewJFrogArtifactServer(
	client http.Client, project string, token string, baseUrl string) *JFrogArtifactServer {

	authHeader := BuildBearerAuthHeader(token)

	return &JFrogArtifactServer{
		client:  client,
		project: project,
		baseUrl: baseUrl,

		authHeader: authHeader,
	}
}

func (s JFrogArtifactServer) Get(uri string, params []StringPair, headers []StringPair) (data []byte, code int, err error) {
	if err := ValidateRelativeUri(s.baseUrl, uri); err != nil {
		return nil, 0, err
	}

	reqUrl, err := url.JoinPath(s.baseUrl, s.project, projectUriSeparator, uri)
	if err != nil {
		slog.Error("failed building url", "base", s.baseUrl, "uri", uri)
		return nil, 0, err
	}

	slog.Debug("getting from artifact server", "url", reqUrl)

	// send token if we have it configured
	if headers == nil {
		headers = []StringPair{s.authHeader}
	} else {
		headers = append(headers, s.authHeader)
	}

	responseData, statusCode, err := sendSealRequest[any](s.client, "GET", reqUrl, nil, headers, params)
	return responseData, statusCode, err
}

func (s JFrogArtifactServer) GetJsonObject(uri string, headers []StringPair, params []StringPair, obj any) (int, error) {
	if obj == nil {
		slog.Error("bad usage - input should be an pointer to allocated object")
		return 0, NilResponseObjectType
	}

	responseData, statusCode, err := s.Get(uri, headers, params)
	if err != nil {
		return 0, err
	}

	err = json.Unmarshal(responseData, obj)
	if err != nil {
		slog.Error("failed unmarshal response body", "body", string(responseData))
		return 0, err
	}

	common.Trace("received json response", "data", string(responseData), "status", statusCode)
	return statusCode, err
}
