package api

import (
	"cli/internal/common"
	"io"
	"net/http"
	"strings"
	"testing"
)

type requestValidatorCallback func(*http.Request)

type FakeRoundTripper struct {
	// data to return for request
	jsonContent string
	statusCode  int
	Validator   requestValidatorCallback
}

func (f FakeRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
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
	fakeRoundTripper := FakeRoundTripper{statusCode: 200, jsonContent: `{"items":[],"total":0,"limit":1,"offset":0}`}
	client := http.Client{Transport: fakeRoundTripper}
	method := "POST"
	url := "https://seal/a/url/endpoint"

	result, statusCode, err := sendSealRequestJson[BulkCheckRequest, Page[PackageVersion]](client, method, url, &request, nil, nil)
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
		Name:           "ejs-escaped",
		Version:        "2.7.4-sp1",
		PackageManager: "NPM",
	}
	request := BulkCheckRequest{
		Entries: []common.Dependency{dependency},
	}

	fakeRoundTripper := FakeRoundTripper{statusCode: 200, jsonContent: `
        {
			"items": [
				{
					"id": "a83e834c-1f7c-4db5-97b1-da8f02c1f95c",
					"recommended_library_version_id": "226353d8-4e42-4e6c-91f7-c5038be2c7fa",
					"recommended_library_version": "2.7.4-sp2",
					"library": {
						"id": "33af4a95-4249-4d3b-9fa5-424184fa4b76",
						"name": "ejs",
						"escaped_name": "ejs-escaped",
						"package_manager": "NPM",
						"source_link": "https://github.com/mde/ejs"
					},
					"version": "2.7.4-sp1",
					"open_vulnerabilities": [
						{
							"cve": "CVE-2024-33883",
							"nvd_score": null,
							"snyk_id": "SNYK-JS-EJS-6689533",
							"snyk_cvss_score": 5.3,
							"github_advisory_id": "GHSA-ghr5-ch3p-vcr6",
							"github_advisory_score": null,
							"unified_score": 5.3,
							"malicious_id": null
						}
					],
					"sealed_vulnerabilities": [
						{
							"cve": "CVE-2022-29078",
							"nvd_score": 9.8,
							"snyk_id": "SNYK-JS-EJS-2803307",
							"snyk_cvss_score": 8.1,
							"github_advisory_id": "GHSA-phwq-j96m-2c2q",
							"github_advisory_score": 9.8,
							"unified_score": 9.8,
							"malicious_id": null
						},
						{
							"cve": null,
							"nvd_score": null,
							"snyk_id": "SNYK-JS-EJS-1049328",
							"snyk_cvss_score": 4.1,
							"github_advisory_id": null,
							"github_advisory_score": null,
							"unified_score": 4.1,
							"malicious_id": null
						}
					],
					"is_hidden": null,
					"is_sealed": null,
					"last_pulled": null,
					"number_of_times_pulled": null,
					"origin_version": "2.7.4",
					"origin_version_id": "6f7a8ea8-c536-4e22-9d5d-69a25ac57899",
					"patch_stage": "uploaded",
					"patch_stage_result": "finished",
					"publish_date": "2024-05-01T07:45:58.493077Z"
				}
			],
			"total": 1,
			"limit": 1,
			"offset": 0
		}`}

	client := http.Client{Transport: fakeRoundTripper}

	method := "POST"
	url := "https://seal/a/url/endpoint"

	result, statusCode, err := sendSealRequestJson[BulkCheckRequest, Page[PackageVersion]](client, method, url, &request, nil, nil)
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
	if vulnerablePackage.VersionId != "a83e834c-1f7c-4db5-97b1-da8f02c1f95c" {
		t.Fatalf("wrong version id %s", vulnerablePackage.VersionId)
	}

	if vulnerablePackage.Library.Id != "33af4a95-4249-4d3b-9fa5-424184fa4b76" {
		t.Fatalf("wrong library id %s", vulnerablePackage.Library.Id)
	}

	if vulnerablePackage.Library.Name != dependency.Name {
		t.Fatalf("wrong library name %s != %s", dependency.Name, vulnerablePackage.Library.Name)
	}

	if vulnerablePackage.Version != dependency.Version {
		t.Fatalf("wrong library version %s != %s", dependency.Version, vulnerablePackage.Version)
	}

	if vulnerablePackage.Library.PackageManager != dependency.PackageManager {
		t.Fatalf("wrong package manager name %s != %s", dependency.PackageManager, vulnerablePackage.Library.PackageManager)
	}

	if vulnerablePackage.RecommendedLibraryVersionId != "226353d8-4e42-4e6c-91f7-c5038be2c7fa" {
		t.Fatalf("wrong recommended id %s", vulnerablePackage.RecommendedLibraryVersionId)
	}

	if vulnerablePackage.RecommendedLibraryVersionString != "2.7.4-sp2" {
		t.Fatalf("wrong recommended version %s", vulnerablePackage.RecommendedLibraryVersionString)
	}

	if len(vulnerablePackage.OpenVulnerabilities) != 1 {
		t.Fatalf("wrong number of open vulnerabilities %v", vulnerablePackage.OpenVulnerabilities)
	}
	if len(vulnerablePackage.SealedVulnerabilities) != 2 {
		t.Fatalf("wrong number of sealed vulnerabilities %v", vulnerablePackage.SealedVulnerabilities)
	}

	if !vulnerablePackage.IsSealed() {
		t.Fatalf("package is not sealed")
	}

	if vulnerablePackage.OriginVersionString != "2.7.4" {
		t.Fatalf("wrong origin version %s", vulnerablePackage.OriginVersionString)
	}

	if vulnerablePackage.OriginVersionId != "6f7a8ea8-c536-4e22-9d5d-69a25ac57899" {
		t.Fatalf("wrong origin version %s", vulnerablePackage.OriginVersionId)
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

	fakeRoundTripper := FakeRoundTripper{statusCode: 200, jsonContent: `{
        "items": [
            {
                "id": "6f7a8ea8-c536-4e22-9d5d-69a25ac57899",
                "library": {
                    "id": "33af4a95-4249-4d3b-9fa5-424184fa4b76",
                    "name": "ejs",
                    "escaped_name": "ejs",
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

	result, statusCode, err := sendSealRequestJson[BulkCheckRequest, Page[PackageVersion]](client, method, url, &request, nil, nil)
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
	fakeRoundTripper := FakeRoundTripper{statusCode: 200, Validator: func(req *http.Request) {
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

	_, _, _ = sendSealRequestJson[any, any](client, method, url, nil, nil, nil)
}

func TestCustomHeaderCliStartTime(t *testing.T) {
	// IMPORTANT: this will fail if run from vs code, see .vscode/settings.json
	fakeRoundTripper := FakeRoundTripper{statusCode: 200, Validator: func(req *http.Request) {

		timeValues := req.Header.Values(SealStartTimeHeader)
		if len(timeValues) == 0 {
			t.Fatalf("no cli start time header")
		}

		if len(timeValues) > 1 {
			t.Fatalf("multple cli start time headers %v", timeValues)
		}

		startTime := timeValues[0]
		if startTime == "" {
			t.Fatalf("empty startTime header value")
		}

		if startTime != common.CliStartTime.Format(common.StartTimeLayout) {
			t.Fatalf("wrong startTime header value - got `%s` expected `%s`", startTime, common.CliStartTime)
		}
	}}

	client := http.Client{Transport: fakeRoundTripper}

	method := "POST"
	url := "https://seal/a/url/endpoint"

	_, _, _ = sendSealRequestJson[any, any](client, method, url, nil, nil, nil)
}

func TestSessionIdHeaderAdded(t *testing.T) {
	// IMPORTANT: this will fail if run from vs code, see .vscode/settings.json
	fakeRoundTripper := FakeRoundTripper{statusCode: 200, Validator: func(req *http.Request) {
		sessionValues := req.Header.Values(SealSessionIdHeader)
		if len(sessionValues) == 0 {
			t.Fatalf("no session session header")
		}

		if len(sessionValues) > 1 {
			t.Fatalf("multple session session headers %v", sessionValues)
		}

		session := sessionValues[0]
		if session == "" {
			t.Fatalf("empty session header value")
		}

		if session != common.SessionId {
			t.Fatalf("wrong session header value - got `%s` expected `%s`", session, common.SessionId)
		}
	}}

	client := http.Client{Transport: fakeRoundTripper}

	method := "POST"
	url := "https://seal/a/url/endpoint"

	_, _, _ = sendSealRequestJson[any, any](client, method, url, nil, nil, nil)
	_, _, _ = sendSealRequestJson[any, any](client, method, url, nil, nil, nil) // check twice is using the same one
}

func TestHeaderUserAgent(t *testing.T) {
	fakeRoundTripper := FakeRoundTripper{statusCode: 200, Validator: func(req *http.Request) {
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

		expected := FormatUserAgent()
		if userAgent != expected {
			t.Fatalf("wrong user-agent header value - got `%s` expected `%s`", userAgent, expected)
		}
	}}

	client := http.Client{Transport: fakeRoundTripper}

	method := "POST"
	url := "https://seal/a/url/endpoint"

	_, _, _ = sendSealRequestJson[any, any](client, method, url, nil, nil, nil)
}

func TestExtraHeaderAdded(t *testing.T) {
	headerName := "Fake-Header"
	headerValue := "headervalue"
	fakeRoundTripper := FakeRoundTripper{statusCode: 200, Validator: func(req *http.Request) {
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

	_, _, _ = sendSealRequestJson[any, any](client,
		method,
		url,
		nil,
		[]StringPair{{Name: headerName, Value: headerValue}},
		nil,
	)
}

func TestQueryParamsAdded(t *testing.T) {
	paramName := "fake-param"
	paramValue := "paramvalue"
	fakeRoundTripper := FakeRoundTripper{statusCode: 200, Validator: func(req *http.Request) {
		query := req.URL.Query()
		if len(query) == 0 {
			t.Fatalf("param not added to request")
		}
		val := query.Get(paramName)
		if val != paramValue {
			t.Fatalf("wrong param value got %s, expected %s", val, paramValue)
		}
	}}

	client := http.Client{Transport: fakeRoundTripper}

	method := "POST"
	url := "https://seal/a/url/endpoint"

	_, _, _ = sendSealRequestJson[any, any](client,
		method,
		url,
		nil,
		nil,
		[]StringPair{{Name: paramName, Value: paramValue}},
	)
}
