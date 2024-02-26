package utils

import (
	"cli/internal/api"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestDownloadNpm(t *testing.T) {
	name := "lodash"
	version := "123-sp1"
	fakePackageContent := `asdf` // sha1(asdf) -> 3da541559918a808c2402bba5012f6c60b27661c
	transparentRoundTripper := api.TransparentRoundTripper{Callback: func(req *http.Request) *http.Response {

		uri := req.URL.Path
		var content string
		switch uri {
		case "/lodash/":
			content = `{
				"versions": {
					"123-sp1": {
						"dist": {
							"shasum": "3da541559918a808c2402bba5012f6c60b27661c",
							"tarball": "https://registry.npmjs.org/lodash/-/lodash-123-sp1.tgz"
						}
					} 
				}
			}`
		case "/lodash/-/lodash-123-sp1.tgz":
			content = fakePackageContent

		default:
			t.Fatalf("unsupported url request `%s`", uri)
		}

		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(content)),
			Request:    req,
		}
	}}

	client := http.Client{Transport: transparentRoundTripper}
	server := api.Server{Client: client}

	data, err := DownloadNPMPackage(server, name, version)
	if err != nil {
		t.Fatalf("got error %v", err)
	}
	if string(data) != fakePackageContent {
		t.Fatalf("got %s, expected %s", string(data), fakePackageContent)
	}
}
