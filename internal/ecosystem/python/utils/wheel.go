package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func EscapePackageName(name string) string {
	return strings.ReplaceAll(name, "-", "_")
}

func DistInfoPath(name string, version string) string {
	return fmt.Sprintf("%s-%s.dist-info", EscapePackageName(name), version)
}

func isEggInfoPath(path string, name string, version string) bool {
	return strings.HasPrefix(path, name+"-"+version) && strings.HasSuffix(path, ".egg-info")
}

func FindEggInfoPath(sitePackages string, name string, version string) string {
	entries, err := os.ReadDir(sitePackages)
	if err != nil {
		return ""
	}
	for _, entry := range entries {
		if isEggInfoPath(entry.Name(), EscapePackageName(name), version) {
			return filepath.Join(sitePackages, entry.Name())
		}
	}
	return ""
}
