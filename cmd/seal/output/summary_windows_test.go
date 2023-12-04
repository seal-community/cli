//go:build windows

package output

import (
	"cli/internal/api"
	"cli/internal/ecosystem/shared"
	"cli/internal/phase"
	"testing"
)

func TestSummarySanity(t *testing.T) {
	projDir := `C:\fuwawa\proj`
	f1 := phase.FixedEntry{Package: &api.PackageVersion{
		Version:                         "1.2.3",
		Library:                         api.Package{Name: "lodash", PackageManager: shared.NpmManager},
		RecommendedLibraryVersionId:     "123123",
		RecommendedLibraryVersionString: "1.2.3-sp1",
	},
		Paths: map[string]bool{
			`C:\fuwawa\proj\node_modules\lodash`:                    true,
			`C:\fuwawa\proj\node_modules\other\node_modules\lodash`: true,
			`C:\fuwawa\zzz\lodash`:                                  true, // using zzz so it will be last one in sorted slice
		}}
	fixes := phase.FixMap{phase.FormatFixKey(f1.Package): &f1}

	s := NewSummary(projDir, fixes)
	if s.Root != projDir {
		t.Fatalf("wrong project dir; expected `%s`, got `%s`", projDir, s.Root)
	}

	if len(s.Fixes) != 1 {
		t.Fatalf("wrong number of fixes; expected `%d`, got `%d`", 1, len(s.Fixes))
	}

	parsedFix1 := s.Fixes[0]
	if parsedFix1.pkg != f1.Package {
		t.Fatalf("wrong package; expected `%v`, got `%v`", f1.Package, parsedFix1.pkg)
	}

	locs := parsedFix1.locations
	if locs[0] != `node_modules\lodash` {
		t.Fatalf("wrong path for standard dep path; got `%s`", locs[0])
	}

	if locs[1] != `node_modules\other\node_modules\lodash` {
		t.Fatalf("wrong path for nested dep path; got `%s`", locs[1])
	}

	if locs[2] != `..\zzz\lodash` {
		t.Fatalf("wrong path for outside proj dir; got `%s`", locs[2])
	}
}
