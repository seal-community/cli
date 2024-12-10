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
	scannedSmolToml := api.PackageVersion{
		Version:                         "1.3.0",
		Library:                         api.Package{NormalizedName: "smol-toml", Name: "smol-toml", PackageManager: mappings.NpmManager},
		RecommendedLibraryVersionId:     "11111",
		RecommendedLibraryVersionString: "1.3.0+sp1",
		OpenVulnerabilities: []api.Vulnerability{
			{GitHubAdvisoryID: "GHSA-pqhp-25j4-6hq9"},
		},
		OriginVersionString: "1.3.0",
	}
	fixedSmolToml := api.PackageVersion{
		Version:             "1.3.0+sp1",
		Library:             api.Package{NormalizedName: "smol-toml", Name: "smol-toml", PackageManager: mappings.NpmManager},
		OpenVulnerabilities: []api.Vulnerability{},
		SealedVulnerabilities: []api.Vulnerability{
			{GitHubAdvisoryID: "GHSA-pqhp-25j4-6hq9"},
		},
		OriginVersionString: "1.3.0",
	}

	scannedEjs := api.PackageVersion{
		Version:                         "3.1.10",
		Library:                         api.Package{NormalizedName: "ejs", Name: "ejs", PackageManager: mappings.NpmManager},
		RecommendedLibraryVersionId:     "2222222",
		RecommendedLibraryVersionString: "3.1.10+sp1",
		OpenVulnerabilities: []api.Vulnerability{
			{CVE: "CVE-2024-33883", GitHubAdvisoryID: "GHSA-ghr5-ch3p-vcr6"},
			{CVE: "CVE-dummy-open-vuln", GitHubAdvisoryID: "GHSA-dummy-open-vuln"},
		},
		OriginVersionString: "3.1.10",
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

	scannedEjsAnotherVersion := api.PackageVersion{
		Version:                         "3.1.9",
		Library:                         api.Package{NormalizedName: "ejs", Name: "ejs", PackageManager: mappings.NpmManager},
		RecommendedLibraryVersionId:     "3333333",
		RecommendedLibraryVersionString: "3.1.9+sp1",
		OpenVulnerabilities: []api.Vulnerability{
			{CVE: "CVE-2024-33883", GitHubAdvisoryID: "GHSA-ghr5-ch3p-vcr6"},
			{CVE: "CVE-dummy-open-vuln", GitHubAdvisoryID: "GHSA-dummy-open-vuln"},
		},
		OriginVersionString: "3.1.9",
	}
	fixedEjsAnotherVersion := api.PackageVersion{
		Version: "3.1.9+sp1",
		Library: api.Package{NormalizedName: "ejs", Name: "ejs", PackageManager: mappings.NpmManager},
		OpenVulnerabilities: []api.Vulnerability{
			{CVE: "CVE-2024-33883", GitHubAdvisoryID: "GHSA-ghr5-ch3p-vcr6"},
		},
		SealedVulnerabilities: []api.Vulnerability{},
		OriginVersionString:   "3.1.9",
	}

	return []shared.DependencyDescriptor{
		{VulnerablePackage: &scannedSmolToml, AvailableFix: &fixedSmolToml, Locations: nil, FixedLocations: nil},
		{VulnerablePackage: &scannedEjs, AvailableFix: &fixedEjs, Locations: nil, FixedLocations: nil},
		{VulnerablePackage: &scannedEjsAnotherVersion, AvailableFix: &fixedEjsAnotherVersion, Locations: nil, FixedLocations: nil},
	}
}

