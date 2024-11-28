package rpm

import (
	"testing"
)

func TestIsDistroSupported(t *testing.T) {
	if !isDistroSupported("centos") {
		t.Fatalf("centos should be supported")
	}

	if !isDistroSupported("rhel") {
		t.Fatalf("rhel should be supported")
	}

	if isDistroSupported("ubuntu") {
		t.Fatalf("ubuntu should not be supported")
	}
}
