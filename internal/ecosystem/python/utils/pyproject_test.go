package utils

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetProjectNameSanity(t *testing.T) {
	tmp := t.TempDir()
	// copy test data to temp dir
	src, err := os.ReadFile("testdata/pyproject.toml")
	if err != nil {
		t.Fatal(err)
	}

	dst := filepath.Join(tmp, "pyproject.toml")
	err = os.WriteFile(dst, src, os.ModePerm)
	if err != nil {
		t.Fatal(err)
	}

	pyproj := GetPyprojectProjectName(tmp)
	if pyproj != "test_project" {
		t.Fatal("failed to load pyproject.toml")
	}
}

func TestGetProjectNamePoetry(t *testing.T) {
	tmp := t.TempDir()
	// copy test data to temp dir
	src, err := os.ReadFile("testdata/poetry_pyproject.toml")
	if err != nil {
		t.Fatal(err)
	}

	dst := filepath.Join(tmp, "pyproject.toml")
	err = os.WriteFile(dst, src, os.ModePerm)
	if err != nil {
		t.Fatal(err)
	}

	pyproj := GetPyprojectProjectName(tmp)
	if pyproj != "poetry_project" {
		t.Fatal("failed to load pyproject.toml")
	}
}

func TestGetProjectNameNoFile(t *testing.T) {
	tmp := t.TempDir()

	pyproj := GetPyprojectProjectName(tmp)
	if pyproj != "" {
		t.Fatal("failed to load pyproject.toml")
	}
}

func TestGetProjectNameNoName(t *testing.T) {
	tmp := t.TempDir()
	// copy test data to temp dir
	src, err := os.ReadFile("testdata/noname_pyproject.toml")
	if err != nil {
		t.Fatal(err)
	}

	dst := filepath.Join(tmp, "pyproject.toml")
	err = os.WriteFile(dst, src, os.ModePerm)
	if err != nil {
		t.Fatal(err)
	}

	pyproj := GetPyprojectProjectName(tmp)
	if pyproj != "" {
		t.Fatal("failed to load pyproject.toml")
	}
}

func TestGetProjectNameProjectWrongType(t *testing.T) {
	tmp := t.TempDir()
	// copy test data to temp dir
	src, err := os.ReadFile("testdata/wrong_type_pyproject.toml")
	if err != nil {
		t.Fatal(err)
	}

	dst := filepath.Join(tmp, "pyproject.toml")
	err = os.WriteFile(dst, src, os.ModePerm)
	if err != nil {
		t.Fatal(err)
	}

	pyproj := GetPyprojectProjectName(tmp)
	if pyproj != "" {
		t.Fatal("failed to load pyproject.toml")
	}
}
