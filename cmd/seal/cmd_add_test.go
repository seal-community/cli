package main

import (
	"cli/internal/api"
	"cli/internal/ecosystem/mappings"
	"cli/internal/phase"
	"testing"
)

func TestFindRuleSanity(t *testing.T) {
	packages := []api.PackageVersion{
		{
			Version:                         "1.2.3",
			Library:                         api.Package{Name: "lodash", PackageManager: mappings.NpmManager},
			RecommendedLibraryVersionId:     "111",
			RecommendedLibraryVersionString: "1.2.3-sp1",
		},
		{
			Version:                         "1.0.0",
			Library:                         api.Package{Name: "semver-regex", PackageManager: mappings.NpmManager},
			RecommendedLibraryVersionId:     "123123",
			RecommendedLibraryVersionString: "1.0.0-sp1",
		},
	}

	toFind := api.PackageVersion{

		Version:                         "1.2.3",
		Library:                         api.Package{Name: "lodash", PackageManager: mappings.NpmManager},
		RecommendedLibraryVersionString: "1.2.3-sp1",
	}

	foundPackage := findRule(toFind, packages)
	if foundPackage == nil {
		t.Fatalf("did not find package")
	}

	if foundPackage.Library.Name != toFind.Library.Name {
		t.Fatalf("found wrong package, got %s expected %s", foundPackage.Library.Name, toFind.Library.Name)
	}

	if foundPackage.Library.PackageManager != toFind.Library.PackageManager {
		t.Fatalf("found wrong package, got %s expected %s", foundPackage.Library.PackageManager, toFind.Library.PackageManager)
	}

	if foundPackage.Version != toFind.Version {
		t.Fatalf("found wrong package, got %s expected %s", foundPackage.Version, toFind.Version)
	}
}

func TestFindRuleNotFoundVersion(t *testing.T) {
	packages := []api.PackageVersion{
		{
			Version:                         "1.2.3",
			Library:                         api.Package{Name: "lodash", PackageManager: mappings.NpmManager},
			RecommendedLibraryVersionId:     "111",
			RecommendedLibraryVersionString: "1.2.3-sp1",
		},
		{
			Version:                         "1.0.0",
			Library:                         api.Package{Name: "semver-regex", PackageManager: mappings.NpmManager},
			RecommendedLibraryVersionId:     "123123",
			RecommendedLibraryVersionString: "1.0.0-sp1",
		},
	}

	toFind := api.PackageVersion{

		Version:                         "1.2.3-sp1",
		Library:                         api.Package{Name: "lodash", PackageManager: mappings.NpmManager},
		RecommendedLibraryVersionString: "1.2.3-sp2",
	}

	foundPackage := findRule(toFind, packages)
	if foundPackage != nil {
		t.Fatalf("should not find package %v", foundPackage)
	}
}

func TestFindRuleNotFoundLibrary(t *testing.T) {
	packages := []api.PackageVersion{
		{
			Version:                         "1.2.3",
			Library:                         api.Package{Name: "lodash", PackageManager: mappings.NpmManager},
			RecommendedLibraryVersionId:     "111",
			RecommendedLibraryVersionString: "1.2.3-sp1",
		},
		{
			Version:                         "1.0.0",
			Library:                         api.Package{Name: "semver-regex", PackageManager: mappings.NpmManager},
			RecommendedLibraryVersionId:     "123123",
			RecommendedLibraryVersionString: "1.0.0-sp1",
		},
	}

	toFind := api.PackageVersion{

		Version:                         "1.2.3",
		Library:                         api.Package{Name: "bloop", PackageManager: mappings.NpmManager},
		RecommendedLibraryVersionString: "1.2.3-sp1",
	}

	foundPackage := findRule(toFind, packages)
	if foundPackage != nil {
		t.Fatalf("should not find package %v", foundPackage)
	}
}

func TestFindRuleNotFoundEmpty(t *testing.T) {
	packages := []api.PackageVersion{}

	toFind := api.PackageVersion{

		Version:                         "1.2.3",
		Library:                         api.Package{Name: "bloop", PackageManager: mappings.NpmManager},
		RecommendedLibraryVersionString: "1.2.3-sp1",
	}

	foundPackage := findRule(toFind, packages)
	if foundPackage != nil {
		t.Fatalf("should not find package %v", foundPackage)
	}
}

