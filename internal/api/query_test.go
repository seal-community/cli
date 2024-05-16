package api

import (
	"cli/internal/common"
	"cli/internal/ecosystem/mappings"
	"io"
	"net/http"
	"sync"
	"testing"
)

func TestBulkQuerySingleChunk(t *testing.T) {
	chunksRequested := 0
	m := &sync.Mutex{}
	fakeRoundTripper := fakeRoundTripper{statusCode: 200,
		jsonContent: `{"items":[],"total":0,"limit":1,"offset":0}`,
		Validator: func(r *http.Request) {
			m.Lock()
			chunksRequested += 1
			m.Unlock()
		}}

	client := http.Client{Transport: fakeRoundTripper}
	s := Server{Client: client}
	_, err := s.FetchPackagesInfo([]common.Dependency{
		{Name: "a", Version: "1.2.3", PackageManager: "mmm"},
	},
		Metadata{},
		OnlyVulnerable,
		nil,
	)

	if err != nil {
		t.Fatalf("failed send unitest %v", err)
	}

	if chunksRequested != 1 {
		t.Fatalf("wrong number of chunks sent: %d", chunksRequested)
	}
}

func TestBulkQueryChunks(t *testing.T) {
	chunksRequested := 0
	m := &sync.Mutex{}
	fakeRoundTripper := fakeRoundTripper{statusCode: 200,
		jsonContent: `{"items":[],"total":0,"limit":1,"offset":0}`,
		Validator: func(r *http.Request) {
			m.Lock()
			chunksRequested += 1
			m.Unlock()
		}}
	client := http.Client{Transport: fakeRoundTripper}
	s := Server{Client: client, BulkChunkSize: 1}
	_, err := s.FetchPackagesInfo([]common.Dependency{
		{Name: "a", Version: "1.2.3", PackageManager: "mmm"},
		{Name: "b", Version: "1.0.0", PackageManager: "mmm"},
		{Name: "c", Version: "0.0.1", PackageManager: "mmm"},
	},
		Metadata{},
		OnlyVulnerable,
		nil,
	)

	if err != nil {
		t.Fatalf("failed send unitest %v", err)
	}

	if chunksRequested != 3 {
		t.Fatalf("wrong number of chunks sent: %d", chunksRequested)
	}
}

func TestRemoteConfigQuerySanity(t *testing.T) {

	recommendedId := "2222"

	query := RemoteOverrideQuery{
		LibraryId:            "000",
		OriginVersionId:      "111",
		RecommendedVersionId: &recommendedId,
	}

	fakeRoundTripper := fakeRoundTripper{statusCode: 200,
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

			if r.URL.Path != "/authenticated/v1/fixes/remote/proj" {
				t.Fatalf("bad url %s", r.URL.Path)
			}
		},
	}

	client := http.Client{Transport: fakeRoundTripper}
	s := Server{Client: client, AuthToken: "asd"}
	page, err := s.sendRemoteFixesQuery([]RemoteOverrideQuery{query}, "proj")

	if err != nil || page == nil {
		t.Fatalf("failed send unitest %v, page: %v", err, page)
	}

	if len(page.Items) != 1 {
		t.Fatalf("bad number of returned items %v", len(page.Items))
	}
}

func TestRemoteConfigQueryNoToken(t *testing.T) {

	recommendedId := "2222"

	query := RemoteOverrideQuery{
		LibraryId:            "000",
		OriginVersionId:      "111",
		RecommendedVersionId: &recommendedId,
	}

	fakeRoundTripper := fakeRoundTripper{statusCode: 200,
		jsonContent: `{"items":[],"total":0,"limit":1,"offset":0}`,
	}

	client := http.Client{Transport: fakeRoundTripper}
	s := Server{Client: client, AuthToken: ""}
	page, err := s.sendRemoteFixesQuery([]RemoteOverrideQuery{query}, "proj")

	if err != MissingTokenForApiRequest || page != nil {
		t.Fatalf("should fail without token %v, page: %v", err, page)
	}
}

func TestRemoteConfigQueryProjectDoesNotExist(t *testing.T) {

	recommendedId := "2222"

	query := RemoteOverrideQuery{
		LibraryId:            "000",
		OriginVersionId:      "111",
		RecommendedVersionId: &recommendedId,
	}

	fakeRoundTripper := fakeRoundTripper{statusCode: 404}

	client := http.Client{Transport: fakeRoundTripper}
	s := Server{Client: client, AuthToken: "asd"}
	page, err := s.sendRemoteFixesQuery([]RemoteOverrideQuery{query}, "non-existent")

	if err != NonExistentProjectError || page != nil {
		t.Fatalf("should fail without token %v, page: %v", err, page)
	}
}

func TestBulkQuery(t *testing.T) {
	dep := common.Dependency{Name: "ejs", Version: "2.7.4", PackageManager: mappings.NpmManager}
	fakeRoundTripper := fakeRoundTripper{statusCode: 200,
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
	s := Server{Client: client, AuthToken: ""}

	page, err := s.sendBulkRequest(&BulkCheckRequest{
		Metadata: nil,
		Entries:  []common.Dependency{dep},
	}, OnlyVulnerable)

	if err != nil || page == nil {
		t.Fatalf("should fail without token %v, page: %v", err, page)
	}
}

func TestBulkQueryAuth(t *testing.T) {
	dep := common.Dependency{Name: "ejs", Version: "2.7.4", PackageManager: mappings.NpmManager}
	proj := "myproj"
	fakeRoundTripper := fakeRoundTripper{statusCode: 200,
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
	s := Server{Client: client, AuthToken: "123123"}

	page, err := s.sendBulkRequestAuth(&BulkCheckRequest{
		Metadata: nil,
		Entries:  []common.Dependency{dep},
	}, OnlyVulnerable, proj, false)

	if err != nil || page == nil {
		t.Fatalf("should fail without token %v, page: %v", err, page)
	}
}

func TestBulkQueryAuthMissingToken(t *testing.T) {
	client := http.Client{Transport: nil}
	s := Server{Client: client, AuthToken: ""}

	_, err := s.sendBulkRequestAuth(nil, OnlyVulnerable, "proj", false)
	if err != MissingTokenForApiRequest {
		t.Fatalf("should fail without token %v", err)
	}
}

func TestBulkQueryAuthGenerateActivity(t *testing.T) {
	dep := common.Dependency{Name: "ejs", Version: "2.7.4", PackageManager: mappings.NpmManager}
	proj := "myproj"
	fakeRoundTripper := fakeRoundTripper{statusCode: 200,
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
	s := Server{Client: client, AuthToken: "123123"}

	page, err := s.sendBulkRequestAuth(&BulkCheckRequest{
		Metadata: nil,
		Entries:  []common.Dependency{dep},
	}, OnlyVulnerable, proj, true)

	if err != nil || page == nil {
		t.Fatalf("should fail without token %v, page: %v", err, page)
	}
}
