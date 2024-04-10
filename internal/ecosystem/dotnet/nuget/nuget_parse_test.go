package nuget

import (
	"cli/internal/config"
	"cli/internal/ecosystem/mappings"
	"os"
	"path/filepath"
	"strings"

	"testing"

	"golang.org/x/exp/maps"
)

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
	parser := dependencyParser{conf}

	dependencies, err := parser.Parse(getTestFile("deps_default.json"), defaultTestProjectDir)
	if err != nil {
		t.Fatalf("parse failed ")
	}

	if len(dependencies) != 8 {
		t.Fatalf("expected 8 dependencies, got: %v", len(dependencies))
	}

	deps := maps.Values(dependencies)
	for _, dep := range deps {
		if len(dep) != 1 {
			t.Fatalf("expected 1 dep, got: %v", len(dep))
		}

		if dep[0].PackageManager != mappings.NugetManager {
			t.Fatalf("wrong package manager %v", dep[0].PackageManager)
		}

		if !dep[0].IsDirect() {
			t.Fatalf("did not detect as direct dep %v", dep)
		}

		if dep[0].Link {
			t.Fatalf("did not detect as non-link dep %v", dep)
		}

		expectedDependencies := map[string]string{
			"DotNet.ReproducibleBuilds":           "1.1.1",
			"Snappier":                            "1.1.0",
			"Microsoft.Build.Tasks.Git":           "1.1.1",
			"Microsoft.SourceLink.AzureRepos.Git": "1.1.1",
			"Microsoft.SourceLink.Bitbucket.Git":  "1.1.1",
			"Microsoft.SourceLink.Common":         "1.1.1",
			"Microsoft.SourceLink.GitHub":         "1.1.1",
			"Microsoft.SourceLink.GitLab":         "1.1.1",
		}

		if expectedVersion, ok := expectedDependencies[dep[0].Name]; ok {
			if dep[0].Version != expectedVersion {
				t.Fatalf("wrong version %v", dep[0].Version)
			}
			
		} else {
			t.Fatalf("unexpected package %v", dep[0].Name)
		}
	}
}

func TestParseEmptyDependenciesDependencies(t *testing.T) {
	conf, _ := config.New(nil)
	parser := dependencyParser{conf}

	dependencies, err := parser.Parse(getTestFile("deps_empty.json"), defaultTestProjectDir)
	if err != nil {
		t.Fatalf("parse failed ")
	}

	if len(dependencies) != 0 {
		t.Fatalf("expected 0 dependencies, got: %v", len(dependencies))
	}
}

func TestParseWithoutRestoreDependencies(t *testing.T) {
	conf, _ := config.New(nil)
	parser := dependencyParser{conf}

	dependencies, err := parser.Parse(getTestFile("deps_no_restore.json"), defaultTestProjectDir)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	if dependencies != nil {
		t.Fatalf("expected nil dependencies, got: %v", dependencies)
	}

	if strings.HasPrefix(err.Error(), "Error: No assets file was found") != true {
		t.Fatalf("expected error message to start with 'Error: No assets file was found', got: %v", err.Error())
	}
}
