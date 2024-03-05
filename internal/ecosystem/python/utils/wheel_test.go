package utils

import "testing"

func TestEscapePackageName(t *testing.T) {
	tests := []struct {
		name     string
		expected string
	}{
		{"foo", "foo"},
		{"foo-bar", "foo_bar"},
		{"foo-bar-baz", "foo_bar_baz"},
		{"foo-bar-baz-qux", "foo_bar_baz_qux"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if result := EscapePackageName(test.name); result != test.expected {
				t.Fatalf("wrong result, expected: `%s` got: `%s`", test.expected, result)
			}
		})
	}
}

func TestDistInfoPath(t *testing.T) {
	tests := []struct {
		name     string
		version  string
		expected string
	}{
		{"foo", "1.2.3", "foo-1.2.3.dist-info"},
		{"foo-bar", "0.1.0", "foo_bar-0.1.0.dist-info"},
		{"foo-bar-baz", "1.2.3", "foo_bar_baz-1.2.3.dist-info"},
		{"foo-bar-baz-qux", "1.2.3.", "foo_bar_baz_qux-1.2.3..dist-info"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if result := DistInfoPath(test.name, test.version); result != test.expected {
				t.Fatalf("wrong result, expected: `%s` got: `%s`", test.expected, result)
			}
		})
	}
}
