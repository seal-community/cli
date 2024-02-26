package pip

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPipManagerDetectionNoRequirementsFile(t *testing.T) {
	target, err := os.MkdirTemp("", "test_seal_cli_*")
	if err != nil {
		panic(err)
	}
	defer os.Remove(target)

	found, err := IsPipProjectDir(target)
	if err != nil {
		t.Fatalf("had error %v", err)
	}

	if found {
		t.Fatal("detected pip")
	}
}

func TestPipManagerDetectionRequirementsFile(t *testing.T) {
	target, err := os.MkdirTemp("", "test_seal_cli_*")
	if err != nil {
		panic(err)
	}
	defer os.Remove(target)

	p := filepath.Join(target, Pipfile)
	f, err := os.Create(p)
	if err != nil {
		panic(err)
	}
	f.Close()
	defer os.Remove(p)

	found, err := IsPipProjectDir(target)
	if err != nil {
		t.Fatalf("had error %v", err)
	}

	if !found {
		t.Fatal("did not detect pip")
	}
}
