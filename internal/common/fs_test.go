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

func TestFindPathsWithSuffix(t *testing.T) {
	target, err := os.MkdirTemp("", "test_seal_cli_*")
	if err != nil {
		panic(err)
	}
	
	defer os.Remove(target)

	inner_target, err := os.MkdirTemp(target, "test_seal_inner_dir_*")
	if err != nil {
		panic(err)
	}

	// create files in target with suffixes
	suffixes := []string{".txt", ".json", ".xml"}
	for _, suffix := range suffixes {
		p := filepath.Join(target, "file"+suffix)
		f, err := os.Create(p)
		if err != nil {
			panic(err)
		}

		f.Close()

		p = filepath.Join(inner_target, "file"+suffix)
		f, err = os.Create(p)
		if err != nil {
			panic(err)
		}

		f.Close()
	}

	for _, suffix := range suffixes {
		paths, err := FindPathsWithSuffix(target, suffix)
		if err != nil {
			t.Fatalf("had error %v", err)
		}

		if len(paths) != 2 {
			t.Fatalf("expected 2 paths, got %v", len(paths))
		}

		for _, path := range paths {
			if !strings.HasSuffix(strings.ToLower(path), strings.ToLower(suffix)) {
				t.Fatalf("expected %v to have suffix %v", path, suffix)
			}
		}
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
