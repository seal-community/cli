package files

import (
	"archive/zip"
	"cli/internal/common"
	"cli/internal/ecosystem/java/utils"
	"cli/internal/ecosystem/mappings"
	"cli/internal/ecosystem/shared"
	"fmt"
	"log/slog"
	"path/filepath"
	"regexp"
	"strings"
)

// regex based on syft's logic
// https://github.com/anchore/syft/blob/5e16e50/syft/pkg/cataloger/java/archive_filename.go#L50
var nameAndVersionPattern = regexp.MustCompile(`(?Ui)^(?P<name>(?:[[:alpha:]][[:word:].]*(?:\.[[:alpha:]][[:word:].]*)*-?)+)(?:-(?P<version>(\d.*|(build\d+.*)|(rc?\d+(?:^[[:alpha:]].*)?))))?$`)
var secondaryVersionPattern = regexp.MustCompile(`(?:[._-](?P<version>(\d.*|(build\d+.*)|(rc?\d+(?:^[[:alpha:]].*)?))))?$`)

func getSubexp(matches []string, subexpName string, re *regexp.Regexp, raw string) (string, error) {
	if len(matches) < 1 {
		slog.Error("no matches found", "raw", raw)
		return "", fmt.Errorf("no matches found")
	}

	index := re.SubexpIndex(subexpName)
	if index < 1 {
		slog.Error("subexp not found", "subexp", subexpName)
		return "", fmt.Errorf("subexp not found")
	}

	// Prevent out-of-range panic
	if len(matches) < index+1 {
		slog.Error("index out of range", "index", index, "matches", matches)
		return "", fmt.Errorf("index out of range")
	}

	return matches[index], nil
}

// extract artifactId and version from a jar path
func parseJarPath(jarPath string) (string, string, error) {
	filename := filepath.Base(jarPath)
	filename = strings.TrimSuffix(filename, filepath.Ext(filename))

	matches := nameAndVersionPattern.FindStringSubmatch(filename)

	name, err := getSubexp(matches, "name", nameAndVersionPattern, jarPath)
	if err != nil {
		return "", "", err
	}

	// some jars get named with different conventions, like `_<version>` or `.<version>`
	version, _ := getSubexp(matches, "version", nameAndVersionPattern, jarPath)
	if version == "" {
		matches = secondaryVersionPattern.FindStringSubmatch(name)
		version, _ = getSubexp(matches, "version", secondaryVersionPattern, jarPath)
		if version != "" {
			name = name[0 : len(name)-len(version)-1]
		}
	}

	return name, version, nil
}

// find pom.properties and pom.xml of the jar
func findPomDataInJar(jarPath string, artifactId string) (*utils.PomProperties, *utils.PomXML, error) {
	origReader, err := zip.OpenReader(jarPath)
	if err != nil {
		slog.Error("failed reading package", "err", err, "path", jarPath)
		return nil, nil, err
	}
	defer origReader.Close()

	var pomProperties *utils.PomProperties
	var pomXML *utils.PomXML

	for _, zipItem := range origReader.File {
		header := zipItem.FileHeader

		if !strings.HasPrefix(header.Name, "META-INF") {
			continue
		}

		currFilePath := filepath.ToSlash(header.Name)
		currFileName := filepath.Base(header.Name)

		// Skip pom files that aren't for this jar, but embedded in it (shaded, bundled, etc.)
		currDir := filepath.Base(filepath.Dir(header.Name))
		if currDir != artifactId {
			continue
		}

		zipItemReader, err := zipItem.Open()
		if err != nil {
			slog.Error("failed opening zip item", "err", err, "path", zipItem.Name)
			return nil, nil, err
		}
		defer zipItemReader.Close()

		if currFileName == utils.PomXMLFileName {
			if pomXML != nil {
				slog.Warn("found multiple pom files", "path", currFilePath)
				continue
			}
			slog.Debug("found pom file", "path", currFilePath)
			pomXML = utils.ReadPomXMLFromFile(zipItemReader)
		} else if currFileName == utils.PomPropertiesFileName {
			if pomProperties != nil {
				slog.Warn("found multiple pom properties files", "path", currFilePath)
				continue
			}
			slog.Debug("found pom properties file", "path", currFilePath)
			pomProperties = utils.ReadPomPropertiesFromFile(zipItemReader)
		}
	}

	return pomProperties, pomXML, nil
}

// Given a jar path, extract a dependency representing that jar.
// We assume the file name is <artifactId>-<version>.jar, and parse the groupId from within the pom.xml/pom.properties file inside the jar.
func getDependencyFromJar(jarPath string, normalizer shared.Normalizer) (*common.Dependency, error) {
	artifactId, version, err := parseJarPath(jarPath)
	if err != nil {
		slog.Error("failed to parse jar path", "err", err, "jarPath", jarPath)
		return nil, err
	}
	if artifactId == "" || version == "" {
		slog.Error("failed to parse jar path", "artifactId", artifactId, "version", version, "jarPath", jarPath)
		return nil, fmt.Errorf("failed to parse jar path %s", jarPath)
	}

	// find groupId from within the jar
	origReader, err := zip.OpenReader(jarPath)
	if err != nil {
		slog.Error("failed reading package", "err", err, "path", jarPath)
		return nil, err
	}
	defer origReader.Close()

	pomProperties, pomXML, err := findPomDataInJar(jarPath, artifactId)
	if err != nil {
		slog.Error("failed finding pom data in jar", "err", err, "jarPath", jarPath)
		return nil, err
	}

	// extract groupId from jar's pom definitions
	var groupId string
	if pomProperties != nil {
		groupId = pomProperties.GroupId
	} else if pomXML != nil {
		groupId = pomXML.GetGroupId()
	} else {
		slog.Warn("failed reading pom file and pom properties", "path", jarPath)
		return nil, fmt.Errorf("failed reading pom file and pom properties %s", jarPath)
	}

	depName := fmt.Sprintf("%s:%s", groupId, artifactId)

	dep := &common.Dependency{
		Name:           depName,
		Version:        version,
		PackageManager: mappings.MavenManager,
		NormalizedName: normalizer.NormalizePackageName(depName),
		DiskPath:       jarPath,
	}

	return dep, nil
}

// Given a jar path, extract all dependencies representing that jar and its shaded dependencies.
// Handles sealed jars too.
func getFileDependencies(jarPath string, normalizer shared.Normalizer) ([]*common.Dependency, error) {
	dependencies := make([]*common.Dependency, 0)
	dep, err := getDependencyFromJar(jarPath, normalizer)
	if err != nil {
		slog.Error("failed getting dependency from jar", "err", err)
		return nil, err
	}

	dependencies = append(dependencies, dep)
	// add shaded deps
	shadedDeps, err := utils.FindShadedDependencies(jarPath, dep, normalizer)
	if err != nil {
		slog.Error("failed finding shaded dependencies", "err", err)
		return nil, err
	}

	dependencies = append(dependencies, shadedDeps...)

	return dependencies, nil
}
