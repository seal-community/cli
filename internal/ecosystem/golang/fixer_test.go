package golang

import (
	"testing"
)

func TestRemoveVersionPath(t *testing.T) {
	path := "github.com/Masterminds/goutils@v1.1.0-sp1/wordutils.go"
	version := "1.1.0-sp1"
	expected := "github.com/Masterminds/goutils/wordutils.go"
	actual := removeVersionPath(path, version)
	if actual != expected {
		t.Fatalf("expected %s, got %s", expected, actual)
	}
}
