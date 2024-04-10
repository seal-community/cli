package nuget

import (
	"cli/internal/common"
	"cli/internal/config"
	"cli/internal/ecosystem/dotnet/utils"
	"cli/internal/ecosystem/mappings"
	"encoding/json"
	"log/slog"
	"path/filepath"
	"strings"
)

type NugetConfig struct {
	Version  int            `json:"version"`
	Projects []NugetProject `json:"projects"`
	Problems []Problem      `json:"problems"`
}

type Problem struct {
	Project string `json:"project"`
	Level   string `json:"level"` // error, etc
	Text    string `json:"text"`
}

type NugetProject struct {
	Path       string           `json:"path"`
	Frameworks []NugetFramework `json:"frameworks"`
}

type NugetFramework struct {
	Framework          string         `json:"frameworks"`
	TopLevelPackages   []NugetPackage `json:"topLevelPackages"`
	TransitivePackages []NugetPackage `json:"transitivePackages"`
}

type NugetPackage struct {
	Name             string `json:"id"`
	RequestedVersion string `json:"version"`
	ResolvedVersion  string `json:"resolvedVersion"`
}

type dependencyParser struct {
	config *config.Config
}

func (parser *dependencyParser) shouldSkip(p *NugetPackage) bool {
	if p.Name == "" || p.ResolvedVersion == "" {
		slog.Warn("empty dependency")
		return true
	}

	return false
}

func (parser *dependencyParser) Parse(nugetOutput string, projectDir string) (common.DependencyMap, error) {
	deps := make(common.DependencyMap)
	dependencyList := NugetConfig{}

	err := json.Unmarshal([]byte(nugetOutput), &dependencyList)
	if err != nil {
		slog.Error("failed to unmarshal nuget output", "err", err)
		return nil, err
	}

	if dependencyList.Problems != nil {
		for _, problem := range dependencyList.Problems {
			if problem.Level == "error" && strings.HasPrefix(problem.Text, "No assets file was found for") {
				slog.Debug("no assets file found", "project", problem.Project)
				return nil, common.NewPrintableError(problem.Text)
			}

			slog.Warn("problem with project", "project", problem.Project, "level", problem.Level, "text", problem.Text)
		}
	}

	for _, project := range dependencyList.Projects {
		for _, framework := range project.Frameworks {
			for _, pkg := range framework.TopLevelPackages {
				if parser.shouldSkip(&pkg) {
					continue
				}

				addDepInstance(deps, &pkg, projectDir)
			}

			for _, pkg := range framework.TransitivePackages {
				if parser.shouldSkip(&pkg) {
					continue
				}
				
				addDepInstance(deps, &pkg, projectDir)
			}
		}
	}

	return deps, nil
}

func addDepInstance(deps common.DependencyMap, p *NugetPackage, projectDir string) *common.Dependency {
	common.Trace("adding dep", "name", p.Name, "version", p.ResolvedVersion)
	packagesPath := utils.GetGlobalPackagesCachePath()
	diskPath := filepath.Join(packagesPath, strings.ToLower(p.Name), p.ResolvedVersion)

	newDep := &common.Dependency{
		Name:           p.Name,
		Version:        p.ResolvedVersion,
		PackageManager: mappings.NugetManager,
		DiskPath:       diskPath,
	}

	key := newDep.Id()
	if _, ok := deps[key]; !ok {
		deps[key] = make([]*common.Dependency, 0, 1)
	}
	deps[key] = append(deps[key], newDep)
	return newDep
}
