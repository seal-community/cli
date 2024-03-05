package utils

import "testing"

func TestGetSitePackages(t *testing.T) {
	tests := []struct {
		name     string
		expected string
	}{
		{"pip 24.0 from /usr/local/lib/python3.7/site-packages/pip (python 3.7)", "/usr/local/lib/python3.7/site-packages/"},
		{"pip 1.2.3 from /usr/local/lib/python3.7/site-packages/pip (python 3.7)", "/usr/local/lib/python3.7/site-packages/"},
		{"pip 24.0 from /usr/local/lib/python3.7/site-packages/pip (python 3.3)", "/usr/local/lib/python3.7/site-packages/"},
		{`pip 24.0 from C:\Python\site-packages\pip (python 3.7)`, `C:\Python\site-packages\`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if result, err := GetSitePackages(test.name); err != nil || result != test.expected {
				t.Fatalf("wrong result, expected: `%s` got: `%s`", test.expected, result)
			}
		})
	}
}

func TestGetVersion(t *testing.T) {
	tests := []struct {
		name     string
		expected string
	}{
		{"pip 24.0 from /usr/local/lib/python3.7/site-packages/pip (python 3.7)", "24.0"},
		{"pip 1.2.3 from /usr/local/lib/python3.7/site-packages/pip (python 3.7)", "1.2.3"},
		{"pip 24.0 from /usr/local/lib/python3.7/site-packages/pip (python 3.3)", "24.0"},
		{`pip 24.0 from C:\Python\site-packages\pip (python 3.7)`, "24.0"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := GetVersion(test.name)
			if result != test.expected {
				t.Fatalf("wrong result, expected: `%s` got: `%s`", test.expected, result)
			}
		})
	}
}
