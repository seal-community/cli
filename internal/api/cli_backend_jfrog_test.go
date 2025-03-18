package api

import (
	"errors"
	"fmt"
	"net/http"
	"testing"
)

func TestJFrogRemoteConfigQueryDisabledResponse(t *testing.T) {
	proj := "proj-id"
	recommendedId := "2222"

	query := RemoteOverrideQuery{
		LibraryId:            "000",
		OriginVersionId:      "111",
		RecommendedVersionId: &recommendedId,
	}

	fakeRoundTripper := FakeRoundTripper{statusCode: 405,
		jsonContent: ``, Validator: func(r *http.Request) {
			if r.Method != "GET" {
				t.Fatalf("bad method %s", r.Method)
			}

			if r.URL.Path != fmt.Sprintf("/artifactory/repository-name/path/to/artifact/fixes/remote/%s", proj) {
				t.Fatalf("bad url %s", r.URL.Path)
			}
		},
	}

	client := http.Client{Transport: fakeRoundTripper}
	server := NewCliJfrogServer(client, proj, "test", "https://seal.jfrog.io/artifactory/repository-name/path/to/artifact")
	page, err := server.QueryRemoteConfig([]RemoteOverrideQuery{query})

	if err == nil {
		t.Fatalf("failed send unitest %v, page: %v", err, page)
	}

	if !errors.Is(err, RemoteOverrideDisabledError) {
		t.Fatalf("bad error %v", err)
	}
}
