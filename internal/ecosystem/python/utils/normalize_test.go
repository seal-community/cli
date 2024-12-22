package utils

import "testing"

func TestNormalizePackageName(t *testing.T) {
	tests := []struct {
		name     string
		expected string
	}{
		{"foo", "foo"},
		{"foo_bar", "foo-bar"},
		{"foo_bar_baz", "foo-bar-baz"},
		{"foo_bar_baz_qux", "foo-bar-baz-qux"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if result := NormalizePackageName(test.name); result != test.expected {
				t.Fatalf("wrong result, expected: `%s` got: `%s`", test.expected, result)
			}
		})
	}
}
