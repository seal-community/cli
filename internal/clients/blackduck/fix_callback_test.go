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

func TestParseKey(t *testing.T) {
	vals := []string{"a", "b", "c", "d"}
	got := parseKey(vals)
	want := "a/b/c/d"

	if got != want {
		t.Errorf("parseKey() = %v, want %v", got, want)
	}
}

func TestBuildSealedVulnerabilitiesMapping(t *testing.T) {
	django := api.PackageVersion{
		Version:                         "3.2.17",
		Library:                         api.Package{Name: "Django", PackageManager: mappings.PythonManager},
		RecommendedLibraryVersionId:     "111",
		RecommendedLibraryVersionString: "3.2.17-sp1",
		SealedVulnerabilities: []api.Vulnerability{
			{CVE: "CVE-2023-41164"},
		},
	}

	fixmap := shared.FixMap{
		"a": &shared.FixedEntry{Package: &django, Paths: map[string]bool{
			filepath.Join("/prj", "pythonstuff/django"): true,
		}},
	}

	got := buildSealedVulnerabilitiesMapping(fixmap)
	want := vulnerabilityMapping{
		"django/3.2.17/pypi/cve-2023-41164": true,
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("buildSealedVulnerabilitiesMapping() = %v, want %v", got, want)
	}
}

func TestHandleAppliedFixes(t *testing.T) {
	django := api.PackageVersion{
		Version:                         "3.2.17",
		Library:                         api.Package{Name: "Django", PackageManager: mappings.PythonManager},
		RecommendedLibraryVersionId:     "111",
		RecommendedLibraryVersionString: "3.2.17-sp1",
		SealedVulnerabilities: []api.Vulnerability{
			{CVE: "CVE-2023-41164"},
		},
	}

	fixmap := shared.FixMap{
		"a": &shared.FixedEntry{Package: &django, Paths: map[string]bool{
			filepath.Join("/prj", "pythonstuff/django"): true,
		}},
	}

	fakeRoundTripper := fakeRoundTripper{
		statusCode: 200,
		jsonContent: map[string]string{
			"https://test.com/api/projects/projects-id/versions/versions-id/components/components-id/versions/versions-id/origins/origins-id/vulnerabilities/CVE-2023-32731/remediation": "{}", // unsealed
			"https://test.com/api/projects/projects-id/versions/versions-id/components/components-id/versions/versions-id/origins/origins-id/vulnerabilities/CVE-2023-41164/remediation": "{}", // sealed
		},
	}

	client := http.Client{Transport: &fakeRoundTripper}
	c := BlackDuckClient{
		Client:          client,
		Url:             "https://test.com",
		Token:           "token",
		BearerToken:     "bearer-token",
		ValidUntil:      time.Now().Add(time.Hour),
		VersionToFilter: "versionName1",
	}

	err := handleAppliedFixes("projectName1", &c, fixmap)
	if err != nil {
		t.Errorf("HandleAppliedFixes() = %v, want nil", err)
	}

	fakeRoundTripper.CheckUrlCounter(
		t,
		map[string]int{
			"https://test.com/api/projects":                                                            1,
			"https://test.com/api/projects/projects-id/versions":                                       1,
			"https://test.com/api/projects/projects-id/versions/versions-id/vulnerable-bom-components": 1,
			"https://test.com/api/projects/projects-id/versions/versions-id/components/components-id/versions/versions-id/origins/origins-id/vulnerabilities/CVE-2023-41164/remediation": 1,
			"https://test.com/api/projects/projects-id/versions/versions-id/components/components-id/versions/versions-id/origins/origins-id/vulnerabilities/CVE-2023-32731/remediation": 1,
		},
	)
}

