package api

import (
	"cli/internal/common"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
)

const SealVersionHeader = "X-Seal-Version"
const SealSessionIdHeader = "X-Seal-CLI-Session-ID"

var BadServerResponseCode = common.NewPrintableError("remote server issue")

func FormatUserAgent() string {
	return fmt.Sprintf("seal-cli/%s", common.CliVersion)
}

func sendSealRequestJson[RequestType any, ResponseType any](client http.Client, method string, url string, body *RequestType, headers []StringPair, params []StringPair) (*ResponseType, int, error) {
	var responseObject ResponseType

	responseData, statusCode, err := sendSealRequest[RequestType](client, method, url, body, headers, params)

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

func sendSealRequest[RequestType any](client http.Client, method string, url string, body *RequestType, headers []StringPair, params []StringPair) ([]byte, int, error) {
	baseHeaders := []StringPair{
		{Name: "Accept", Value: "application/json"},
		{Name: "Content-Type", Value: "application/json"},
		{Name: SealVersionHeader, Value: common.CliVersion},
		{Name: SealSessionIdHeader, Value: common.SessionId},
		{Name: "User-Agent", Value: FormatUserAgent()},
	}

	headers = append(headers, baseHeaders...)
	return SendHttpRequest(client, method, url, body, headers, params)
}
