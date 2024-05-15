package maven

import (
	"os"
	"path/filepath"
	"testing"
)


func TestEmptyParseDependencies(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "empty_deps.txt"))
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	depList, err := parseDependencies(string(data))
	if len(depList) != 0 || err != nil {
		t.Fatalf("expected 0 deps, got %v", len(depList))
	}
}

func TestFullParseDependencies(t *testing.T) {
	expectedDependencies := map[string]bool{
		"com.example.app:example-app:jar:1.0-SNAPSHOT": true,
		"junit:junit:jar:4.11:test": true,
		"org.springframework:spring-beans:jar:5.3.12:compile": true,
		"net.minidev:json-smart:jar:2.4.8:compile": true,
		"org.hamcrest:hamcrest-core:jar:1.3:test": true,
		"org.springframework:spring-core:jar:5.3.12:compile": true,
		"org.springframework:spring-jcl:jar:5.3.12:compile": true,
		"net.minidev:accessors-smart:jar:2.4.8:compile": true,
		"org.ow2.asm:asm:jar:9.1:compile": true,
	}
	data, err := os.ReadFile(filepath.Join("testdata", "full_deps.txt"))
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	resultDependencies, err := parseDependencies(string(data))
	if len(resultDependencies) != len(expectedDependencies) || err != nil {
		t.Fatalf("expected %v deps, got %v", len(expectedDependencies), len(resultDependencies))
	}
	for dep := range resultDependencies {
		if _, ok := expectedDependencies[dep]; !ok {
			t.Fatalf("unexpected dependency %v", dep)
		}
	}
}

func TestModulesParseDependencies(t *testing.T) {
	expectedDependencies := map[string]bool{
		"com.example.app:parent-project:pom:1.0-SNAPSHOT": true,
		"com.example.app:module-a:jar:1.0-SNAPSHOT": true,
		"com.example.app:module-b:jar:1.0-SNAPSHOT": true,
		"junit:junit:jar:3.8.1:test": true,
		"org.springframework:spring-beans:jar:5.3.12:compile": true,
		"net.minidev:json-smart:jar:2.4.8:compile": true,
		"org.springframework:spring-core:jar:5.3.12:compile": true,
		"org.springframework:spring-jcl:jar:5.3.12:compile": true,
		"net.minidev:accessors-smart:jar:2.4.8:compile": true,
		"org.ow2.asm:asm:jar:9.1:compile": true,
	}
	data, err := os.ReadFile(filepath.Join("testdata", "modules_deps.txt"))
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	resultDependencies, err := parseDependencies(string(data))
	if len(resultDependencies) != len(expectedDependencies) || err != nil {
		t.Fatalf("expected %v deps, got %v", len(expectedDependencies), len(resultDependencies))
	}
	for dep := range resultDependencies {
		if _, ok := expectedDependencies[dep]; !ok {
			t.Fatalf("unexpected dependency %v", dep)
		}
	}
}
