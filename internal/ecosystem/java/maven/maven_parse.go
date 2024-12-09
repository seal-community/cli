package maven

import (
	"archive/zip"
	"cli/internal/common"
	"cli/internal/config"
	"cli/internal/ecosystem/java/utils"
	"cli/internal/ecosystem/mappings"
	"cli/internal/ecosystem/shared"
	"fmt"
	"log/slog"
	"maps"
	"path/filepath"
	"slices"
	"strings"

	"gonum.org/v1/gonum/graph/formats/dot"
	"gonum.org/v1/gonum/graph/formats/dot/ast"
)

var prodBuildScopes = []string{"compile", "runtime", ""}

type dependencyParser struct {
	config     *config.Config
	cacheDir   string
	normalizer shared.Normalizer
}

type shadedDependency struct {
	name    string
	version string
}

func formatString(v ast.Vertex) string {
	return strings.Trim(v.String(), "\"")
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
		zipItemReader, err := zipItem.Open()
		if err != nil {
			slog.Error("failed opening zip item", "err", err, "path", zipItem.Name)
			return nil, nil, err
		}
		defer zipItemReader.Close()

		header := zipItem.FileHeader

		currFilePath := filepath.ToSlash(header.Name)
		currFileName := filepath.Base(header.Name)
		if currFileName == utils.PomXMLFileName {
			slog.Debug("found pom file", "path", currFilePath)
			pom := utils.ReadPomXMLFromFile(zipItemReader)
			if pom == nil {
				slog.Warn("failed reading pom file", "path", currFilePath)
				continue
			}

			dep := shadedDependency{
				name:    fmt.Sprintf("%s:%s", pom.GetGroupId(), pom.GetArtifactId()),
				version: pom.GetVersion(),
			}
			pomXmlDeps[dep] = true
			slog.Info("found shaded dependencies", "package", dep)

		} else if currFileName == utils.PomPropertiesFileName {
			slog.Debug("found pom properties file", "path", currFilePath)
			pomProperties := utils.ReadPomPropertiesFromFile(zipItemReader)
			if pomProperties == nil {
				slog.Warn("failed reading pom.properties file", "path", currFilePath)
				continue
			}

			dep := shadedDependency{
				name:    fmt.Sprintf("%s:%s", pomProperties.GroupId, pomProperties.ArtifactId),
				version: pomProperties.Version,
			}
			pomPropertiesDeps[dep] = true
			slog.Info("found shaded dependencies", "package", dep)

		}
	}

	return pomXmlDeps, pomPropertiesDeps, nil
}

// Finds the shaded dependencies in a jar file and returns them as a list of dependencies
// Java only supports one level of shading, so we can assume that the dependencies are direct under `parent`
func (parser *dependencyParser) findShadedDependencies(jarPath string, parent *common.Dependency) ([]*common.Dependency, error) {
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
		if dep.version == "" || dep.name == "" {
			slog.Info("failed getting shaded dep info, skipping", "path", jarPath, "package", dep)
			continue
		}

		newDep := &common.Dependency{
			Name:           dep.name,
			NormalizedName: parser.normalizer.NormalizePackageName(dep.name),
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

func (parser *dependencyParser) addDepInstance(deps common.DependencyMap, packageInfo *utils.JavaPackageInfo, prodOnly bool, cacheDir string, projCacheDir string) (err error) {
	slog.Info("adding dependency", "packageInfo", packageInfo)
	if prodOnly && !slices.Contains(prodBuildScopes, packageInfo.Scope) {
		slog.Debug("skipping dependency", "packageInfo", packageInfo)
		return
	}

	packageName := fmt.Sprintf("%s:%s", packageInfo.OrgName, packageInfo.ArtifactName)
	packagePath := utils.GetJavaPackagePath(cacheDir, packageName, packageInfo.Version)
	metadataPath := filepath.Join(packagePath, shared.SealMetadataFileName)
	artifactFileName := utils.GetPackageFileName(packageInfo.ArtifactName, packageInfo.Version)
	artifactPath := filepath.Join(packagePath, artifactFileName)
	sealMetadata, err := shared.LoadPackageSealMetadata(metadataPath)

	// no error if the file does not exist
	if err != nil {
		return
	}

	if sealMetadata != nil {
		packageInfo.Version = sealMetadata.SealedVersion
		slog.Info("found sealed package", "packageInfo", packageInfo)
	}

	newDep := &common.Dependency{
		Name:           packageName,
		NormalizedName: parser.normalizer.NormalizePackageName(packageName),
		Version:        packageInfo.Version,
		PackageManager: mappings.MavenManager,
		DiskPath:       artifactPath, // Note that this is the path only AFTER the cache copy
	}

	key := newDep.Id()
	if _, ok := deps[key]; !ok {
		deps[key] = []*common.Dependency{newDep}
	} else {
		deps[key] = append(deps[key], newDep)
	}

	// Add shaded dependencies
	originalPackagePath := utils.GetJavaPackagePath(projCacheDir, packageName, packageInfo.Version)
	originalArtifactPath := filepath.Join(originalPackagePath, artifactFileName)
	shadedDeps, err := parser.findShadedDependencies(originalArtifactPath, newDep)
	if err != nil {
		slog.Error("failed finding shaded dependencies", "err", err)
		return
	}

	for _, dep := range shadedDeps {
		key := dep.Id()
		if _, ok := deps[key]; !ok {
			deps[key] = []*common.Dependency{dep}
		} else {
			deps[key] = append(deps[key], dep)
		}
	}

	return
}

func (parser *dependencyParser) Parse(mavenOutput string, projectDir string) (common.DependencyMap, error) {
	deps := make(common.DependencyMap)

	projCacheDir := utils.GetCacheDir(projectDir)
	if projCacheDir == "" {
		slog.Warn("failed getting maven cache dir")
		return nil, fmt.Errorf("failed getting maven cache dir")
	}

	depsRawMap, err := parseDependencies(mavenOutput)
	if err != nil {
		return nil, common.NewPrintableError("failed to parse dependency list")
	}

	for dep := range depsRawMap {
		info, err := utils.CreateJavaPackageInfo(dep)
		if err != nil {
			slog.Error("failed creating package info", "err", err)
			return nil, err
		}
		err = parser.addDepInstance(deps, info, parser.config.Maven.ProdOnlyDeps, parser.cacheDir, projCacheDir)
		if err != nil {
			slog.Error("failed adding dependency instance", "err", err)
			return nil, err
		}
	}

	return deps, nil
}

// mavenOutput is the output of the maven dependency:tree command in a dot format
// the dependencies identifiers (orgName:artifactName:buildType:version:scope) are the vertices in the graph
// parseDependencies parses the graph and returns a map of identifiers of the dependencies
func parseDependencies(mavenOutput string) (map[string]bool, error) {
	f, err := dot.ParseString(mavenOutput)
	if err != nil {
		slog.Error("failed unmarshal dependency graph", "err", err)
		return nil, err
	}

	deps := make(map[string]bool)
	for _, graph := range f.Graphs {
		for _, stmt := range graph.Stmts {
			switch root := stmt.(type) {
			case *ast.EdgeStmt:
				// Ignore the root node, it's not a dependency
				if root.From.String() != graph.ID {
					deps[formatString(root.From)] = true
				}

				next := root.To
				for next != nil {
					deps[formatString(next.Vertex)] = true
					next = next.To
				}
			default:
				slog.Warn("shouldnt be here", "stmt", stmt)
			}
		}
	}
	return deps, nil
}