func getVulnerableResults() []api.PackageVersion {
	vulnerableSmolToml := api.PackageVersion{
		Version: "1.3.0",
		Library: api.Package{NormalizedName: "smol-toml", Name: "smol-toml", PackageManager: mappings.NpmManager},
		OpenVulnerabilities: []api.Vulnerability{
			{GitHubAdvisoryID: "GHSA-pqhp-25j4-6hq9"},
		},
	}
	vulnerableEjs := api.PackageVersion{
		Version: "3.1.10",
		Library: api.Package{NormalizedName: "ejs", Name: "ejs", PackageManager: mappings.NpmManager},
		OpenVulnerabilities: []api.Vulnerability{
			{CVE: "CVE-2024-33883", GitHubAdvisoryID: "GHSA-ghr5-ch3p-vcr6"},
			{CVE: "CVE-dummy-open-vuln", GitHubAdvisoryID: "GHSA-dummy-open-vuln"},
		},
	}
	vulnerableEjsAnotherVersion := api.PackageVersion{
		Version: "3.1.9",
		Library: api.Package{NormalizedName: "ejs", Name: "ejs", PackageManager: mappings.NpmManager},
		OpenVulnerabilities: []api.Vulnerability{
			{CVE: "CVE-2024-33883", GitHubAdvisoryID: "GHSA-ghr5-ch3p-vcr6"},
			{CVE: "CVE-dummy-open-vuln", GitHubAdvisoryID: "GHSA-dummy-open-vuln"},
		},
	}
	return []api.PackageVersion{
		vulnerableSmolToml,
		vulnerableEjs,
		vulnerableEjsAnotherVersion,
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

	err := handleAppliedFixes(&c, fixResults, getVulnerableResults())
	if err != nil {
		t.Errorf("HandleAppliedFixes() = %v, want nil", err)
	}

	fakeRoundTripper.CheckUrlCounter(
		t,
		map[string]int{
			"https://api.github.com/repos/owner-id/repo-id/dependabot/alerts":    2,
			"https://api.github.com/repos/owner-id/repo-id/dependabot/alerts/30": 1, // sealed
			"https://api.github.com/repos/owner-id/repo-id/dependabot/alerts/29": 0, // not sealed
			"https://api.github.com/repos/owner-id/repo-id/dependabot/alerts/28": 0, // 1 sealed 1 not sealed
			"https://api.github.com/repos/owner-id/repo-id/dependabot/alerts/1":  0, // not in request
		},
	)
}

func TestPatchVulnInDependabot(t *testing.T) {
	scannedSmolToml := api.PackageVersion{
		Version:                         "1.3.0",
		Library:                         api.Package{NormalizedName: "smol-toml", Name: "smol-toml", PackageManager: mappings.NpmManager},
		RecommendedLibraryVersionId:     "11111",
		RecommendedLibraryVersionString: "1.3.0+sp1",
		OpenVulnerabilities: []api.Vulnerability{
			{GitHubAdvisoryID: "GHSA-pqhp-25j4-6hq9"},
		},
		OriginVersionString: "1.3.0",
	}
	fixedSmolToml := api.PackageVersion{
		Version: "1.3.0+sp1",
		Library: api.Package{NormalizedName: "smol-toml", Name: "smol-toml", PackageManager: mappings.NpmManager},
		OpenVulnerabilities: []api.Vulnerability{
			{GitHubAdvisoryID: "GHSA-pqhp-25j4-6hq9"},
		},
		SealedVulnerabilities: []api.Vulnerability{},
		OriginVersionString:   "1.3.0",
	}
	fixResults := []shared.DependencyDescriptor{
		{VulnerablePackage: &scannedSmolToml, AvailableFix: &fixedSmolToml, Locations: nil, FixedLocations: nil},
	}
	vulnerableSmolToml := api.PackageVersion{
		Version: "1.3.0",
		Library: api.Package{NormalizedName: "smol-toml", Name: "smol-toml", PackageManager: mappings.NpmManager},
		OpenVulnerabilities: []api.Vulnerability{
			{GitHubAdvisoryID: "GHSA-pqhp-25j4-6hq9"},
		},
	}
	vulnerable := []api.PackageVersion{
		vulnerableSmolToml,
	}

	fakeRoundTripper := fakeRoundTripper{}

	client := http.Client{Transport: &fakeRoundTripper}
	c := DependabotClient{
		Client: client,
		Url:    "https://api.github.com",
		Token:  "token",
		Owner:  "owner-id",
		Repo:   "repo-id",
	}

	err := handleAppliedFixes(&c, fixResults, vulnerable)
	if err != nil {
		t.Errorf("HandleAppliedFixes() = %v, want nil", err)
	}

	fakeRoundTripper.CheckUrlCounter(
		t,
		map[string]int{
			"https://api.github.com/repos/owner-id/repo-id/dependabot/alerts":    2,
			"https://api.github.com/repos/owner-id/repo-id/dependabot/alerts/30": 1, // sealed
		},
	)
}

func TestBuildSealedVulnerabilitiesMappingOriginVersionWithSp1(t *testing.T) {
	// we pulled the original version and we have sp1 affected by GHSA-X - we expect Dependabot to close alert
	scannedEjs := api.PackageVersion{
		Version:                         "3.1.10",
		Library:                         api.Package{NormalizedName: "ejs", Name: "ejs", PackageManager: mappings.NpmManager},
		RecommendedLibraryVersionId:     "2222222",
		RecommendedLibraryVersionString: "3.1.10+sp1",
		OpenVulnerabilities: []api.Vulnerability{
			{CVE: "CVE-2024-33883", GitHubAdvisoryID: "GHSA-ghr5-ch3p-vcr6"},
			{CVE: "CVE-dummy-open-vuln", GitHubAdvisoryID: "GHSA-dummy-open-vuln"},
		},
		OriginVersionString: "3.1.10",
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
	fixResults := []shared.DependencyDescriptor{
		{VulnerablePackage: &scannedEjs, AvailableFix: &fixedEjs, Locations: nil, FixedLocations: nil},
	}
	vulnerableEjs := api.PackageVersion{
		Version: "3.1.10",
		Library: api.Package{NormalizedName: "ejs", Name: "ejs", PackageManager: mappings.NpmManager},
		OpenVulnerabilities: []api.Vulnerability{
			{GitHubAdvisoryID: "GHSA-ghr5-ch3p-vcr6"},
			{GitHubAdvisoryID: "GHSA-dummy-open-vuln"},
		},
	}
	vulnerable := []api.PackageVersion{
		vulnerableEjs,
	}

	got := buildSealedVulnerabilitiesMapping(fixResults, vulnerable)
	want := vulnerabilityMapping{
		"npm/ejs/ghsa-ghr5-ch3p-vcr6": true,
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("buildSealedVulnerabilitiesMapping() = %v, want %v", got, want)
	}
}

func TestBuildSealedVulnerabilitiesMappingPulledSp1NoSp2(t *testing.T) {
	// we pulled the sealed version, which is vulnerable to GHSA-X, and we don't have sp2 - we expect Dependabot to not close alert
	scannedEjs := api.PackageVersion{
		Version:                     "3.1.10+sp1",
		Library:                     api.Package{NormalizedName: "ejs", Name: "ejs", PackageManager: mappings.NpmManager},
		RecommendedLibraryVersionId: "2222222",
		OpenVulnerabilities: []api.Vulnerability{
			{CVE: "CVE-dummy-open-vuln", GitHubAdvisoryID: "GHSA-dummy-open-vuln"},
		},
		OriginVersionString: "3.1.10",
	}
	fixResults := []shared.DependencyDescriptor{
		{VulnerablePackage: &scannedEjs, AvailableFix: nil, Locations: nil, FixedLocations: nil},
	}
	vulnerableEjs := api.PackageVersion{
		Version: "3.1.10+sp1",
		Library: api.Package{NormalizedName: "ejs", Name: "ejs", PackageManager: mappings.NpmManager},
		OpenVulnerabilities: []api.Vulnerability{
			{GitHubAdvisoryID: "GHSA-dummy-open-vuln"},
		},
	}
	vulnerable := []api.PackageVersion{
		vulnerableEjs,
	}

	got := buildSealedVulnerabilitiesMapping(fixResults, vulnerable)
	want := vulnerabilityMapping{}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("buildSealedVulnerabilitiesMapping() = %v, want %v", got, want)
	}
}

func TestBuildSealedVulnerabilitiesMappingPulledSp1WithSp2(t *testing.T) {
	// we pulled the sealed version (sp1), which is vulnerable to GHSA-X, and we do have +sp2 - we expect Dependabot to close alert
	scannedEjs := api.PackageVersion{
		Version:                         "3.1.10+sp1",
		Library:                         api.Package{NormalizedName: "ejs", Name: "ejs", PackageManager: mappings.NpmManager},
		RecommendedLibraryVersionId:     "2222222",
		RecommendedLibraryVersionString: "3.1.10+sp2",
		OpenVulnerabilities: []api.Vulnerability{
			{CVE: "CVE-2024-33883", GitHubAdvisoryID: "GHSA-ghr5-ch3p-vcr6"},
		},
		OriginVersionString: "3.1.10",
	}
	fixedEjs := api.PackageVersion{
		Version:             "3.1.10+sp2",
		Library:             api.Package{NormalizedName: "ejs", Name: "ejs", PackageManager: mappings.NpmManager},
		OpenVulnerabilities: []api.Vulnerability{},
		SealedVulnerabilities: []api.Vulnerability{
			{CVE: "CVE-2024-33883", GitHubAdvisoryID: "GHSA-ghr5-ch3p-vcr6"},
		},
		OriginVersionString: "3.1.10",
	}
	fixResults := []shared.DependencyDescriptor{
		{VulnerablePackage: &scannedEjs, AvailableFix: &fixedEjs, Locations: nil, FixedLocations: nil},
	}
	vulnerableEjs := api.PackageVersion{
		Version: "3.1.10+sp1",
		Library: api.Package{NormalizedName: "ejs", Name: "ejs", PackageManager: mappings.NpmManager},
		OpenVulnerabilities: []api.Vulnerability{
			{GitHubAdvisoryID: "GHSA-ghr5-ch3p-vcr6"},
		},
	}
	vulnerable := []api.PackageVersion{
		vulnerableEjs,
	}

	got := buildSealedVulnerabilitiesMapping(fixResults, vulnerable)
	want := vulnerabilityMapping{
		"npm/ejs/ghsa-ghr5-ch3p-vcr6": true,
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("buildSealedVulnerabilitiesMapping() = %v, want %v", got, want)
	}
}

func TestBuildSealedVulnerabilitiesMappingOriginVersionNoSp1(t *testing.T) {
	// we pulled the original version and we don't have sp1 - we expect Dependabot to not close any alert
	scannedEjs := api.PackageVersion{
		Version:                         "3.1.10",
		Library:                         api.Package{NormalizedName: "ejs", Name: "ejs", PackageManager: mappings.NpmManager},
		RecommendedLibraryVersionId:     "2222222",
		RecommendedLibraryVersionString: "3.1.10+sp1",
		OpenVulnerabilities: []api.Vulnerability{
			{CVE: "CVE-dummy-open-vuln", GitHubAdvisoryID: "GHSA-dummy-open-vuln"},
		},
		OriginVersionString: "3.1.10",
	}
	fixResults := []shared.DependencyDescriptor{
		{VulnerablePackage: &scannedEjs, AvailableFix: nil, Locations: nil, FixedLocations: nil},
	}
	vulnerableEjs := api.PackageVersion{
		Version: "3.1.10",
		Library: api.Package{NormalizedName: "ejs", Name: "ejs", PackageManager: mappings.NpmManager},
		OpenVulnerabilities: []api.Vulnerability{
			{GitHubAdvisoryID: "GHSA-dummy-open-vuln"},
		},
	}
	vulnerable := []api.PackageVersion{
		vulnerableEjs,
	}

	got := buildSealedVulnerabilitiesMapping(fixResults, vulnerable)
	want := vulnerabilityMapping{}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("buildSealedVulnerabilitiesMapping() = %v, want %v", got, want)
	}
}

