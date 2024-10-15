package yum

import "testing"

func TestBuildRpmName(t *testing.T) {
	tests := []struct {
		name     string
		version  string
		arch     string
		expected string
	}{
		{
			name:     "test1",
			version:  "1.0.0",
			arch:     "x86_64",
			expected: "test1-1.0.0.x86_64.rpm",
		},
		{
			name:     "a-b-c",
			version:  "1.0.0-545.el7_9",
			arch:     "noarch",
			expected: "a-b-c-1.0.0-545.el7_9.noarch.rpm",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := buildRpmName(tt.name, tt.version, tt.arch)
			if actual != tt.expected {
				t.Fatalf("expected %s, got %s", tt.expected, actual)
			}
		})
	}
}
