package composer

import (
	"cli/internal/ecosystem/shared"
	"path/filepath"
	"strings"
)

func normalizePackageName(name string) string {
	return strings.TrimSpace(strings.ToLower(name))
}

func normalizePackageVersion(version string) string {
	// DB doesn't store the version with the 'v' prefix
	return strings.TrimLeft(version, "v")
}

func getMetadataDepFile(targetDir string, name string) string {
	// composer package metadatas are stored in vendor/<vendor>/<name>/.seal-metadata
	nameParts := strings.SplitN(name, "/", 2)
	vendor, name := nameParts[0], nameParts[1]
	return filepath.Join(targetDir, composerModulesDirName, vendor, name, shared.SealMetadataFileName)
}

type ComposerDependency struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type ComposerDependencyList struct {
	Dependencies []ComposerDependency `json:"locked"`
}

const composerModulesDirName = "vendor"
