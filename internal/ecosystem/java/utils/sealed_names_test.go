package utils

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func getTestFile(name string) io.ReadCloser {
	// fetch file from current package's testdata folder
	// ref: https://pkg.go.dev/cmd/go/internal/test
	p := filepath.Join("testdata", name)
	data, err := os.ReadFile(p)
	if err != nil {
		panic(err)
	}

	return io.NopCloser(bytes.NewReader(data))
}

func TestGetSealedPomProperties(t *testing.T) {
	tests := []struct {
		artifactId string
		version    string
		expected   string
	}{
		{"artifact", "1.2.3", "artifactId=artifact\ngroupId=seal\nversion=1.2.3\n"},
		{"artifact", "1.2.3-SNAPSHOT", "artifactId=artifact\ngroupId=seal\nversion=1.2.3-SNAPSHOT\n"},
	}
	for _, test := range tests {
		t.Run(test.artifactId, func(t *testing.T) {
			result := getSealedPomProperties(test.artifactId, test.version)
			resultBytes, _ := io.ReadAll(result)
			if string(resultBytes) != test.expected {
				t.Fatalf("wrong result, expected: `%s` got: `%s`", test.expected, result)
			}
		})
	}
}

func TestGetSealedManifest(t *testing.T) {
	tests := []struct {
		filename   string
		artifactId string
	}{
		{"MANIFEST_WITH_SYMBOLIC.MF", "jackson-databind"},
		{"MANIFEST_WITHOUT_SYMBOLIC.MF", "jackson-databind"},
	}
	for _, test := range tests {
		t.Run(test.artifactId, func(t *testing.T) {
			inFile := getTestFile(test.filename)
			result := getSealedManifest(inFile, test.artifactId)
			resultData, _ := io.ReadAll(result)
			expected, _ := io.ReadAll(getTestFile(test.filename + ".EXPECTED"))
			expectedString := strings.ReplaceAll(string(expected), "\r", "")
			if string(resultData) != expectedString {
				t.Fatalf("wrong result for %s, expected: `%s` got: `%s`", test.filename, expected, resultData)
			}
		})
	}
}
