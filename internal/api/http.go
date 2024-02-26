package api

import (
	"bytes"
	"cli/internal/common"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
)

const BaseURL = "https://api.sealsecurity.io"

type StringPair struct {
	Name  string
	Value string
}

var BadServerResponseCode = common.NewPrintableError("remote server issue")

const SealVersionHeader = "X-Seal-Version"

func formatUserAgent() string {
	return fmt.Sprintf("seal-cli/%s", common.CliVersion)
}

func sendApiRequest[RequestType any, ResponseType any](client http.Client, method string, path string, body *RequestType, headers []StringPair, params []StringPair) (*ResponseType, int, error) {
	reqUrl, err := url.JoinPath(BaseURL, path)

	if err != nil {
		slog.Error("failed joining url path", "err", err)
		return nil, 0, err
	}

	return SendRequestJson[RequestType, ResponseType](client, method, reqUrl, body, headers, params)
}

func SendRequestJson[RequestType any, ResponseType any](client http.Client, method string, url string, body *RequestType, headers []StringPair, params []StringPair) (*ResponseType, int, error) {
	var responseObject ResponseType

	responseData, statusCode, err := SendRequest[RequestType](client, method, url, body, headers, params)

	if err != nil {
		return nil, statusCode, err
	}

	if len(responseData) == 0 {
		return nil, statusCode, nil
	}

	err = json.Unmarshal(responseData, &responseObject)
	if err != nil {
		slog.Error("failed unmarshal response body", "body", string(responseData))
		return nil, 0, err
	}

	common.Trace("received json response", "data", string(responseData), "status", statusCode)
	return &responseObject, statusCode, nil

}

func SendRequest[RequestType any](client http.Client, method string, url string, body *RequestType, headers []StringPair, params []StringPair) ([]byte, int, error) {
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

	req.Header.Add("Accept", "application/json")
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add(SealVersionHeader, common.CliVersion)
	req.Header.Add("User-Agent", formatUserAgent())

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

	if len(responseData) == 0 {
		return nil, res.StatusCode, nil
	}

	return responseData, res.StatusCode, nil
}
