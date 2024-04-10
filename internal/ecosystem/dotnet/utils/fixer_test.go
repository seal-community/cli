package utils

import (
	"os"
	"path/filepath"
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

func TestSaveFiles(t *testing.T) {
	target, err := os.MkdirTemp("", "test_seal_cli_*")
	if err != nil {
		panic(err)
	}

	defer os.Remove(target)

	inner_target, err := os.MkdirTemp(target, "test_seal_inner_dir_*")
	if err != nil {
		panic(err)
	}

	defer os.Remove(inner_target)

	// create a file inside the inner directory
	currentFilename := "testfile.txt"
	p := filepath.Join(inner_target, currentFilename)
	f, err := os.Create(p)
	if err != nil {
		panic(err)
	}
	
	f.Close()

	packageName := "Snappier.1.1.0-sp1.nupkg"
	nupkgData := []byte(getTestFile("test_package.nupkg"))
	err = savePackageFiles(target, packageName, nupkgData)
	if err != nil {
		t.Fatalf("failed saving package files %v", err)
	}

	// check that the inner target was removed along with its contents
	if _, err := os.Stat(inner_target); err == nil {
		t.Fatalf("expected inner target to be removed")
	}

	// expect target to have this files: COPYING.txt, snappier.nuspec, Snappier.1.1.0-sp1.nupkg, Snappier.1.1.0-sp1.nupkg.sha512
	files, err := os.ReadDir(target)
	if err != nil {
		t.Fatalf("had error %v", err)
	}

	expectedFiles := []string{"COPYING.txt", "snappier.nuspec", "snappier.1.1.0-sp1.nupkg", "snappier.1.1.0-sp1.nupkg.sha512"}
	for _, expectedFile := range expectedFiles {
		found := false
		for _, file := range files {
			if file.Name() == expectedFile {
				found = true
				break
			}
		}

		if !found {
			t.Fatalf("expected file %v not found", expectedFile)
		}
	}
}
