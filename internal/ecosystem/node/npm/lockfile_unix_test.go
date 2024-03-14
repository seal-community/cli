//go:build !windows

package npm

import (
	"bytes"
	"cli/internal/api"
	"cli/internal/ecosystem/mappings"
	"cli/internal/ecosystem/shared"
	"path/filepath"
	"testing"
)

// only formatting tests

func TestLockfileUpdateV1Formatting(t *testing.T) {
	lock := _loadf("5.10.0.nested.package-lock.json")
	expectedJsonAfter := string(_readf("5.10.0.nested.after.package-lock.json"))
	minimist1 := api.PackageVersion{
		Version:                         "0.0.8",
		Library:                         api.Package{Name: "minimist", PackageManager: mappings.NpmManager},
		RecommendedLibraryVersionId:     "111",
		RecommendedLibraryVersionString: "0.0.8-sp1",
	}
	minimist2 := api.PackageVersion{
		Version:                         "1.2.0",
		Library:                         api.Package{Name: "minimist", PackageManager: mappings.NpmManager},
		RecommendedLibraryVersionId:     "111",
		RecommendedLibraryVersionString: "1.2.0-sp1",
	}

	lodash := api.PackageVersion{
		Version:                         "4.17.11",
		Library:                         api.Package{Name: "lodash", PackageManager: mappings.NpmManager},
		RecommendedLibraryVersionId:     "111",
		RecommendedLibraryVersionString: "4.17.11-sp1",
	}

	projectDir := "/prj"

	fixmap := shared.FixMap{
		"a": &shared.FixedEntry{Package: &minimist1, Paths: map[string]bool{
			filepath.Join(projectDir, "node_modules/mkdirp/node_modules/minimist"): true,
		}},
		"b": &shared.FixedEntry{Package: &minimist2, Paths: map[string]bool{
			filepath.Join(projectDir, "node_modules/minimist"): true,
		}},
		"c": &shared.FixedEntry{Package: &lodash, Paths: map[string]bool{
			filepath.Join(projectDir, "node_modules/lodash"): true,
		}},
	}

	if err := UpdateLockfile(lock, fixmap, projectDir); err != nil {
		t.Fatalf("failed updating: %v", err)
	}

	w := bytes.NewBufferString("")
	if err := dumpLockfile(lock, w); err != nil {
		t.Fatalf("failed dumping: %v", err)
	}

	result := w.String()

	if result != expectedJsonAfter {
		t.Fatal("different json than expected after patching")
	}
}

func TestLockfileUpdateV2Formatting(t *testing.T) {
	lock := _loadf("7.24.2.complex.package-lock.json")
	expectedJsonAfter := string(_readf("7.24.2.complex.after.package-lock.json"))
	merge := api.PackageVersion{
		Version:                         "1.2.1",
		Library:                         api.Package{Name: "merge", PackageManager: mappings.NpmManager},
		RecommendedLibraryVersionId:     "111",
		RecommendedLibraryVersionString: "1.2.1-sp1",
	}
	minimist := api.PackageVersion{
		Version:                         "1.2.0",
		Library:                         api.Package{Name: "minimist", PackageManager: mappings.NpmManager},
		RecommendedLibraryVersionId:     "111",
		RecommendedLibraryVersionString: "1.2.0-sp1",
	}

	lodash := api.PackageVersion{
		Version:                         "4.17.11",
		Library:                         api.Package{Name: "lodash", PackageManager: mappings.NpmManager},
		RecommendedLibraryVersionId:     "111",
		RecommendedLibraryVersionString: "4.17.11-sp1",
	}

	projectDir := "/prj"
	fixmap := shared.FixMap{
		"a": &shared.FixedEntry{Package: &merge, Paths: map[string]bool{
			filepath.Join(projectDir, "node_modules/merge"): true,
		}},
		"b": &shared.FixedEntry{Package: &minimist, Paths: map[string]bool{
			filepath.Join(projectDir, "node_modules/minimist"): true,
		}},
		"c": &shared.FixedEntry{Package: &lodash, Paths: map[string]bool{
			filepath.Join(projectDir, "node_modules/lodash"): true,
		}},
	}
	if err := UpdateLockfile(lock, fixmap, projectDir); err != nil {
		t.Fatalf("failed updating: %v", err)
	}

	w := bytes.NewBufferString("")
	if err := dumpLockfile(lock, w); err != nil {
		t.Fatalf("failed dumping: %v", err)
	}

	result := w.String()

	if result != expectedJsonAfter {
		t.Fatal("different json than expected after patching")
	}
}

func TestLockfileUpdateV3Formatting(t *testing.T) {
	lock := _loadf("10.1.0.multi-versions.package-lock.json")
	expectedJsonAfter := string(_readf("10.1.0.multi-versions.after.package-lock.json"))
	merge := api.PackageVersion{
		Version:                         "1.2.1",
		Library:                         api.Package{Name: "merge", PackageManager: mappings.NpmManager},
		RecommendedLibraryVersionId:     "111",
		RecommendedLibraryVersionString: "1.2.1-sp1",
	}
	minimist := api.PackageVersion{
		Version:                         "1.2.0",
		Library:                         api.Package{Name: "minimist", PackageManager: mappings.NpmManager},
		RecommendedLibraryVersionId:     "111",
		RecommendedLibraryVersionString: "1.2.0-sp1",
	}
	minimist2 := api.PackageVersion{
		Version:                         "0.0.8",
		Library:                         api.Package{Name: "minimist", PackageManager: mappings.NpmManager},
		RecommendedLibraryVersionId:     "111",
		RecommendedLibraryVersionString: "0.0.8-sp1",
	}

	lodash := api.PackageVersion{
		Version:                         "4.17.11",
		Library:                         api.Package{Name: "lodash", PackageManager: mappings.NpmManager},
		RecommendedLibraryVersionId:     "111",
		RecommendedLibraryVersionString: "4.17.11-sp1",
	}

	lodash2 := api.PackageVersion{
		Version:                         "4.17.5",
		Library:                         api.Package{Name: "lodash", PackageManager: mappings.NpmManager},
		RecommendedLibraryVersionId:     "111",
		RecommendedLibraryVersionString: "4.17.5-sp1",
	}

	projectDir := "/prj"
	fixmap := shared.FixMap{
		"a": &shared.FixedEntry{Package: &merge, Paths: map[string]bool{
			filepath.Join(projectDir, "node_modules/merge"): true,
		}},
		"b": &shared.FixedEntry{Package: &minimist, Paths: map[string]bool{
			filepath.Join(projectDir, "node_modules/minimist"): true,
		}},
		"bb": &shared.FixedEntry{Package: &minimist2, Paths: map[string]bool{
			filepath.Join(projectDir, "node_modules/mkdirp/node_modules/minimist"): true,
		}},
		"c": &shared.FixedEntry{Package: &lodash, Paths: map[string]bool{
			filepath.Join(projectDir, "node_modules/commitizen/node_modules/lodash"): true,
			filepath.Join(projectDir, "node_modules/cypress/node_modules/lodash"):    true,
		}},
		"d": &shared.FixedEntry{Package: &lodash2, Paths: map[string]bool{
			filepath.Join(projectDir, "node_modules/lodash"): true,
		}},
	}

	if err := UpdateLockfile(lock, fixmap, projectDir); err != nil {
		t.Fatalf("failed updating: %v", err)
	}

	w := bytes.NewBufferString("")
	if err := dumpLockfile(lock, w); err != nil {
		t.Fatalf("failed dumping: %v", err)
	}

	result := w.String()
	if result != expectedJsonAfter {
		t.Fatal("different json than expected after patching")
	}
}
