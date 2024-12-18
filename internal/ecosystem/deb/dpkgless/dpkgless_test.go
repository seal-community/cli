//go:build !windows

package dpkgless

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseControlFile(t *testing.T) {
	content, err := os.ReadFile(filepath.Join("testdata", "controlfile.txt"))
	if err != nil {
		t.Fatalf("failed to open file: %v", err)
	}

	name, version, arch, err := parseControlFile(content)
	if err != nil {
		t.Fatalf("failed to parse control file: %v", err)
	}

	if name != "zlib1g" || version != "1:1.2.11.dfsg-2+deb11u2+sp1" || arch != "amd64" {
		t.Fatalf("got %s %s %s, expected %s %s %s", name, version, arch, "zlib1g", "1:1.2.11.dfsg-2+deb11u2+sp1", "amd64")
	}
}

func TestGetControlQueryLine(t *testing.T) {
	content, err := os.ReadFile(filepath.Join("testdata", "controlfile.txt"))
	if err != nil {
		t.Fatalf("failed to open file: %v", err)
	}

	packageLine, err := getControlFormattedLine(content)
	if err != nil {
		t.Fatalf("failed to parse control file: %v", err)
	}

	if packageLine != "zlib1g 1:1.2.11.dfsg-2+deb11u2+sp1 amd64 install ok installed\n" {
		t.Fatalf("got %s", packageLine)
	}
}

func TestGetPackageName(t *testing.T) {
	content, err := os.ReadFile(filepath.Join("testdata", "controlfile.txt"))
	if err != nil {
		t.Fatalf("failed to open file: %v", err)
	}

	name := getPackageNameFromControlFile(content)
	if err != nil {
		t.Fatalf("failed to parse control file: %v", err)
	}

	if name != "zlib1g" {
		t.Fatalf("got %s", name)
	}
}

func TestGetFilesListFromHashesFile(t *testing.T) {
	file, err := os.Open(filepath.Join("testdata", "md5sums.txt"))
	if err != nil {
		t.Fatalf("failed to open file: %v", err)
	}

	filesList, err := getFilesListFromHashesFile(file)
	if err != nil {
		t.Fatalf("failed to parse hashes file: %v", err)
	}

	if len(filesList) != 4 {
		t.Fatalf("expected array of 4, got %v", filesList)
	}

	expected := map[string]bool{
		"/lib/x86_64-linux-gnu/libz.so.1.2.11":      true,
		"/usr/share/doc/zlib1g/changelog.Debian.gz": true,
		"/usr/share/doc/zlib1g/changelog.gz":        true,
		"/usr/share/doc/zlib1g/copyright":           true,
	}

	for _, file := range filesList {
		if _, ok := expected[file]; !ok {
			t.Fatalf("unexpected file %s", file)
		}
	}
}
