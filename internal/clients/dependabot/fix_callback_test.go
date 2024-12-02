package dependabot

import (
	"cli/internal/api"
	"cli/internal/ecosystem/mappings"
	"cli/internal/ecosystem/shared"
	"net/http"
	"reflect"
	"testing"
)

func getFixAndScanResult() []shared.DependencyDescriptor {
	scannedEjs := api.PackageVersion{
		Version:                         "3.1.10",
		Library:                         api.Package{NormalizedName: "ejs", Name: "ejs", PackageManager: mappings.NpmManager},
		RecommendedLibraryVersionId:     "11111",
		RecommendedLibraryVersionString: "3.1.10+sp1",
		OpenVulnerabilities: []api.Vulnerability{
			{CVE: "CVE-2024-33883", GitHubAdvisoryID: "GHSA-ghr5-ch3p-vcr6"},
		},
		OriginVersionString: "1.52.0",
	}

	fixedEjs := api.PackageVersion{
		Version:             "3.1.10+sp1",
		Library:             api.Package{NormalizedName: "ejs", Name: "ejs", PackageManager: mappings.NpmManager},
		OpenVulnerabilities: []api.Vulnerability{},
		SealedVulnerabilities: []api.Vulnerability{
			{CVE: "CVE-2024-33883", GitHubAdvisoryID: "GHSA-ghr5-ch3p-vcr6"},
		},
		OriginVersionString: "3.1.10",
	}

	return []shared.DependencyDescriptor{
		{VulnerablePackage: &scannedEjs, AvailableFix: &fixedEjs, Locations: nil, FixedLocations: nil},
	}
}

func TestBuildSealedVulnerabilitiesMapping(t *testing.T) {
	fixResults := getFixAndScanResult()

	got := buildSealedVulnerabilitiesMapping(fixResults)
	want := vulnerabilityMapping{
		"npm/ejs/ghsa-ghr5-ch3p-vcr6": true,
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("buildSealedVulnerabilitiesMapping() = %v, want %v", got, want)
	}
}
func TestHandleAppliedMultipleFixes(t *testing.T) {
	fixResults := getFixAndScanResult()

	fakeRoundTripper := fakeRoundTripper{}

	client := http.Client{Transport: &fakeRoundTripper}
	c := DependabotClient{
		Client: client,
		Url:    "https://api.github.com",
		Token:  "token",
		Owner:  "owner-id",
		Repo:   "repo-id",
	}

	err := handleAppliedFixes(&c, fixResults)
	if err != nil {
		t.Errorf("HandleAppliedFixes() = %v, want nil", err)
	}

	fakeRoundTripper.CheckUrlCounter(
		t,
		map[string]int{
			"https://api.github.com/repos/owner-id/repo-id/dependabot/alerts":    2,
			"https://api.github.com/repos/owner-id/repo-id/dependabot/alerts/28": 1, // sealed
			"https://api.github.com/repos/owner-id/repo-id/dependabot/alerts/27": 0, // not sealed
			"https://api.github.com/repos/owner-id/repo-id/dependabot/alerts/1":  0, // not in request
		},
	)
}
