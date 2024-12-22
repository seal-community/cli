package common

import (
	"archive/zip"
	"os"
	"path/filepath"
	"slices"
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

func TestIsDirEmpty(t *testing.T) {
	target, err := os.MkdirTemp("", "test_seal_cli_*")
	if err != nil {
		panic(err)
	}

	defer os.Remove(target)

	res, err := IsDirEmpty(target)
	if err != nil {
		t.Fatalf("had error %v", err)
	}

	if !res {
		t.Fatalf("expected %v to be empty", target)
	}

	// create a file in target
	p := filepath.Join(target, "file.txt")
	f, err := os.Create(p)
	if err != nil {
		panic(err)
	}

	f.Close()

	res, err = IsDirEmpty(target)
	if err != nil {
		t.Fatalf("had error %v", err)
	}

	if res {
		t.Fatalf("expected %v to not be empty", target)
	}
}

func TestDumpBytes(t *testing.T) {
	d, err := os.MkdirTemp("", "test_seal_cli_*")
	if err != nil {
		panic(err)
	}

	defer os.Remove(d)

	content := "as;lfal1"
	fpath := filepath.Join(d, "f.bin")

	if err := DumpBytes(fpath, []byte(content)); err != nil {
		t.Fatalf("failed dump bytes %v", err)
	}

	data, err := os.ReadFile(fpath)
	if err != nil {
		t.Fatalf("failed read file %v", err)
	}

	datastr := string(data)
	if datastr != content {
		t.Fatalf("failed wrong content got `%s` expected `%s`", data, content)
	}
}

func TestDumpBytesExists(t *testing.T) {
	d, err := os.MkdirTemp("", "test_seal_cli_*")
	if err != nil {
		panic(err)
	}

	defer os.Remove(d)

	content := "as;lfal1"
	contentNew := "123123"
	fpath := filepath.Join(d, "f.bin")

	if err := DumpBytes(fpath, []byte(content)); err != nil {
		t.Fatalf("failed dump bytes %v", err)
	}

	if err := DumpBytes(fpath, []byte(contentNew)); err != nil {
		t.Fatalf("failed dump bytes %v", err)
	}

	dataNew, err := os.ReadFile(fpath)
	if err != nil {
		t.Fatalf("failed read file %v", err)
	}

	datastrNew := string(dataNew)
	if datastrNew != contentNew {
		t.Fatalf("failed wrong content got `%s` expected `%s`", datastrNew, contentNew)
	}
}

func TestListDir(t *testing.T) {
	d, err := os.MkdirTemp("", "test_seal_cli_*")
	if err != nil {
		panic(err)
	}

	defer os.Remove(d)

	content := "as;lfal1"
	fname := "f.bin"
	dname := "dd"
	fpath := filepath.Join(d, fname)
	dpath := filepath.Join(d, dname)

	if err := DumpBytes(fpath, []byte(content)); err != nil {
		panic(err)
	}

	if err := os.Mkdir(dpath, os.ModePerm); err != nil {
		panic(err)

	}

	entries, err := ListDir(d)
	if err != nil {
		t.Fatalf("failed: %v", err)
	}

	if !slices.Equal(entries, []string{dname, fname}) {
		t.Fatalf("failed list: %v", entries)
	}
}
