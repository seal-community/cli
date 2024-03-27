package nuget

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNugetManagerDetectionNoNugetFile(t *testing.T) {
	target, err := os.MkdirTemp("", "test_seal_cli_*")
	if err != nil {
		panic(err)
	}
	defer os.Remove(target)

	found, err := GetNugetIndicatorFile(target)
	if err != nil {
		t.Fatalf("had error %v", err)
	}

	if found != "" {
		t.Fatal("detected nuget")
	}
}

func TestNugetManagerDetectionNugetFile(t *testing.T) {
	target, err := os.MkdirTemp("", "test_seal_cli_*")
	if err != nil {
		panic(err)
	}
	defer os.Remove(target)

	for _, suffixIndicator := range nugetSuffixIndicators {
		currentFilename := "testfile" + suffixIndicator
		p := filepath.Join(target, currentFilename)
		f, err := os.Create(p)
		if err != nil {
			panic(err)
		}
		f.Close()

		func() {
			defer os.Remove(p)
			found, err := GetNugetIndicatorFile(target)
			if err != nil {
				t.Fatalf("had error %v", err)
			}

			if filepath.Base(found) != currentFilename {
				t.Fatalf("did not detect nuget %v, found %v", currentFilename, found)
			}
		}()
	}
}
