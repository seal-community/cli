package api

import (
	"cli/internal/common"
	"cli/internal/ecosystem/mappings"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"testing"
)

func TestQueryPackages(t *testing.T) {
	authToken := "thisisjustatribute"
	proj := "proj-id"

	dep := common.Dependency{Name: "ejs", Version: "2.7.4", PackageManager: mappings.NpmManager}
	fakeRoundTripper := FakeRoundTripper{statusCode: 200,
		Validator: func(req *http.Request) {
			if fixedParam := req.URL.Query().Get("fixed"); fixedParam != "0" {
				t.Fatalf("did not set fixed param correctly for only vulnerable: %s", fixedParam)
			}

			if req.URL.Path != "/unauthenticated/v1/bulk" {
				t.Fatalf("bad endpoint uri: %s", req.URL.RawPath)
			}

			body, err := io.ReadAll(req.Body)
			if err != nil {
				t.Fatalf("failed to read body: %v", err)
			}

			expected := "{\"entries\":[{\"library_name\":\"ejs\",\"library_version\":\"2.7.4\",\"library_package_manager\":\"NPM\"}],\"metadata\":null}"
			if string(body) != expected {
				t.Fatalf("expected %s, got %s", expected, string(body))
			}
		},
		jsonContent: `{"items":[],"total":0,"limit":1,"offset":0}`,
	}

	client := http.Client{Transport: fakeRoundTripper}
	server := NewCliServer(authToken, proj, client)

	page, err := server.QueryPackages(&BulkCheckRequest{
		Metadata: nil,
		Entries:  []common.Dependency{dep},
	}, OnlyVulnerable)

	if err != nil || page == nil {
		t.Fatalf("should fail without token %v, page: %v", err, page)
	}
}

func TestBulkQueryAuthMissingToken(t *testing.T) {
	client := http.Client{Transport: nil}
	server := NewCliServer("", "proj", client)

	_, err := server.QueryPackagesAuth(nil, OnlyVulnerable, false)
	if err != MissingTokenForApiRequest {
		t.Fatalf("should fail without token %v", err)
	}
}

func TestBulkQueryAuthGenerateActivity(t *testing.T) {
	dep := common.Dependency{Name: "ejs", Version: "2.7.4", PackageManager: mappings.NpmManager}
	proj := "myproj"
	authToken := "thisisjustatribute"
	fakeRoundTripper := FakeRoundTripper{statusCode: 200,
		Validator: func(req *http.Request) {
			if req.URL.Path != "/authenticated/v1/scan/myproj" {
				t.Fatalf("bad endpoint uri: %s", req.URL.RawPath)
			}

			if storeParam := req.URL.Query().Get("store"); storeParam != "1" {
				t.Fatalf("did not set store param correctly for generating activity: %s", storeParam)
			}

			body, err := io.ReadAll(req.Body)
			if err != nil {
				t.Fatalf("failed to read body: %v", err)
			}

			expected := "{\"entries\":[{\"library_name\":\"ejs\",\"library_version\":\"2.7.4\",\"library_package_manager\":\"NPM\"}],\"metadata\":null}"
			if string(body) != expected {
				t.Fatalf("expected %s, got %s", expected, string(body))
			}
		},
		jsonContent: `{"items":[],"total":0,"limit":1,"offset":0}`,
	}

	client := http.Client{Transport: fakeRoundTripper}
	server := NewCliServer(authToken, proj, client)

	page, err := server.QueryPackagesAuth(&BulkCheckRequest{
		Metadata: nil,
		Entries:  []common.Dependency{dep},
	}, OnlyVulnerable, true)

	if err != nil || page == nil {
		t.Fatalf("should fail without token %v, page: %v", err, page)
	}
}

