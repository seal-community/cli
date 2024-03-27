package utils

import (
	"log/slog"
	"os"
	"path/filepath"
)

func GetNugetCacheLocation() string {
	dirname, err := os.UserHomeDir()
	if err != nil {
		slog.Error("failed getting user home dir", "err", err)
	}
	return filepath.Join(dirname, ".nuget")
}

func GetGlobalPackagesCachePath() string {
	cacheLocation := GetNugetCacheLocation()
	return filepath.Join(cacheLocation, "packages")
}
