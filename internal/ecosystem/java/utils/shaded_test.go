package utils

import "testing"

func TestShouldSkipDependency(t *testing.T) {
	tests := []struct {
		dep      shadedDependency
		expected bool
	}{
		{
			dep: shadedDependency{
				name:    "org.apache.commons:commons-lang3",
				version: "3.11",
			},
			expected: false,
		},
		{
			dep: shadedDependency{
				name:    "org.apache.commons:commons-lang3",
				version: "",
			},
			expected: true,
		},
		{
			dep: shadedDependency{
				name:    "",
				version: "3.11",
			},
			expected: true,
		},
	}
	for _, test := range tests {
		t.Run(test.dep.name, func(t *testing.T) {
			if shouldSkipDependency(test.dep) != test.expected {
				t.Fatalf("unexpected result")
			}
		})
	}
}
