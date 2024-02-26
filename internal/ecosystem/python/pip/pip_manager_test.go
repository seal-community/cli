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

	found, err := GetPythonIndicatorFile(target)
	if err != nil {
		t.Fatalf("had error %v", err)
	}

	if found != "" {
		t.Fatal("detected pip")
	}
}

func TestPipManagerDetectionRequirementsFile(t *testing.T) {
	target, err := os.MkdirTemp("", "test_seal_cli_*")
	if err != nil {
		panic(err)
	}
	defer os.Remove(target)

	for _, indicator := range pythonIndicators {
		p := filepath.Join(target, indicator)
		f, err := os.Create(p)
		if err != nil {
			panic(err)
		}
		f.Close()

		func() {
			defer os.Remove(p)
			found, err := GetPythonIndicatorFile(target)
			if err != nil {
				t.Fatalf("had error %v", err)
			}

			if found != indicator {
				t.Fatalf("did not detect pip %v, found %v", indicator, found)
			}
		}()
	}
}