func TestBulkQueryAuth(t *testing.T) {
	dep := common.Dependency{Name: "ejs", Version: "2.7.4", PackageManager: mappings.NpmManager}
	authToken := "thisisjustatribute"
	proj := "myproj"

	fakeRoundTripper := FakeRoundTripper{statusCode: 200,
		Validator: func(req *http.Request) {
			if req.URL.Path != "/authenticated/v1/scan/myproj" {
				t.Fatalf("bad endpoint uri: %s", req.URL.RawPath)
			}

			if storeParam := req.URL.Query().Get("store"); storeParam != "" {
				t.Fatalf("found store param without generating activity: %s", storeParam)
			}

			body, err := io.ReadAll(req.Body)
			if err != nil {
				t.Fatalf("failed to read body: %v", err)
			}

			expected := "{\"entries\":[{\"library_name\":\"ejs\",\"library_version\":\"2.7.4\",\"library_package_manager\":\"NPM\"}],\"metadata\":null}"
			if string(body) != expected {
				t.Fatalf("expected %s, got %s", expected, string(body))
			}
		},
		jsonContent: `{"items":[],"total":0,"limit":1,"offset":0}`,
	}

	client := http.Client{Transport: fakeRoundTripper}
	server := NewCliServer(authToken, proj, client)

	page, err := server.QueryPackagesAuth(&BulkCheckRequest{
		Metadata: nil,
		Entries:  []common.Dependency{dep},
	}, OnlyVulnerable, false)

	if err != nil || page == nil {
		t.Fatalf("should fail without token %v, page: %v", err, page)
	}
}

func TestInitializeProject(t *testing.T) {
	authToken := "thisisjustatribute"
	proj := "proj-id"
	projName := "pretty"

	fakeRoundTripper := FakeRoundTripper{statusCode: 200, Validator: func(req *http.Request) {
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

		expected := fmt.Sprintf("Basic %s", buildAuthToken(authToken, proj))
		if auth != expected {
			t.Fatalf("bad token value in header; got %s, expected %s", auth, expected)
		}

		if req.URL.Path != "/authenticated/v1/project" {
			t.Fatalf("bad request uri: %s", req.URL.RawPath)
		}

		if req.Method != "POST" {
			t.Fatalf("bad request method: %s", req.Method)
		}

		var payload []byte
		payload, err := io.ReadAll(req.Body)
		if err != nil {
			t.Fatalf("failed reading body: %v", err)
		}

		if string(payload) != fmt.Sprintf(`{"name":"%s","tag":"%s"}`, projName, proj) {
			t.Fatalf("bad body: `%s`", string(payload))
		}
	}, jsonContent: fmt.Sprintf(`{"name":"%s","tag":"%s","is_new":true}`, projName, proj)}

	client := http.Client{Transport: fakeRoundTripper}
	server := NewCliServer(authToken, proj, client)

	desc, err := server.InitializeProject(projName)
	if err != nil {
		t.Fatalf("got error %v", err)
	}

	if !desc.New {
		t.Fatalf("bad is_new value %t", desc.New)
	}

	if desc.Name != projName {
		t.Fatalf("bad response name %s; expected %s", desc.Name, projName)
	}

	if desc.Tag != proj {
		t.Fatalf("bad response tag %s; expected %s", desc.Tag, proj)
	}
}

func TestAuthentication(t *testing.T) {
	authToken := "thisisjustatribute"
	proj := "proj-id"
	fakeRoundTripper := FakeRoundTripper{statusCode: 200, Validator: func(req *http.Request) {
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

		expected := fmt.Sprintf("Basic %s", buildAuthToken(authToken, proj))
		if auth != expected {
			t.Fatalf("bad token value in header; got %s, expected %s", auth, expected)
		}
	}}

	client := http.Client{Transport: fakeRoundTripper}
	server := NewCliServer(authToken, proj, client)

	err := server.CheckAuthenticationValid()
	if err != nil {
		t.Fatalf("got error %v", err)
	}

}

func TestAuthenticaionFailureOnStatusCode(t *testing.T) {
	authToken := "thisisjustatribute"
	statusCodes := []struct {
		code int
		ok   bool
	}{{100, false}, {101, false}, {200, true}, {201, true}, {300, false}, {301, false}, {400, false}, {403, false}, {404, false}, {500, false}, {501, false}, {502, false}}

	for _, testCase := range statusCodes {

		t.Run(fmt.Sprintf("code_%d", testCase.code), func(t *testing.T) {

			fakeRoundTripper := FakeRoundTripper{statusCode: testCase.code}
			client := http.Client{Transport: fakeRoundTripper}
			server := NewCliServer(authToken, "proj", client)

			err := server.CheckAuthenticationValid()
			if testCase.ok && err != nil {
				t.Fatalf("got error %v for code %d", err, testCase.code)
			}

			if !testCase.ok && err == nil {
				t.Fatalf("expected error for code %d", testCase.code)
			}
		})
	}
}

