package api

import (
	"bytes"
	"cli/internal/common"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"slices"
)

type StringPair struct {
	Name  string
	Value string
}

// returns length of query string, e.g.:
//
//	len("param=value&param2=value2")
func calculateQuerystringLength(params []StringPair) int {
	q := url.Values{}
	for _, p := range params {
		q.Add(p.Name, p.Value)
	}

	qs := q.Encode()
	return len(qs)
}

func paramExists(params []StringPair, name string) bool {
	idx := slices.IndexFunc(params, func(p StringPair) bool {
		return p.Name == name
	})

	return idx > -1
}

func SendHttpRequest[RequestType any](client http.Client, method string, url string, body *RequestType, headers []StringPair, params []StringPair) ([]byte, int, error) {
	var err error
	encodedBody := []byte{}
	if body != nil {
		encodedBody, err = json.Marshal(body)
		if err != nil {
			slog.Error("failed serializing body to json", "err", err)
			return nil, 0, err
		}
	}

	req, err := http.NewRequest(method, url, bytes.NewReader(encodedBody)) // possible improvement: NewRequestWithContext, add timeout / cancel from caller
	if err != nil {
		slog.Error("failed creating new request", "err", err)
		return nil, 0, err
	}

	for _, header := range headers {
		if req.Header.Get(header.Name) != "" {
			slog.Warn("adding multiple header value", "name", header.Name)
		}
		req.Header.Add(header.Name, header.Value)
	}

	if len(params) > 0 {
		query := req.URL.Query()
		for _, param := range params {
			query.Add(param.Name, param.Value)
		}
		req.URL.RawQuery = query.Encode()
	}

	common.Trace("raw query", "value", req.URL.RawQuery)
	slog.Debug("sending request", "method", req.Method, "url", req.URL.String())

	if len(encodedBody) > 0 {
		common.Trace("sending body data", "body", string(encodedBody))
	}

	res, err := client.Do(req)
	if err != nil {
		slog.Error("failed performing request", "err", err)
		return nil, 0, err
	}

	slog.Debug("received response", "status", res.StatusCode)

	responseData, err := io.ReadAll(res.Body)
	if err != nil {
		slog.Error("failed reading body", "err", err)
		return nil, 0, err
	}

	defer res.Body.Close()

	slog.Debug("body size", "status", len(responseData))

	if len(responseData) == 0 {
		return nil, res.StatusCode, nil
	}

	return responseData, res.StatusCode, nil
}
