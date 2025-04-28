package ox

import (
	"cli/internal/config"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

type requestValidatorCallback func(*http.Request)

type fakeRoundTripper struct {
	statusCode  map[string]int
	urlCounter  map[string]int
	jsonContent map[string]string
	Validator   requestValidatorCallback
}

var pathToJsonFile = map[string]string{
	"https://test.ox.security": "get_issues.json",
}

func getTestFile(name string) []byte {
	p := filepath.Join("testdata", name)
	data, err := os.ReadFile(p)
	if err != nil {
		panic(err)
	}
	return data
}

func (f *fakeRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	url := req.URL.String()

	if f.urlCounter == nil {
		f.urlCounter = make(map[string]int)
	}
	f.urlCounter[url]++

	statusCode := f.statusCode[url]
	if statusCode == 0 {
		statusCode = 200
	}

	fileName := pathToJsonFile[url]
	if fileName == "" {
		panic("no file found for url " + url)
	}
	body := string(getTestFile(fileName))

	resp := &http.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
	resp.Header.Set("Content-Type", "application/json")

	if f.Validator != nil {
		f.Validator(req)
	}

	return resp, nil
}

func (f *fakeRoundTripper) CheckUrlCounter(t *testing.T, expected map[string]int) {
	if !reflect.DeepEqual(f.urlCounter, expected) {
		t.Errorf("URL counter mismatch:\ngot: %v\nwant: %v", f.urlCounter, expected)
	}
}

func TestNewClient(t *testing.T) {
	tests := []struct {
		name     string
		config   config.OxConfig
		expected *OxClient
	}{
		{
			name: "valid config",
			config: config.OxConfig{
				Url:                          "https://test.ox.security",
				Token:                        config.SensitiveString("test-token"),
				Application:                  "test-app",
				ExcludeWhenHighCriticalFixed: true,
			},
			expected: &OxClient{
				Url:                          "https://test.ox.security",
				Token:                        "test-token",
				Application:                  "test-app",
				ExcludeWhenHighCriticalFixed: true,
			},
		},
		{
			name: "empty config",
			config: config.OxConfig{
				Url:                          "",
				Token:                        config.SensitiveString(""),
				Application:                  "",
				ExcludeWhenHighCriticalFixed: false,
			},
			expected: &OxClient{
				Url:                          "",
				Token:                        "",
				Application:                  "",
				ExcludeWhenHighCriticalFixed: false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewClient(tt.config)

			if got.Url != tt.expected.Url {
				t.Errorf("Url = %v, want %v", got.Url, tt.expected.Url)
			}
			if got.Token != tt.expected.Token {
				t.Errorf("Token = %v, want %v", got.Token, tt.expected.Token)
			}
			if got.Application != tt.expected.Application {
				t.Errorf("Application = %v, want %v", got.Application, tt.expected.Application)
			}
			if got.ExcludeWhenHighCriticalFixed != tt.expected.ExcludeWhenHighCriticalFixed {
				t.Errorf("ExcludeWhenHighCriticalFixed = %v, want %v",
					got.ExcludeWhenHighCriticalFixed, tt.expected.ExcludeWhenHighCriticalFixed)
			}
		})
	}
}

func TestGetIssues(t *testing.T) {
	fakeRoundTripper := fakeRoundTripper{
		Validator: func(req *http.Request) {
			if req.Header.Get("Authorization") != "test-token" {
				t.Errorf("expected Authorization header 'test-token', got %s", req.Header.Get("Authorization"))
			}
			if req.Header.Get("Content-Type") != "application/json" {
				t.Errorf("expected Content-Type header 'application/json', got %s", req.Header.Get("Content-Type"))
			}
		},
	}

	client := http.Client{Transport: &fakeRoundTripper}
	c := OxClient{
		Client:      client,
		Url:         "https://test.ox.security",
		Token:       "test-token",
		Application: "test-app",
	}

	input := GetIssuesInput{
		Offset: 0,
		Limit:  10,
		ConditionalFilters: []GetIssuesFilter{
			{
				Condition: "AND",
				FieldName: "uniqueLibs",
				Values:    []string{"axios@1.6.0"},
			},
		},
	}

	resp, err := c.GetIssues(input)
	if err != nil {
		t.Fatalf("GetIssues failed: %v", err)
	}

	if len(resp.Data.GetIssues.Issues) != 1 {
		t.Errorf("expected 1 issue, got %d", len(resp.Data.GetIssues.Issues))
	}

	issue := resp.Data.GetIssues.Issues[0]
	if issue.IssueID != "issue-1" {
		t.Errorf("expected issue ID 'issue-1', got %s", issue.IssueID)
	}

	fakeRoundTripper.CheckUrlCounter(t, map[string]int{
		"https://test.ox.security": 1,
	})
}

func TestGetIssuesFailed(t *testing.T) {
	fakeRoundTripper := fakeRoundTripper{
		statusCode: map[string]int{
			"https://test.ox.security": 500,
		},
		Validator: func(req *http.Request) {
			if req.Header.Get("Authorization") != "test-token" {
				t.Errorf("expected Authorization header 'test-token', got %s", req.Header.Get("Authorization"))
			}
			if req.Header.Get("Content-Type") != "application/json" {
				t.Errorf("expected Content-Type header 'application/json', got %s", req.Header.Get("Content-Type"))
			}
		},
	}

	client := http.Client{Transport: &fakeRoundTripper}
	c := OxClient{
		Client:      client,
		Url:         "https://test.ox.security",
		Token:       "test-token",
		Application: "test-app",
	}

	input := GetIssuesInput{
		Offset: 0,
		Limit:  10,
		ConditionalFilters: []GetIssuesFilter{
			{
				Condition: "AND",
				FieldName: "uniqueLibs",
				Values:    []string{"axios@1.6.0"},
			},
		},
	}

	_, err := c.GetIssues(input)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	fakeRoundTripper.CheckUrlCounter(t, map[string]int{
		"https://test.ox.security": 1,
	})
}

func TestExcludeIssues(t *testing.T) {
	fakeRoundTripper := fakeRoundTripper{
		statusCode: map[string]int{
			"https://test.ox.security": 200,
		},
		jsonContent: map[string]string{
			"https://test.ox.security": `{"data":{"excludeBulkAlerts":[{"totalExclusions":2}]}}`,
		},
		Validator: func(req *http.Request) {
			if req.Header.Get("Authorization") != "test-token" {
				t.Errorf("expected Authorization header 'test-token', got %s", req.Header.Get("Authorization"))
			}
			if req.Header.Get("Content-Type") != "application/json" {
				t.Errorf("expected Content-Type header 'application/json', got %s", req.Header.Get("Content-Type"))
			}
		},
	}

	client := http.Client{Transport: &fakeRoundTripper}
	c := OxClient{
		Client:      client,
		Url:         "https://test.ox.security",
		Token:       "test-token",
		Application: "test-app",
	}

	issues := []ExcludedIssue{
		{
			Issue: Issue{
				ID:      "1",
				IssueID: "issue-1",
			},
			Reason: "Test exclusion comment",
		},
		{
			Issue: Issue{
				ID:      "2",
				IssueID: "issue-2",
			},
			Reason: "Test exclusion comment",
		},
	}

	err := c.ExcludeIssues(issues)
	if err != nil {
		t.Fatalf("ExcludeIssues failed: %v", err)
	}

	fakeRoundTripper.CheckUrlCounter(t, map[string]int{
		"https://test.ox.security": 1,
	})
}

func TestExcludeIssuesFailed(t *testing.T) {
	fakeRoundTripper := fakeRoundTripper{
		statusCode: map[string]int{
			"https://test.ox.security": 500,
		},
		Validator: func(req *http.Request) {
			if req.Header.Get("Authorization") != "test-token" {
				t.Errorf("expected Authorization header 'test-token', got %s", req.Header.Get("Authorization"))
			}
			if req.Header.Get("Content-Type") != "application/json" {
				t.Errorf("expected Content-Type header 'application/json', got %s", req.Header.Get("Content-Type"))
			}
		},
	}

	client := http.Client{Transport: &fakeRoundTripper}
	c := OxClient{
		Client:      client,
		Url:         "https://test.ox.security",
		Token:       "test-token",
		Application: "test-app",
	}

	issues := []ExcludedIssue{
		{
			Issue: Issue{
				ID:      "1",
				IssueID: "issue-1",
			},
			Reason: "Test exclusion comment",
		},
		{
			Issue: Issue{
				ID:      "2",
				IssueID: "issue-2",
			},
			Reason: "Test exclusion comment",
		},
	}

	err := c.ExcludeIssues(issues)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	fakeRoundTripper.CheckUrlCounter(t, map[string]int{
		"https://test.ox.security": 1,
	})
}

func TestAddCommentToIssue(t *testing.T) {
	fakeRoundTripper := fakeRoundTripper{
		Validator: func(req *http.Request) {
			if req.Header.Get("Authorization") != "test-token" {
				t.Errorf("expected Authorization header 'test-token', got %s", req.Header.Get("Authorization"))
			}
			if req.Header.Get("Content-Type") != "application/json" {
				t.Errorf("expected Content-Type header 'application/json', got %s", req.Header.Get("Content-Type"))
			}
		},
	}

	client := http.Client{Transport: &fakeRoundTripper}
	c := OxClient{
		Client:      client,
		Url:         "https://test.ox.security",
		Token:       "test-token",
		Application: "test-app",
	}

	issue := Issue{
		ID:        "1",
		IssueID:   "issue-1",
		MainTitle: "Test Issue",
		Severity:  "HIGH",
		App: struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		}{
			ID:   "app-1",
			Name: "test-app",
		},
		Category: struct {
			Name       string `json:"name"`
			CategoryID int    `json:"categoryId"`
		}{
			Name:       "Security",
			CategoryID: 1,
		},
		ScaVulnerabilities: []ScaVulnerability{
			{
				Cve:        "CVE-2023-1234",
				OxSeverity: "HIGH",
				LibName:    "axios",
				LibVersion: "1.6.0",
			},
		},
	}
	comment := "Test comment"

	err := c.AddCommentToIssue(issue, comment)
	if err != nil {
		t.Fatalf("AddCommentToIssue failed: %v", err)
	}

	fakeRoundTripper.CheckUrlCounter(t, map[string]int{
		"https://test.ox.security": 1,
	})
}

