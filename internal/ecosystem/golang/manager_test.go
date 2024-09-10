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

func TestIsGolangIndicatorFileWrong(t *testing.T) {
	if IsGolangIndicatorFile("/src/test/xdxd/requirements.txt") {
		t.Fatal("wrongfully detected as golang indicator")
	}
}

func TestIsGolangIndicatorFileWindowsWrong(t *testing.T) {
	if IsGolangIndicatorFile("C:\\x.y") {
		t.Fatal("wrongfully detected as golang indicator")
	}
}

func TestIsGolangIndicatorFile(t *testing.T) {
	if !IsGolangIndicatorFile("/src/test/xdxd/go.mod") {
		t.Fatal("didnt detected as golang indicator")
	}
}

func TestIsGolangIndicatorFileWindows(t *testing.T) {
	if !IsGolangIndicatorFile("C:\\go.mod") {
		t.Fatal("didnt detected as golang indicator")
	}
}
