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
		groupId    string
	}{
		{"MANIFEST_WITH_SYMBOLIC.MF", "jackson-databind", "com.fasterxml.jackson.core"},
		{"MANIFEST_WITHOUT_SYMBOLIC.MF", "jackson-databind", "com.fasterxml.jackson.core"},
	}
	for _, test := range tests {
		t.Run(test.artifactId, func(t *testing.T) {
			inFile := getTestFile(test.filename)
			result := getSilencedManifest(inFile, test.artifactId, test.groupId)
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

func TestGetTempJarFile(t *testing.T) {
	origFile, err := os.CreateTemp("", "test")
	if err != nil {
		t.Fatalf("failed creating temp file: %v", err)
	}
	defer os.Remove(origFile.Name())

	tempJarFile, err := getTempJarFile(origFile.Name())
	if err != nil {
		t.Fatalf("failed getting temp jar file: %v", err)
	}

	tmpFileStat, err := origFile.Stat()
	if err != nil {
		t.Fatalf("failed getting temp jar file stat: %v", err)
	}

	tempJarFileStat, err := tempJarFile.Stat()
	if err != nil {
		t.Fatalf("failed getting temp jar file stat: %v", err)
	}

	if tmpFileStat.Mode() != tempJarFileStat.Mode() {
		t.Fatalf("wrong mode, expected: %v got: %v", tmpFileStat.Mode(), tempJarFileStat.Mode())
	}

	origJarPath := origFile.Name()

	tmpJarPath := tempJarFile.Name()
	perIdx := strings.LastIndex(tmpJarPath, ".")
	if perIdx == -1 {
		t.Fatalf("bad path %s", tmpJarPath)
	}

	supposedorigPath := tmpJarPath[:perIdx]
	tmpSfx := tmpJarPath[perIdx:]
	if supposedorigPath != origJarPath {
		t.Fatalf("wrong temp path, orig jar: %s new: %s", origJarPath, tmpJarPath)
	}

	if tmpSfx == "" {
		t.Fatalf("bad temp path %s - sfx %s", tmpJarPath, tmpSfx)
	}

}

func TestGetSealedGroupId(t *testing.T) {
	groupIdToSeal := "com.fasterxml.jackson.core"
	expectedSealedGroupId := sealGroupId + "." + groupIdToSeal
	resultSealedGroupId := getSealedGroupId(groupIdToSeal)
	if resultSealedGroupId != expectedSealedGroupId {
		t.Fatalf("wrong result, expected: `%s` got: `%s`", resultSealedGroupId, expectedSealedGroupId)
	}
}
