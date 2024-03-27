//go:build !windows

package npm

import (
	"cli/internal/common"
	"cli/internal/config"
	"cli/internal/ecosystem/mappings"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/exp/maps"
)

const PlatformTestDir = "unix"
const defaultTestProjectDir = "/Users/fuwawa/proj"

func getTestFile(name string) string {
	// fetch file from current package's testdata folder
	// ref: https://pkg.go.dev/cmd/go/internal/test
	p := filepath.Join("testdata", PlatformTestDir, name)
	data, err := os.ReadFile(p)
	if err != nil {
		panic(err)
	}

	return string(data)
}

// the following tests could be ported to windows too with appropriate paths
func TestEmptyDepObject(t *testing.T) {
	conf, _ := config.New(nil)
	parser := dependencyParser{conf}
	deps, err := parser.Parse(getTestFile("empty_dep.json"),
		defaultTestProjectDir)
	if err != nil {
		t.Fatalf("failed parsing with empty dep %v", err)
	}
	if len(deps) != 0 {
		t.Fatalf("wrong number of deps %d", len(deps))

	}
}

func TestDupVersionInTree(t *testing.T) {
	conf, _ := config.New(nil)
	parser := dependencyParser{conf}
	deps, err := parser.Parse(getTestFile("dup_versions.json"),
		defaultTestProjectDir)
	if err != nil {
		t.Fatalf("failed parsing with dup deps %v", err)
	}

	if len(deps) != 2 {
		// has cypress(direct) which requires lodash
		t.Fatalf("wrong number of deps %d", len(deps))
	}

	lodashKey := common.DependencyId(mappings.NpmManager, "lodash", "4.17.11")
	depInstances := deps[lodashKey]

	if len(depInstances) != 2 {
		t.Fatalf("wrong number of instances of dep %v", depInstances)
	}
}

func TestAliasDependency(t *testing.T) {
	conf, _ := config.New(nil)
	parser := dependencyParser{conf}
	deps, err := parser.Parse(getTestFile("6.14.18_alias_dep.json"),
		"/test/alias_test")
	if err != nil {
		t.Fatalf("failed parsing with dup deps %v", err)
	}

	replacedKey := common.DependencyId(mappings.NpmManager, "semver-regex", "0.1.1")
	replacedDeps, ok := deps[replacedKey]
	if !ok {
		t.Fatalf("did not find deps for %s", replacedKey)
	}

	if len(replacedDeps) != 2 {
		// occurs twice because it is a direct dep, as well as 'transitive'
		t.Fatalf("wrong number of deps for %v", replacedDeps)
	}

	dep := replacedDeps[0]
	if dep.NameAlias != "is-extendable" {
		t.Fatalf("did not detect alias %v", dep)
	}
}

func TestExtraneousDepNotIgnored(t *testing.T) {
	conf, _ := config.New(nil)
	parser := dependencyParser{conf}
	deps, err := parser.Parse(getTestFile("9.9.0_extraneous.json"),
		"/test/output/extraneous_test")
	if err != nil {
		t.Fatalf("failed parsing %v", err)
	}

	if len(deps) == 0 {
		t.Fatalf("ignored extraneous dep")
	}

	instances := deps[common.DependencyId(mappings.NpmManager, "semver-regex", "1.0.0")]
	if len(instances) != 1 {
		t.Fatalf("wrong number of deps for extraneous: %v", instances)
	}

	depInstance := instances[0]
	if depInstance.Name != "semver-regex" || depInstance.Version != "1.0.0" {
		t.Fatalf("wrong dep %v", depInstance)
	}

	if !depInstance.Extraneous {
		t.Fatalf("not marked as extraneous %v", depInstance)
	}
}

func TestExtraneousDepSkipped(t *testing.T) {
	conf, _ := config.New(nil)
	conf.Npm.IgnoreExtraneous = true
	parser := dependencyParser{conf}
	deps, err := parser.Parse(getTestFile("9.9.0_extraneous.json"),
		"/test/output/extraneous_test")
	if err != nil {
		t.Fatalf("failed parsing %v", err)
	}

	if len(deps) != 0 {
		t.Fatalf("did not ignore extraneous dep; found: %v", deps)
	}
}

func TestLocalFolderPackageInstallLinks(t *testing.T) {
	// Test parsing installation with --install-links is parsed correctly, even though running npm ll returns an error
	conf, _ := config.New(nil)
	parser := dependencyParser{conf}
	dependencies, err := parser.Parse(getTestFile("10.2.0_folder_dep.json"),
		"/test/output/folder_test")
	if err != nil {
		t.Fatalf("failed parsing %v", err)
	}

	if len(dependencies) != 1 {
		t.Fatalf("got more than 1 dependency: %v", len(dependencies))
	}

	deps := maps.Values(dependencies)
	dep := deps[0][0]

	if dep.PackageManager != mappings.NpmManager {
		t.Fatalf("wrong package manager %v", dep.PackageManager)
	}

	if dep.Version != "6.0.0" {
		t.Fatalf("wrong version %v", dep.Version)
	}

	if dep.Name != "is-number" {
		t.Fatalf("wrong version %v", dep.Version)
	}

	if !dep.IsDirect() {
		t.Fatalf("did not detect as direct dep %v", dep)
	}
}


