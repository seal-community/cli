package utils

import "testing"

func TestGetOSVersion(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Test 1",
			input:    "1.2.3-4.el7",
			expected: "7",
		},
		{
			name:     "Test 2",
			input:    "5.9-14.20130511.el7_4+sp1",
			expected: "7",
		},
		{
			name:     "Test 3",
			input:    "2.7.5-94.el7_9",
			expected: "7",
		},
		{
			name:     "Test 4",
			input:    "3.4.3-168.el7.centos",
			expected: "7",
		},
		{
			name:     "Test 5",
			input:    "4:5.16.3-299.el7_9",
			expected: "7",
		},
		{
			name:     "Test 6",
			input:    "1.2.3-4.el10",
			expected: "10",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getOsVersion(tt.input)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestBuildUri(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected string
	}{
		{
			name:     "Test 1",
			input:    []string{"acl", "2.2.53-3.el8", "x86_64"},
			expected: "centos/8/x86_64/Packages/acl-2.2.53-3.el8.x86_64.rpm",
		},
		{
			name:     "Test 2",
			input:    []string{"acl", "2.2.53-3.el7", "x86_64"},
			expected: "centos/7/x86_64/Packages/acl-2.2.53-3.el7.x86_64.rpm",
		},
		{
			name:     "Test 3",
			input:    []string{"acl", "2.2.53-3.el6", "x86_64"},
			expected: "centos/6/x86_64/Packages/acl-2.2.53-3.el6.x86_64.rpm",
		},
		{
			name:     "Test 4",
			input:    []string{"acl", "2.2.53-3.el5", "x86_64"},
			expected: "centos/5/x86_64/Packages/acl-2.2.53-3.el5.x86_64.rpm",
		},
		{
			name:     "Test 5",
			input:    []string{"acl", "2.2.53-3.el4", "noarch"},
			expected: "centos/4/noarch/Packages/acl-2.2.53-3.el4.noarch.rpm",
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
