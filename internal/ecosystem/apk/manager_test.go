package apk

import (
	"testing"
)

func TestIsDistroSupported(t *testing.T) {
	if !isDistroSupported("alpine") {
		t.Fatalf("centos should be supported")
	}
}
