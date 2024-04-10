package main

import (
	"cli/internal/actions"
	"cli/internal/api"
	"cli/internal/ecosystem/mappings"
	"slices"
	"testing"
)

func TestOverrideFilterSanity(t *testing.T) {
	vulnPackages := []api.PackageVersion{
		{
			Version:                         "1.2.3",
			Library:                         api.Package{Name: "lodash", PackageManager: mappings.NpmManager},
			RecommendedLibraryVersionId:     "123123",
			RecommendedLibraryVersionString: "1.2.3-sp1",
			OpenVulnerabilities:             []api.Vulnerability{{CVE: "CVE-2012-0865", UnifiedScore: 9.8}},
		},
		{
			Version:                         "1.0.0",
			Library:                         api.Package{Name: "semver-regex", PackageManager: mappings.NpmManager},
			RecommendedLibraryVersionId:     "123123",
			RecommendedLibraryVersionString: "1.0.0-sp1",
			OpenVulnerabilities:             []api.Vulnerability{{CVE: "CVE-2012-12312", UnifiedScore: 9.8}},
		},
	}

	overriddenVersion := "1.2.3-myoverride"
	overrides := actions.LibraryOverrideMap{"lodash": actions.VersionOverrideMap{"1.2.3": actions.Override{Version: overriddenVersion}}}
	proj := actions.ProjectSection{Manager: actions.ProjectManagerSection{
		Ecosystem: "node",
		Name:      mappings.NpmManager,
	}, Overrides: overrides}

	result := filterVulnerablePackageForProject(vulnPackages, proj)
	if len(result) != 1 {
		t.Fatalf("wrong result length filtered %d", len(result))
	}

	overriddenPackage := result[0]
	vulnPackage := vulnPackages[0]
	if overriddenPackage.RecommendedLibraryVersionId != vulnPackage.RecommendedLibraryVersionId {
		// should not clear it for now, since it is being used to filter out what to install later
		t.Fatalf("removed recommended id from overridden package %s", overriddenPackage.RecommendedLibraryVersionId)
	}

	if overriddenPackage.Library.Name != vulnPackage.Library.Name {
		t.Fatalf("wrong library name %s", overriddenPackage.Library.Name)
	}

	if overriddenPackage.Library.PackageManager != vulnPackage.Library.PackageManager {
		t.Fatalf("wrong package manager %s", overriddenPackage.Library.PackageManager)
	}

	if !slices.Equal(overriddenPackage.OpenVulnerabilities, vulnPackage.OpenVulnerabilities) {
		t.Fatalf("wrong open vulns %v; expected %v", overriddenPackage.Library.PackageManager, vulnPackage.OpenVulnerabilities)
	}

	if overriddenPackage.RecommendedLibraryVersionString != overriddenVersion {
		// should not clear it for now, since it is being used to filter out what to install later
		t.Fatalf("wrong overidden version `%s` instead of `%s`", overriddenPackage.RecommendedLibraryVersionString, overriddenVersion)
	}

	if string(overriddenPackage.OverrideMethod) != string(api.OverriddenFromLocal) {
		// should not clear it for now, since it is being used to filter out what to install later
		t.Fatalf("wrong overidden method `%s` ", overriddenPackage.OverrideMethod)
	}
}

func TestOverrideFilterNoMatchLibrary(t *testing.T) {
	vulnPackages := []api.PackageVersion{
		{
			Version:                         "1.2.3",
			Library:                         api.Package{Name: "lodash", PackageManager: mappings.NpmManager},
			RecommendedLibraryVersionId:     "123123",
			RecommendedLibraryVersionString: "1.2.3-sp1",
			OpenVulnerabilities:             []api.Vulnerability{{CVE: "CVE-2012-0865", UnifiedScore: 9.8}},
		},
	}

	overriddenVersion := "1.2.3-myoverride"
	overrides := actions.LibraryOverrideMap{"semver-regex": actions.VersionOverrideMap{"1.0.0": actions.Override{Version: overriddenVersion}}}
	proj := actions.ProjectSection{Manager: actions.ProjectManagerSection{
		Ecosystem: "node",
		Name:      mappings.NpmManager,
	}, Overrides: overrides}

	result := filterVulnerablePackageForProject(vulnPackages, proj)
	if len(result) != 0 {
		t.Fatalf("wrong result length filtered %v", result)
	}
}

