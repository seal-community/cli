package blackduck

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type requestValidatorCallback func(*http.Request)

type fakeRoundTripper struct {
	// data to return for request
	jsonContent map[string]string
	statusCode  map[string]int
	Validator   requestValidatorCallback
	UrlCounter  map[string]int
}

var pathToJsonFile = map[string]string{
	"https://test.com/api/projects":                                                            "get_projects.json",
	"https://test.com/api/projects/projects-id/versions":                                       "get_versions.json",
	"https://test.com/api/projects/projects-id/versions/versions-id/vulnerable-bom-components": "get_vulnerable_bom_components.json",
}

func (f *fakeRoundTripper) CheckUrlCounter(t *testing.T, expected map[string]int) {
	for url := range expected {
		f.UrlCounter[url]--
	}

	for url, calls := range f.UrlCounter {
		if calls != 0 {
			moreOrLess := "less"
			if calls > 0 {
				moreOrLess = "more"
			}
			t.Errorf("got %d to %s calls to %s", calls, moreOrLess, url)
		}
	}
}

func (f *fakeRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	resp := new(http.Response)
	url := fmt.Sprintf("%s://%s%s", req.URL.Scheme, req.URL.Host, req.URL.Path)

	if f.UrlCounter == nil {
		f.UrlCounter = make(map[string]int)
		f.UrlCounter[url] = 0
	}
	f.UrlCounter[url]++

	content := ""
	if f.jsonContent != nil {
		content = f.jsonContent[url]
	}

	if content == "" && url == "https://test.com/api/tokens/authenticate" {
		content = "{}"
	}

	if content == "" {
		// fetch file from current package's testdata folder
		fileName := pathToJsonFile[url]
		if fileName == "" {
			panic("no file found for url " + url)
		}
		content = string(getTestFile(fileName))
	}

	statusCode := f.statusCode[url]
	if statusCode == 0 {
		statusCode = 200 // default status code
	}

	resp.Body = io.NopCloser(strings.NewReader(content))
	resp.StatusCode = statusCode
	resp.Request = req

	if f.Validator != nil {
		f.Validator(req)
	}

	return resp, nil
}

func getTestFile(name string) []byte {
	// fetch file from current package's testdata folder
	// ref: https://pkg.go.dev/cmd/go/internal/test
	p := filepath.Join("testdata", name)
	data, err := os.ReadFile(p)
	if err != nil {
		panic(err)
	}

	return data
}

func TestAuthenticate(t *testing.T) {
	authRes := bdAPITokenResponse{
		BearerToken:           "my-token",
		ExpiresInMilliseconds: 10000,
	}

	jsonContent, err := json.Marshal(authRes)
	if err != nil {
		t.Fatalf("failed to marshal json: %v", err)
	}

	fakeRoundTripper := fakeRoundTripper{
		jsonContent: map[string]string{
			"https://test.com/api/tokens/authenticate": string(jsonContent),
		},
	}
	client := http.Client{Transport: &fakeRoundTripper}
	c := BlackDuckClient{
		Client: client,
		Url:    "https://test.com",
		Token:  "token",
	}

	bearer, err := c.getBearerAuth()
	if err != nil {
		t.Fatalf("failed to get bearer token: %v", err)
	}

	if bearer != "my-token" {
		t.Fatalf("expected my-token, got %s", bearer)
	}

	if c.ValidUntil.IsZero() {
		t.Fatalf("expected non-zero time, got zero")
	}

	if c.BearerToken != "my-token" {
		t.Fatalf("expected my-token, got %s", c.BearerToken)
	}

	// check if the token is cached
	c.BearerToken = "new-token"
	bearer, err = c.getBearerAuth()
	if err != nil {
		t.Fatalf("failed to get bearer token: %v", err)
	}

	if bearer != "my-token" {
		t.Fatalf("expected my-token, got %s", bearer)
	}
}

