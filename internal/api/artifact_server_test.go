package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
)

var baseUrl = "https://artifact-server.sealsecurity.io"

func TestValidateRelativeUri(t *testing.T) {
	options := []string{
		"a/b/c/d.txt",
		"/x.txt",
		"/x.txt?a=1",
	}

	for i, uri := range options {
		t.Run(fmt.Sprintf("code_%d", i), func(tt *testing.T) {
			if err := ValidateRelativeUri(baseUrl, uri); err != nil {
				tt.Fatalf("failed validating `%s` %v", uri, err)
			}
		})
	}

}

func TestValidateRelativeUriBad(t *testing.T) {
	options := []string{
		"http://a/b/c/d.txt",
		"https://www.google.com",
		"ftp://www.google.com",
	}

	for i, uri := range options {
		t.Run(fmt.Sprintf("code_%d", i), func(tt *testing.T) {
			if err := ValidateRelativeUri(baseUrl, uri); err == nil {
				tt.Fatalf("failed validating bad uri `%s` %v", uri, err)
			}
		})
	}

}

func TestGetBadUri(t *testing.T) {
	baseurl := "http://example.com"
	uri := "https://my/uri/a.txt"

	server := NewArtifactServer(baseurl, "", "proj-id", http.Client{})

	_, _, err := server.Get(uri, nil, nil)

	if err != ArtifactServerUnsupportedMethod {
		t.Fatalf("err %v", err)
	}

}

func TestGetNoToken(t *testing.T) {
	jsonData := `{"text":"abcdefgh"}`
	baseurl := "http://example.com"
	uri := "my/uri/a.txt"
	fakeRoundTripper := FakeRoundTripper{statusCode: 200,
		Validator: func(req *http.Request) {
			if req.URL.Path != "/my/uri/a.txt" {
				t.Fatalf("bad endpoint uri: %s", req.URL.Path)
			}

			if req.Host != "example.com" {
				t.Fatalf("bad endpoint host: %s", req.URL.Host)
			}

			if req.URL.Scheme != "http" {
				t.Fatalf("bad scheme host: %s", req.URL.Scheme)
			}

			authValues := req.Header.Values("Authorization")
			if len(authValues) != 0 {
				t.Fatalf("got auth header: %v", authValues)
			}
		},
		jsonContent: jsonData,
	}

	client := http.Client{Transport: fakeRoundTripper}

	server := NewArtifactServer(baseurl, "", "proj-id", client)

	data, statusCode, err := server.Get(uri, nil, nil)

	if err != nil {
		t.Fatalf("err %v", err)
	}

	if string(data) != jsonData {
		t.Fatalf("bad data: `%s` expected `%s`", string(data), jsonData)
	}

	if statusCode != 200 {
		t.Fatalf("bad code %d", statusCode)
	}
}

func TestGetToken(t *testing.T) {
	jsonData := `{"text":"abcdefgh"}`
	baseurl := "http://example.com"
	uri := "my/uri/a.txt"
	token := "my-token"
	projId := "proj-id"
	fakeRoundTripper := FakeRoundTripper{statusCode: 200,
		Validator: func(req *http.Request) {
			if req.URL.Path != "/my/uri/a.txt" {
				t.Fatalf("bad endpoint uri: %s", req.URL.Path)
			}

			if req.Host != "example.com" {
				t.Fatalf("bad endpoint host: %s", req.URL.Host)
			}

			if req.URL.Scheme != "http" {
				t.Fatalf("bad scheme host: %s", req.URL.Scheme)
			}

			authValues := req.Header.Values("Authorization")
			if len(authValues) == 0 {
				t.Fatalf("no auth header")
			}

			if len(authValues) > 1 {
				t.Fatalf("multple auth headers %v", authValues)
			}
			auth := authValues[0]
			if auth == "" {
				t.Fatalf("empty auth header value")
			}

			expected := fmt.Sprintf("Basic %s", buildAuthToken(token, projId))
			if auth != expected {
				t.Fatalf("bad token value in header; got %s, expected %s", auth, expected)
			}
		},
		jsonContent: jsonData,
	}

	client := http.Client{Transport: fakeRoundTripper}

	server := NewArtifactServer(baseurl, token, projId, client)

	data, statusCode, err := server.Get(uri, nil, nil)

	if err != nil {
		t.Fatalf("err %v", err)
	}

	if string(data) != jsonData {
		t.Fatalf("bad data: `%s` expected `%s`", string(data), jsonData)
	}

	if statusCode != 200 {
		t.Fatalf("bad code %d", statusCode)
	}
}

func TestGetJson(t *testing.T) {
	jsonData := `{"text":"abcdefgh"}`
	baseurl := "http://example.com"
	uri := "my/uri/a.txt"
	fakeRoundTripper := FakeRoundTripper{statusCode: 200,
		Validator: func(req *http.Request) {
			if req.URL.Path != "/my/uri/a.txt" {
				t.Fatalf("bad endpoint uri: %s", req.URL.Path)
			}

			if req.Host != "example.com" {
				t.Fatalf("bad endpoint host: %s", req.URL.Host)
			}

			if req.URL.Scheme != "http" {
				t.Fatalf("bad scheme host: %s", req.URL.Scheme)
			}

		},
		jsonContent: jsonData,
	}

	client := http.Client{Transport: fakeRoundTripper}

	server := NewArtifactServer(baseurl, "my-token", "proj-id", client)

	type Response struct {
		Text string `json:"text"`
	}

	var resp Response

	statusCode, err := server.GetJsonObject(uri, nil, nil, &resp)

	if err != nil {
		t.Fatalf("err %v", err)
	}

	if statusCode != 200 {
		t.Fatalf("bad code %d", statusCode)
	}

	if resp.Text != "abcdefgh" {
		t.Fatalf("bad data: `%s`", resp.Text)
	}
}

func TestGetJsonBadObjNil(t *testing.T) {

	baseurl := "http://example.com"
	uri := "my/uri/a.txt"

	server := NewArtifactServer(baseurl, "my-token", "proj-id", http.Client{})

	_, err := server.GetJsonObject(uri, nil, nil, nil)

	if err != NilResponseObjectType {
		t.Fatalf("err %v", err)
	}
}

func TestGetJsonGetErr(t *testing.T) {

	baseurl := "http://example.com"
	uri := "http://my/uri/a.txt" // should fail the inner get func

	server := NewArtifactServer(baseurl, "my-token", "proj-id", http.Client{})

	type Response struct {
		Text string `json:"text"`
	}

	var resp Response

	_, err := server.GetJsonObject(uri, nil, nil, &resp)
	if err != ArtifactServerUnsupportedMethod {
		t.Fatalf("bad err: %v", err)
	}
}

func TestGetJsonBadObj(t *testing.T) {

	jsonData := `{"text":"abcdefgh"}`
	baseurl := "http://example.com"
	uri := "my/uri/a.txt"
	fakeRoundTripper := FakeRoundTripper{statusCode: 200, jsonContent: jsonData}

	client := http.Client{Transport: fakeRoundTripper}

	server := NewArtifactServer(baseurl, "my-token", "proj-id", client)

	type Response struct {
		Text string `json:"text"`
	}
	var resp Response
	_, err := server.GetJsonObject(uri, nil, nil, resp)

	if _, ok := err.(*json.InvalidUnmarshalError); !ok {
		t.Fatalf("bad err: %v", err)
	}
}
