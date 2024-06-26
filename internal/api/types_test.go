package api

import (
	"cli/internal/ecosystem/mappings"
	"testing"
)

func TestIsMalicious(t *testing.T) {
	openVulnerabilities := []Vulnerability{
		{
			MaliciousID: "MAL-2022-7421",
		},
	}

	vulnerablePackage := PackageVersion{
		OpenVulnerabilities: openVulnerabilities,
		Version:             "1.2.3",
		Library: Package{
			Name:           "ejs",
			NormalizedName: "ejs",
			PackageManager: "NPM",
		},
	}

	if !vulnerablePackage.IsMalicious() {
		t.Fatalf("expected malicious")
	}
}

func TestIsNotMalicious(t *testing.T) {
	openVulnerabilities := []Vulnerability{
		{
			CVE: "CVE-123",
		},
	}

	vulnerablePackage := PackageVersion{
		OpenVulnerabilities: openVulnerabilities,
		Version:             "1.2.3",
		Library: Package{
			Name:           "ejs",
			NormalizedName: "ejs",
			PackageManager: "NPM",
		},
	}

	if vulnerablePackage.IsMalicious() {
		t.Fatalf("expected not malicious")
	}
}

func TestCanBeFixed(t *testing.T) {
	packageVersion := PackageVersion{
		RecommendedLibraryVersionId: "123",
	}

	if !packageVersion.CanBeFixed() {
		t.Fatalf("expected can be fixed")
	}
}

func TestCanNotBeFixed(t *testing.T) {
	packageVersion := PackageVersion{}
	if packageVersion.CanBeFixed() {
		t.Fatalf("expected can not be fixed")
	}
}

func TestPreferredIdMalicious(t *testing.T) {
	vulnerability := Vulnerability{
		MaliciousID:      "MAL-2022-7421",
		CVE:              "CVE-123",
		GitHubAdvisoryID: "GHSA-123",
		SnykID:           "SNYK-123",
	}

	if vulnerability.PreferredId() != "MAL-2022-7421" {
		t.Fatalf("expected MAL-2022-7421")
	}
}

func TestPreferredIdCVE(t *testing.T) {
	vulnerability := Vulnerability{
		CVE:              "CVE-123",
		GitHubAdvisoryID: "GHSA-123",
		SnykID:           "SNYK-123",
	}

	if vulnerability.PreferredId() != "CVE-123" {
		t.Fatalf("expected CVE-123")
	}
}

func TestPreferredIdGitHub(t *testing.T) {
	vulnerability := Vulnerability{
		GitHubAdvisoryID: "GHSA-123",
		SnykID:           "SNYK-123",
	}

	if vulnerability.PreferredId() != "GHSA-123" {
		t.Fatalf("expected GHSA-123")
	}
}

func TestPreferredIdSnyk(t *testing.T) {
	vulnerability := Vulnerability{
		SnykID: "SNYK-123",
	}

	if vulnerability.PreferredId() != "SNYK-123" {
		t.Fatalf("expected SNYK-123")
	}
}

func TestOriginId(t *testing.T) {
	pv := PackageVersion{
		VersionId: "1111",
		Library: Package{
			PackageManager: mappings.NugetManager,
			Name:           "Library",
			NormalizedName: "library",
		},
		Version:                         "version",
		RecommendedLibraryVersionId:     "2222",
		RecommendedLibraryVersionString: "recommended",
		OriginVersionId:                 "3333",
		OriginVersionString:             "origin",
	}

	if oid := pv.OriginId(); oid != "NuGet|library@origin" {
		t.Fatalf("got %s", oid)
	}
}

func TestRecommendedId(t *testing.T) {
	pv := PackageVersion{
		VersionId: "1111",
		Library: Package{
			PackageManager: mappings.NugetManager,
			Name:           "Library",
			NormalizedName: "library",
		},
		Version:                         "version",
		RecommendedLibraryVersionId:     "2222",
		RecommendedLibraryVersionString: "recommended",
		OriginVersionId:                 "3333",
		OriginVersionString:             "origin",
	}

	if rid := pv.RecommendedId(); rid != "NuGet|library@recommended" {
		t.Fatalf("got %s", rid)
	}
}

func TestOriginIdForOrigin(t *testing.T) {
	pv := PackageVersion{
		VersionId: "1111",
		Library: Package{
			PackageManager: mappings.NugetManager,
			Name:           "Library",
			NormalizedName: "library",
		},
		Version:                         "version",
		RecommendedLibraryVersionId:     "2222",
		RecommendedLibraryVersionString: "recommended",
	}

	if oid := pv.OriginId(); oid != "NuGet|library@version" {
		t.Fatalf("got %s", oid)
	}
}

func TestVersionId(t *testing.T) {
	pv := PackageVersion{
		VersionId: "1111",
		Library: Package{
			PackageManager: mappings.NugetManager,
			Name:           "Library",
			NormalizedName: "library",
		},
		Version:                         "version",
		RecommendedLibraryVersionId:     "2222",
		RecommendedLibraryVersionString: "recommended",
		OriginVersionId:                 "3333",
		OriginVersionString:             "origin",
	}

	if vid := pv.Id(); vid != "NuGet|library@version" {
		t.Fatalf("got %s", vid)
	}
}
