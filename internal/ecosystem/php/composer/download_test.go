package composer

import (
	"cli/internal/api"
	"io"
	"net/http"
	"strings"
	"testing"
)

const fakePackageContent = `asdf`

func TestBuildUriSanity(t *testing.T) {
	name := "package"
	version := "0.0.1+sp1"
	expected := "/download/package/0.0.1+sp1/package-0.0.1+sp1.zip"
	uri := buildUri(name, version)
	if uri != expected {
		t.Fatalf("got %s, expected %s", uri, expected)
	}
}

func TestBuildUriVendor(t *testing.T) {
	name := "vendor/package"
	version := "0.0.1+sp1"
	uri := buildUri(name, version)
	expected := "/download/vendor/package/0.0.1+sp1/vendor-package-0.0.1+sp1.zip"
	if uri != expected {
		t.Fatalf("got %s, expected %s", uri, expected)
	}
}

func TestBuildUriVersionNormalization(t *testing.T) {
	name := "vendor/package"
	version := "v0.0.1+sp1"
	uri := buildUri(name, version)
	expected := "/download/vendor/package/0.0.1+sp1/vendor-package-0.0.1+sp1.zip"
	if uri != expected {
		t.Fatalf("got %s, expected %s", uri, expected)
	}
}

func TestDownloadComposerSanity(t *testing.T) {
	name := "vendor/package"
	version := "0.0.1+sp1"
	token := "this-a-token"
	project := "this-a-proj"
	transparentRoundTripper := api.TransparentRoundTripper{Callback: func(req *http.Request) *http.Response {

		uri := req.URL.Path
		var content string

		if uri == "/download/vendor/package/0.0.1+sp1/vendor-package-0.0.1+sp1.zip" {
			content = fakePackageContent
		} else {
			t.Fatalf("unsupported url request `%s`", uri)
		}

		return &http.Response{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(content)),
			Request:    req,
		}
	}}

	client := http.Client{Transport: transparentRoundTripper}
	server := api.NewArtifactServer("http://baseurl.com", token, project, client)

	data, err := downloadPackage(server, name, version)
	if err != nil {
		t.Fatalf("got error %v", err)
	}
	if string(data) != fakePackageContent {
		t.Fatalf("got %s, expected %s", string(data), fakePackageContent)
	}
}
