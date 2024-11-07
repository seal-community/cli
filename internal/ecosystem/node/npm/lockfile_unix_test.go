//go:build !windows

package npm

import (
	"bytes"
	"cli/internal/api"
	"cli/internal/common"
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
		Library:                         api.Package{NormalizedName: "minimist", Name: "minimist", PackageManager: mappings.NpmManager},
		RecommendedLibraryVersionId:     "111",
		RecommendedLibraryVersionString: "0.0.8-sp1",
	}

	minimist2 := api.PackageVersion{
		Version:                         "1.2.0",
		Library:                         api.Package{NormalizedName: "minimist", Name: "minimist", PackageManager: mappings.NpmManager},
		RecommendedLibraryVersionId:     "111",
		RecommendedLibraryVersionString: "1.2.0-sp1",
	}

	lodash := api.PackageVersion{
		Version:                         "4.17.11",
		Library:                         api.Package{NormalizedName: "lodash", Name: "lodash", PackageManager: mappings.NpmManager},
		RecommendedLibraryVersionId:     "111",
		RecommendedLibraryVersionString: "4.17.11-sp1",
	}

	projectDir := "/prj"
	fixes := []shared.DependencyDescriptor{
		{
			VulnerablePackage: &minimist1,
			FixedLocations:    []string{filepath.Join(projectDir, "node_modules/mkdirp/node_modules/minimist")},
		},
		{
			VulnerablePackage: &minimist2,
			FixedLocations:    []string{filepath.Join(projectDir, "node_modules/minimist")},
		},
		{
			VulnerablePackage: &lodash,
			FixedLocations:    []string{filepath.Join(projectDir, "node_modules/lodash")},
		},
	}

	if err := UpdateLockfile(lock, fixes, projectDir); err != nil {
		t.Fatalf("failed updating: %v", err)
	}

	w := bytes.NewBufferString("")
	if err := common.JsonDump(lock, w); err != nil {
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
		Library:                         api.Package{NormalizedName: "merge", Name: "merge", PackageManager: mappings.NpmManager},
		RecommendedLibraryVersionId:     "111",
		RecommendedLibraryVersionString: "1.2.1-sp1",
	}

	minimist := api.PackageVersion{
		Version:                         "1.2.0",
		Library:                         api.Package{NormalizedName: "minimist", Name: "minimist", PackageManager: mappings.NpmManager},
		RecommendedLibraryVersionId:     "111",
		RecommendedLibraryVersionString: "1.2.0-sp1",
	}

	lodash := api.PackageVersion{
		Version:                         "4.17.11",
		Library:                         api.Package{NormalizedName: "lodash", Name: "lodash", PackageManager: mappings.NpmManager},
		RecommendedLibraryVersionId:     "111",
		RecommendedLibraryVersionString: "4.17.11-sp1",
	}

	projectDir := "/prj"
	fixes := []shared.DependencyDescriptor{
		{
			VulnerablePackage: &merge,
			FixedLocations:    []string{filepath.Join(projectDir, "node_modules/merge")},
		},
		{
			VulnerablePackage: &minimist,
			FixedLocations:    []string{filepath.Join(projectDir, "node_modules/minimist")},
		},
		{
			VulnerablePackage: &lodash,
			FixedLocations:    []string{filepath.Join(projectDir, "node_modules/lodash")},
		},
	}

	if err := UpdateLockfile(lock, fixes, projectDir); err != nil {
		t.Fatalf("failed updating: %v", err)
	}

	w := bytes.NewBufferString("")
	if err := common.JsonDump(lock, w); err != nil {
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
		Library:                         api.Package{NormalizedName: "merge", Name: "merge", PackageManager: mappings.NpmManager},
		RecommendedLibraryVersionId:     "111",
		RecommendedLibraryVersionString: "1.2.1-sp1",
	}

	minimist := api.PackageVersion{
		Version:                         "1.2.0",
		Library:                         api.Package{NormalizedName: "minimist", Name: "minimist", PackageManager: mappings.NpmManager},
		RecommendedLibraryVersionId:     "111",
		RecommendedLibraryVersionString: "1.2.0-sp1",
	}

	minimist2 := api.PackageVersion{
		Version:                         "0.0.8",
		Library:                         api.Package{NormalizedName: "minimist", Name: "minimist", PackageManager: mappings.NpmManager},
		RecommendedLibraryVersionId:     "111",
		RecommendedLibraryVersionString: "0.0.8-sp1",
	}

	lodash := api.PackageVersion{
		Version:                         "4.17.11",
		Library:                         api.Package{NormalizedName: "lodash", Name: "lodash", PackageManager: mappings.NpmManager},
		RecommendedLibraryVersionId:     "111",
		RecommendedLibraryVersionString: "4.17.11-sp1",
	}

	lodash2 := api.PackageVersion{
		Version:                         "4.17.5",
		Library:                         api.Package{NormalizedName: "lodash", Name: "lodash", PackageManager: mappings.NpmManager},
		RecommendedLibraryVersionId:     "111",
		RecommendedLibraryVersionString: "4.17.5-sp1",
	}

	projectDir := "/prj"
	fixes := []shared.DependencyDescriptor{
		{
			VulnerablePackage: &merge,
			FixedLocations:    []string{filepath.Join(projectDir, "node_modules/merge")},
		},
		{
			VulnerablePackage: &minimist,
			FixedLocations:    []string{filepath.Join(projectDir, "node_modules/minimist")},
		},
		{
			VulnerablePackage: &minimist2,
			FixedLocations:    []string{filepath.Join(projectDir, "node_modules/mkdirp/node_modules/minimist")},
		},
		{
			VulnerablePackage: &lodash,
			FixedLocations: []string{
				filepath.Join(projectDir, "node_modules/commitizen/node_modules/lodash"),
				filepath.Join(projectDir, "node_modules/cypress/node_modules/lodash"),
			},
		},
		{
			VulnerablePackage: &lodash2,
			FixedLocations: []string{
				filepath.Join(projectDir, "node_modules/lodash"),
			},
		},
	}

	if err := UpdateLockfile(lock, fixes, projectDir); err != nil {
		t.Fatalf("failed updating: %v", err)
	}

	w := bytes.NewBufferString("")
	if err := common.JsonDump(lock, w); err != nil {
		t.Fatalf("failed dumping: %v", err)
	}

	result := w.String()
	if result != expectedJsonAfter {
		t.Fatal("different json than expected after patching")
	}
}
