package maven

import (
	"cli/internal/common"
	"cli/internal/config"
	"cli/internal/ecosystem/java/utils"
	"cli/internal/ecosystem/mappings"
	"cli/internal/ecosystem/shared"
	"fmt"
	"log/slog"
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

func formatString(v ast.Vertex) string {
	return strings.Trim(v.String(), "\"")
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
	shadedDeps, err := utils.FindShadedDependencies(originalArtifactPath, newDep, parser.normalizer)
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
