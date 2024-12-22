//go:build !windows

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
	return utils.NormalizePackageName(name)
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

func getTestFileReplaceSitePackage(name string, newpath string) string {
	// fetch file from current package's testdata folder
	// ref: https://pkg.go.dev/cmd/go/internal/test
	data := getTestFile(name)
	data = strings.Replace(data, " /usr/local/lib/python3.7/site-packages/pip ", fmt.Sprintf(" %s ", filepath.Join(newpath, "pip")), 1)

	return data
}

func makeSitePackagesDir(basedir string, packageFolderNames ...string) string {
	sitePackages := filepath.Join(basedir, "site-packages")
	for _, name := range packageFolderNames {
		err := os.MkdirAll(filepath.Join(sitePackages, name), os.ModePerm)
		if err != nil {
			panic(err)
		}
	}

	return sitePackages
}

func getBaseDir() string {
	base, err := os.MkdirTemp("", "test_seal_cli_*")
	if err != nil {
		panic(err)
	}

	return base
}

func TestParseDefaultDependencies(t *testing.T) {
	conf, _ := config.New(nil)
	parser := dependencyParser{conf, fakeNormalizer{}}

	baseDir := getBaseDir()
	sitePackages := makeSitePackagesDir(baseDir, "pip-24.0.dist-info", "setuptools-69.1.0.dist-info", "wheel-0.42.0.dist-info")
	output := getTestFileReplaceSitePackage("24.0_default_deps", sitePackages)

	dependencies, err := parser.Parse(output, defaultTestProjectDir)
	if err != nil {
		t.Fatalf("parse failed")
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

		if dep[0].DiskPath != filepath.Join(sitePackages, fmt.Sprintf("%s-%s.dist-info", dep[0].Name, dep[0].Version)) {
			t.Fatalf("wrong disk path %v", dep[0].DiskPath)
		}
	}
}

func TestParseEditable(t *testing.T) {
	conf, _ := config.New(nil)
	parser := dependencyParser{conf, fakeNormalizer{}}

	baseDir := getBaseDir()
	sitePackages := makeSitePackagesDir(baseDir, "fastapi-0.101.0.dist-info")
	output := getTestFileReplaceSitePackage("24.0_editable", sitePackages)

	dependencies, err := parser.Parse(output, defaultTestProjectDir)
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
	baseDir := getBaseDir()
	sitePackages := makeSitePackagesDir(baseDir)
	output := getTestFileReplaceSitePackage("24.0_no_deps", sitePackages)

	dependencies, err := parser.Parse(output, defaultTestProjectDir)
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

	baseDir := getBaseDir()
	sitePackages := makeSitePackagesDir(baseDir, "python-multipart-0.0.5+sp1.dist-info")
	output := getTestFileReplaceSitePackage("24.0_sp_version", sitePackages)

	dependencies, err := parser.Parse(output, defaultTestProjectDir)
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

	expected := filepath.Join(sitePackages, fmt.Sprintf("%s-%s.dist-info", dep.Name, dep.Version))
	if dep.DiskPath != expected {
		t.Fatalf("wrong disk path `%s` instead of `%s`", dep.DiskPath, expected)
	}
}
