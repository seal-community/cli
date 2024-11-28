package deb

import (
	"testing"
)

func TestIsDistroSupportedSupported(t *testing.T) {
	if !isDistroSupported("ubuntu") {
		t.Fatalf("ubuntu should be supported")
	}

	if !isDistroSupported("debian") {
		t.Fatalf("ubuntu should be supported")
	}
}

func TestIsDistroSupportedUnsupported(t *testing.T) {
	if isDistroSupported("centos") {
		t.Fatalf("centos should not be supported")
	}

	if isDistroSupported("rhel") {
		t.Fatalf("rhel should not be supported")
	}

	if isDistroSupported("") {
		t.Fatalf("empty string should not be supported")
	}
}
