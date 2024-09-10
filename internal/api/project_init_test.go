package api

import (
	"io"
	"net/http"
	"testing"
)

func TestInitializeProjectSanity(t *testing.T) {

	fakeRoundTripper := fakeRoundTripper{statusCode: 200,
		jsonContent: `{"is_new": true, "name": "pretty-name", "tag":"proj-id"}`,
		Validator: func(req *http.Request) {

			if req.Method != "POST" {
				t.Fatalf("bad method: %s", req.Method)
			}

			if req.URL.Path != "/authenticated/v1/project" {
				t.Fatalf("bad endpoint uri: %s", req.URL.RawPath)
			}

			body, err := io.ReadAll(req.Body)
			if err != nil {
				t.Fatalf("failed to read body: %v", err)
			}

			expected := `{"name":"pretty-name","tag":"proj-id"}`
			if string(body) != expected {
				t.Fatalf("expected %s, got %s", expected, string(body))
			}
		},
	}

	client := http.Client{Transport: fakeRoundTripper}
	s := Server{Client: client, AuthToken: "AAA"}
	result, err := s.InitializeProject("proj-id", "pretty-name")

	if err != nil || result == nil {
		t.Fatalf("failed %v, res: %v", err, result)
	}

	if result.Name != "pretty-name" {
		t.Fatalf("wrong name: %s", result.Name)
	}

	if result.Tag != "proj-id" {
		t.Fatalf("wrong tag: %s", result.Tag)
	}

	if !result.New {
		t.Fatalf("not new")
	}
}

func TestInitializeProjectNoTokenFails(t *testing.T) {

	fakeRoundTripper := fakeRoundTripper{statusCode: 200,
		jsonContent: `{"is_new": true, "name": "pretty-name", "tag":"proj-id"}`,
	}

	client := http.Client{Transport: fakeRoundTripper}
	s := Server{Client: client, AuthToken: ""}
	result, err := s.InitializeProject("proj-id", "pretty-name")

	if err != MissingTokenForApiRequest || result != nil {
		t.Fatalf("should fail without token %v, page: %v", err, result)
	}
}
