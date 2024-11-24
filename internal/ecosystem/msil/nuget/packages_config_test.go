package nuget

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestParsePackagesConfig(t *testing.T) {
	data := `<?xml version="1.0" encoding="utf-8"?>
<packages>
  <package id="EntityFramework" version="6.1.3" targetFramework="net452" />
  <package id="log4net" version="2.0.5" targetFramework="net452" />
</packages>`

	pkgsDir := "/packages_dir"
	depMap, err := parsePackagesConfig(strings.NewReader(data), pkgsDir)
	// since we don't actually have any of the packages on disk, it will return an error
	if err != NoPackagesFoundError || len(depMap) == 0 {
		t.Fatalf("err %v", err)
	}

	if len(depMap) != 2 {
		t.Fatal("wrong number of deps", "count", len(depMap))
	}

	entityDeps, exists := depMap["NuGet|entityframework@6.1.3"]
	if !exists {
		t.Fatal("does not exist")
	}

	if len(entityDeps) != 1 {
		t.Fatal("wrong number of entity deps", "count", len(entityDeps))
	}

	entityDiskPath := filepath.Join(pkgsDir, "EntityFramework.6.1.3")
	if entityDeps[0].DiskPath != entityDiskPath {
		t.Fatalf("wrong disk path: expected `%s` got `%s`", entityDiskPath, entityDeps[0].DiskPath)
	}

	log4netDeps, exists := depMap["NuGet|log4net@2.0.5"]
	if !exists {
		t.Fatal("does not exist")
	}

	if len(log4netDeps) != 1 {
		t.Fatal("wrong number of entity deps", "count", len(entityDeps))
	}

	log4netDiskPath := filepath.Join(pkgsDir, "log4net.2.0.5")
	if log4netDeps[0].DiskPath != log4netDiskPath {
		t.Fatalf("wrong disk path: expected `%s` got `%s`", log4netDiskPath, log4netDeps[0].DiskPath)
	}

}

func TestParsePackagesConfigDot(t *testing.T) {
	data := `<?xml version="1.0" encoding="utf-8"?>
<packages>
  <package id="jQuery.Validation" version="1.19.4" targetFramework="net452" />
</packages>`

	pkgsDir := "/packages_dir"
	depMap, err := parsePackagesConfig(strings.NewReader(data), pkgsDir)
	// since we don't actually have any of the packages on disk, it will return an error
	if err != NoPackagesFoundError || len(depMap) != 1 {
		t.Fatalf("err %v", err)
	}

	entityDeps, exists := depMap["NuGet|jquery.validation@1.19.4"]
	if !exists {
		t.Fatal("does not exist")
	}

	if len(entityDeps) != 1 {
		t.Fatal("wrong number of entity deps", "count", len(entityDeps))
	}

	entityDiskPath := filepath.Join(pkgsDir, "jQuery.Validation.1.19.4")
	if entityDeps[0].DiskPath != entityDiskPath {
		t.Fatalf("wrong disk path: expected `%s` got `%s`", entityDiskPath, entityDeps[0].DiskPath)
	}

}

func TestFormatDependencyDiskPath(t *testing.T) {
	if res := formatDependencyDiskPath("my-root", "lib-NAME", "1.2.3"); res != filepath.Join("my-root", "lib-NAME.1.2.3") {
		t.Fatalf("bad path, got `%s`", res)
	}
}
