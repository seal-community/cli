//go:build !windows

package output

import (
	"cli/internal/api"
	"cli/internal/common"
	"cli/internal/ecosystem/mappings"
	"cli/internal/ecosystem/shared"
	"reflect"
	"testing"
)

func TestSummarySanity(t *testing.T) {
	projDir := "/Users/fuwawa/proj"
	descLodash := shared.DependnecyDescriptor{
		VulnerablePackage: &api.PackageVersion{
			Version:                         "1.2.3",
			Library:                         api.Package{NormalizedName: "lodash", Name: "lodash", PackageManager: mappings.NpmManager},
			RecommendedLibraryVersionId:     "123123",
			RecommendedLibraryVersionString: "1.2.3-sp1",
		},
		AvailableFix: &api.PackageVersion{
			Version:   "1.2.3-sp1",
			Library:   api.Package{NormalizedName: "lodash", Name: "lodash", PackageManager: mappings.NpmManager},
			VersionId: "123123",
		},
		Locations: map[string]common.Dependency{
			"/Users/fuwawa/proj/node_modules/lodash":                    {},
			"/Users/fuwawa/zzz/lodash":                                  {}, // using zzz to test it will be last one in sorted slice
			"/Users/fuwawa/proj/node_modules/other/node_modules/lodash": {},
		},
		FixedLocations: []string{
			"/Users/fuwawa/proj/node_modules/lodash",
			"/Users/fuwawa/proj/node_modules/other/node_modules/lodash",
			"/Users/fuwawa/zzz/lodash",
		},
	}

	descGlob := shared.DependnecyDescriptor{
		VulnerablePackage: &api.PackageVersion{
			Version:                         "3.1.0",
			Library:                         api.Package{NormalizedName: "glob-parent", Name: "glob-parent", PackageManager: mappings.NpmManager},
			RecommendedLibraryVersionId:     "1111",
			RecommendedLibraryVersionString: "3.1.0-sp1",
		},
		AvailableFix: &api.PackageVersion{
			Version:   "3.1.0-sp1",
			Library:   api.Package{NormalizedName: "glob-parent", Name: "glob-parent", PackageManager: mappings.NpmManager},
			VersionId: "1111",
		},
		Locations: map[string]common.Dependency{
			"/Users/fuwawa/proj/node_modules/glob-parent": {},
		},
		FixedLocations: []string{
			"/Users/fuwawa/proj/node_modules/glob-parent",
		},
	}

	s := NewSummary(projDir, []shared.DependnecyDescriptor{descLodash, descGlob})
	if s.Root != projDir {
		t.Fatalf("wrong project dir; expected `%s`, got `%s`", projDir, s.Root)
	}

	if len(s.Fixes) != 2 {
		t.Fatalf("wrong number of fixes; expected `%d`, got `%d`", 2, len(s.Fixes))
	}

	parsedLodash := s.Fixes[1] // results are ordered by lib name
	if !reflect.DeepEqual(parsedLodash.dep, descLodash) {
		t.Fatalf("wrong package; expected `%v`, got `%v`", descLodash.VulnerablePackage, parsedLodash.dep.AvailableFix)
	}

	locsLodash := parsedLodash.locations
	if len(locsLodash) != 3 {
		t.Fatalf("wrong number of paths got `%d`", len(locsLodash))
	}

	if locsLodash[0] != "node_modules/lodash" {
		t.Fatalf("wrong path for standard dep path; got `%s`", locsLodash[0])
	}

	if locsLodash[1] != "node_modules/other/node_modules/lodash" {
		t.Fatalf("wrong path for nested dep path; got `%s`", locsLodash[1])
	}

	if locsLodash[2] != "../zzz/lodash" {
		t.Fatalf("wrong path for outside proj dir; got `%s`", locsLodash[2])
	}

	parsedGlob := s.Fixes[0]
	if !reflect.DeepEqual(parsedGlob.dep, descGlob) {
		t.Fatalf("wrong package; expected `%v`, got `%v`", descGlob.VulnerablePackage, parsedGlob.dep.AvailableFix)
	}

	locsGlob := parsedGlob.locations
	if len(locsGlob) != 1 {
		t.Fatalf("wrong number of paths got `%d`", len(locsGlob))
	}

	if locsGlob[0] != "node_modules/glob-parent" {
		t.Fatalf("wrong path for standard dep path; got `%s`", locsGlob[0])
	}
}
