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

func getFixAndScanResult() (shared.FixMap, []api.PackageVersion) {
	scannedDjango := api.PackageVersion{
		Version:                         "3.2.17+sp1",
		Library:                         api.Package{Name: "Django", PackageManager: mappings.PythonManager},
		RecommendedLibraryVersionId:     "111",
		RecommendedLibraryVersionString: "3.2.17+sp2",
		OpenVulnerabilities: []api.Vulnerability{
			{CVE: "CVE-2024-27351"},
			{CVE: "CVE-2023-46695"},
			{CVE: "CVE-2023-43665"},
		},
		SealedVulnerabilities: []api.Vulnerability{
			{CVE: "CVE-2023-41164"},
		},
		OriginVersion: "3.2.17",
	}

	scannedGrpcio := api.PackageVersion{
		Version:                         "1.52.0",
		Library:                         api.Package{Name: "grpcio", PackageManager: mappings.PythonManager},
		RecommendedLibraryVersionId:     "222",
		RecommendedLibraryVersionString: "1.52.0+sp1",
		OpenVulnerabilities: []api.Vulnerability{
			{CVE: "CVE-2023-1428"},
			{CVE: "CVE-2023-32731"},
		},
		OriginVersion: "1.52.0",
	}

	scannedRequests := api.PackageVersion{
		Version:                         "2.26.0",
		Library:                         api.Package{Name: "requests", PackageManager: mappings.PythonManager},
		RecommendedLibraryVersionId:     "333",
		RecommendedLibraryVersionString: "2.26.0+sp1",
		OpenVulnerabilities: []api.Vulnerability{
			{CVE: "CVE-2023-32681"},
		},
		OriginVersion: "2.26.0",
	}

	fixedDjango := api.PackageVersion{
		Version: "3.2.17+sp2",
		Library: api.Package{Name: "Django", PackageManager: mappings.PythonManager},
		SealedVulnerabilities: []api.Vulnerability{
			{CVE: "CVE-2024-27351"},
			{CVE: "CVE-2023-46695"},
			{CVE: "CVE-2023-43665"},
			{CVE: "CVE-2023-41164"},
		},
		OriginVersion: "3.2.17",
	}

	fixedGrpcio := api.PackageVersion{
		Version: "1.52.0+sp1",
		Library: api.Package{Name: "grpcio", PackageManager: mappings.PythonManager},
		OpenVulnerabilities: []api.Vulnerability{
			{CVE: "CVE-2023-1428"},
		},
		SealedVulnerabilities: []api.Vulnerability{
			{CVE: "CVE-2023-32731"},
		},
		OriginVersion: "1.52.0",
	}

	fixedRequests := api.PackageVersion{
		Version: "2.26.0+sp1",
		Library: api.Package{Name: "requests", PackageManager: mappings.PythonManager},
		SealedVulnerabilities: []api.Vulnerability{
			{CVE: "CVE-2023-32681"},
		},
		OriginVersion: "2.26.0",
	}

	fixmap := shared.FixMap{
		"a": &shared.FixedEntry{Package: &scannedDjango, Paths: map[string]bool{
			filepath.Join("/prj", "pythonstuff/django"): true,
		}},
		"b": &shared.FixedEntry{Package: &scannedGrpcio, Paths: map[string]bool{
			filepath.Join("/prj", "pythonstuff/grpcio"): true,
		}},
		"c": &shared.FixedEntry{Package: &scannedRequests, Paths: map[string]bool{
			filepath.Join("/prj", "pythonstuff/requests"): true,
		}},
	}

	fixResults := []api.PackageVersion{fixedDjango, fixedGrpcio, fixedRequests}

	return fixmap, fixResults
}

func TestBuildSealedVulnerabilitiesMapping(t *testing.T) {
	_, fixResults := getFixAndScanResult()

	got := buildSealedVulnerabilitiesMapping(fixResults)
	want := vulnerabilityMapping{
		"pypi/django/3.2.17/cve-2024-27351":   true,
		"pypi/django/3.2.17/cve-2023-46695":   true,
		"pypi/django/3.2.17/cve-2023-43665":   true,
		"pypi/django/3.2.17/cve-2023-41164":   true,
		"pypi/grpcio/1.52.0/cve-2023-32731":   true,
		"pypi/requests/2.26.0/cve-2023-32681": true,
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("buildSealedVulnerabilitiesMapping() = %v, want %v", got, want)
	}
}

