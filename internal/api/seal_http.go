package api

import (
	"cli/internal/common"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
)

const SealVersionHeader = "X-Seal-Version"
const SealSessionIdHeader = "X-Seal-CLI-Session-ID"

var BadServerResponseCode = common.NewPrintableError("remote server issue")

func FormatUserAgent() string {
	return fmt.Sprintf("seal-cli/%s", common.CliVersion)
}

func sendSealApiRequest[RequestType any, ResponseType any](client http.Client, method string, path string, body *RequestType, headers []StringPair, params []StringPair) (*ResponseType, int, error) {
	reqUrl, err := url.JoinPath(BaseURL, path)

	if err != nil {
		slog.Error("failed joining url path", "err", err)
		return nil, 0, err
	}

	return SendSealRequestJson[RequestType, ResponseType](client, method, reqUrl, body, headers, params)
}

func SendSealRequestJson[RequestType any, ResponseType any](client http.Client, method string, url string, body *RequestType, headers []StringPair, params []StringPair) (*ResponseType, int, error) {
	var responseObject ResponseType

	responseData, statusCode, err := SendSealRequest[RequestType](client, method, url, body, headers, params)

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

func SendSealRequest[RequestType any](client http.Client, method string, url string, body *RequestType, headers []StringPair, params []StringPair) ([]byte, int, error) {
	baseHeaders := []StringPair{
		{Name: "Accept", Value: "application/json"},
		{Name: "Content-Type", Value: "application/json"},
		{Name: SealVersionHeader, Value: common.CliVersion},
		{Name: SealSessionIdHeader, Value: common.SessionId},
		{Name: "User-Agent", Value: FormatUserAgent()},
	}

	headers = append(headers, baseHeaders...)
	return BaseSendRequest(client, method, url, body, headers, params)
}
