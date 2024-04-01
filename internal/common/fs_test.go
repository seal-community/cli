package common

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strings"

	"testing"
)

func getTestFile(name string) string {
	// fetch file from current package's testdata folder
	// ref: https://pkg.go.dev/cmd/go/internal/test
	p := filepath.Join("testdata", name)
	data, err := os.ReadFile(p)
	if err != nil {
		panic(err)
	}

	return string(data)
}

func TestFindFileWithSuffix(t *testing.T) {
	target, err := os.MkdirTemp("", "test_seal_dir_*")
	if err != nil {
		panic(err)
	}
	defer os.Remove(target)

	inner_target, err := os.MkdirTemp(target, "test_seal_inner_dir_*")
	if err != nil {
		panic(err)
	}
	defer os.Remove(inner_target)

	for _, suffixIndicator := range []string{"lock", "txt", "toml"} {
		currentFilename := "testfile." + suffixIndicator
		p := filepath.Join(inner_target, currentFilename)
		f, err := os.Create(p)
		if err != nil {
			panic(err)
		}
		f.Close()

		found, err := FindFileWithSuffix(target, suffixIndicator)
		if err != nil {
			t.Fatalf("had error %v", err)
		}

		if filepath.Base(found) != currentFilename {
			t.Fatalf("did not detect file %v, found %v", currentFilename, found)
		}
	}

	_, err = FindFileWithSuffix(target, "notfound")
	if err == nil {
		t.Fatalf("expected error")
	}

	if !strings.Contains(err.Error(), "no file found with suffix") {
		t.Fatalf("expected error")
	}
}

func TestUnzipFiles(t *testing.T) {
	target, err := os.MkdirTemp("", "test_seal_cli_*")
	if err != nil {
		panic(err)
	}
	defer os.Remove(target)

	// unzip 'testdata/test.zip' to target using UnzipFile
	zipFile := getTestFile("test.zip")
	zipReader, err := zip.NewReader(strings.NewReader(zipFile), int64(len(zipFile)))
	if err != nil {
		panic(err)
	}

	for _, file := range zipReader.File {
		err = UnzipFile(file, target)
		if err != nil {
			t.Fatalf("had error %v", err)
		}
	}

	expectedPaths := []string{
		"a",
		filepath.Join("a", "k"),
		filepath.Join("a", "b", "c"),
		"i",
		"j",
		"fileB.txt",
		filepath.Join("i", "fileC.txt"),
		filepath.Join("a", "k", "fileA.txt"),
	}
	for _, expectedPath := range expectedPaths {
		p := filepath.Join(target, expectedPath)
		_, err := os.Stat(p)
		if err != nil {
			t.Fatalf("expected %v to exist", p)
		}
	}
}
