package pip

import (
	"cli/internal/common"
	"cli/internal/config"
	"cli/internal/ecosystem/mappings"
	"cli/internal/ecosystem/python/utils"
	"cli/internal/ecosystem/shared"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
)

type PythonPackage struct {
	Version                 string `json:"version"`
	Name                    string `json:"name"`
	EditableProjectLocation string `json:"editable_project_location"`
}

type dependencyParser struct {
	config     *config.Config // in the future we might want to only pass the pip specific config object
	normalizer shared.Normalizer
}

func (parser *dependencyParser) shouldSkip(p *PythonPackage) bool {
	if p.Name == "" || p.Version == "" {
		slog.Debug("empty dependency")
		return true
	}
	if p.EditableProjectLocation != "" {
		slog.Info("skipping link dependency", "name", p.Name, "version", p.Version, "editableProjectLocation", p.EditableProjectLocation)
		return true
	}

	return false
}

func (parser *dependencyParser) addDepInstance(deps common.DependencyMap, p *PythonPackage, sitePackages string) error {
	common.Trace("adding dep", "name", p.Name, "version", p.Version, "editableProjectLocation", p.EditableProjectLocation)

	// find the install directory of the package, either a '.dist-info' or 'egg-info' folder
	diskPath, err := utils.FindSitePackagesFolderForPackage(sitePackages, p.Name, p.Version)
	if err != nil {
		slog.Error("failed getting site packages folder", "err", err, "name", p.Name, "version", p.Version)
		return err
	}

	newDep := &common.Dependency{
		Name:           p.Name,
		NormalizedName: parser.normalizer.NormalizePackageName(p.Name),
		Version:        p.Version,
		PackageManager: mappings.PythonManager,
		DiskPath:       diskPath,
	}

	key := newDep.Id()
	if _, ok := deps[key]; !ok {
		deps[key] = make([]*common.Dependency, 0, 1)
	}

	deps[key] = append(deps[key], newDep)
	return nil
}

func (parser *dependencyParser) Parse(pipOutput string, projectDir string) (common.DependencyMap, error) {
	deps := make(common.DependencyMap)

	dependencyList := make([]PythonPackage, 0)

	pipResult := strings.Split(pipOutput, pipResultSeparator)
	if len(pipResult) != 2 {
		slog.Error("failed splitting pip result", "result", pipOutput)
		return nil, fmt.Errorf("failed splitting pip result")
	}

	versionOutput, listOutput := pipResult[0], pipResult[1]

	sitePackages, err := utils.GetSitePackages(versionOutput)
	if err != nil {
		slog.Error("failed getting site packages", "err", err)
		return nil, fmt.Errorf("failed getting site packages")
	}
	slog.Info("site packages", "path", sitePackages)

	err = json.Unmarshal([]byte(listOutput), &dependencyList)
	if err != nil {
		slog.Error("failed unmarshal list output", "err", err)
		return nil, fmt.Errorf("failed unmarshal list output")
	}

	for i, p := range dependencyList {
		if parser.shouldSkip(&p) {
			slog.Warn("skipping dep", "name", p.Name, "version", p.Version, "index", i)
			continue
		}

		if err := parser.addDepInstance(deps, &p, sitePackages); err != nil {
			slog.Warn("could not add dep instance - skipping", "err", err, "package", p)
		}
	}

	slog.Info("root package", "direct_deps", len(dependencyList))

	return deps, nil
}