func TestAuthenticateFailed(t *testing.T) {
	fakeRoundTripper := fakeRoundTripper{
		statusCode: map[string]int{
			"https://test.com/api/tokens/authenticate": 500,
		},
	}
	client := http.Client{Transport: &fakeRoundTripper}
	c := BlackDuckClient{
		Client: client,
		Url:    "https://test.com",
		Token:  "token",
	}

	_, err := c.getBearerAuth()
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestExecutePutRequestFailed(t *testing.T) {
	fakeRoundTripper := fakeRoundTripper{
		statusCode: map[string]int{
			"https://test.com/": 500,
		},
		jsonContent: map[string]string{
			"https://test.com/": "{}",
		},
	}
	client := http.Client{Transport: &fakeRoundTripper}
	c := BlackDuckClient{
		Client:      client,
		Url:         "https://test.com",
		Token:       "token",
		BearerToken: "bearer-token",
		ValidUntil:  time.Now().Add(time.Hour),
	}

	_, err := c.executePut("https://test.com/", nil, nil, nil)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestExecuteGetRequestFailed(t *testing.T) {
	fakeRoundTripper := fakeRoundTripper{
		statusCode: map[string]int{
			"https://test.com/": 500,
		},
		jsonContent: map[string]string{
			"https://test.com/": "{}",
		},
	}
	client := http.Client{Transport: &fakeRoundTripper}
	c := BlackDuckClient{
		Client:      client,
		Url:         "https://test.com",
		Token:       "token",
		BearerToken: "bearer-token",
		ValidUntil:  time.Now().Add(time.Hour),
	}

	_, err := c.executeGet("https://test.com/", nil, nil)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestGetProjects(t *testing.T) {
	// the example file (get_projects.json) is using v4 API while the code is using v6 API
	// When i will get a v6 API response, i will update the test file and the test
	fakeRoundTripper := fakeRoundTripper{}

	client := http.Client{Transport: &fakeRoundTripper}
	c := BlackDuckClient{
		Client:      client,
		Url:         "https://test.com",
		Token:       "token",
		BearerToken: "bearer-token",
		ValidUntil:  time.Now().Add(time.Hour),
	}

	projects, err := c.getProjects(nil)
	if err != nil {
		t.Fatalf("failed to get projects: %v", err)
	}

	if len(projects.Items) != 1 {
		t.Fatalf("expected 0 projects, got %d", len(projects.Items))
	}

	if projects.TotalCount != 1 {
		t.Fatalf("expected 0 projects, got %d", projects.TotalCount)
	}

	fakeRoundTripper.CheckUrlCounter(
		t,
		map[string]int{
			"https://test.com/api/projects": 1,
		},
	)
}

func TestGetProjectByName(t *testing.T) {
	// the example file (get_projects.json) is using v4 API while the code is using v6 API
	// When i will get a v6 API response, i will update the test file and the test
	fakeRoundTripper := fakeRoundTripper{
		Validator: func(req *http.Request) {
			if req.URL.Query().Get("q") != "name:projectName1" {
				t.Fatalf("expected name:projectName1, got %s", req.URL.Query().Get("q"))
			}
		},
	}

	client := http.Client{Transport: &fakeRoundTripper}
	c := BlackDuckClient{
		Client:      client,
		Url:         "https://test.com",
		Token:       "token",
		BearerToken: "bearer-token",
		ValidUntil:  time.Now().Add(time.Hour),
	}

	project, err := c.getProjectByName("projectName1")
	if err != nil {
		t.Fatalf("failed to get project: %v", err)
	}

	if project == nil {
		t.Fatalf("expected project, got nil")
	}

	if project.Name != "projectName1" {
		t.Fatalf("expected projectName1, got %s", project.Name)
	}

	fakeRoundTripper.CheckUrlCounter(
		t,
		map[string]int{
			"https://test.com/api/projects": 1,
		},
	)
}

func TestGetProjectVersions(t *testing.T) {
	project := bdProject{
		Name:                     "project",
		Description:              "description",
		ProjectTier:              1,
		ProjectLevelAdjustments:  true,
		SnippetAdjustmentApplied: true,
		CreatedAt:                "now",
		CreatedBy:                "user",
		UpdatedAt:                "now",
		UpdatedBy:                "user",
		CreatedByUser:            "user",
		UpdatedByUser:            "user",
		Meta: bdMeta{
			Href: "https://test.com/api/projects/projects-id",
			Links: []bdLink{
				{
					Rel:  "rel",
					Href: "https://test.com/api/projects/projects-id",
				}},
		},
	}

	fakeRoundTripper := fakeRoundTripper{}

	client := http.Client{Transport: &fakeRoundTripper}
	c := BlackDuckClient{
		Client:      client,
		Url:         "https://test.com",
		Token:       "token",
		BearerToken: "bearer-token",
		ValidUntil:  time.Now().Add(time.Hour),
	}

	versions, err := c.getProjectVersions(&project, 10, 0)
	if err != nil {
		t.Fatalf("failed to get project versions: %v", err)
	}

	if len(versions.Items) != 3 {
		t.Fatalf("expected 2 versions, got %d", len(versions.Items))
	}

	if versions.TotalCount != 3 {
		t.Fatalf("expected 2 versions, got %d", versions.TotalCount)
	}

	fakeRoundTripper.CheckUrlCounter(
		t,
		map[string]int{
			"https://test.com/api/projects/projects-id/versions": 1,
		},
	)
}

func TestGetLink(t *testing.T) {
	version := bdVersion{
		VersionName:  "version",
		Phase:        "phase",
		Distribution: "distribution",
		Source:       "source",
		Meta: bdMeta{
			Href: "https://test.com/api/projects/projects-id",
			Links: []bdLink{
				{
					Rel:  "rel",
					Href: "https://test.com/api/projects/projects-id",
				}},
		},
	}

	fakeRoundTripper := fakeRoundTripper{}

	client := http.Client{Transport: &fakeRoundTripper}
	c := BlackDuckClient{
		Client: client,
		Url:    "https://test.com",
		Token:  "token",
	}

	link := c.getLink(version, "rel")
	if link != "https://test.com/api/projects/projects-id" {
		t.Fatalf("expected href, got %s", link)
	}
}

func TestGetVulnerableComponents(t *testing.T) {
	fakeRoundTripper := fakeRoundTripper{
		Validator: func(req *http.Request) {
			if req.Header.Get("Accept") != "application/vnd.blackducksoftware.bill-of-materials-6+json" {
				t.Fatalf("expected application/vnd.blackducksoftware.bill-of-materials-6+json, got %s", req.Header.Get("Accept"))
			}
			if req.Header.Get("Content-Type") != "application/vnd.blackducksoftware.bill-of-materials-6+json" {
				t.Fatalf("expected application/vnd.blackducksoftware.bill-of-materials-6+json, got %s", req.Header.Get("Content-Type"))
			}
		},
	}
	client := http.Client{Transport: &fakeRoundTripper}
	c := BlackDuckClient{
		Client:      client,
		Url:         "https://test.com",
		Token:       "token",
		BearerToken: "bearer-token",
		ValidUntil:  time.Now().Add(time.Hour),
	}

	vulnerableComponents, err := c.getVulnerableComponents("https://test.com/api/projects/projects-id/versions/versions-id/vulnerable-bom-components", 10, 0)
	if err != nil {
		t.Fatalf("failed to get vulnerable components: %v", err)
	}

	if len(vulnerableComponents.Items) != 4 {
		t.Fatalf("expected 4 vulnerable component, got %d", len(vulnerableComponents.Items))
	}

	if vulnerableComponents.TotalCount != 4 {
		t.Fatalf("expected 4 vulnerable component, got %d", vulnerableComponents.TotalCount)
	}

	fakeRoundTripper.CheckUrlCounter(
		t,
		map[string]int{
			"https://test.com/api/projects/projects-id/versions/versions-id/vulnerable-bom-components": 1,
		},
	)
}

func TestUpdateVulnerability(t *testing.T) {
	fakeRoundTripper := fakeRoundTripper{
		statusCode: map[string]int{
			"https://test.com/api/projects/projects-id/versions/versions-id/vulnerable-bom-components": 202,
		},
		Validator: func(req *http.Request) {
			if req.Header.Get("Content-Type") != "application/json" {
				t.Fatalf("expected application/json, got %s", req.Header.Get("Content-Type"))
			}
			if req.Header.Get("Accept") != "application/vnd.blackducksoftware.bill-of-materials-6+json" {
				t.Fatalf("expected application/vnd.blackducksoftware.bill-of-materials-6+json, got %s", req.Header.Get("Accept"))
			}
		},
	}
	client := http.Client{Transport: &fakeRoundTripper}
	c := BlackDuckClient{
		Client:      client,
		Url:         "https://test.com",
		Token:       "token",
		BearerToken: "bearer-token",
		ValidUntil:  time.Now().Add(time.Hour),
	}

	update := bdUpdateBOMComponentVulnerabilityRemediation{
		RemediationStatus: "NEW",
		Comment:           "I AM SEAL",
	}

	err := c.updateVuln("https://test.com/api/projects/projects-id/versions/versions-id/vulnerable-bom-components", update)
	if err != nil {
		t.Fatalf("failed to update vulnerability: %v", err)
	}

	fakeRoundTripper.CheckUrlCounter(
		t,
		map[string]int{
			"https://test.com/api/projects/projects-id/versions/versions-id/vulnerable-bom-components": 1,
		},
	)
}

func TestGetAllVunerabilitiesInProject(t *testing.T) {
	project := bdProject{
		Name: "project",
		Meta: bdMeta{
			Href: "https://test.com/api/projects/projects-id",
			Links: []bdLink{
				{
					Rel:  "rel",
					Href: "https://test.com/api/projects/projects-id",
				}},
		},
	}

	fakeRoundTripper := fakeRoundTripper{}
	client := http.Client{Transport: &fakeRoundTripper}
	c := BlackDuckClient{
		Client:          client,
		Url:             "https://test.com",
		Token:           "token",
		BearerToken:     "bearer-token",
		ValidUntil:      time.Now().Add(time.Hour),
		VersionToFilter: "versionName1",
	}

	vulnerabilitiesChannel := make(chan bdVulnerableBOMComponent, 10)
	err := c.getAllVulnsInProject(&project, vulnerabilitiesChannel)
	if err != nil {
		t.Fatalf("failed to get all vulnerabilities in project: %v", err)
	}

	close(vulnerabilitiesChannel)

	if len(vulnerabilitiesChannel) != 4 {
		t.Fatalf("expected 4 vulnerabilities, got %d", len(vulnerabilitiesChannel))
	}

	counter := map[string]int{
		"Django/3.2.17":   1,
		"grpcio/1.52.0":   2,
		"requests/2.26.0": 1,
		"CVE-2023-41164":  1,
		"CVE-2023-1428":   1,
		"CVE-2023-32731":  1,
		"CVE-2023-32681":  1,
	}

	for i := 0; i < 4; i++ {
		v := <-vulnerabilitiesChannel
		counter[v.ComponentVersionOriginId]--
		counter[v.VulnerabilityWithRemediation.VulnerabilityName]--
	}

	for k, v := range counter {
		if v != 0 {
			t.Fatalf("expected 0 %s, got %d", k, v)
		}
	}

	fakeRoundTripper.CheckUrlCounter(
		t,
		map[string]int{
			"https://test.com/api/projects/projects-id/versions":                                       1,
			"https://test.com/api/projects/projects-id/versions/versions-id/vulnerable-bom-components": 1,
		},
	)
}
