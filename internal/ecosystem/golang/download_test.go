package golang

import "testing"

func TestBuildUri(t *testing.T) {
	// Test cases
	tests := []struct {
		// Input
		name    string
		version string
		// Output
		expected string
	}{
		{
			name:     "github.com/psf13/cobra",
			version:  "1.0.0",
			expected: "github.com/psf13/cobra/@v/v1.0.0.zip",
		},
		{
			name:     "github.com/Masterminds/semver",
			version:  "1.0.0",
			expected: "github.com/!masterminds/semver/@v/v1.0.0.zip",
		},
		{
			name:     "github.com/Masterminds/semver",
			version:  "1.0.0-sp1",
			expected: "github.com/!masterminds/semver/@v/v1.0.0-sp1.zip",
		},
		{
			name:     "github.com/Masterminds/semver",
			version:  "1.0.0-RELEASE",
			expected: "github.com/!masterminds/semver/@v/v1.0.0-!r!e!l!e!a!s!e.zip",
		},
	}

	for i, test := range tests {
		// Test
		result := buildUri(test.name, test.version)

		// Compare
		if result != test.expected {
			t.Fatalf("Test %d failed: expected %s, got %s", i, test.expected, result)
		}
	}
}
