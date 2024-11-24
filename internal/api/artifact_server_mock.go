//go:build mock
// +build mock

// this will only be bundled when running unit-tests

package api

import "encoding/json"

type GetValidatorType func(uri string, params []StringPair, extraHdrs []StringPair) (data []byte, code int, err error)
type GetJsonObjectValidatorType func(url string, headers []StringPair, params []StringPair, obj any) (int, error)

type FakeArtifactServer struct {
	ExtraHeaders []StringPair

	// return this unless validator is provided
	Data []byte
	Code int
	Err  error

	GetValidator           GetValidatorType
	GetJsonObjectValidator GetJsonObjectValidatorType
}

func (s *FakeArtifactServer) Get(uri string, params []StringPair, extraHdrs []StringPair) (data []byte, code int, err error) {
	if s.GetValidator != nil {
		return s.GetValidator(uri, params, extraHdrs)
	}

	return s.Data, s.Code, s.Err
}

func (s *FakeArtifactServer) GetJsonObject(url string, headers []StringPair, params []StringPair, obj any) (int, error) {
	if s.GetJsonObjectValidator != nil {
		return s.GetJsonObjectValidator(url, headers, params, obj)
	}

	err := json.Unmarshal(s.Data, obj)
	if err != nil {
		return s.Code, err
	}

	return s.Code, s.Err
}

func (s *FakeArtifactServer) SetExtraHeaders(headers []StringPair) {
	s.ExtraHeaders = headers
}
