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

func (parser *dependencyParser) addDepInstance(deps common.DependencyMap, packageInfo *utils.JavaPackageInfo, prodOnly bool, cacheDir string) *common.Dependency {
	slog.Info("adding dependency", "packageInfo", packageInfo)
	if prodOnly && !slices.Contains(prodBuildScopes, packageInfo.Scope) {
		slog.Debug("skipping dependency", "packageInfo", packageInfo)
		return nil
	}

	packageName := fmt.Sprintf("%s:%s", packageInfo.OrgName, packageInfo.ArtifactName)
	packagePath := utils.GetJavaPackagePath(cacheDir, packageName, packageInfo.Version)
	metadataPath := filepath.Join(packagePath, shared.SealMetadataFileName)
	artifactFileName := utils.GetPackageFileName(packageInfo.ArtifactName, packageInfo.Version)
	artifactPath := filepath.Join(packagePath, artifactFileName)
	sealMetadata, err := shared.LoadPackageSealMetadata(metadataPath)

	// no error if the file does not exist
	if err != nil {
		return nil
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
	// Since the dependencies have only one instance, we can just add them once
	deps[key] = []*common.Dependency{newDep}

	return newDep
}

func (parser *dependencyParser) Parse(mavenOutput string, projectDir string) (common.DependencyMap, error) {
	deps := make(common.DependencyMap)

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
		parser.addDepInstance(deps, info, parser.config.Maven.ProdOnlyDeps, parser.cacheDir)
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
				deps[formatString(root.From)] = true
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
