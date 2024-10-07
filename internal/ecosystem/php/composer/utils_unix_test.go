//go:build !windows

package composer

import (
	"testing"
)

type TestPair struct {
	input    string
	expected string
}

func TestNormalizePackageName(t *testing.T) {
	tests := []TestPair{
		{"aa/aaa", "aa/aaa"},
		{"aaA/Aa", "aaa/aa"},
		{"PACKAGE/VENDOR", "package/vendor"},
		{"  package/vendor  ", "package/vendor"},
	}

	for _, test := range tests {
		result := normalizePackageName(test.input)
		if result != test.expected {
			t.Fatalf("got %s, expected %s", result, test.expected)
		}
	}
}

func TestGetMetadataDepFileSanityUnix(t *testing.T) {
	output := getMetadataDepFile("/a/b/c", "vendor/package")
	expected := "/a/b/c/vendor/vendor/package/.seal-metadata.yaml"
	if output != expected {
		t.Fatalf("got %s, expected %s", output, expected)
	}
}
