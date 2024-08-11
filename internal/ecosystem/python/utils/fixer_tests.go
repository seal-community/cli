//go:build !windows

package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestGetSourceName(t *testing.T) {
	// read tar.gz package from testsdata and verify the result matches
	p := filepath.Join("testdata", "six-1.16.0.tar.gz")
	data, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("failed %v", err)
	}

	name, err := getSourceName(data)
	if err != nil {
		t.Fatalf("failed %v", err)
	}

	if name != "six-1.16.0" {
		t.Fatalf("got wrong name %v", name)
	}
}

func TestFindSitePackagesInfo(t *testing.T) {
	sitePackagesPath := t.TempDir()
	sixName := "six-1.16.0"
	sixPath := fmt.Sprintf("%s.dist-info", sixName)

	err := os.MkdirAll(filepath.Join(sitePackagesPath, sixPath), os.ModeDir)
	if err != nil {
		t.Fatalf("failed %v", err)
	}

	info, err := findSitePackagesInfo(sitePackagesPath, sixName)
	if err != nil {
		t.Fatalf("failed %v", err)
	}

	if info != sixPath {
		t.Fatalf("got wrong path %v", info)
	}
}

func TestFindSitePackagesInfoEggInfo(t *testing.T) {
	sitePackagesPath := t.TempDir()
	sixName := "six-1.16.0"
	sixPath := fmt.Sprintf("%s-py3.5.egg-info", sixName)

	err := os.MkdirAll(filepath.Join(sitePackagesPath, sixPath), os.ModeDir)
	if err != nil {
		t.Fatalf("failed %v", err)
	}

	info, err := findSitePackagesInfo(sitePackagesPath, sixName)
	if err != nil {
		t.Fatalf("failed %v", err)
	}

	if info != sixPath {
		t.Fatalf("got wrong path %v", info)
	}
}
