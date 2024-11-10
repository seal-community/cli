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
	descLodash := shared.DependencyDescriptor{
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

	descGlob := shared.DependencyDescriptor{
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

	s := NewSummary(projDir, []shared.DependencyDescriptor{descLodash, descGlob}, map[string][]string{"ejs@1.2.3": {"/Users/fuwawa/proj/node_modules/ejs"}})
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

	if len(s.Silenced) != 1 {
		t.Fatalf("wrong number of silenced; got `%d`", len(s.Silenced))
	}

	if s.Silenced[0].descriptor != "ejs@1.2.3" {
		t.Fatalf("wrong silenced descriptor; got `%s`", s.Silenced[0].descriptor)
	}

	if len(s.Silenced[0].locations) != 1 {
		t.Fatalf("wrong number of silenced paths; got `%d`", len(s.Silenced[0].locations))
	}

	if s.Silenced[0].locations[0] != "node_modules/ejs" {
		t.Fatalf("wrong path for silenced dep path; got `%s`", s.Silenced[0].locations[0])
	}
}

func TestGetRemediatedCVEs(t *testing.T) {
	tests := []struct {
		name        string
		input       summaryFix
		expectedIds []string
	}{
		{
			name: "Successful remediation",
			input: summaryFix{
				dep: shared.DependencyDescriptor{
					VulnerablePackage: &api.PackageVersion{
						OpenVulnerabilities: []api.Vulnerability{
							{SnykID: "Sneaky123"},
							{CVE: "CVE-2023-0002"},
						},
					},
					AvailableFix: &api.PackageVersion{
						SealedVulnerabilities: []api.Vulnerability{
							{SnykID: "Sneaky123"},
						},
					},
				},
			},
			expectedIds: []string{"Sneaky123"},
		},
		{
			name: "No remediation available",
			input: summaryFix{
				dep: shared.DependencyDescriptor{
					VulnerablePackage: &api.PackageVersion{
						OpenVulnerabilities: []api.Vulnerability{
							{CVE: "CVE-2023-0001"},
						},
					},
					AvailableFix: nil,
				},
			},
			expectedIds: []string{},
		},
		{
			name: "No vulnerable package",
			input: summaryFix{
				dep: shared.DependencyDescriptor{
					VulnerablePackage: nil,
					AvailableFix: &api.PackageVersion{
						SealedVulnerabilities: []api.Vulnerability{
							{CVE: "CVE-2023-0001"},
						},
					},
				},
			},
			expectedIds: []string{},
		},
		{
			name: "No matching CVEs between vulnerable and sealed",
			input: summaryFix{
				dep: shared.DependencyDescriptor{
					VulnerablePackage: &api.PackageVersion{
						OpenVulnerabilities: []api.Vulnerability{
							{CVE: "CVE-2023-0001"},
						},
					},
					AvailableFix: &api.PackageVersion{
						SealedVulnerabilities: []api.Vulnerability{
							{CVE: "CVE-2023-0002"},
						},
					},
				},
			},
			expectedIds: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			remediatedVulnerabilityIds := GetRemediatedVulnerabilityIds(tt.input)
			if !(len(tt.expectedIds) == 0 && len(remediatedVulnerabilityIds) == 0) && !reflect.DeepEqual(tt.expectedIds, remediatedVulnerabilityIds) {
				t.Fatalf("remediatedCVEs output: %s is not the same as expected: %s", remediatedVulnerabilityIds, tt.expectedIds)
			}
		})
	}
}