func TestHandleAppliedFixes(t *testing.T) {
	_, fixResults := getFixAndScanResult()
	fixResults[1].SealedVulnerabilities = []api.Vulnerability{}

	fakeRoundTripper := fakeRoundTripper{
		statusCode: map[string]int{
			"https://test.com/api/projects/projects-id/versions/versions-id/components/components-id/versions/versions-id/origins/origins-id/vulnerabilities/CVE-2023-41164/remediation": 202,
			"https://test.com/api/projects/projects-id/versions/versions-id/components/components-id/versions/versions-id/origins/origins-id/vulnerabilities/CVE-2023-32681/remediation": 202,
			"https://test.com/api/projects/projects-id/versions/versions-id/components/components-id/versions/versions-id/origins/origins-id/vulnerabilities/CVE-2023-32731/remediation": 202, // unsealed
		},
		jsonContent: map[string]string{
			"https://test.com/api/projects/projects-id/versions/versions-id/components/components-id/versions/versions-id/origins/origins-id/vulnerabilities/CVE-2023-41164/remediation": "{}", // sealed
			"https://test.com/api/projects/projects-id/versions/versions-id/components/components-id/versions/versions-id/origins/origins-id/vulnerabilities/CVE-2023-32681/remediation": "{}", // sealed
			"https://test.com/api/projects/projects-id/versions/versions-id/components/components-id/versions/versions-id/origins/origins-id/vulnerabilities/CVE-2023-32731/remediation": "{}", // unsealed
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

	err := handleAppliedFixes("projectName1", &c, fixResults)
	if err != nil {
		t.Errorf("HandleAppliedFixes() = %v, want nil", err)
	}

	fakeRoundTripper.CheckUrlCounter(
		t,
		map[string]int{
			"https://test.com/api/projects":                                                            1,
			"https://test.com/api/projects/projects-id/versions":                                       1,
			"https://test.com/api/projects/projects-id/versions/versions-id/vulnerable-bom-components": 1,
			"https://test.com/api/projects/projects-id/versions/versions-id/components/components-id/versions/versions-id/origins/origins-id/vulnerabilities/CVE-2024-27351/remediation": 0,
			"https://test.com/api/projects/projects-id/versions/versions-id/components/components-id/versions/versions-id/origins/origins-id/vulnerabilities/CVE-2023-46695/remediation": 0,
			"https://test.com/api/projects/projects-id/versions/versions-id/components/components-id/versions/versions-id/origins/origins-id/vulnerabilities/CVE-2023-43665/remediation": 0,
			"https://test.com/api/projects/projects-id/versions/versions-id/components/components-id/versions/versions-id/origins/origins-id/vulnerabilities/CVE-2023-41164/remediation": 1,
			"https://test.com/api/projects/projects-id/versions/versions-id/components/components-id/versions/versions-id/origins/origins-id/vulnerabilities/CVE-2023-32681/remediation": 1,
			"https://test.com/api/projects/projects-id/versions/versions-id/components/components-id/versions/versions-id/origins/origins-id/vulnerabilities/CVE-2023-32731/remediation": 1,
		},
	)
}

func TestHandleAppliedMultipleFixes(t *testing.T) {
	_, fixResults := getFixAndScanResult()

	fakeRoundTripper := fakeRoundTripper{
		statusCode: map[string]int{
			"https://test.com/api/projects/projects-id/versions/versions-id/components/components-id/versions/versions-id/origins/origins-id/vulnerabilities/CVE-2023-41164/remediation": 202,
			"https://test.com/api/projects/projects-id/versions/versions-id/components/components-id/versions/versions-id/origins/origins-id/vulnerabilities/CVE-2023-32681/remediation": 202,
			"https://test.com/api/projects/projects-id/versions/versions-id/components/components-id/versions/versions-id/origins/origins-id/vulnerabilities/CVE-2023-32731/remediation": 202, // unsealed
		},
		jsonContent: map[string]string{
			"https://test.com/api/projects/projects-id/versions/versions-id/components/components-id/versions/versions-id/origins/origins-id/vulnerabilities/CVE-2023-41164/remediation": "{}", // sealed
			"https://test.com/api/projects/projects-id/versions/versions-id/components/components-id/versions/versions-id/origins/origins-id/vulnerabilities/CVE-2023-32681/remediation": "{}", // sealed
			"https://test.com/api/projects/projects-id/versions/versions-id/components/components-id/versions/versions-id/origins/origins-id/vulnerabilities/CVE-2023-32731/remediation": "{}", // unsealed
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

	err := handleAppliedFixes("projectName1", &c, fixResults)
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
			"https://test.com/api/projects/projects-id/versions/versions-id/components/components-id/versions/versions-id/origins/origins-id/vulnerabilities/CVE-2023-32681/remediation": 1,
			"https://test.com/api/projects/projects-id/versions/versions-id/components/components-id/versions/versions-id/origins/origins-id/vulnerabilities/CVE-2023-32731/remediation": 1,
		},
	)
}
