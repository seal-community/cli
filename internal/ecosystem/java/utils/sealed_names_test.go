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
			result := getSilencedManifest(inFile, test.artifactId)
			resultData, _ := io.ReadAll(result)
			expected, _ := io.ReadAll(getTestFile(test.filename + ".EXPECTED"))
			expectedString := strings.ReplaceAll(string(expected), "\r", "")
			if string(resultData) != expectedString {
				t.Fatalf("wrong result for %s, expected: `%s` got: `%s`", test.filename, expected, resultData)
			}
		})
	}
}

func TestGetSealedPomXML(t *testing.T) {
	inFile := getTestFile("pom.xml")
	result := getSilencedPomXML(inFile)
	resultData, _ := io.ReadAll(result)
	expected, _ := io.ReadAll(getTestFile("pom.xml.expected"))
	expectedString := strings.ReplaceAll(string(expected), "\r", "")
	resultString := strings.ReplaceAll(string(resultData), "\r", "")
	if resultString != expectedString {
		t.Fatalf("wrong result, expected: `%s` got: `%s`", expected, resultData)
	}
}