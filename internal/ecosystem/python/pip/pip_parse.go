package pip

import (
	"cli/internal/common"
	"cli/internal/config"
	"cli/internal/ecosystem/mappings"
	"cli/internal/ecosystem/python/utils"
	"encoding/json"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
)

type PythonPackage struct {
	Version                 string `json:"version"`
	Name                    string `json:"name"`
	EditableProjectLocation string `json:"editable_project_location"`
}

type dependencyParser struct {
	config *config.Config // in the future we might want to only pass the pip specific config object
}

func (parser *dependencyParser) shouldSkip(p *PythonPackage) bool {
	if p.Name == "" || p.Version == "" {
		slog.Warn("empty dependency")
		return true
	}
	if p.EditableProjectLocation != "" {
		slog.Info("skipping link dependency", "name", p.Name, "version", p.Version, "editableProjectLocation", p.EditableProjectLocation)
		return true
	}

	return false
}

func addDepInstance(deps common.DependencyMap, p *PythonPackage, sitePackages string) *common.Dependency {
	common.Trace("adding dep", "name", p.Name, "version", p.Version, "editableProjectLocation", p.EditableProjectLocation)
	diskPath := filepath.Join(sitePackages, utils.DistInfoPath(p.Name, p.Version))

	newDep := &common.Dependency{
		Name:           p.Name,
		Version:        p.Version,
		PackageManager: mappings.PythonManager,
		DiskPath:       diskPath,
	}

	key := newDep.Id()
	if _, ok := deps[key]; !ok {
		deps[key] = make([]*common.Dependency, 0, 1)
	}

	deps[key] = append(deps[key], newDep)
	return newDep
}

func (parser *dependencyParser) Parse(pipOutput string, projectDir string) (common.DependencyMap, error) {
	deps := make(common.DependencyMap)

	dependendyList := make([]PythonPackage, 0)

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

	err = json.Unmarshal([]byte(listOutput), &dependendyList)
	if err != nil {
		slog.Error("failed unmarshal list output", "err", err)
		return nil, fmt.Errorf("failed unmarshal list output")
	}

	for i, p := range dependendyList {
		if parser.shouldSkip(&p) {
			slog.Warn("skipping dep", "name", p.Name, "version", p.Version, "index", i)
			continue
		}
		_ = addDepInstance(deps, &p, sitePackages)
	}

	slog.Info("root package", "direct_deps", len(dependendyList))

	return deps, nil
}
