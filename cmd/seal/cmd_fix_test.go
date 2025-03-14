package main

import (
	"cli/internal/actions"
	"cli/internal/api"
	"cli/internal/ecosystem/mappings"
	"cli/internal/phase"
	"reflect"
	"testing"
)

type npmFakeNormalizer struct{}

func (f npmFakeNormalizer) NormalizePackageName(name string) string {
	return name
}

func TestOverrideFilterSanity(t *testing.T) {
	vulnPackages := []api.PackageVersion{
		{
			Version:                         "1.2.3",
			Library:                         api.Package{NormalizedName: "lodash", Name: "lodash", PackageManager: mappings.NpmManager},
			RecommendedLibraryVersionId:     "123123",
			RecommendedLibraryVersionString: "1.2.3-sp1",
			OpenVulnerabilities:             []api.Vulnerability{{CVE: "CVE-2012-0865", UnifiedScore: 9.8}},
		},
		{
			Version:                         "1.0.0",
			Library:                         api.Package{NormalizedName: "semver-regex", Name: "semver-regex", PackageManager: mappings.NpmManager},
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

	meta := actions.MetaSection{SchemaVersion: "0.1.0", CreatedOn: actions.IsoTime{Time: actions.IsoTime{}.Time}, CliVersion: "0.1.0"}

	af := actions.ActionsFile{Meta: meta, Projects: map[string]actions.ProjectSection{"project": proj}}

	ov := convertActionsOverride(&af, npmFakeNormalizer{})

	result := filterVulnerablePackageForOverrides(vulnPackages, ov)
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

	if overriddenPackage.Library.NormalizedName != vulnPackage.Library.NormalizedName {
		t.Fatalf("wrong library normalized name %s", overriddenPackage.Library.NormalizedName)
	}

	if overriddenPackage.Library.PackageManager != vulnPackage.Library.PackageManager {
		t.Fatalf("wrong package manager %s", overriddenPackage.Library.PackageManager)
	}

	if !reflect.DeepEqual(overriddenPackage.OpenVulnerabilities, vulnPackage.OpenVulnerabilities) {
		t.Fatalf("wrong open vulns %v; expected %v", overriddenPackage.Library.PackageManager, vulnPackage.OpenVulnerabilities)
	}

	if overriddenPackage.RecommendedLibraryVersionString != overriddenVersion {
		// should not clear it for now, since it is being used to filter out what to install later
		t.Fatalf("wrong overidden version `%s` instead of `%s`", overriddenPackage.RecommendedLibraryVersionString, overriddenVersion)
	}
}

func TestOverrideFilterNoMatchLibrary(t *testing.T) {
	vulnPackages := []api.PackageVersion{
		{
			Version:                         "1.2.3",
			Library:                         api.Package{NormalizedName: "lodash", Name: "lodash", PackageManager: mappings.NpmManager},
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

	meta := actions.MetaSection{SchemaVersion: "0.1.0", CreatedOn: actions.IsoTime{Time: actions.IsoTime{}.Time}, CliVersion: "0.1.0"}

	af := actions.ActionsFile{Meta: meta, Projects: map[string]actions.ProjectSection{"project": proj}}

	ov := convertActionsOverride(&af, npmFakeNormalizer{})

	result := filterVulnerablePackageForOverrides(vulnPackages, ov)
	if len(result) != 0 {
		t.Fatalf("wrong result length filtered %v", result)
	}
}

func TestOverrideFilterNoMatchVersion(t *testing.T) {
	vulnPackages := []api.PackageVersion{
		{
			Version:                         "1.2.3",
			Library:                         api.Package{NormalizedName: "lodash", Name: "lodash", PackageManager: mappings.NpmManager},
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

	meta := actions.MetaSection{SchemaVersion: "0.1.0", CreatedOn: actions.IsoTime{Time: actions.IsoTime{}.Time}, CliVersion: "0.1.0"}

	af := actions.ActionsFile{Meta: meta, Projects: map[string]actions.ProjectSection{"project": proj}}

	ov := convertActionsOverride(&af, npmFakeNormalizer{})

	result := filterVulnerablePackageForOverrides(vulnPackages, ov)
	if len(result) != 0 {
		t.Fatalf("wrong result length filtered %v", result)
	}
}

func TestOverrideFilterDoesNotAllowOverrideNonSealable(t *testing.T) {
	vulnPackages := []api.PackageVersion{
		{
			Version:                         "1.2.3",
			Library:                         api.Package{NormalizedName: "lodash", Name: "lodash", PackageManager: mappings.NpmManager},
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

	meta := actions.MetaSection{SchemaVersion: "0.1.0", CreatedOn: actions.IsoTime{Time: actions.IsoTime{}.Time}, CliVersion: "0.1.0"}

	af := actions.ActionsFile{Meta: meta, Projects: map[string]actions.ProjectSection{"project": proj}}

	ov := convertActionsOverride(&af, npmFakeNormalizer{})

	result := filterVulnerablePackageForOverrides(vulnPackages, ov)
	if len(result) != 0 {
		t.Fatalf("wrong result length filtered %v", result)
	}
}
func TestOverrideFilterWithNoAllowedOverrides(t *testing.T) {
	vulnPackages := []api.PackageVersion{
		{
			Version:                         "1.2.3",
			Library:                         api.Package{NormalizedName: "lodash", Name: "lodash", PackageManager: mappings.NpmManager},
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

	meta := actions.MetaSection{SchemaVersion: "0.1.0", CreatedOn: actions.IsoTime{Time: actions.IsoTime{}.Time}, CliVersion: "0.1.0"}

	af := actions.ActionsFile{Meta: meta, Projects: map[string]actions.ProjectSection{"project": proj}}

	ov := convertActionsOverride(&af, npmFakeNormalizer{})

	result := filterVulnerablePackageForOverrides(vulnPackages, ov)
	if len(result) != 0 {
		t.Fatalf("wrong result length filtered %v", result)
	}
}

func TestFixModeParsing(t *testing.T) {
	f := fixModeFromString("local")
	if f != phase.FixModeLocal {
		t.Fatalf("failed to parse local mode")
	}

	f = fixModeFromString("all")
	if f != phase.FixModeAll {
		t.Fatalf("failed to parse all mode")
	}

	f = fixModeFromString("remote")
	if f != phase.FixModeRemote {
		t.Fatalf("failed to parse remote mode")
	}

	f = fixModeFromString("fail")
	if f != "" {
		t.Fatalf("failed to parse unknown mode")
	}
}

func TestGetSilenceRules(t *testing.T) {
	tests := []struct {
		input    []string
		expected []api.SilenceRule
	}{
		{[]string{"name@version"}, []api.SilenceRule{{Library: "name", Version: "version"}}},
		{[]string{"name@version@other"}, nil},
		{[]string{"name"}, nil},
	}

	for _, test := range tests {
		rules, err := getSilenceRules(test.input)
		if test.expected == nil {
			if err == nil || rules != nil {
				t.Fatalf("failed to parse `%v`", test.input)
			}
			continue
		}
		if len(rules) != len(test.expected) {
			for i, r := range rules {
				if r != test.expected[i] {
					t.Fatalf("failed to parse `%s`", test.input)
				}
			}
		}
	}
}
