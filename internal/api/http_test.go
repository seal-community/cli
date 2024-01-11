package api

import (
	"cli/internal/common"
	"io"
	"net/http"
	"strings"
	"testing"
)

type requestValidatorCallback func(*http.Request)
type roundTriphandler func(*http.Request) *http.Response
type transparentRoundTripper struct {
	Callback roundTriphandler
}

func (t transparentRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return t.Callback(req), nil
}

type fakeRoundTripper struct {
	// data to return for request
	jsonContent string
	statusCode  int
	Validator   requestValidatorCallback
}

func (f fakeRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	resp := new(http.Response)
	content := f.jsonContent
	resp.Body = io.NopCloser(strings.NewReader(content))
	resp.StatusCode = f.statusCode
	resp.Request = req

	if f.Validator != nil {
		f.Validator(req)
	}

	return resp, nil
}

func TestEmpty(t *testing.T) {
	request := BulkCheckRequest{
		Entries: []common.Dependency{},
	}
	fakeRoundTripper := fakeRoundTripper{statusCode: 200, jsonContent: `{"items":[],"total":0,"limit":1,"offset":0}`}
	client := http.Client{Transport: fakeRoundTripper}
	method := "POST"
	url := "https://seal/a/url/endpoint"

	result, statusCode, err := sendRequestJson[BulkCheckRequest, Page[PackageVersion]](client, method, url, &request)
	if err != nil {
		t.Fatalf("got error %v", err)
	}

	if statusCode != 200 {
		t.Fatalf("got wrong status code %v", statusCode)
	}

	if len(result.Items) != 0 {
		t.Fatalf("wrong number of items %v", result.Items)
	}

	if result.Total != 0 {
		t.Fatalf("wrong number of total items %v", result.Total)
	}
}

func TestSanity(t *testing.T) {
	dependency := common.Dependency{
		Name:           "ejs",
		Version:        "2.7.4",
		PackageManager: "NPM",
	}
	request := BulkCheckRequest{
		Entries: []common.Dependency{dependency},
	}

	fakeRoundTripper := fakeRoundTripper{statusCode: 200, jsonContent: `{
        "items": [
            {
                "id": "6f7a8ea8-c536-4e22-9d5d-69a25ac57899",
                "recommended_library_version_id": "a83e834c-1f7c-4db5-97b1-da8f02c1f95c",
                "recommended_library_version": "2.7.4-sp1",
                "library": {
                    "id": "33af4a95-4249-4d3b-9fa5-424184fa4b76",
                    "name": "ejs",
                    "package_manager": "NPM",
                    "source_link": "https://github.com/mde/ejs"
                },
                "version": "2.7.4",
                "open_vulnerabilities": [
                    {
                        "cve": "CVE-2022-29078",
                        "nvd_score": 9.8,
                        "snyk_id": "SNYK-JS-EJS-2803307",
                        "snyk_cvss_score": 8.1,
                        "github_advisory_id": "GHSA-phwq-j96m-2c2q",
                        "github_advisory_score": 9.8,
                        "unified_score": 9.8
                    },
                    {
                        "cve": null,
                        "nvd_score": null,
                        "snyk_id": "SNYK-JS-EJS-1049328",
                        "snyk_cvss_score": 4.1,
                        "github_advisory_id": null,
                        "github_advisory_score": null,
                        "unified_score": 4.1
                    }
                ],
                "sealed_vulnerabilities": [],
                "is_hidden": null,
                "is_sealed": null,
                "last_pulled": null,
                "number_of_times_pulled": null
            }
        ],
        "total": 1,
        "limit": 1,
        "offset": 0
    }`}

	client := http.Client{Transport: fakeRoundTripper}

	method := "POST"
	url := "https://seal/a/url/endpoint"

	result, statusCode, err := sendRequestJson[BulkCheckRequest, Page[PackageVersion]](client, method, url, &request)
	if err != nil {
		t.Fatalf("got error %v", err)
	}

	if statusCode != 200 {
		t.Fatalf("got wrong status code %v", statusCode)
	}

	if len(result.Items) != 1 {
		t.Fatalf("wrong number of items %v", result.Items)
	}

	if result.Total != 1 {
		t.Fatalf("wrong number of total items %v", result.Total)
	}

	vulnerablePackage := result.Items[0]
	if vulnerablePackage.Library.Name != dependency.Name {
		t.Fatalf("wrong library name %s != %s", dependency.Name, vulnerablePackage.Library.Name)

	}

	if vulnerablePackage.Version != dependency.Version {
		t.Fatalf("wrong library name %s != %s", dependency.Version, vulnerablePackage.Version)
	}

	if vulnerablePackage.Version != dependency.Version {
		t.Fatalf("wrong package manager name %s != %s", dependency.PackageManager, vulnerablePackage.Library.PackageManager)
	}
}

