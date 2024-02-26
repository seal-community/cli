package mappings

import (
	"fmt"
	"testing"
)

func TestEcosystemConversion(t *testing.T) {
	maps := [][]string{
		{NpmManager, "node"},
		{PythonManager, "python"},
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
