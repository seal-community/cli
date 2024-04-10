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
)

type requestValidatorCallback func(*http.Request)

type fakeRoundTripper struct {
	// data to return for request
	jsonContent map[string]string
	statusCode  int
	Validator   requestValidatorCallback
}

var pathToJsonFile = map[string]string{
	"https://test.com/api/projects/projects-id/versions":                                       "get_versions.json",
	"https://test.com/api/projects/projects-id/versions/versions-id/vulnerable-bom-components": "get_vulnerable_bom_components.json",
}

func (f fakeRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	resp := new(http.Response)
	url := fmt.Sprintf("%s://%s%s", req.URL.Scheme, req.URL.Host, req.URL.Path)

	// if in jsonContent conent=jsonContent[url]
	content := f.jsonContent[url]
	if content == "" {
		// fetch file from current package's testdata folder
		fileName := pathToJsonFile[url]
		if fileName == "" {
			panic("no file found for url " + url)
		}
		content = string(getTestFile(fileName))
	}

	resp.Body = io.NopCloser(strings.NewReader(content))
	resp.StatusCode = f.statusCode
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

func TestGetProjects(t *testing.T) {
	// the example file (get_projects.json) is using v4 API while the code is using v6 API
	// When i will get a v6 API response, i will update the test file and the test
	resData := bdProjects{
		Items: []bdProject{
			{
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
						},
					},
				},
			},
		},
		TotalCount: 1,
	}
	jsonContent, err := json.Marshal(resData)
	if err != nil {
		t.Fatalf("failed to marshal json: %v", err)
	}
	fakeRoundTripper := fakeRoundTripper{
		statusCode: 200,
		jsonContent: map[string]string{
			"https://test.com/api/projects": string(jsonContent),
		},
	}
	client := http.Client{Transport: fakeRoundTripper}
	c := BlackDuckClient{
		Client: client,
		Url:    "https://test.com",
		Token:  "token",
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
}