func TestInitializeProjectNoTokenFails(t *testing.T) {

	s := NewCliServer("", "proj-id", http.Client{})
	result, err := s.InitializeProject("pretty-name")

	if err != MissingTokenForApiRequest || result != nil {
		t.Fatalf("should fail without token %v, page: %v", err, result)
	}
}

func TestRemoteConfigQuerySanity(t *testing.T) {
	authToken := "thisisjustatribute"
	proj := "proj-id"
	recommendedId := "2222"

	query := RemoteOverrideQuery{
		LibraryId:            "000",
		OriginVersionId:      "111",
		RecommendedVersionId: &recommendedId,
	}

	fakeRoundTripper := FakeRoundTripper{statusCode: 200,
		jsonContent: `{
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
		}`, Validator: func(r *http.Request) {
			if r.Method != "POST" {
				t.Fatalf("bad method %s", r.Method)
			}

			if r.URL.Path != fmt.Sprintf("/authenticated/v1/fixes/remote/%s", proj) {
				t.Fatalf("bad url %s", r.URL.Path)
			}
		},
	}

	client := http.Client{Transport: fakeRoundTripper}
	server := NewCliServer(authToken, proj, client)
	page, err := server.QueryRemoteConfig([]RemoteOverrideQuery{query})

	if err != nil || page == nil {
		t.Fatalf("failed send unitest %v, page: %v", err, page)
	}

	if len(page.Items) != 1 {
		t.Fatalf("bad number of returned items %v", len(page.Items))
	}
}

func TestRemoteConfigQueryNoToken(t *testing.T) {

	recommendedId := "2222"
	proj := "proj-id"

	query := RemoteOverrideQuery{
		LibraryId:            "000",
		OriginVersionId:      "111",
		RecommendedVersionId: &recommendedId,
	}

	fakeRoundTripper := FakeRoundTripper{statusCode: 200,
		jsonContent: `{"items":[],"total":0,"limit":1,"offset":0}`,
	}

	client := http.Client{Transport: fakeRoundTripper}
	server := NewCliServer("", proj, client)
	page, err := server.QueryRemoteConfig([]RemoteOverrideQuery{query})

	if err != MissingTokenForApiRequest || page != nil {
		t.Fatalf("should fail without token %v, page: %v", err, page)
	}
}

func TestRemoteConfigQueryProjectDoesNotExist(t *testing.T) {

	recommendedId := "2222"
	authToken := "thisisjustatribute"
	proj := "proj-id-does-not-exist"

	query := RemoteOverrideQuery{
		LibraryId:            "000",
		OriginVersionId:      "111",
		RecommendedVersionId: &recommendedId,
	}

	fakeRoundTripper := FakeRoundTripper{statusCode: 404}

	client := http.Client{Transport: fakeRoundTripper}
	server := NewCliServer(authToken, proj, client)
	page, err := server.QueryRemoteConfig([]RemoteOverrideQuery{query})

	if err != NonExistentProjectError || page != nil {
		t.Fatalf("should fail without token %v, page: %v", err, page)
	}
}

func TestPayloadify(t *testing.T) {
	d := struct {
		Name   string
		hidden string
	}{Name: "ahoy", hidden: "hidden"}
	b64 := payloadify(d)

	jsonData, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		t.Fatalf("failed decoding base64: %v", err)
	}

	if string(jsonData) != `{"Name":"ahoy"}` {
		t.Fatalf("bad payload: `%s`", jsonData)
	}
}

func TestQuerySilenceRules(t *testing.T) {
	authToken := "thisisjustatribute"
	proj := "proj-id"

	fakeRoundTripper := FakeRoundTripper{statusCode: 200,
		jsonContent: `[
			{
				"library": "name",
				"version": "version",
				"manager": "npm"
			}
		]`,
		Validator: func(r *http.Request) {
			if r.Method != "GET" {
				t.Fatalf("bad method %s", r.Method)
			}

			if r.URL.Path != fmt.Sprintf("/authenticated/v1/scanner_exclusions/%s", proj) {
				t.Fatalf("bad url %s", r.URL.Path)
			}
		},
	}

	client := http.Client{Transport: fakeRoundTripper}
	server := NewCliServer(authToken, proj, client)
	rules, err := server.QuerySilenceRules()

	if err != nil || len(rules) != 1 {
		t.Fatalf("failed send unitest %v, rules: %v", err, rules)
	}
	if rules[0].Library != "name" || rules[0].Version != "version" || rules[0].Manager != "npm" {
		t.Fatalf("bad rule %v", rules[0])
	}
}
