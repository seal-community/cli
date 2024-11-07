package npm

import (
	"cli/internal/api"
	"cli/internal/ecosystem/mappings"
	"cli/internal/ecosystem/shared"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/iancoleman/orderedmap"
)

func getNested(om *orderedmap.OrderedMap, membs ...string) orderedmap.OrderedMap {
	itr := *om
	for _, m := range membs {
		o, exists := itr.Get(m)
		if !exists {
			panic(fmt.Sprintf("%s does not exist in ordered map", m))
		}
		itr = o.(orderedmap.OrderedMap)
	}

	return itr
}

func _get[T any](om orderedmap.OrderedMap, m string) T {
	v, exists := om.Get(m)
	if !exists {
		panic("does not exist")
	}

	return v.(T)
}

func _load(s string) *orderedmap.OrderedMap {
	var d orderedmap.OrderedMap
	if err := json.Unmarshal([]byte(s), &d); err != nil {
		fmt.Printf("err: %v", err)
		panic(err)
	}

	return &d
}
func _readf(filename string) []byte {
	p := filepath.Join("testdata/lockfiles", filename)
	data, err := os.ReadFile(p)
	if err != nil {
		panic(err)
	}

	return data
}

func _loadf(filename string) *orderedmap.OrderedMap {
	p := filepath.Join("testdata/lockfiles", filename)
	data, err := os.ReadFile(p)
	if err != nil {
		panic(err)
	}

	return _load(string(data))
}

func TestLockfileVersionDetectionV1(t *testing.T) {
	wanted := lockV1
	lock := `{ "lockfileVersion": 1 }`

	if ver := getLockfileVersion(_load(lock)); ver != wanted {
		t.Fatalf("got %v, expected %v", ver, wanted)
	}
}

func TestLockfileVersionDetectionV2(t *testing.T) {
	wanted := lockV2
	lock := `{
		"lockfileVersion": 2
	}`

	if ver := getLockfileVersion(_load(lock)); ver != wanted {
		t.Fatalf("got %v, expected %v", ver, wanted)
	}
}

func TestLockfileVersionDetectionV3(t *testing.T) {
	wanted := lockV3
	lock := `{
		"lockfileVersion": 3
	}`

	if ver := getLockfileVersion(_load(lock)); ver != wanted {
		t.Fatalf("got %v, expected %v", ver, wanted)
	}
}

func TestLockfileVersionDetectionNonExistant(t *testing.T) {
	wanted := lockOld
	lock := `{}`

	if ver := getLockfileVersion(_load(lock)); ver != wanted {
		t.Fatalf("got %v, expected %v", ver, wanted)
	}
}

func TestLockfileVersionDetectionBadType(t *testing.T) {
	wanted := lockBadVersion
	lock := `{
		"lockfileVersion": "aaa"
	}`
	if ver := getLockfileVersion(_load(lock)); ver != wanted {
		t.Fatalf("got %v, expected %v", ver, wanted)
	}
}

func TestLockfileVersionDetectionBadTypeInt(t *testing.T) {
	wanted := lockBadVersion
	lock := `{
		"lockfileVersion": "1"
	}`
	if ver := getLockfileVersion(_load(lock)); ver != wanted {
		t.Fatalf("got %v, expected %v", ver, wanted)
	}
}

func TestLockfileVersionDetectionWrongValue(t *testing.T) {
	wanted := lockBadVersion
	lock := `{
		"lockfileVersion": 123
	}`

	if ver := getLockfileVersion(_load(lock)); ver != wanted {
		t.Fatalf("got %v, expected %v", ver, wanted)
	}
}

func TestLockfileVersionDetectionWrongZero(t *testing.T) {
	wanted := lockBadVersion
	lock := `{
		"lockfileVersion": 0
	}`

	if ver := getLockfileVersion(_load(lock)); ver != wanted {
		t.Fatalf("got %v, expected %v", ver, wanted)
	}
}

func TestLockfileVersionDetectionWrongMax(t *testing.T) {
	wanted := lockBadVersion
	lock := fmt.Sprintf(`{
		"lockfileVersion": %d
	}`, lockSupportedVersion)

	if ver := getLockfileVersion(_load(lock)); ver != wanted {
		t.Fatalf("got %v, expected %v", ver, wanted)
	}
}

