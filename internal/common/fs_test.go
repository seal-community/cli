package common

import (
	"os"
	"path/filepath"
	"strings"

	"testing"
)

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