func TestOverrideFilterNoMatchVersion(t *testing.T) {
	vulnPackages := []api.PackageVersion{
		{
			Version:                         "1.2.3",
			Library:                         api.Package{Name: "lodash", PackageManager: mappings.NpmManager},
			RecommendedLibraryVersionId:     "123123",
			RecommendedLibraryVersionString: "1.2.3-sp1",
			OpenVulnerabilities:             []api.Vulnerability{{CVE: "CVE-2012-0865", UnifiedScore: 9.8}},
		},
	}

	overriddenVersion := "1.2.3-myoverride"
	overrides := actions.LibraryOverrideMap{"lodash": actions.VersionOverrideMap{"4.5.15": actions.Override{Version: overriddenVersion}}}
	proj := actions.ProjectSection{Manager: actions.ProjectManagerSection{
		Ecosystem: "node",
		Name:      mappings.NpmManager,
	}, Overrides: overrides}

	result := filterVulnerablePackageForProject(vulnPackages, proj)
	if len(result) != 0 {
		t.Fatalf("wrong result length filtered %v", result)
	}
}

func TestOverrideFilterDoesNotAllowOverrideNonSealable(t *testing.T) {
	vulnPackages := []api.PackageVersion{
		{
			Version:                         "1.2.3",
			Library:                         api.Package{Name: "lodash", PackageManager: mappings.NpmManager},
			RecommendedLibraryVersionId:     "",
			RecommendedLibraryVersionString: "",
			OpenVulnerabilities:             []api.Vulnerability{{CVE: "CVE-2012-0865", UnifiedScore: 9.8}},
		},
	}

	overriddenVersion := "1.2.3-myoverride"
	overrides := actions.LibraryOverrideMap{"lodash": actions.VersionOverrideMap{"1.2.3": actions.Override{Version: overriddenVersion}}}
	proj := actions.ProjectSection{Manager: actions.ProjectManagerSection{
		Ecosystem: "node",
		Name:      mappings.NpmManager,
	}, Overrides: overrides}

	result := filterVulnerablePackageForProject(vulnPackages, proj)
	if len(result) != 0 {
		t.Fatalf("wrong result length filtered %v", result)
	}
}
func TestOverrideFilterWithNoAllowedOverrides(t *testing.T) {
	vulnPackages := []api.PackageVersion{
		{
			Version:                         "1.2.3",
			Library:                         api.Package{Name: "lodash", PackageManager: mappings.NpmManager},
			RecommendedLibraryVersionId:     "",
			RecommendedLibraryVersionString: "",
			OpenVulnerabilities:             []api.Vulnerability{{CVE: "CVE-2012-0865", UnifiedScore: 9.8}},
		},
	}
	
	overrides := actions.LibraryOverrideMap{}
	proj := actions.ProjectSection{
		Manager: actions.ProjectManagerSection{
			Ecosystem: "node",
			Name:      mappings.NpmManager,
		},
		Overrides: overrides,
	}

	result := filterVulnerablePackageForProject(vulnPackages, proj)
	if len(result) != 0 {
		t.Fatalf("wrong result length filtered %v", result)
	}
}

func TestFixModeParsing(t *testing.T) {
	f := fixModeFromString("local")
	if f != localMode {
		t.Fatalf("failed to parse local mode")
	}

	f = fixModeFromString("all")
	if f != allMode {
		t.Fatalf("failed to parse all mode")
	}

	f = fixModeFromString("fail")
	if f != "" {
		t.Fatalf("failed to parse unknown mode")
	}
}