func TestMalicious(t *testing.T) {
	dependency := common.Dependency{
		Name:           "ejs",
		Version:        "2.7.4",
		PackageManager: "NPM",
	}
	request := BulkCheckRequest{
		Entries: []common.Dependency{dependency},
	}

	fakeRoundTripper := fakeRoundTripper{statusCode: 200, jsonContent: `{
        "items": [
            {
                "id": "6f7a8ea8-c536-4e22-9d5d-69a25ac57899",
                "library": {
                    "id": "33af4a95-4249-4d3b-9fa5-424184fa4b76",
                    "name": "ejs",
                    "package_manager": "NPM",
                    "source_link": "https://github.com/mde/ejs"
                },
                "version": "2.7.4",
                "open_vulnerabilities": [
                    {
						"malicious_id": "MAL-2022-7421",
                        "unified_score": 10.0
                    }
                ],
                "sealed_vulnerabilities": [],
                "is_hidden": null,
                "is_sealed": null,
                "last_pulled": null,
                "number_of_times_pulled": null
            }
        ],
        "total": 1,
        "limit": 1,
        "offset": 0
    }`}

	client := http.Client{Transport: fakeRoundTripper}

	method := "POST"
	url := "https://seal/a/url/endpoint"

	result, statusCode, err := sendRequestJson[BulkCheckRequest, Page[PackageVersion]](client, method, url, &request)
	if err != nil {
		t.Fatalf("got error %v", err)
	}

	if statusCode != 200 {
		t.Fatalf("got wrong status code %v", statusCode)
	}

	if len(result.Items) != 1 {
		t.Fatalf("wrong number of items %v", result.Items)
	}

	if result.Total != 1 {
		t.Fatalf("wrong number of total items %v", result.Total)
	}

	vulnerablePackage := result.Items[0]

	if vulnerablePackage.OpenVulnerabilities[0].MaliciousID != "MAL-2022-7421" {
		t.Fatalf("wrong malicious id %s", vulnerablePackage.OpenVulnerabilities[0].MaliciousID)
	}
}

func TestCustomHeaderCliVersion(t *testing.T) {
	// IMPORTANT: this will fail if run from vs code, see .vscode/settings.json
	fakeRoundTripper := fakeRoundTripper{statusCode: 200, Validator: func(req *http.Request) {
		versionValues := req.Header.Values(SealVersionHeader)
		if len(versionValues) == 0 {
			t.Fatalf("no cli version header")
		}

		if len(versionValues) > 1 {
			t.Fatalf("multple cli version headers %v", versionValues)
		}
		version := versionValues[0]
		if version == "" {
			t.Fatalf("empty version header value")
		}

		if version != common.CliVersion {
			t.Fatalf("wrong version header value - got `%s` expected `%s`", version, common.CliVersion)
		}
	}}

	client := http.Client{Transport: fakeRoundTripper}

	method := "POST"
	url := "https://seal/a/url/endpoint"

	_, _, _ = sendRequestJson[any, any](client, method, url, nil)
}

func TestHeaderUserAgent(t *testing.T) {
	fakeRoundTripper := fakeRoundTripper{statusCode: 200, Validator: func(req *http.Request) {
		userAgentValues := req.Header.Values("User-Agent")
		if len(userAgentValues) == 0 {
			t.Fatalf("no user-agent header")
		}

		if len(userAgentValues) > 1 {
			t.Fatalf("multple user-agent headers %v", userAgentValues)
		}
		userAgent := userAgentValues[0]
		if userAgent == "" {
			t.Fatalf("empty user-agent header value")
		}

		expected := formatUserAgent()
		if userAgent != expected {
			t.Fatalf("wrong user-agent header value - got `%s` expected `%s`", userAgent, expected)
		}
	}}

	client := http.Client{Transport: fakeRoundTripper}

	method := "POST"
	url := "https://seal/a/url/endpoint"

	_, _, _ = sendRequestJson[any, any](client, method, url, nil)
}

func TestExtraHeaderAdded(t *testing.T) {
	headerName := "Fake-Header"
	headerValue := "headervalue"
	fakeRoundTripper := fakeRoundTripper{statusCode: 200, Validator: func(req *http.Request) {
		vals := req.Header.Values(headerName)
		if len(vals) == 0 {
			t.Fatalf("extra header not added to request")
		}

		if len(vals) > 1 {
			t.Fatalf("wrong number of header blues %v", vals)
		}

		if vals[0] != headerValue {
			t.Fatalf("wrong headar value got %s, expected %s", headerValue, vals[0])
		}
	}}

	client := http.Client{Transport: fakeRoundTripper}

	method := "POST"
	url := "https://seal/a/url/endpoint"

	_, _, _ = sendRequestJson[any, any](client,
		method,
		url,
		nil,
		HeaderPair{Name: headerName, Value: headerValue},
	)
}
