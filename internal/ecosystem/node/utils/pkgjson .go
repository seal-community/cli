package utils

import (
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
)

const PackageJsonFile = "package.json"

func GetProjectName(dir string) string {
	pgk := loadPackageJson(dir)
	val, ok := pgk["name"]
	if !ok {
		slog.Warn("name not found in package json", "dir", dir)
		return ""
	}

	sVal, ok := val.(string)
	if !ok {
		slog.Warn("name value is bad type", "dir", dir)
		return ""
	}

	return sVal
}

func GetVersion(dir string) string {
	pgk := loadPackageJson(dir)
	val, ok := pgk["version"]
	if !ok {
		slog.Warn("version not found in package json", "dir", dir)
		return ""
	}

	sVal, ok := val.(string)
	if !ok {
		slog.Warn("version value is bad type", "dir", dir)
		return ""
	}

	return sVal
}

func loadPackageJson(dir string) map[string]any {
	var pkg map[string]any
	p := filepath.Join(dir, PackageJsonFile)
	data, err := os.ReadFile(p)
	if err != nil {
		slog.Error("failed opening package json file", "err", err, "path", p)
		return nil
	}

	if err := json.Unmarshal(data, &pkg); err != nil {
		slog.Error("failed loading json", "err", err, "path", p)
		return nil
	}
	
	return pkg
}
