package utils

import (
	"fmt"
	"testing"
)

func TestFormatEscapedPackageSitePackages(t *testing.T) {
	if res := formatEscapedPackageSitePackages("multi-part", "1.2.3"); res != "multi-part-1.2.3" {
		t.Fatalf("bad escaped, got `%s`", res)
	}

	if res := formatEscapedPackageSitePackages("multi_part", "1.2.3"); res != "multi-part-1.2.3" {
		t.Fatalf("bad escaped, got `%s`", res)
	}

	if res := formatEscapedPackageSitePackages("MULTI_part", "1.2.3"); res != "multi-part-1.2.3" {
		t.Fatalf("bad escaped, got `%s`", res)
	}
}

func TestFindSitePackagesInfoNotFound(t *testing.T) {

	info := findMatchingDistInfoOrEggInfoFolder([]string{"blah"}, "six", "1.16.0")
	if info != "" {
		t.Fatalf("got wrong path %v", info)
	}
}

func TestFindSitePackagesInfoEmpty(t *testing.T) {

	info := findMatchingDistInfoOrEggInfoFolder(nil, "six", "1.16.0")
	if info != "" {
		t.Fatalf("got wrong path %v", info)
	}
}

func TestFindSitePackagesEscapedName(t *testing.T) {
	name := "Pillow"
	version := "9.3.0"
	folder := "pillow-9.3.0.dist-info"

	info := findMatchingDistInfoOrEggInfoFolder([]string{folder}, name, version)
	if info != folder {
		t.Fatalf("got wrong path %v", info)
	}
}

func TestFindSitePackagesEscapedFolder(t *testing.T) {
	name := "pillow"
	version := "9.3.0"
	folder := "Pillow-9.3.0.dist-info"

	info := findMatchingDistInfoOrEggInfoFolder([]string{folder}, name, version)
	if info != folder {
		t.Fatalf("got wrong path %v", info)
	}
}

func TestFindSitePackagesInfo(t *testing.T) {
	name := "six"
	version := "1.16.0"
	folder := fmt.Sprintf("%s-%s.dist-info", name, version)

	info := findMatchingDistInfoOrEggInfoFolder([]string{folder, "blah"}, name, version)
	if info != folder {
		t.Fatalf("got wrong path %v", info)
	}
}

func TestFindSitePackagesInfoEggInfo(t *testing.T) {
	name := "six"
	version := "1.16.0"
	folder := fmt.Sprintf("%s-%s-py3.5.egg-info", name, version)

	info := findMatchingDistInfoOrEggInfoFolder([]string{folder}, name, version)
	if info != folder {
		t.Fatalf("got wrong path %v", info)
	}
}

func TestFindMatchingDistInfoOrEggInfoFolder(t *testing.T) {
}
