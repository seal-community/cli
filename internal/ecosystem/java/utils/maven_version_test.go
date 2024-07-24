package utils

import "testing"
import "fmt"

func TestGetMavenVersion(t *testing.T) {
	tests := []struct {
		versionLine string
		expected    string
	}{
		{"", ""},
		{"garbage stirng here", ""},
		{"Apache Maven 3.9.6 (asdasfasd)\nMaven home: /tmp", "3.9.6"},
		{"Apache Maven 3.6.3 (asdasfasd)\nMaven home: /tmp", "3.6.3"},
		{"Apache Maven 4.0.0-alpha-13 (0a6a5617fe5e)", "4.0.0-alpha-13"},
		{"Apache Maven 3.1.2  (cccccc)\nMaven home: /tmp", "3.1.2"},
		{"\x1b[1mApache Maven 3.1.2  (cccccc)\nMaven home: /tmp", "3.1.2"},
		{"\x1b[1mApache Maven 3.1.2\r\n  (cccccc)\nMaven home: /tmp", "3.1.2"},
		{"\x1b[1mApache Maven 3.1.2\n  (cccccc)\nMaven home: /tmp", "3.1.2"},
		{"\x1b[1mApache Maven 3.1.2\x1b[m\nMaven home: /tmp", "3.1.2"},
		{"\x1b[1mApache Maven 3.1.2\x1b[m\r\nMaven home: /tmp", "3.1.2"},
	}

	for idx, test := range tests {
		t.Run(fmt.Sprintf("%d", idx), func(t *testing.T) {
			if result := parseMavenVersion(test.versionLine); result != test.expected {
				t.Fatalf("wrong result, expected: `%s` got: `%s`", test.expected, result)
			}
		})
	}
}