func TestUpsertAddedNew(t *testing.T) {
	existing := []api.PackageVersion{
		{
			Version:                         "1.2.3",
			Library:                         api.Package{Name: "lodash", PackageManager: mappings.NpmManager},
			RecommendedLibraryVersionId:     "111",
			RecommendedLibraryVersionString: "1.2.3-sp1",
		},
		{
			Version:                         "1.0.0",
			Library:                         api.Package{Name: "semver-regex", PackageManager: mappings.NpmManager},
			RecommendedLibraryVersionId:     "123123",
			RecommendedLibraryVersionString: "1.0.0-sp1",
		},
	}

	resolved := phase.ResolvedRule{
		From: api.PackageVersion{
			Version:                         "2.7.4",
			Library:                         api.Package{Name: "ejs", PackageManager: mappings.NpmManager},
			RecommendedLibraryVersionString: "2.7.4-sp1",
		},
		To: &api.PackageVersion{
			Version:                         "2.7.4-sp1",
			Library:                         api.Package{Name: "ejs", PackageManager: mappings.NpmManager},
			RecommendedLibraryVersionString: "",
		},
	}

	_, modified, found := upsertRule(resolved, &existing) // must have To set, checked in caller
	if found || !modified || len(existing) != 3 {
		t.Fatalf("did not add new rule")
	}
}

func TestUpsertFoundExact(t *testing.T) {
	existing := []api.PackageVersion{
		{
			Version:                         "1.2.3",
			Library:                         api.Package{Name: "lodash", PackageManager: mappings.NpmManager},
			RecommendedLibraryVersionString: "1.2.3-sp1",
		},
		{
			Version:                         "2.7.4",
			Library:                         api.Package{Name: "ejs", PackageManager: mappings.NpmManager},
			RecommendedLibraryVersionString: "2.7.4-sp1",
		},
	}

	resolved := phase.ResolvedRule{
		From: api.PackageVersion{
			Version:                         "2.7.4",
			Library:                         api.Package{Name: "ejs", PackageManager: mappings.NpmManager},
			RecommendedLibraryVersionString: "2.7.4-sp1",
		},
		To: &api.PackageVersion{
			Version:                         "2.7.4-sp1",
			Library:                         api.Package{Name: "ejs", PackageManager: mappings.NpmManager},
			RecommendedLibraryVersionString: "",
		},
	}

	_, modified, found := upsertRule(resolved, &existing) // must have To set, checked in caller
	if !found || modified || len(existing) != 2 {
		t.Fatalf("added rule")
	}
}

func TestUpsertModifiedNotExact(t *testing.T) {
	existing := []api.PackageVersion{
		{
			Version:                         "1.2.3",
			Library:                         api.Package{Name: "lodash", PackageManager: mappings.NpmManager},
			RecommendedLibraryVersionString: "1.2.3-sp1",
		},
		{
			Version:                         "2.7.4",
			Library:                         api.Package{Name: "ejs", PackageManager: mappings.NpmManager},
			RecommendedLibraryVersionString: "2.7.4-sp1",
		},
	}

	resolved := phase.ResolvedRule{
		From: api.PackageVersion{
			Version:                         "2.7.4",
			Library:                         api.Package{Name: "ejs", PackageManager: mappings.NpmManager},
			RecommendedLibraryVersionString: "2.7.4-sp2",
			RecommendedLibraryVersionId:     "recommended-id",
		},
		To: &api.PackageVersion{
			VersionId:                       "recommended-id",
			Version:                         "2.7.4-sp2",
			Library:                         api.Package{Name: "ejs", PackageManager: mappings.NpmManager},
			RecommendedLibraryVersionString: "",
		},
	}

	_, modified, found := upsertRule(resolved, &existing) // must have To set, checked in caller
	if !modified || !found || len(existing) != 2 {
		t.Fatalf("did not find similar rule")
	}
	if existing[1].RecommendedLibraryVersionString != "2.7.4-sp2" {
		t.Fatal("did not update version string")
	}
	if existing[1].RecommendedLibraryVersionId != "recommended-id" {
		t.Fatal("did not update version id")
	}
}

func TestParseRuleArgsEmpty(t *testing.T) {
	r, err := parseRule([]string{})
	if err == nil || r != nil {
		t.Fatalf("failed detecting empty input")
	}
}

func TestParseRuleArgsMissingOne(t *testing.T) {
	r, err := parseRule([]string{"ejs"})
	if err == nil || r != nil {
		t.Fatalf("failed detecting empty input")
	}
}

func TestParseRuleArgsTooMany(t *testing.T) {
	r, err := parseRule([]string{"ejs", "1.2", "ejs2"})
	if err == nil || r != nil {
		t.Fatalf("failed detecting empty input")
	}
}
func TestParseRuleArgsSanity(t *testing.T) {
	r, err := parseRule([]string{"ejs", "1.2"})
	if err != nil || r == nil {
		t.Fatalf("failed parsing input")
	}

	if r.From.Library != "ejs" {
		t.Fatalf("bad parsing, got %s", r.From.Library)
	}
	if r.From.Version != "1.2" {
		t.Fatalf("bad parsing, got %s", r.From.Version)
	}

	if r.To == nil {
		t.Fatalf("wrong initial version for To")
	}

	if r.To.Library != "" || r.To.Version != "" {
		t.Fatalf("bad initial values for To, got %v", r.To)
	}
}
