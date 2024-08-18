package golang

import (
	"path/filepath"
	"testing"
)

func TestParseGoVersion(t *testing.T) {
	version := ParseGoVersion("go version go1.21.4 darwin/arm64")
	if version != "1.21.4" {
		t.Fatalf("expected 1.21.4, got %s", version)
	}

	version = ParseGoVersion("go version go1.16.15 linux/amd64")
	if version != "1.16.15" {
		t.Fatalf("expected 1.16.15, got %s", version)
	}
}

func TestBuildDependencyMapSimple(t *testing.T) {
	goMod, err := ParseGoModFile(filepath.Join("testdata", "simple_go.mod"))
	if err != nil {
		t.Fatalf("failed parsing go.mod file: %v", err)
	}

	deps := BuildDependencyMap(goMod)
	if len(deps) != 2 {
		t.Fatalf("expected 2 dependencies, got %v", len(deps))
	}

	deps1, ok := deps["GO|github.com/masterminds/semver/v3@3.2.1"]
	if !ok {
		t.Fatalf("expected dependency not found")
	}
	if len(deps1) != 1 {
		t.Fatalf("expected 1 version, got %v", len(deps1))
	}
	dep1 := deps1[0]
	if dep1.Name != "github.com/Masterminds/semver/v3" {
		t.Fatalf("expected github.com/Masterminds/semver/v3, got %s", dep1.Name)
	}
	if dep1.Version != "3.2.1" {
		t.Fatalf("expected 3.2.1, got %s", dep1.Version)
	}
	if dep1.PackageManager != "GO" {
		t.Fatalf("expected Go, got %s", dep1.PackageManager)
	}

	deps2, ok := deps["GO|golang.org/x/exp@0.0.0-20231110203233-9a3e6036ecaa"]
	if !ok {
		t.Fatalf("expected dependency not found")
	}
	if len(deps2) != 1 {
		t.Fatalf("expected 1 version, got %v", len(deps2))
	}
	dep2 := deps2[0]
	if dep2.Name != "golang.org/x/exp" {
		t.Fatalf("expected golang.org/x/exp, got %s", dep2.Name)
	}
	if dep2.Version != "0.0.0-20231110203233-9a3e6036ecaa" {
		t.Fatalf("expected 0.0.0-20231110203233-9a3e6036ecaa, got %s", dep2.Version)
	}
	if dep2.PackageManager != "GO" {
		t.Fatalf("expected Go, got %s", dep2.PackageManager)
	}
}

func TestBuildDependencyMapReplace1(t *testing.T) {
	goMod, err := ParseGoModFile(filepath.Join("testdata", "replace1_go.mod"))
	if err != nil {
		t.Fatalf("failed parsing go.mod file: %v", err)
	}

	deps := BuildDependencyMap(goMod)
	if len(deps) != 1 {
		t.Fatalf("expected 1 dependency, got %v", len(deps))
	}

	deps1, ok := deps["GO|github.com/deepfabric/etcd@3.3.17+incompatible"]
	if !ok {
		t.Fatalf("expected dependency not found")
	}
	if len(deps1) != 1 {
		t.Fatalf("expected 1 version, got %v", len(deps1))
	}
}

func TestBuildDependencyMapReplace2(t *testing.T) {
	goMod, err := ParseGoModFile(filepath.Join("testdata", "replace2_go.mod"))
	if err != nil {
		t.Fatalf("failed parsing go.mod file: %v", err)
	}

	deps := BuildDependencyMap(goMod)
	if len(deps) != 1 {
		t.Fatalf("expected 1 dependency, got %v", len(deps))
	}

	deps1, ok := deps["GO|github.com/deepfabric/etcd@3.3.17+incompatible"]
	if !ok {
		t.Fatalf("expected dependency not found")
	}
	if len(deps1) != 1 {
		t.Fatalf("expected 1 version, got %v", len(deps1))
	}
}

func TestBuildDependencyMapReplace3(t *testing.T) {
	goMod, err := ParseGoModFile(filepath.Join("testdata", "replace3_go.mod"))
	if err != nil {
		t.Fatalf("failed parsing go.mod file: %v", err)
	}

	deps := BuildDependencyMap(goMod)
	if len(deps) != 0 {
		t.Fatalf("expected 2 dependencies, got %v", len(deps))
	}
}

func TestBuildDependencyMapReplace4(t *testing.T) {
	goMod, err := ParseGoModFile(filepath.Join("testdata", "replace4_go.mod"))
	if err != nil {
		t.Fatalf("failed parsing go.mod file: %v", err)
	}

	deps := BuildDependencyMap(goMod)
	if len(deps) != 0 {
		t.Fatalf("expected 2 dependencies, got %v", len(deps))
	}
}
