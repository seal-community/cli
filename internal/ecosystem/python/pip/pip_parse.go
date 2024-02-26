package pip

import (
	"cli/internal/common"
	"cli/internal/config"
	"cli/internal/ecosystem/mappings"
	"encoding/json"
	"fmt"
	"log/slog"
	"path/filepath"
	"regexp"
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

func escapePackageName(name string) string {
	return strings.ReplaceAll(name, "-", "_")
}

func addDepInstance(deps common.DependencyMap, p *PythonPackage, sitePackages string) *common.Dependency {
	common.Trace("adding dep", "name", p.Name, "version", p.Version, "editableProjectLocation", p.EditableProjectLocation)
	diskPath := filepath.Join(sitePackages, fmt.Sprintf("%s-%s.dist-info", escapePackageName(p.Name), p.Version))

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

func getSitePackages(pipOutput string) (string, error) {
	// Parse pip version result, example:
	// pip 10.0.0 from /usr/local/lib/python3.7/site-packages/pip (python 3.7)
	r, err := regexp.Compile(`pip (?:[0-9.]+) from (.+) \(python [0-9.]+\)`)
	if err != nil {
		slog.Error("failed compiling regex", "err", err)
		return "", err
	}

	matches := r.FindStringSubmatch(pipOutput)
	if len(matches) != 2 {
		slog.Error("failed matching regex", "result", pipOutput)
		return "", fmt.Errorf("failed matching regex")
	}
	pipSitePackages := matches[1]
	if pipSitePackages == "" {
		slog.Error("failed matching regex", "result", pipOutput)
		return "", fmt.Errorf("failed matching regex")
	}

	sitePackagesPath := strings.TrimSuffix(pipSitePackages, "pip")

	return sitePackagesPath, nil
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

	sitePackages, err := getSitePackages(versionOutput)
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
