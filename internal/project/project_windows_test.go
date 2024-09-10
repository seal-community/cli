//go:build windows

package project

import (
	"testing"
)

func TestFormatProjectIdPathWindows(t *testing.T) {
	payload := formatProjectIdFallback("myproj", "src\\requirements.txt", "my-proj")
	if payload != "myproj/src/requirements.txt/my-proj" {
		t.Fatalf("got bad payload: `%s`", payload)
	}
}

func TestFormatProjectIdRepoPathWindows(t *testing.T) {
	payload := formatProjectIdForRepo("path\\to\\src\\requirements.txt", "https://github.com/seal-community/cli")
	if payload == "" {
		t.Fatalf("failed -  pid: %s", payload)
	}

	if payload != "seal-community/cli/path/to/src/requirements.txt" {
		t.Fatalf("wrong project id %s", payload)
	}
}
