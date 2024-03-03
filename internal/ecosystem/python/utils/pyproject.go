package utils

import (
	"log/slog"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

const PyProjectTomlFile = "pyproject.toml"

func GetProjectName(dir string) string {
	pyproj := loadPyProjectToml(dir)
	if pyproj == nil {
		return ""
	}

	if name := getPoetryProjectName(pyproj); name != "" {
		return name
	}

	if name := getPyProjectName(pyproj); name != "" {
		return name
	}

	return ""
}

func getPyProjectName(pyproj map[string]any) string {
	val, ok := pyproj["project"]
	if !ok {
		slog.Warn("project not found in pyproject.toml")
		return ""
	}

	proj, ok := val.(map[string]any)
	if !ok {
		slog.Warn("project value is bad type")
		return ""
	}

	val, ok = proj["name"]
	if !ok {
		slog.Warn("name not found in project")
		return ""
	}

	sVal, ok := val.(string)
	if !ok {
		slog.Warn("name value is bad type")
		return ""
	}
	return sVal
}

func getPoetryProjectName(pyproj map[string]any) string {
	val, ok := pyproj["tool"]
	if !ok {
		slog.Warn("tool not found in pyproject.toml")
		return ""
	}

	tool, ok := val.(map[string]any)
	if !ok {
		slog.Warn("tool value is bad type")
		return ""
	}

	val, ok = tool["poetry"]
	if !ok {
		slog.Warn("poetry not found in tool")
		return ""
	}

	poetry, ok := val.(map[string]any)
	if !ok {
		slog.Warn("poetry value is bad type")
		return ""
	}

	val, ok = poetry["name"]
	if !ok {
		slog.Warn("name not found in poetry")
		return ""
	}

	sVal, ok := val.(string)
	if !ok {
		slog.Warn("name value is bad type")
		return ""
	}

	return sVal
}

func loadPyProjectToml(dir string) map[string]any {
	var pyproj map[string]any

	p := filepath.Join(dir, PyProjectTomlFile)
	data, err := os.ReadFile(p)
	if err != nil {
		slog.Error("failed opening pyproject.toml file", "err", err, "path", p)
		return nil
	}

	if err := toml.Unmarshal(data, &pyproj); err != nil {
		slog.Error("failed loading toml", "err", err, "path", p)
		return nil
	}

	return pyproj
}