func TestAddCommentToIssueFailed(t *testing.T) {
	fakeRoundTripper := fakeRoundTripper{
		statusCode: map[string]int{
			"https://test.ox.security": 500,
		},
		Validator: func(req *http.Request) {
			if req.Header.Get("Authorization") != "test-token" {
				t.Errorf("expected Authorization header 'test-token', got %s", req.Header.Get("Authorization"))
			}
			if req.Header.Get("Content-Type") != "application/json" {
				t.Errorf("expected Content-Type header 'application/json', got %s", req.Header.Get("Content-Type"))
			}
		},
	}

	client := http.Client{Transport: &fakeRoundTripper}
	c := OxClient{
		Client:      client,
		Url:         "https://test.ox.security",
		Token:       "test-token",
		Application: "test-app",
	}

	issue := Issue{
		ID:        "1",
		IssueID:   "issue-1",
		MainTitle: "Test Issue",
		Severity:  "HIGH",
		App: struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		}{
			ID:   "app-1",
			Name: "test-app",
		},
		Category: struct {
			Name       string `json:"name"`
			CategoryID int    `json:"categoryId"`
		}{
			Name:       "Security",
			CategoryID: 1,
		},
		ScaVulnerabilities: []ScaVulnerability{
			{
				Cve:        "CVE-2023-1234",
				OxSeverity: "HIGH",
				LibName:    "axios",
				LibVersion: "1.6.0",
			},
		},
	}
	comment := "Test comment"

	err := c.AddCommentToIssue(issue, comment)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	fakeRoundTripper.CheckUrlCounter(t, map[string]int{
		"https://test.ox.security": 1,
	})
}
