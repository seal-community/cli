package pip

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
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

func TestParseWheelTags(t *testing.T) {
	tests := []struct {
		name     string
		expected []string
	}{
		{`Wheel-Version: 1.0
Generator: bdist_wheel (0.40.0)
Root-Is-Purelib: false
Tag: cp38-cp38-manylinux_2_17_x86_64
Tag: cp38-cp38-manylinux2014_x86_64`, []string{"cp38-cp38-manylinux_2_17_x86_64", "cp38-cp38-manylinux2014_x86_64"}},
		{`Wheel-Version: 1.9
Generator: bdist_wheel 1.9
Root-Is-Purelib: true
Tag: py2-none-any
Tag: py3-none-any
Build: 1
Install-Paths-To: wheel/_paths.py
Install-Paths-To: wheel/_paths.json
`, []string{"py2-none-any", "py3-none-any"}},
		// Windows newlines
		{"Wheel-Version: 1.9\r\nGenerator: bdist_wheel 1.9\r\nRoot-Is-Purelib: true\r\nTag: py2-none-any\r\nTag: py3-none-any\r\nBuild: 1\r\nInstall-Paths-To: wheel/_paths.py\r\nInstall-Paths-To: wheel/_paths.json\r\n", []string{"py2-none-any", "py3-none-any"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := parseWheelTags(test.name)
			if !reflect.DeepEqual(result, test.expected) {
				t.Fatalf("wrong result, expected: `%s` got: `%s`", test.expected, result)
			}
		})
	}
}

func TestIndicatorMatches(t *testing.T) {
	ps := []string{"/b/poetry.lock", "/b/pipfile.lock", "/b/requirements.txt", "/b/pyproject.toml", "/b/pipfile"}

	for i, p := range ps {
		t.Run(fmt.Sprintf("pth_%d", i), func(t *testing.T) {
			if !IsPythonIndicatorFile(p) {
				t.Fatalf("failed to detect indicator path `%s`", p)
			}
		})
	}
}
