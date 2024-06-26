package mappings

import (
	"fmt"
	"testing"
)

func TestEcosystemConversion(t *testing.T) {
	maps := [][]string{
		{NpmManager, "node"},
		{PythonManager, "python"},
		{NugetManager, ".NET"},
		{MavenManger, "java"},
		{"asdasdasda", ""},
	}

	for i, m := range maps {
		t.Run(fmt.Sprintf("map_%d", i), func(t *testing.T) {
			given := m[0]
			expected := m[1]
			if result := BackendManagerToEcosystem(given); result != expected {
				t.Fatalf("wrong ecosystem, expected: `%s` got: `%s`", expected, result)
			}
		})
	}

}

func TestManagerToEcosystemConversion(t *testing.T) {
	maps := [][]string{
		{NodeEcosystem, "NPM"},
		{PythonEcosystem, "PyPI"},
		{DotnetEcosystem, "NuGet"},
		{JavaEcosystem, "Maven"},
		{"asdasdasda", ""},
	}

	for i, m := range maps {
		t.Run(fmt.Sprintf("map_%d", i), func(t *testing.T) {
			given := m[0]
			expected := m[1]
			if result := EcosystemToBackendManager(given); result != expected {
				t.Fatalf("wrong manager, expected: `%s` got: `%s`", expected, result)
			}
		})
	}
}
