//go:build mock
// +build mock

package api

import "net/http"

type roundTriphandler func(*http.Request) *http.Response
type TransparentRoundTripper struct {
	Callback roundTriphandler
}

func (t TransparentRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return t.Callback(req), nil
}
