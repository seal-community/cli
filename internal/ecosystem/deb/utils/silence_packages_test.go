package utils

import (
	"testing"
)

func TestModifyPackageStatusForSilence(t *testing.T) {
	pkg := StatusFilePackage{
		Package: "libquadmath0",
		Version: "4.9.2-10",
		Source:  "gcc-4.9",
	}
	modifyPackageStatusForSilence(&pkg)
	assertEqual(t, "seal-libquadmath0", pkg.Package)
	assertEqual(t, "seal-gcc-4.9", pkg.Source)
	assertEqual(t, "libquadmath0 (= 4.9.2-10)", pkg.Provides)
	assertEqual(t, "libquadmath0 (<= 4.9.2-10)", pkg.Conflicts)
	assertEqual(t, "libquadmath0 (<= 4.9.2-10)", pkg.Breaks)
	assertEqual(t, "libquadmath0 (<= 4.9.2-10)", pkg.Replaces)
}

func TestGetInfoFilePath(t *testing.T) {
	assertEqual(t, "/var/lib/dpkg/info/libquadmath0", getInfoFilePath("/var/lib/dpkg/info", "libquadmath0"))
}

func TestGetNewInfoFilePath(t *testing.T) {
	assertEqual(t, "/var/lib/dpkg/info/seal-libquadmath0", getNewInfoFilePath("/var/lib/dpkg/info", "libquadmath0"))
}

func TestIsFileRelatedToPackage(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"InvalidSuffix", "libquadmath0:amd64.md5sums", false},
		{"ValidSuffixWithColon", ":amd64.md5sums", true},
		{"ValidSuffixWithoutColon", ".md5sums", true},
		{"ValidSuffixWithoutColon", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidPackageFileSuffix(tt.input)
			if result != tt.expected {
				t.Errorf("For input %q, expected %v but got %v", tt.input, tt.expected, result)
			}
		})
	}
}
