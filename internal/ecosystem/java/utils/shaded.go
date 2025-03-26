package utils

import (
	"archive/zip"
	"cli/internal/common"
	"cli/internal/ecosystem/mappings"
	"cli/internal/ecosystem/shared"
	"fmt"
	"log/slog"
	"maps"
	"path/filepath"
)

type shadedDependency struct {
	name    string
	version string
}

// Given a jar path, find the shaded dependencies in the jar
// Returns 2 sets: one for dependencies found via pom.xml and one for dependencies found via pom.properties
// Note: This will find the shading dependency too.
func findShadedDependenciesFromJar(jarPath string) (map[shadedDependency]bool, map[shadedDependency]bool, error) {
	pomXmlDeps := make(map[shadedDependency]bool, 0)
	pomPropertiesDeps := make(map[shadedDependency]bool, 0)

	origReader, err := zip.OpenReader(jarPath)
	if err != nil {
		slog.Error("failed reading package", "err", err, "path", jarPath)
		return nil, nil, err
	}
	defer origReader.Close()

	for _, zipItem := range origReader.File {
		header := zipItem.FileHeader

		currFilePath := filepath.ToSlash(header.Name)
		currFileName := filepath.Base(header.Name)

		if currFileName != PomXMLFileName && currFileName != PomPropertiesFileName {
			// we skip non pom files here to make sure we're not trying to open zip items
			// that we don't need to.
			// it previously failed on a file with `ErrFormat` originated from:
			// https://cs.opensource.google/go/go/+/refs/tags/go1.23.4:src/archive/zip/reader.go;l=338
			slog.Debug("skipping non pom file", "path", currFilePath)
			continue
		}

		zipItemReader, err := zipItem.Open()
		if err != nil {
			slog.Error("failed opening zip item", "err", err, "path", zipItem.Name)
			return nil, nil, err
		}
		defer zipItemReader.Close()

		if currFileName == PomXMLFileName {
			slog.Debug("found pom file", "path", currFilePath)
			pom := ReadPomXMLFromFile(zipItemReader)
			if pom == nil {
				slog.Warn("failed reading pom file", "path", currFilePath)
				continue
			}

			dep := shadedDependency{
				name:    fmt.Sprintf("%s:%s", pom.GetGroupId(), pom.GetArtifactId()),
				version: pom.GetVersion(),
			}
			pomXmlDeps[dep] = true
			slog.Info("found shaded dependencies from pom.xml", "package", dep)

		} else if currFileName == PomPropertiesFileName {
			slog.Debug("found pom properties file", "path", currFilePath)
			pomProperties := ReadPomPropertiesFromFile(zipItemReader)
			if pomProperties == nil {
				slog.Warn("failed reading pom.properties file", "path", currFilePath)
				continue
			}

			dep := shadedDependency{
				name:    fmt.Sprintf("%s:%s", pomProperties.GroupId, pomProperties.ArtifactId),
				version: pomProperties.Version,
			}
			pomPropertiesDeps[dep] = true
			slog.Info("found shaded dependencies from pom.properties", "package", dep)

		}
	}

	return pomXmlDeps, pomPropertiesDeps, nil
}

func shouldSkipDependency(dep shadedDependency) bool {
	return dep.version == "" || dep.name == ""
}

// Finds the shaded dependencies in a jar file and returns them as a list of dependencies
// Java only supports one level of shading, so we can assume that the dependencies are direct under `parent`
func FindShadedDependencies(jarPath string, parent *common.Dependency, normalizer shared.Normalizer) ([]*common.Dependency, error) {
	slog.Info("finding shaded dependencies", "jarPath", jarPath)
	exists, err := common.PathExists(jarPath)
	if err != nil {
		return nil, err
	}
	if !exists {
		slog.Info("jar file not found, not extracting shaded dependencies", "path", jarPath, "parent", parent.Id())
		return nil, nil
	}

	pomXmlDeps, pomPropertiesDeps, err := findShadedDependenciesFromJar(jarPath)
	if err != nil {
		return nil, err
	}

	deps := make(map[shadedDependency]bool, 0)
	maps.Copy(deps, pomXmlDeps)
	maps.Copy(deps, pomPropertiesDeps)

	shadedDeps := make([]*common.Dependency, 0)
	for dep := range deps {
		if shouldSkipDependency(dep) {
			slog.Warn("failed getting shaded dep info, skipping", "path", jarPath, "package", dep)
			continue
		}

		newDep := &common.Dependency{
			Name:           dep.name,
			NormalizedName: normalizer.NormalizePackageName(dep.name),
			Version:        dep.version,
			PackageManager: mappings.MavenManager,
			DiskPath:       parent.DiskPath, // Note that this is the path only AFTER the cache copy
			Parent:         parent,
			IsShaded:       true,
		}

		// Skip the parent dependency
		if newDep.Id() != parent.Id() {
			shadedDeps = append(shadedDeps, newDep)
		}

		_, inPomXml := pomXmlDeps[dep]
		_, inPomProperties := pomPropertiesDeps[dep]
		if !inPomXml && inPomProperties {
			slog.Warn("found dependency in pom properties that is not in pom xml", "dependency", dep)
		} else if inPomXml && !inPomProperties {
			slog.Warn("found dependency in pom xml that is not in pom properties", "dependency", dep)
		}
	}

	return shadedDeps, nil

}
