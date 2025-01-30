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
			input:    []string{"acl", "2.3.2-r0", "x86_64"},
			expected: "seal/seal/x86_64/acl-2.3.2-r0.apk",
		},
		{
			name:     "Test 2",
			input:    []string{"libc++-static", "1:17.0.6-r1", "aarch64"},
			expected: "seal/seal/aarch64/libc++-static-17.0.6-r1.apk",
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
