package dpkg

import "testing"

func TestNormalizePackageName(t *testing.T) {
	tests := []struct {
		name     string
		expected string
	}{
		{
			name:     "test1",
			expected: "test1",
		},
		{
			name:     "a-b-c",
			expected: "a-b-c",
		},
		{
			name:     "a_b_c",
			expected: "a_b_c",
		},
	}

	normalizer := NewDPKGManager(nil, "")
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := normalizer.NormalizePackageName(tt.name)
			if actual != tt.expected {
				t.Fatalf("expected %s, got %s", tt.expected, actual)
			}
		})
	}
}
