package dependabot

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

type requestValidatorCallback func(*http.Request)

var pathToJsonFile = map[string]string{
	"https://api.github.com/repos/owner-id/repo-id/dependabot/alerts":    "get_alerts.json",
	"https://api.github.com/repos/owner-id/repo-id/dependabot/alerts/28": "post_dismiss.json",
	"https://api.github.com/repos/owner-id/repo-id/dependabot/alerts/30": "post_dismiss.json",
}

type fakeRoundTripper struct {
	// data to return for request
	jsonContent map[string]string
	statusCode  map[string]int
	Validator   requestValidatorCallback
	UrlCounter  map[string]int
}

func (f *fakeRoundTripper) CheckUrlCounter(t *testing.T, expected map[string]int) {
	for url, calls := range expected {
		if f.UrlCounter[url] != calls {
			t.Errorf("expected %d calls to %s, got %d", calls, url, f.UrlCounter[url])
		}
	}

	for url, calls := range f.UrlCounter {
		if _, ok := expected[url]; !ok {
			t.Errorf("unexpected call to %s, should have been called %d times", url, calls)
		}
	}
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

func RemoveObjectsFromJSON(jsonContent string, numberOfObjectsToRemove int) (string, error) {
	var data interface{}
	if err := json.Unmarshal([]byte(jsonContent), &data); err != nil {
		return "", fmt.Errorf("error parsing JSON: %w", err)
	}

	// Check if the data is a JSON array
	arr, isArray := data.([]interface{})
	if !isArray {
		return "", errors.New("input JSON is not an array")
	}

	// Determine how many objects can actually be removed
	removeCount := min(numberOfObjectsToRemove, len(arr))

	// Update the array by removing elements
	updatedArray := arr[removeCount:]

	// Marshal the updated array back to JSON
	updatedJSON, err := json.Marshal(updatedArray)
	if err != nil {
		return "", fmt.Errorf("error generating JSON: %w", err)
	}

	return string(updatedJSON), nil
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

	if content == "" {
		// fetch file from current package's testdata folder
		fileName := pathToJsonFile[url]
		if fileName == "" {
			panic("no file found for url " + url)
		}

		content = string(getTestFile(fileName))
		pageStr := req.URL.Query().Get("page")
		page, err := strconv.Atoi(pageStr)
		if err != nil {
			page = 1
		}
		if page > 1 {
			modifiedContent, err := RemoveObjectsFromJSON(content, 10*f.UrlCounter[url])
			if err != nil {
				panic("could not remove paginated objects from content")
			}
			content = modifiedContent
		}
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

func TestUpdateVulnRequestFailed(t *testing.T) {
	fakeRoundTripper := fakeRoundTripper{
		statusCode: map[string]int{
			"https://api.github.com/": 500,
		},
		jsonContent: map[string]string{
			"https://api.github.com/": "{}",
		},
	}
	client := http.Client{Transport: &fakeRoundTripper}
	c := DependabotClient{
		Client: client,
		Url:    "https://api.github.com",
		Token:  "token",
		Owner:  "owner-id",
		Repo:   "repo-id",
	}

	update := dependabotUpdateComponentVulnerabilityRemediation{
		State:            "open",
		DismissedReason:  "fix_started",
		DismissedComment: "vulnerability patched by seal-security",
	}

	err := c.updateVuln("https://api.github.com/", &update)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestExecuteGetRequestFailed(t *testing.T) {
	fakeRoundTripper := fakeRoundTripper{
		statusCode: map[string]int{
			"https://api.github.com/": 500,
		},
		jsonContent: map[string]string{
			"https://api.github.com/": "{}",
		},
	}
	client := http.Client{Transport: &fakeRoundTripper}
	c := DependabotClient{
		Client: client,
		Url:    "https://api.github.com",
		Token:  "token",
		Owner:  "owner-id",
		Repo:   "repo-id",
	}

	_, err := c.executeGet("https://api.github.com/", nil, nil)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}
func TestGetProjectAlerts(t *testing.T) {
	fakeRoundTripper := fakeRoundTripper{}

	client := http.Client{Transport: &fakeRoundTripper}
	c := DependabotClient{
		Client: client,
		Url:    "https://api.github.com",
		Token:  "token",
		Owner:  "owner-id",
		Repo:   "repo-id",
	}

	projects, err := c.getProjectAlerts(100, 0)
	if err != nil {
		t.Fatalf("failed to get projects: %v", err)
	}

	if len(*projects) != 15 {
		t.Log(*projects)
		t.Fatalf("expected 15 projects, got %d", len(*projects))
	}

	fakeRoundTripper.CheckUrlCounter(
		t,
		map[string]int{
			"https://api.github.com/repos/owner-id/repo-id/dependabot/alerts": 1,
		},
	)
}

func TestUpdateVulnerability(t *testing.T) {
	fakeRoundTripper := fakeRoundTripper{
		statusCode: map[string]int{
			"https://api.github.com/repos/owner-id/repo-id/dependabot/alerts/28": 200,
		},
		Validator: func(req *http.Request) {
			if req.Header.Get("X-GitHub-Api-Version") != "2022-11-28" {
				t.Fatalf("expected X-GitHub-Api-Version=2022-11-28, got %s", req.Header.Get("X-GitHub-Api-Version"))
			}

			if req.Header.Get("Accept") != "application/vnd.github+json" {
				t.Fatalf("expected application/vnd.github+json, got %s", req.Header.Get("Accept"))
			}

			body, err := io.ReadAll(req.Body)
			if err != nil {
				t.Fatalf("failed to read body: %v", err)
			}

			expected := "{\"state\":\"dismissed\",\"dismissed_reason\":\"fix_started\",\"dismissed_comment\":\"vulnerability patched by seal-security\"}"
			if string(body) != expected {
				t.Fatalf("expected %s, got %s", expected, string(body))
			}

			if req.ContentLength != int64(len(expected)) {
				t.Fatalf("expected %d, got %d", len(body), req.ContentLength)
			}
		},
	}

	client := http.Client{Transport: &fakeRoundTripper}
	c := DependabotClient{
		Client: client,
		Url:    "https://api.github.com",
		Token:  "token",
		Owner:  "owner",
		Repo:   "repo",
	}

	update := dependabotUpdateComponentVulnerabilityRemediation{
		State:            "dismissed",
		DismissedReason:  "fix_started",
		DismissedComment: "vulnerability patched by seal-security",
	}

	err := c.updateVuln("https://api.github.com/repos/owner-id/repo-id/dependabot/alerts/28", &update)
	if err != nil {
		t.Fatalf("failed to update vulnerability: %v", err)
	}

	fakeRoundTripper.CheckUrlCounter(
		t,
		map[string]int{
			"https://api.github.com/repos/owner-id/repo-id/dependabot/alerts/28": 1,
		},
	)
}

func TestGetAllVunerabilitiesInProject(t *testing.T) {
	fakeRoundTripper := fakeRoundTripper{}

	client := http.Client{Transport: &fakeRoundTripper}
	c := DependabotClient{
		Client: client,
		Url:    "https://api.github.com",
		Token:  "token",
		Owner:  "owner-id",
		Repo:   "repo-id",
	}

	vulnerabilitiesChannel := make(chan dependabotVulnerableComponent, 10)
	answerChannel := make(chan int, 1)
	go func() {
		foundVulns := 0
		for {
			_, more := <-vulnerabilitiesChannel
			if !more {
				break
			}
			foundVulns++
		}

		answerChannel <- foundVulns
		close(answerChannel)
	}()

	err := c.getAllVulnsInProject(vulnerabilitiesChannel)
	if err != nil {
		t.Fatalf("failed to get all vulnerabilities in project: %v", err)
	}

	close(vulnerabilitiesChannel)

	foundVulns := <-answerChannel
	if foundVulns != 15 {
		t.Fatalf("Expected 15, got %d", foundVulns)
	}

	fakeRoundTripper.CheckUrlCounter(
		t,
		map[string]int{
			"https://api.github.com/repos/owner-id/repo-id/dependabot/alerts": 2,
		},
	)
}
