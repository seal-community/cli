package composer

import (
	"cli/internal/common"
	"cli/internal/ecosystem/mappings"
	"cli/internal/ecosystem/shared"
	"encoding/json"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
)

func parseComposerVersion(composerVersionOutput string) string {
	version := strings.Split(composerVersionOutput, " ")[2]
	return version
}

func ParseComposerDependencies(dependencyTreeString string, targetDir string) (common.DependencyMap, error) {
	// When there are no dependencies, the output is an empty array
	var emptyDependencies []interface{}
	if err := json.Unmarshal([]byte(dependencyTreeString), &emptyDependencies); err == nil {
		slog.Warn("empty composer dependencies")
		return common.DependencyMap{}, nil
	}

	var composerDependencyTree ComposerDependencyList
	err := json.Unmarshal([]byte(dependencyTreeString), &composerDependencyTree)
	if err != nil {
		slog.Error("failed unmarshalling `composer show` output", "err", err, "output", dependencyTreeString)
		return nil, err
	}

	deps := make(common.DependencyMap)
	for _, dep := range composerDependencyTree.Dependencies {
		// check if the package is already sealed
		if err := replaceSealedVersion(&dep, targetDir); err != nil {
			slog.Error("failed to replace sealed version in dependency", "err", err)
			continue
		}

		depNormalizedName := normalizePackageName(dep.Name)
		newDep := &common.Dependency{
			Name:           dep.Name,
			NormalizedName: depNormalizedName,
			Version:        normalizePackageVersion(dep.Version),
			PackageManager: mappings.ComposerManager,
			// the disk path of a composer package always uses the lower case name
			DiskPath: getDiskPath(targetDir, depNormalizedName),
		}
		key := newDep.Id()
		deps[key] = []*common.Dependency{newDep}
	}

	return deps, nil
}

func replaceSealedVersion(dependency *ComposerDependency, targetDir string) error {
	// if we stored the sealed version, we should use it
	metadataPath := getMetadataDepFile(targetDir, dependency.Name)
	sealMetadata, err := shared.LoadPackageSealMetadata(metadataPath)
	// no error if the file does not exist
	if err != nil {
		return err
	}

	if sealMetadata != nil {
		slog.Info("using sealed version from metadata", "packageInfo", fmt.Sprintf("%s=%s", dependency.Name, sealMetadata.SealedVersion))
		dependency.Version = sealMetadata.SealedVersion
	}

	return nil
}

func getDiskPath(targetDir string, dependencyName string) string {
	depNameParts := strings.SplitN(dependencyName, "/", 2)
	depVendor, depProjectName := depNameParts[0], depNameParts[1]
	return filepath.Join(targetDir, composerModulesDirName, depVendor, depProjectName)
}
