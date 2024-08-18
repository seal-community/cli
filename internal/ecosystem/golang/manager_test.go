package golang

import (
	"testing"
)

func TestIsVersionSupported(t *testing.T) {
	m := &GolangPackageManager{}
	if !m.IsVersionSupported("1.17") {
		t.Fatalf("expected true")
	}

	if m.IsVersionSupported("1.16") {
		t.Fatalf("expected false")
	}
}
