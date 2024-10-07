package composer

import (
	"testing"
)

func TestGetDistPathWindows(t *testing.T) {
	if getDiskPath("dir", "vendor/package") != "dir\\vendor\\vendor\\package" {
		t.Fatalf("wrong disk path")
	}
}
