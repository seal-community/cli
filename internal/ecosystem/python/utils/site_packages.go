package utils

import (
	"cli/internal/common"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
)

// the normalized base name in format matching entries in site-packages
// normalizes both name and version for comparison when looking in site-packages directory
func formatEscapedPackageSitePackages(name, version string) string {
	return NormalizePackageName(fmt.Sprintf("%s-%s", name, version))
}

func findMatchingDistInfoOrEggInfoFolder(dirEntries []string, name, version string) string {
	depEscBase := formatEscapedPackageSitePackages(name, version)
	slog.Debug("searching for escaped entry in site packages items", "target", depEscBase, "len", len(dirEntries))

	for _, name := range dirEntries {
		ext := filepath.Ext(name)
		basename := strings.TrimSuffix(name, ext)
		escapedBase := NormalizePackageName(basename)

		common.Trace("checking entry in site package", "name", name, "escaped", escapedBase, "ext", ext)
		if ext == ".dist-info" && escapedBase == depEscBase {
			slog.Debug("found dist info", "name", name)
			return name
		}

		if ext == ".egg-info" && strings.HasPrefix(escapedBase, depEscBase) {
			slog.Debug("found egg info", "name", name)
			return name
		}
	}

	slog.Debug("did not find any match in site packages", "target", depEscBase)
	return ""
}

// Since a python dependency defaults to the dist-info disk path, we need to check if it exists
// and in the low chance it doesn't, and there's an egg-info instead, we should replace
// the disk path value to the egg-info path.
func FindSitePackagesFolderForPackage(sitePackages string, name string, version string) (string, error) {
	entries, err := common.ListDir(sitePackages)
	if err != nil {
		slog.Error("failed listing site packages dir", "path", sitePackages)
		return "", err
	}

	entry := findMatchingDistInfoOrEggInfoFolder(entries, name, version)
	if entry == "" {
		slog.Error("failed finding disk path", "name", name, "version", version)
		return "", fmt.Errorf("failed finding disk path for %s@%s", name, version)
	}

	diskPath := filepath.Join(sitePackages, entry)
	exists, err := common.PathExists(diskPath)
	if err != nil {
		slog.Error("failed checking exists", "path", diskPath)
		return "", err
	}

	if !exists {
		slog.Error("diskpath for dist-info or egg-info is wrong", "path", diskPath)
		return "", fmt.Errorf("discovered new path does not exist: %s", diskPath)
	}

	return diskPath, nil
}
