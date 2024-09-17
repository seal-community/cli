package utils

import (
	"cli/internal/api"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestDownloadMaven(t *testing.T) {
	name := "com.example.package:package"
	version := "1.2.3+sp1"
	fakePackageContent := `asdf`
	token := "this-a-token"
	project := "this-a-proj"
	fakePackageSha1 := `3da541559918a808c2402bba5012f6c60b27661c`
	transparentRoundTripper := api.TransparentRoundTripper{Callback: func(req *http.Request) *http.Response {

		uri := req.URL.Path
		var content string
		switch uri {
		case "/com/example/package/package/1.2.3+sp1/package-1.2.3+sp1.jar":
			content = fakePackageContent
		case "/com/example/package/package/1.2.3+sp1/package-1.2.3+sp1.jar.sha1":
			content = fakePackageSha1

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
	server := api.NewArtifactServer("http://baseurl.com", token, project, client)

	data, err := DownloadMavenPackage(server, name, version)
	if err != nil {
		t.Fatalf("got error %v", err)
	}

	if string(data) != fakePackageContent {
		t.Fatalf("got %s, expected %s", string(data), fakePackageContent)
	}
}
