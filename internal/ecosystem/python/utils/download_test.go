package utils

import (
	"cli/internal/api"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

const singlePythonMultipartResponse = `<html><head>
<meta name="pypi:repository-version" content="1.1">
<title>Links for python-multipart</title>
</head>
<body>
<h1>Links for python-multipart</h1>
<a href="https://pypi.sealsecurity.io/simple/python-multipart/python_multipart-0.0.6+sp1-py3-none-any.whl#sha256=f0e4c2f76c58916ec258f246851bea091d14d4247a2fc3e18694461b1816e13b" data-requires-python=">=3.7" data-dist-info-metadata="sha256=2785907fdf571d24a0fc40c6edf1a0246a2d4bf1e9ed5882b69638ad0d8e8323" data-core-metadata="sha256=2785907fdf571d24a0fc40c6edf1a0246a2d4bf1e9ed5882b69638ad0d8e8323">python_multipart-0.0.6+sp1-py3-none-any.whl</a><br>


</body></html>`

const multiplePythonMultipartResponse = `<html><head>
<meta name="pypi:repository-version" content="1.1">
<title>Links for python-multipart</title>
</head>
<body>
<h1>Links for python-multipart</h1>
<a href="https://pypi.sealsecurity.io/simple/python-multipart/python_multipart-0.0.6+sp1-py3-none-any.whl" data-requires-python=">=3.7" data-dist-info-metadata="sha256=2785907fdf571d24a0fc40c6edf1a0246a2d4bf1e9ed5882b69638ad0d8e8323" data-core-metadata="sha256=2785907fdf571d24a0fc40c6edf1a0246a2d4bf1e9ed5882b69638ad0d8e8323">python_multipart-0.0.6+sp1-py3-none-any.whl</a><br>
<a href="https://pypi.sealsecurity.io/simple/python-multipart/python_multipart-0.0.6+sp1.tar.gz" data-requires-python=">=3.7">python_multipart-0.0.6+sp1.tar.gz</a><br>
<a href="https://files.pythonhosted.org/packages/94/35/142fff3d85da49377ada6936ad9b776263549ab22656969b2fcd0bdb10f7/python_multipart-0.0.7-py3-none-any.whl#sha256=b1fef9a53b74c795e2347daac8c54b252d9e0df9c619712691c1cc8021bd3c49" data-requires-python=">=3.7" data-dist-info-metadata="sha256=57c902c68d6038600c5f1947c451cdca5e8cae7edbfddcc75322806dff9efbcc" data-core-metadata="sha256=57c902c68d6038600c5f1947c451cdca5e8cae7edbfddcc75322806dff9efbcc">python_multipart-0.0.7-py3-none-any.whl</a><br>
<a href="https://files.pythonhosted.org/packages/67/94/bb4778be5d4c18329d60276d4e58b3974e2dce8ec3bee5569bfe9c81f36e/python_multipart-0.0.7.tar.gz#sha256=288a6c39b06596c1b988bb6794c6fbc80e6c369e35e5062637df256bee0c9af9" data-requires-python=">=3.7">python_multipart-0.0.7.tar.gz</a><br>
<a href="https://files.pythonhosted.org/packages/c0/3e/9fbfd74e7f5b54f653f7ca99d44ceb56e718846920162165061c4c22b71a/python_multipart-0.0.8-py3-none-any.whl#sha256=999725bf08cf7a071073d157a27cc34f8669af98da0d2435bde1cc1493a50ec3" data-requires-python=">=3.7" data-dist-info-metadata="sha256=f1d28f94347f1402076718203ec1ff53ae1893a11f3eaf12a61cba1038f43ddd" data-core-metadata="sha256=f1d28f94347f1402076718203ec1ff53ae1893a11f3eaf12a61cba1038f43ddd">python_multipart-0.0.8-py3-none-any.whl</a><br>


</body></html>`

func TestDownloadPython(t *testing.T) {
	name := "python-multipart"
	version := "0.0.6+sp1"
	fakePackageContent := `asdf` // sha256(asdf) -> f0e4c2f76c58916ec258f246851bea091d14d4247a2fc3e18694461b1816e13b
	transparentRoundTripper := api.TransparentRoundTripper{Callback: func(req *http.Request) *http.Response {

		uri := req.URL.Path
		var content string
		switch uri {
		case "/simple/python-multipart/":
			content = singlePythonMultipartResponse
		case "/simple/python-multipart/python_multipart-0.0.6+sp1-py3-none-any.whl":
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

	data, err := DownloadPythonPackage(server, name, version, []string{"py3-none-any"})
	if err != nil {
		t.Fatalf("got error %v", err)
	}
	if string(data) != fakePackageContent {
		t.Fatalf("got %s, expected %s", string(data), fakePackageContent)
	}
}

func TestDownloadPythonNoTag(t *testing.T) {
	name := "python-multipart"
	version := "0.0.6+sp1"
	fakePackageContent := `asdf` // sha256(asdf) -> f0e4c2f76c58916ec258f246851bea091d14d4247a2fc3e18694461b1816e13b
	transparentRoundTripper := api.TransparentRoundTripper{Callback: func(req *http.Request) *http.Response {

		uri := req.URL.Path
		var content string
		switch uri {
		case "/simple/python-multipart/":
			content = singlePythonMultipartResponse
		case "/simple/python-multipart/python_multipart-0.0.6+sp1-py3-none-any.whl":
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

	_, err := DownloadPythonPackage(server, name, version, []string{"py2-none-any"})
	if err == nil {
		t.Fatalf("got error %v", err)
	}
}

func TestGetVersionUrl(t *testing.T) {
	version := "0.0.6+sp1"
	tags := []string{"py3-none-any"}
	expected, err := url.Parse("https://pypi.sealsecurity.io/simple/python-multipart/python_multipart-0.0.6+sp1-py3-none-any.whl")
	if err != nil {
		t.Fatalf("failed parsing url %v", err)
	}
	url, err := getVersionUrl([]byte(multiplePythonMultipartResponse), version, tags)
	if err != nil || url != *expected {
		t.Fatalf("got %s, expected %s", url.String(), expected)
	}
}

func TestGetVersionUrlNoVersionMatch(t *testing.T) {
	version := "0.0.7+sp1"
	tags := []string{"py3-none-any"}
	_, err := getVersionUrl([]byte(multiplePythonMultipartResponse), version, tags)
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestGetVersionUrlNoTagMatch(t *testing.T) {
	version := "0.0.6+sp1"
	tags := []string{"py2-none-any"}
	_, err := getVersionUrl([]byte(multiplePythonMultipartResponse), version, tags)
	if err == nil {
		t.Fatalf("expected error")
	}
}