func TestUpdatedNameFormat(t *testing.T) {
	for original, expected := range map[string]string{
		// v1 format
		"pkg":        "seal-pkg",
		"@owner/pkg": "@owner/seal-pkg",
		// v2 format
		"node_modules/pkg":                               "node_modules/seal-pkg", // for newer version that use relative path
		"node_modules/@fastify/multipart":                "node_modules/@fastify/seal-multipart",
		"node_modules/base/node_modules/define-property": "node_modules/base/node_modules/seal-define-property", // nested
	} {
		t.Run(fmt.Sprintf("updated_name_%s", original), func(t *testing.T) {
			if result := formatUpdatedPackageName(original); result != expected {
				t.Fatalf("got %s instead of %s for %s", result, expected, original)
			}

		})

	}
}

func TestLockfileUpdateV1(t *testing.T) {
	lock := _loadf("5.10.0.package-lock.json")
	pkg := api.PackageVersion{
		Version:                         "1.0.0",
		Library:                         api.Package{NormalizedName: "semver-regex", Name: "semver-regex", PackageManager: mappings.NpmManager},
		RecommendedLibraryVersionId:     "111",
		RecommendedLibraryVersionString: "1.0.0-sp1",
	}

	projectDir := "/prj"
	fixmap := map[string]*api.PackageVersion{
		filepath.Join(projectDir, "node_modules", pkg.Library.Name): &pkg,
	}

	if err := updateV1(lock, fixmap, projectDir); err != nil {
		t.Fatalf("failed updating: %v", err)
	}

	oldName := pkg.Library.Name
	newName := formatUpdatedPackageName(oldName)

	depMap := getNested(lock, "dependencies")
	if _, exists := depMap.Get(oldName); exists {
		t.Fatalf("found old name %s after update", oldName)
	}

	dep := getNested(lock, "dependencies", newName)
	ver := _get[string](dep, "version")

	if ver != pkg.RecommendedLibraryVersionString {
		t.Fatalf("bad version: %v", ver)
	}
}

func TestLockfileUpdateV1MultipleLocations(t *testing.T) {
	lock := _loadf("5.10.0.multi-versions.package-lock.json")
	pkg := api.PackageVersion{
		Version:                         "4.17.11",
		Library:                         api.Package{NormalizedName: "lodash", Name: "lodash", PackageManager: mappings.NpmManager},
		RecommendedLibraryVersionId:     "111",
		RecommendedLibraryVersionString: "4.17.11-sp1",
	}

	projectDir := "/prj"
	fixmap := map[string]*api.PackageVersion{
		filepath.Join(projectDir, "node_modules/commitizen/node_modules/lodash"): &pkg,
		filepath.Join(projectDir, "node_modules/cypress/node_modules/lodash"):    &pkg,
	}

	if err := updateV1(lock, fixmap, projectDir); err != nil {
		t.Fatalf("failed updating: %v", err)
	}

	lodash1 := getNested(lock, "dependencies", "commitizen", "dependencies", "seal-lodash")
	lodash2 := getNested(lock, "dependencies", "cypress", "dependencies", "seal-lodash")

	if lodash1_ver := _get[string](lodash1, "version"); lodash1_ver != pkg.RecommendedLibraryVersionString {
		t.Fatalf("bad version: %v", lodash1_ver)
	}

	if lodash2_ver := _get[string](lodash2, "version"); lodash2_ver != pkg.RecommendedLibraryVersionString {
		t.Fatalf("bad version: %v", lodash2_ver)
	}
}

