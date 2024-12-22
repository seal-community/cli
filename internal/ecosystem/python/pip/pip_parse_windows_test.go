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
	data = strings.Replace(data, ` C:\Python\site-packages `, fmt.Sprintf(" %s ", filepath.Join(newpath, "pip")), 1)

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

func TestParseWindowsDependencies(t *testing.T) {
	conf, _ := config.New(nil)
	parser := dependencyParser{conf, fakeNormalizer{}}

	baseDir := getBaseDir()
	sitePackages := makeSitePackagesDir(baseDir, "pip-24.0.dist-info")
	output := getTestFileReplaceSitePackage("24.0_windows", sitePackages)

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

	if dep.Name != "pip" {
		t.Fatalf("wrong package %v", dep.Name)
	}

	if dep.Version != "24.0" {
		t.Fatalf("wrong version %v", dep.Version)
	}

	expected := filepath.Join(sitePackages, fmt.Sprintf("%s-%s.dist-info", dep.Name, dep.Version))
	if dep.DiskPath != expected {
		t.Fatalf("wrong disk path `%s` instead of `%s`", dep.DiskPath, expected)
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
