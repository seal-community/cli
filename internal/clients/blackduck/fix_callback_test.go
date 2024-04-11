package blackduck

import (
	"cli/internal/api"
	"cli/internal/ecosystem/mappings"
	"cli/internal/ecosystem/shared"
	"net/http"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func getFixMap() shared.FixMap {
	django := api.PackageVersion{
		Version:                         "3.2.17",
		Library:                         api.Package{Name: "Django", PackageManager: mappings.PythonManager},
		RecommendedLibraryVersionId:     "111",
		RecommendedLibraryVersionString: "3.2.17-sp1",
		SealedVulnerabilities: []api.Vulnerability{
			{CVE: "CVE-2023-41164"},
		},
	}

	projectDir := "/prj"
	fixmap := shared.FixMap{
		"a": &shared.FixedEntry{Package: &django, Paths: map[string]bool{
			filepath.Join(projectDir, "pythonstuff/django"): true,
		}},
	}

	return fixmap
}

func TestParseKey(t *testing.T) {
	vals := []string{"a", "b", "c", "d"}
	got := parseKey(vals)
	want := "a|b|c|d"

	if got != want {
		t.Errorf("parseKey() = %v, want %v", got, want)
	}
}

func TestBuildSealedVulnerabilitiesMapping(t *testing.T) {
	fixmap := getFixMap()

	got := buildSealedVulnerabilitiesMapping(fixmap)
	want := vulnerabilityMapping{
		"django|3.2.17|pypi|cve-2023-41164": true,
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("buildSealedVulnerabilitiesMapping() = %v, want %v", got, want)
	}
}

func TestHandleAppliedFixes(t *testing.T) {
	fixmap := getFixMap()

	fakeRoundTripper := fakeRoundTripper{
		statusCode: 200,
		jsonContent: map[string]string{
			"https://test.com/api/projects/projects-id/versions/versions-id/components/components-id/versions/versions-id/origins/origin-id/vulnerabilities/CVE-2023-41164/remediation": "{}",
		},
	}

	client := http.Client{Transport: fakeRoundTripper}
	c := BlackDuckClient{
		Client:      client,
		Url:         "https://test.com",
		Token:       "token",
		BearerToken: "bearer-token",
		ValidUntil:  time.Now().Add(time.Hour),
	}

	err := handleAppliedFixes("projectName1", &c, fixmap)
	if err != nil {
		t.Errorf("HandleAppliedFixes() = %v, want nil", err)
	}
}
