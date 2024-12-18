package utils

import "testing"

func TestBuildDebName(t *testing.T) {
	tests := []struct {
		name     string
		version  string
		arch     string
		expected string
	}{
		{
			name:     "test1",
			version:  "1.0.0",
			arch:     "amd64",
			expected: "test1_1.0.0_amd64.deb",
		},
		{
			name:     "a-b-c",
			version:  "1.0.0-545",
			arch:     "noarch",
			expected: "a-b-c_1.0.0-545_noarch.deb",
		},
		{
			name:     "a_b_c",
			version:  "1:1.0.0-545",
			arch:     "noarch",
			expected: "a_b_c_1.0.0-545_noarch.deb",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := BuildDebName(tt.name, tt.version, tt.arch)
			if actual != tt.expected {
				t.Fatalf("expected %s, got %s", tt.expected, actual)
			}
		})
	}
}
