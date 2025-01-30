package apk

import "testing"

func TestBuildAPKName(t *testing.T) {
	tests := []struct {
		name     string
		version  string
		arch     string
		expected string
	}{
		{
			name:     "font-adobe-100dpi",
			version:  "1.0.4-r2",
			expected: "font-adobe-100dpi-1.0.4-r2.apk",
		},
		{
			name:     "a-b-c",
			version:  "1.0.0-r0",
			expected: "a-b-c-1.0.0-r0.apk",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := buildApkName(tt.name, tt.version)
			if actual != tt.expected {
				t.Fatalf("expected %s, got %s", tt.expected, actual)
			}
		})
	}
}
