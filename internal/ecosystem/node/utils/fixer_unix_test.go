//go:build !windows

package utils

import (
	"testing"
)

func TestTempPathForDep(t *testing.T) {
	expected := "/fuwawa/proj/.seal/node_modules/semver-regex"
	newPath, err := getDepRollbackDir("/fuwawa/proj", "/fuwawa/proj/.seal", "/fuwawa/proj/node_modules/semver-regex")
	if err != nil {
		t.Fatalf("failed generating temp path for dep %s", err)
	}

	if newPath != expected {
		t.Fatalf("expected %s; got %s", expected, newPath)
	}
}
func TestTempPathForDepTrailingSlash(t *testing.T) {
	expected := "/fuwawa/proj/.seal/node_modules/semver-regex"
	newPath, err := getDepRollbackDir("/fuwawa/proj/", "/fuwawa/proj/.seal", "/fuwawa/proj/node_modules/semver-regex")
	if err != nil {
		t.Fatalf("failed generating temp path for dep %s", err)
	}
	
	if newPath != expected {
		t.Fatalf("expected %s; got %s", expected, newPath)
	}
}
