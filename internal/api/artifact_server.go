package api

import (
	"cli/internal/common"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/url"
)

type SealArtifactServer struct {
	client       http.Client
	authToken    string
	baseUrl      string
	extraHeaders []StringPair
}

func (s *SealArtifactServer) SetExtraHeaders(headers []StringPair) {
	s.extraHeaders = headers
}

func NewArtifactServer(
	baseUrl string, token string, project string, client http.Client) *SealArtifactServer {
	return &SealArtifactServer{
		client:    client,
		authToken: buildAuthToken(token, project), // allow empty, in case used for accessing other servers
		baseUrl:   baseUrl,
	}
}

func ValidateRelativeUri(base string, uri string) error {
	parsedUri, err := url.Parse(uri)
	if err != nil {
		slog.Error("failed parsing uri", "err", err, "uri", uri)
		return err
	}

	// checks if scheme is passed (e.g. 'https')
	// it will not detect cases passing a hostname without scheme
	// passing url without scheme causes undefined behavior, so no need to support it here
	if parsedUri.IsAbs() {
		slog.Error("non-relative uri provided", "scheme", parsedUri.Scheme, "host", parsedUri.Host)
		return ArtifactServerUnsupportedMethod
	}

	return nil
}

func (s SealArtifactServer) Get(uri string, params []StringPair, headers []StringPair) (data []byte, code int, err error) {
	if err := ValidateRelativeUri(s.baseUrl, uri); err != nil {
		return nil, 0, err
	}

	reqUrl, err := url.JoinPath(s.baseUrl, uri)
	if err != nil {
		slog.Error("failed building url", "base", s.baseUrl, "uri", uri)
		return nil, 0, err
	}

	slog.Debug("getting from artifact server", "url", reqUrl)

	if s.authToken != "" {
		authHdr := BuildBasicAuthHeader(s.authToken)
		// send token if we have it configured
		if headers == nil {
			headers = []StringPair{authHdr}
		} else {
			headers = append(headers, authHdr)
		}
		common.Trace("sending auth header in bulk request")
	}

	if len(s.extraHeaders) > 0 {
		headers = append(headers, s.extraHeaders...)
		common.Trace("adding extra headers to request", "headers", s.extraHeaders)
	}

	responseData, statusCode, err := sendSealRequest[any](s.client, "GET", reqUrl, nil, headers, params)
	return responseData, statusCode, err
}

func (s SealArtifactServer) GetJsonObject(uri string, headers []StringPair, params []StringPair, obj any) (int, error) {

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
