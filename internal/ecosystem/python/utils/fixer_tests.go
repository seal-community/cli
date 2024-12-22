//go:build !windows

package utils

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetSourceName(t *testing.T) {
	// read tar.gz package from testsdata and verify the result matches
	p := filepath.Join("testdata", "six-1.16.0.tar.gz")
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("failed %v", err)
	}

	name, err := getSourceName(data)
	if err != nil {
		t.Fatalf("failed %v", err)
	}

	if name != "six-1.16.0" {
		t.Fatalf("got wrong name %v", name)
	}
}