func TestBuildSealedVulnerabilitiesMappingMultipleVersionsOneAlert(t *testing.T) {
	// we pulled the 2 versions with the same vulnerability, only one fixed - we expect Dependabot not to close any alert
	scannedEjs := api.PackageVersion{
		Version:                         "3.1.10",
		Library:                         api.Package{NormalizedName: "ejs", Name: "ejs", PackageManager: mappings.NpmManager},
		RecommendedLibraryVersionId:     "2222222",
		RecommendedLibraryVersionString: "3.1.10+sp1",
		OpenVulnerabilities: []api.Vulnerability{
			{CVE: "CVE-dummy-sealed-once", GitHubAdvisoryID: "GHSA-dummy-sealed-once"},
			{CVE: "CVE-dummy-not-sealed", GitHubAdvisoryID: "GHSA-dummy-not-sealed"},
		},
		OriginVersionString: "3.1.10",
	}
	fixedEjs := api.PackageVersion{
		Version:             "3.1.10+sp1",
		Library:             api.Package{NormalizedName: "ejs", Name: "ejs", PackageManager: mappings.NpmManager},
		OpenVulnerabilities: []api.Vulnerability{},
		SealedVulnerabilities: []api.Vulnerability{
			{CVE: "CVE-dummy-sealed-once", GitHubAdvisoryID: "GHSA-dummy-sealed-once"},
		},
		OriginVersionString: "3.1.10",
	}

	scannedAnotherEjs := api.PackageVersion{
		Version:                         "3.1.9",
		Library:                         api.Package{NormalizedName: "ejs", Name: "ejs", PackageManager: mappings.NpmManager},
		RecommendedLibraryVersionId:     "2222222",
		RecommendedLibraryVersionString: "3.1.9+sp1",
		OpenVulnerabilities: []api.Vulnerability{
			{CVE: "CVE-dummy-sealed-once", GitHubAdvisoryID: "GHSA-dummy-sealed-once"},
			{CVE: "CVE-dummy-another-sealed", GitHubAdvisoryID: "GHSA-dummy-another-sealed"},
		},
		OriginVersionString: "3.1.9",
	}
	fixedAnotherEjs := api.PackageVersion{
		Version:             "3.1.9+sp1",
		Library:             api.Package{NormalizedName: "ejs", Name: "ejs", PackageManager: mappings.NpmManager},
		OpenVulnerabilities: []api.Vulnerability{},
		SealedVulnerabilities: []api.Vulnerability{
			{CVE: "CVE-dummy-another-sealed", GitHubAdvisoryID: "GHSA-dummy-another-sealed"},
		},
		OriginVersionString: "3.1.9",
	}
	fixResults := []shared.DependencyDescriptor{
		{VulnerablePackage: &scannedEjs, AvailableFix: &fixedEjs, Locations: nil, FixedLocations: nil},
		{VulnerablePackage: &scannedAnotherEjs, AvailableFix: &fixedAnotherEjs, Locations: nil, FixedLocations: nil},
	}

	vulnerableEjs := api.PackageVersion{
		Version: "3.1.10",
		Library: api.Package{NormalizedName: "ejs", Name: "ejs", PackageManager: mappings.NpmManager},
		OpenVulnerabilities: []api.Vulnerability{
			{GitHubAdvisoryID: "GHSA-dummy-sealed-once"},
			{GitHubAdvisoryID: "GHSA-dummy-not-sealed"},
		},
	}
	vulnerableAnotherEjs := api.PackageVersion{
		Version: "3.1.9",
		Library: api.Package{NormalizedName: "ejs", Name: "ejs", PackageManager: mappings.NpmManager},
		OpenVulnerabilities: []api.Vulnerability{
			{GitHubAdvisoryID: "GHSA-dummy-sealed-once"},
			{GitHubAdvisoryID: "GHSA-dummy-another-sealed"},
		},
	}
	vulnerable := []api.PackageVersion{
		vulnerableEjs,
		vulnerableAnotherEjs,
	}

	got := buildSealedVulnerabilitiesMapping(fixResults, vulnerable)
	want := vulnerabilityMapping{
		"npm/ejs/ghsa-dummy-another-sealed": true,
	}

	if !reflect.DeepEqual(got, want) {
		t.Errorf("buildSealedVulnerabilitiesMapping() = %v, want %v", got, want)
	}
}