func TestGetProjectByName(t *testing.T) {
	// the example file (get_projects.json) is using v4 API while the code is using v6 API
	// When i will get a v6 API response, i will update the test file and the test
	resData := bdProjects{
		Items: []bdProject{
			{
				Name:                     "project1",
				Description:              "description1",
				ProjectTier:              1,
				ProjectLevelAdjustments:  true,
				SnippetAdjustmentApplied: true,
				CreatedAt:                "now1",
				CreatedBy:                "user1",
				UpdatedAt:                "now1",
				UpdatedBy:                "user1",
				CreatedByUser:            "user1",
				UpdatedByUser:            "user1",
				Meta: bdMeta{
					Href: "href1",
					Links: []bdLink{
						{
							Rel:  "rel1",
							Href: "href1",
						}},
				},
			},
			{
				Name:                     "project2",
				Description:              "description2",
				ProjectTier:              2,
				ProjectLevelAdjustments:  true,
				SnippetAdjustmentApplied: true,
				CreatedAt:                "now2",
				CreatedBy:                "user2",
				UpdatedAt:                "now2",
				UpdatedBy:                "user2",
				CreatedByUser:            "user2",
				UpdatedByUser:            "user2",
				Meta: bdMeta{
					Href: "https://test.com/api/projects/projects-id",
					Links: []bdLink{
						{
							Rel:  "rel",
							Href: "https://test.com/api/projects/projects-id",
						},
					},
				},
			},
		},
		TotalCount: 2,
	}
	jsonContent, err := json.Marshal(resData)
	if err != nil {
		t.Fatalf("failed to marshal json: %v", err)
	}
	fakeRoundTripper := fakeRoundTripper{
		statusCode: 200,
		jsonContent: map[string]string{
			"https://test.com/api/projects": string(jsonContent),
		},
		Validator: func(req *http.Request) {
			if req.URL.Query().Get("q") != "name:project2" {
				t.Fatalf("expected name:project2, got %s", req.URL.Query().Get("q"))
			}
		},
	}
	client := http.Client{Transport: fakeRoundTripper}
	c := BlackDuckClient{
		Client: client,
		Url:    "https://test.com",
		Token:  "token",
	}

	project, err := c.getProjectByName("project2")
	if err != nil {
		t.Fatalf("failed to get project: %v", err)
	}
	if project == nil {
		t.Fatalf("expected project, got nil")
	}
	if project.Name != "project2" {
		t.Fatalf("expected project2, got %s", project.Name)
	}
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

	fakeRoundTripper := fakeRoundTripper{
		statusCode:  200,
		jsonContent: map[string]string{},
	}
	client := http.Client{Transport: fakeRoundTripper}
	c := BlackDuckClient{
		Client: client,
		Url:    "https://test.com",
		Token:  "token",
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

	c := NewClient("https://test.com", "token")
	link := c.getLink(version, "rel")
	if link != "https://test.com/api/projects/projects-id" {
		t.Fatalf("expected href, got %s", link)
	}
}

func TestGetVulnerableComponents(t *testing.T) {
	fakeRoundTripper := fakeRoundTripper{
		statusCode:  200,
		jsonContent: map[string]string{},
		Validator: func(req *http.Request) {
			if req.Header.Get("Accept") != "application/vnd.blackducksoftware.bill-of-materials-6+json" {
				t.Fatalf("expected application/vnd.blackducksoftware.bill-of-materials-6+json, got %s", req.Header.Get("Accept"))
			}
			if req.Header.Get("Content-Type") != "application/vnd.blackducksoftware.bill-of-materials-6+json" {
				t.Fatalf("expected application/vnd.blackducksoftware.bill-of-materials-6+json, got %s", req.Header.Get("Content-Type"))
			}
		},
	}
	client := http.Client{Transport: fakeRoundTripper}
	c := BlackDuckClient{
		Client: client,
		Url:    "https://test.com",
		Token:  "token",
	}

	vulnerableComponents, err := c.getVulnerableComponents("https://test.com/api/projects/projects-id/versions/versions-id/vulnerable-bom-components", 10, 0)
	if err != nil {
		t.Fatalf("failed to get vulnerable components: %v", err)
	}
	if len(vulnerableComponents.Items) != 3 {
		t.Fatalf("expected 3 vulnerable component, got %d", len(vulnerableComponents.Items))
	}
	if vulnerableComponents.TotalCount != 3 {
		t.Fatalf("expected 3 vulnerable component, got %d", vulnerableComponents.TotalCount)
	}
}

func TestUpdateVulnerability(t *testing.T) {
	fakeRoundTripper := fakeRoundTripper{
		statusCode:  200,
		jsonContent: map[string]string{},
		Validator: func(req *http.Request) {
			if req.Header.Get("Content-Type") != "application/json" {
				t.Fatalf("expected application/json, got %s", req.Header.Get("Content-Type"))
			}
			if req.Header.Get("Accept") != "application/vnd.blackducksoftware.bill-of-materials-6+json" {
				t.Fatalf("expected application/vnd.blackducksoftware.bill-of-materials-6+json, got %s", req.Header.Get("Accept"))
			}
		},
	}
	client := http.Client{Transport: fakeRoundTripper}
	c := BlackDuckClient{
		Client: client,
		Url:    "https://test.com",
		Token:  "token",
	}

	update := bdUpdateBOMComponentVulnerabilityRemediation{
		RemediationStatus: "NEW",
		Comment:           "I AM SEAL",
	}
	err := c.updateVuln("https://test.com/api/projects/projects-id/versions/versions-id/vulnerable-bom-components", update)
	if err != nil {
		t.Fatalf("failed to update vulnerability: %v", err)
	}
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

	fakeRoundTripper := fakeRoundTripper{
		statusCode:  200,
		jsonContent: map[string]string{},
	}
	client := http.Client{Transport: fakeRoundTripper}
	c := BlackDuckClient{
		Client: client,
		Url:    "https://test.com",
		Token:  "token",
	}

	vulnerabilitiesChannel := make(chan bdVulnerableBOMComponent, 10)
	err := c.getAllVulnsInProject(&project, vulnerabilitiesChannel)
	if err != nil {
		t.Fatalf("failed to get all vulnerabilities in project: %v", err)
	}

	close(vulnerabilitiesChannel)

	if len(vulnerabilitiesChannel) != 9 {
		t.Fatalf("expected 3 vulnerabilities, got %d", len(vulnerabilitiesChannel))
	}

	counter := map[string]int{
		"Django/3.2.17":  3,
		"grpcio/1.52.0":  6,
		"CVE-2023-41164": 3,
		"CVE-2023-1428":  3,
		"CVE-2023-32731": 3,
	}

	for i := 0; i < 9; i++ {
		v := <-vulnerabilitiesChannel
		counter[v.ComponentVersionOriginId]--
		counter[v.VulnerabilityWithRemediation.VulnerabilityName]--
	}
	for k, v := range counter {
		if v != 0 {
			t.Fatalf("expected 0 %s, got %d", k, v)
		}
	}
}
