package files

import (
	"archive/zip"
	"cli/internal/api"
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

type depCandidate struct {
	groupId    string // may be empty, in which case we query BE
	artifactId string
	version    string
	path       string
}

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

	// some usages add the groupId to the jar path, ignore it here
	splitByDot := strings.Split(name, ".")
	name = splitByDot[len(splitByDot)-1]

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

// Given a jar path, extract a dependency candidate representing that jar.
// We assume the file name is <artifactId>-<version>.jar, and parse the groupId from within the pom.xml/pom.properties file inside the jar.
func getDepCandidateFromJar(jarPath string) (*depCandidate, error) {
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
	}

	depCandidate := &depCandidate{
		groupId:    groupId,
		artifactId: artifactId,
		version:    version,
		path:       jarPath,
	}

	return depCandidate, nil
}

func buildDependencyFromCandidate(candidate depCandidate, normalizer shared.Normalizer) (*common.Dependency, error) {
	name := strings.Join([]string{candidate.groupId, candidate.artifactId}, ":")
	dep := &common.Dependency{
		PackageManager: mappings.MavenManager,
		Name:           name,
		Version:        candidate.version,
		NormalizedName: normalizer.NormalizePackageName(name),
		DiskPath:       candidate.path,
	}

	return dep, nil
}

// query BE for missing groupIds, in chunks
func queryGroupIds(candidates []depCandidate, be api.Backend) ([]api.MavenGroupIDLookupResult, error) {
	allResults := make([]api.MavenGroupIDLookupResult, 0)
	chunkSize := be.GetPackageChunkSize()

	err := common.ConcurrentChunks(candidates, chunkSize,
		func(chunk []depCandidate, chunkIdx int) (*api.Page[api.MavenGroupIDLookupResult], error) {
			var request api.MavenGroupIDLookupList

			for _, candidate := range chunk {
				request.Queries = append(request.Queries, api.MavenGroupIDLookup{ArtifactId: candidate.artifactId, Version: candidate.version})
			}

			return be.QueryMavenGroupIds(&request)
		},
		func(data *api.Page[api.MavenGroupIDLookupResult], chunkIdx int) error {
			// safe to perform, run from inside mutex
			allResults = append(allResults, data.Items...)
			return nil
		})

	return allResults, err

}

// Since some jar files don't have a groupId in their pom.xml/pom.properties,
// we query BE to resolve the missing groupIds.
// skip candidates that we failed to resolve.
func resolveGroupId(candidates []depCandidate, be api.Backend) ([]depCandidate, error) {
	resolvedCandidates := make([]depCandidate, 0)
	toResolve := make([]depCandidate, 0)

	for _, candidate := range candidates {
		if candidate.groupId == "" {
			toResolve = append(toResolve, candidate)
		}
	}

	if len(toResolve) == 0 {
		slog.Debug("all groupIds resolved already")
		return candidates, nil
	}

	results, err := queryGroupIds(toResolve, be)
	if err != nil {
		slog.Error("failed querying groupIds", "err", err)
		return nil, err
	}

	for _, candidate := range candidates {
		if candidate.groupId == "" {
			var groupId string
			for _, result := range results {
				if result.ArtifactId == candidate.artifactId && result.Version == candidate.version {
					if groupId != "" {
						slog.Warn("found multiple groupIds for candidate", "candidate", candidate, "groupId", groupId, "result", result)
						groupId = ""
						break
					}

					groupId = result.GroupId
				}
			}

			candidate.groupId = groupId
		}

		if candidate.groupId != "" {
			resolvedCandidates = append(resolvedCandidates, candidate)
		}
	}

	slog.Debug("resolved candidates", "resolvedCandidates", resolvedCandidates)
	return resolvedCandidates, nil
}

func getDependencies(jarPaths []string, normalizer shared.Normalizer, be api.Backend) ([]*common.Dependency, error) {
	dependencies := make([]*common.Dependency, 0)

	candidates := make([]depCandidate, 0)
	for _, path := range jarPaths {
		candidate, err := getDepCandidateFromJar(path)
		if err != nil {
			slog.Warn("failed getting dep candidate", "err", err, "path", path)
			continue
		}
		candidates = append(candidates, *candidate)
	}

	// query BE for missing groupIds
	candidates, err := resolveGroupId(candidates, be)
	if err != nil {
		slog.Error("failed resolving groupIds", "err", err)
		return nil, err
	}

	// build dependencies from results
	for _, candidate := range candidates {
		dep, err := buildDependencyFromCandidate(candidate, normalizer)
		if err != nil {
			slog.Warn("failed building dependency from candidate", "err", err, "candidate", candidate)
			continue
		}
		dependencies = append(dependencies, dep)
	}

	// find shaded dependencies
	for _, dep := range dependencies {
		shadedDeps, err := utils.FindShadedDependencies(dep.DiskPath, dep, normalizer)
		if err != nil {
			slog.Warn("failed finding shaded dependencies", "err", err)
			continue
		}

		dependencies = append(dependencies, shadedDeps...)
	}

	return dependencies, nil
}