func TestHandleAppliedMultipleFixes(t *testing.T) {
	django := api.PackageVersion{
		Version:                         "3.2.17",
		Library:                         api.Package{Name: "Django", PackageManager: mappings.PythonManager},
		RecommendedLibraryVersionId:     "111",
		RecommendedLibraryVersionString: "3.2.17-sp1",
		SealedVulnerabilities: []api.Vulnerability{
			{CVE: "CVE-2023-41164"},
		},
	}

	grpcio := api.PackageVersion{
		Version:                         "1.52.0",
		Library:                         api.Package{Name: "grpcio", PackageManager: mappings.PythonManager},
		RecommendedLibraryVersionId:     "222",
		RecommendedLibraryVersionString: "1.52.0-sp1",
		SealedVulnerabilities: []api.Vulnerability{
			{CVE: "CVE-2023-1428"},
			{CVE: "CVE-2023-32731"},
		},
	}

	requests := api.PackageVersion{
		Version:                         "2.26.0",
		Library:                         api.Package{Name: "requests", PackageManager: mappings.PythonManager},
		RecommendedLibraryVersionId:     "333",
		RecommendedLibraryVersionString: "2.26.0-sp1",
		SealedVulnerabilities: []api.Vulnerability{
			{CVE: "CVE-2023-32681"},
		},
	}

	fixmap := shared.FixMap{
		"a": &shared.FixedEntry{Package: &django, Paths: map[string]bool{
			filepath.Join("/prj", "pythonstuff/django"): true,
		}},
		"b": &shared.FixedEntry{Package: &grpcio, Paths: map[string]bool{
			filepath.Join("/prj", "pythonstuff/grpcio"): true,
		}},
		"c": &shared.FixedEntry{Package: &requests, Paths: map[string]bool{
			filepath.Join("/prj", "pythonstuff/requests"): true,
		}},
	}

	fakeRoundTripper := fakeRoundTripper{
		statusCode: 200,
		jsonContent: map[string]string{
			"https://test.com/api/projects/projects-id/versions/versions-id/components/components-id/versions/versions-id/origins/origins-id/vulnerabilities/CVE-2023-41164/remediation": "{}",
			"https://test.com/api/projects/projects-id/versions/versions-id/components/components-id/versions/versions-id/origins/origins-id/vulnerabilities/CVE-2023-1428/remediation":  "{}",
			"https://test.com/api/projects/projects-id/versions/versions-id/components/components-id/versions/versions-id/origins/origins-id/vulnerabilities/CVE-2023-32731/remediation": "{}",
			"https://test.com/api/projects/projects-id/versions/versions-id/components/components-id/versions/versions-id/origins/origins-id/vulnerabilities/CVE-2023-32681/remediation": "{}",
		},
	}

	client := http.Client{Transport: &fakeRoundTripper}
	c := BlackDuckClient{
		Client:          client,
		Url:             "https://test.com",
		Token:           "token",
		BearerToken:     "bearer-token",
		ValidUntil:      time.Now().Add(time.Hour),
		VersionToFilter: "versionName1",
	}

	err := handleAppliedFixes("projectName1", &c, fixmap)
	if err != nil {
		t.Errorf("HandleAppliedFixes() = %v, want nil", err)
	}

	fakeRoundTripper.CheckUrlCounter(
		t,
		map[string]int{
			"https://test.com/api/projects":                                                            1,
			"https://test.com/api/projects/projects-id/versions":                                       1,
			"https://test.com/api/projects/projects-id/versions/versions-id/vulnerable-bom-components": 1,
			"https://test.com/api/projects/projects-id/versions/versions-id/components/components-id/versions/versions-id/origins/origins-id/vulnerabilities/CVE-2023-41164/remediation": 1,
			"https://test.com/api/projects/projects-id/versions/versions-id/components/components-id/versions/versions-id/origins/origins-id/vulnerabilities/CVE-2023-1428/remediation":  1,
			"https://test.com/api/projects/projects-id/versions/versions-id/components/components-id/versions/versions-id/origins/origins-id/vulnerabilities/CVE-2023-32731/remediation": 1,
			"https://test.com/api/projects/projects-id/versions/versions-id/components/components-id/versions/versions-id/origins/origins-id/vulnerabilities/CVE-2023-32681/remediation": 1,
		},
	)
}
