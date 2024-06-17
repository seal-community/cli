package pip

import (
	"cli/internal/config"
	"cli/internal/ecosystem/mappings"
	"cli/internal/ecosystem/python/utils"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"testing"

	"golang.org/x/exp/maps"
)

type fakeNormalizer struct{}

func (f fakeNormalizer) NormalizePackageName(name string) string {
	return strings.Replace(strings.ToLower(name), "_", "-", -1)
}

const defaultTestProjectDir = "/Users/fuwawa/proj"

func getTestFile(name string) string {
	// fetch file from current package's testdata folder
	// ref: https://pkg.go.dev/cmd/go/internal/test
	p := filepath.Join("testdata", name)
	data, err := os.ReadFile(p)
	if err != nil {
		panic(err)
	}

	return string(data)
}

func TestParseDefaultDependencies(t *testing.T) {
	conf, _ := config.New(nil)
	parser := dependencyParser{conf, fakeNormalizer{}}

	dependencies, err := parser.Parse(getTestFile("24.0_default_deps"),
		defaultTestProjectDir)
	if err != nil {
		t.Fatalf("parse failed ")
	}

	if len(dependencies) != 3 {
		t.Fatalf("expected 3 dependencies, got: %v", len(dependencies))
	}

	deps := maps.Values(dependencies)
	for _, dep := range deps {
		if len(dep) != 1 {
			t.Fatalf("expected 1 dep, got: %v", len(dep))
		}

		if dep[0].PackageManager != mappings.PythonManager {
			t.Fatalf("wrong package manager %v", dep[0].PackageManager)
		}

		if !dep[0].IsDirect() {
			t.Fatalf("did not detect as direct dep %v", dep)
		}

		if dep[0].Link {
			t.Fatalf("did not detect as non-link dep %v", dep)
		}

		if dep[0].Name == "pip" {
			if dep[0].Version != "24.0" {
				t.Fatalf("wrong version %v", dep[0].Version)
			}
		} else if dep[0].Name == "setuptools" {
			if dep[0].Version != "69.1.0" {
				t.Fatalf("wrong version %v", dep[0].Version)
			}
		} else if dep[0].Name == "wheel" {
			if dep[0].Version != "0.42.0" {
				t.Fatalf("wrong version %v", dep[0].Version)
			}
		} else {
			t.Fatalf("unexpected package %v", dep[0].Name)
		}

		if dep[0].DiskPath != filepath.Join("/usr/local/lib/python3.7/site-packages", fmt.Sprintf("%s-%s.dist-info", utils.EscapePackageName(dep[0].Name), dep[0].Version)) {
			t.Fatalf("wrong disk path %v", dep[0].DiskPath)
		}
	}
}

func TestParseEditable(t *testing.T) {
	conf, _ := config.New(nil)
	parser := dependencyParser{conf, fakeNormalizer{}}

	dependencies, err := parser.Parse(getTestFile("24.0_editable"),
		defaultTestProjectDir)
	if err != nil {
		t.Fatalf("parse failed ")
	}

	if len(dependencies) != 0 {
		t.Fatalf("expected 0 dependencies, got: %v", len(dependencies))
	}
}

func TestParseNoDependencies(t *testing.T) {
	// Can't really happen, pip is always a dep, but here for good measures
	conf, _ := config.New(nil)
	parser := dependencyParser{conf, fakeNormalizer{}}

	dependencies, err := parser.Parse(getTestFile("24.0_no_deps"),
		defaultTestProjectDir)
	if err != nil {
		t.Fatalf("parse failed ")
	}

	if len(dependencies) != 0 {
		t.Fatalf("expected 0 dependencies, got: %v", len(dependencies))
	}
}

func TestParseSP(t *testing.T) {
	conf, _ := config.New(nil)
	parser := dependencyParser{conf, fakeNormalizer{}}

	dependencies, err := parser.Parse(getTestFile("24.0_sp_version"),
		defaultTestProjectDir)
	if err != nil {
		t.Fatalf("parse failed ")
	}

	if len(dependencies) != 1 {
		t.Fatalf("expected 1 dependencies, got: %v", len(dependencies))
	}

	deps := maps.Values(dependencies)
	if len(deps[0]) != 1 {
		t.Fatalf("expected 1 dep, got: %v", len(deps[0]))
	}

	dep := deps[0][0]
	if dep.PackageManager != mappings.PythonManager {
		t.Fatalf("wrong package manager %v", dep.PackageManager)
	}

	if !dep.IsDirect() {
		t.Fatalf("did not detect as direct dep %v", dep)
	}

	if dep.Name != "python-multipart" {
		t.Fatalf("wrong package %v", dep.Name)
	}

	if dep.Version != "0.0.5+sp1" {
		t.Fatalf("wrong version %v", dep.Version)
	}

	if dep.DiskPath != filepath.Join("/usr/local/lib/python3.7/site-packages", utils.DistInfoPath(dep.Name, dep.Version)) {
		t.Fatalf("wrong disk path %v", dep.DiskPath)
	}
}

func TestParseWindowsDependencies(t *testing.T) {
	conf, _ := config.New(nil)
	parser := dependencyParser{conf, fakeNormalizer{}}

	dependencies, err := parser.Parse(getTestFile("24.0_windows"),
		defaultTestProjectDir)
	if err != nil {
		t.Fatalf("parse failed ")
	}

	if len(dependencies) != 1 {
		t.Fatalf("expected 1 dependencies, got: %v", len(dependencies))
	}

	deps := maps.Values(dependencies)
	if len(deps[0]) != 1 {
		t.Fatalf("expected 1 dep, got: %v", len(deps[0]))
	}

	dep := deps[0][0]
	if dep.PackageManager != mappings.PythonManager {
		t.Fatalf("wrong package manager %v", dep.PackageManager)
	}

	if !dep.IsDirect() {
		t.Fatalf("did not detect as direct dep %v", dep)
	}

	if dep.Name != "pip" {
		t.Fatalf("wrong package %v", dep.Name)
	}

	if dep.Version != "24.0" {
		t.Fatalf("wrong version %v", dep.Version)
	}

	if dep.DiskPath != filepath.Join(`C:\Python\site-packages`, utils.DistInfoPath(dep.Name, dep.Version)) {
		t.Fatalf("wrong disk path %v", dep.DiskPath)
	}
}

func TestShouldFix(t *testing.T) {
	conf, _ := config.New(nil)
	parser := dependencyParser{conf, fakeNormalizer{}}
	noSkip := &PythonPackage{
		Name:                    "pip",
		Version:                 "24.0",
		EditableProjectLocation: "",
	}
	skipEmpty := &PythonPackage{
		Name:                    "",
		Version:                 "",
		EditableProjectLocation: "",
	}
	skipEditable := &PythonPackage{
		Name:                    "pip",
		Version:                 "24.0",
		EditableProjectLocation: "/Users/fuwawa/proj",
	}

	if parser.shouldSkip(noSkip) {
		t.Fatalf("should not skip")
	}

	if !parser.shouldSkip(skipEmpty) {
		t.Fatalf("should skip")
	}

	if !parser.shouldSkip(skipEditable) {
		t.Fatalf("should not skip")
	}
}
