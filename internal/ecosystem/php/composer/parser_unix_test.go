//go:build !windows

package composer

import (
	"cli/internal/ecosystem/mappings"
	"testing"

	"golang.org/x/exp/maps"
)

func TestParseNoDeps(t *testing.T) {
	dependencies, err := ParseComposerDependencies("[]", "")
	if err != nil {
		t.Fatalf("parse failed ")
	}

	if len(dependencies) != 0 {
		t.Fatalf("got more than 0 dependency: %v %v", len(dependencies), dependencies)
	}
}

func TestParseSingleDependency(t *testing.T) {
	dependencies, err := ParseComposerDependencies(getTestFile("single_dep.json"), "")
	if err != nil {
		t.Fatalf("parse failed ")
	}

	if len(dependencies) != 1 {
		t.Fatalf("got more than 1 dependency: %v", len(dependencies))
	}

	deps := maps.Values(dependencies)
	dep := deps[0][0]

	if dep.PackageManager != mappings.ComposerManager {
		t.Fatalf("wrong package manager %v", dep.PackageManager)
	}

	if dep.Version != "1.1.1" {
		t.Fatalf("wrong version %v", dep.Version)
	}

	if dep.Name != "vendor/package" {
		t.Fatalf("wrong version %v", dep.Version)
	}

	if !dep.IsDirect() {
		t.Fatalf("did not detect as direct dep %v", dep)
	}
}

func TestParseOldComposer(t *testing.T) {
	dependencies, err := ParseComposerDependencies(getTestFile("old_composer.json"), "")
	if err != nil {
		t.Fatalf("parse failed ")
	}

	if len(dependencies) != 2 {
		t.Fatalf("got more than 2 dependency: %v", len(dependencies))
	}

	deps := maps.Values(dependencies)
	dep1 := deps[0][0]
	dep2 := deps[1][0]

	if dep1.Name != "vendor1/package1" && dep2.Name == "vendor1/package1" {
		dep1, dep2 = dep2, dep1
	}

	if dep1.PackageManager != mappings.ComposerManager {
		t.Fatalf("wrong package manager %v", dep1.PackageManager)
	}

	if dep1.Version != "1.1.1" {
		t.Fatalf("wrong version %v", dep1.Version)
	}

	if dep1.Name != "vendor1/package1" {
		t.Fatalf("wrong library %v", dep1.Version)
	}

	if dep2.PackageManager != mappings.ComposerManager {
		t.Fatalf("wrong package manager %v", dep2.PackageManager)
	}

	if dep2.Version != "2.2.2+sp2" {
		t.Fatalf("wrong version %v", dep2.Version)
	}

	if dep2.Name != "vendor2/package2" {
		t.Fatalf("wrong library %v", dep2.Version)
	}
}

func TestGetDistPathUnix(t *testing.T) {
	if getDiskPath("dir", "vendor/package") != "dir/vendor/vendor/package" {
		t.Fatalf("wrong disk path")
	}
}
