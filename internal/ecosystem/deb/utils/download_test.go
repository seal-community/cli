package utils

import "testing"

func TestBuildUri(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected string
	}{
		{
			name:     "Test 1",
			input:    []string{"zlib", "1.2.8.dfsg-2+b1", "amd64"},
			expected: "pool/main/s/seal/zlib_1.2.8.dfsg-2+b1_amd64.deb",
		},
		{
			name:     "Test 2",
			input:    []string{"zlib", "1:1.2.8.dfsg-2+b1", "noarch"},
			expected: "pool/main/s/seal/zlib_1.2.8.dfsg-2+b1_noarch.deb",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildUri(tt.input[0], tt.input[1], tt.input[2])
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}