func TestLockfileUpdateV2MultipleLocations(t *testing.T) {
	lock := _loadf("7.24.2.multi-versions.package-lock.json")
	pkg := api.PackageVersion{
		Version:                         "4.17.11",
		Library:                         api.Package{NormalizedName: "lodash", Name: "lodash", PackageManager: mappings.NpmManager},
		RecommendedLibraryVersionId:     "111",
		RecommendedLibraryVersionString: "4.17.11-sp1",
	}

	projectDir := "/prj"

	fixes := []shared.DependencyDescriptor{
		{VulnerablePackage: &pkg,
			FixedLocations: []string{
				filepath.Join(projectDir, "node_modules/commitizen/node_modules/lodash"),
				filepath.Join(projectDir, "node_modules/cypress/node_modules/lodash"),
			},
		},
	}

	if err := UpdateLockfile(lock, fixes, projectDir); err != nil {
		t.Fatalf("failed updating: %v", err)
	}

	// backwards compatible structure
	lodash1 := getNested(lock, "dependencies", "commitizen", "dependencies", "seal-lodash")
	lodash2 := getNested(lock, "dependencies", "cypress", "dependencies", "seal-lodash")
	if lodash1_ver := _get[string](lodash1, "version"); lodash1_ver != pkg.RecommendedLibraryVersionString {
		t.Fatalf("bad version: %v", lodash1_ver)
	}

	if lodash2_ver := _get[string](lodash2, "version"); lodash2_ver != pkg.RecommendedLibraryVersionString {
		t.Fatalf("bad version: %v", lodash2_ver)
	}

	// new structure
	lodash1_new := getNested(lock, "packages", "node_modules/commitizen/node_modules/seal-lodash")
	lodash2_new := getNested(lock, "packages", "node_modules/cypress/node_modules/seal-lodash")

	if lodash1_new_ver := _get[string](lodash1_new, "version"); lodash1_new_ver != pkg.RecommendedLibraryVersionString {
		t.Fatalf("bad version: %v", lodash1_new_ver)
	}

	if lodash2_new_ver := _get[string](lodash2_new, "version"); lodash2_new_ver != pkg.RecommendedLibraryVersionString {
		t.Fatalf("bad version: %v", lodash2_new_ver)
	}
}
func TestLockfileUpdateV3MultipleLocations(t *testing.T) {
	lock := _loadf("10.1.0.multi-versions.package-lock.json")
	pkg := api.PackageVersion{
		Version:                         "4.17.11",
		Library:                         api.Package{NormalizedName: "lodash", Name: "lodash", PackageManager: mappings.NpmManager},
		RecommendedLibraryVersionId:     "111",
		RecommendedLibraryVersionString: "4.17.11-sp1",
	}

	projectDir := "/prj"

	fixes := []shared.DependencyDescriptor{
		{VulnerablePackage: &pkg,
			FixedLocations: []string{
				filepath.Join(projectDir, "node_modules/commitizen/node_modules/lodash"),
				filepath.Join(projectDir, "node_modules/cypress/node_modules/lodash"),
			},
		},
	}

	if err := UpdateLockfile(lock, fixes, projectDir); err != nil {
		t.Fatalf("failed updating: %v", err)
	}

	// only has new structure
	lodash1_new := getNested(lock, "packages", "node_modules/commitizen/node_modules/seal-lodash")
	lodash2_new := getNested(lock, "packages", "node_modules/cypress/node_modules/seal-lodash")

	if lodash1_new_ver := _get[string](lodash1_new, "version"); lodash1_new_ver != pkg.RecommendedLibraryVersionString {
		t.Fatalf("bad version: %v", lodash1_new_ver)
	}

	if lodash2_new_ver := _get[string](lodash2_new, "version"); lodash2_new_ver != pkg.RecommendedLibraryVersionString {
		t.Fatalf("bad version: %v", lodash2_new_ver)
	}
}

func TestLockfileOldNotSupported(t *testing.T) {
	lock := _loadf("4.6.1.npm-shrinkwrap.json")

	semverregex := api.PackageVersion{
		Version:                         "1.0.0",
		Library:                         api.Package{NormalizedName: "semver-regex", Name: "semver-regex", PackageManager: mappings.NpmManager},
		RecommendedLibraryVersionId:     "111",
		RecommendedLibraryVersionString: "1.0.0-sp1",
	}

	projectDir := "/prj"
	fixes := []shared.DependencyDescriptor{
		{VulnerablePackage: &semverregex,
			FixedLocations: []string{
				filepath.Join(projectDir, "node_modules/semver-regex"),
			},
		},
	}

	err := UpdateLockfile(lock, fixes, projectDir)
	if err != UnsupportedLockfileVersion {
		t.Fatalf("got %v, expected  unsupported error", err)
	}
}
