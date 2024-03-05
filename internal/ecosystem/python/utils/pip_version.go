package utils

import (
	"fmt"
	"log/slog"
	"regexp"
	"strings"
)

func GetSitePackages(pipVersionOutput string) (string, error) {
	// Parse pip version result, example:
	// pip 10.0.0 from /usr/local/lib/python3.7/site-packages/pip (python 3.7)
	r, err := regexp.Compile(`pip (?:[0-9.]+) from (.+) \(python [0-9.]+\)`)
	if err != nil {
		slog.Error("failed compiling regex", "err", err)
		return "", err
	}

	matches := r.FindStringSubmatch(pipVersionOutput)
	if len(matches) != 2 {
		slog.Error("failed matching regex", "result", pipVersionOutput)
		return "", fmt.Errorf("failed matching regex")
	}
	pipSitePackages := matches[1]
	if pipSitePackages == "" {
		slog.Error("failed matching regex", "result", pipVersionOutput)
		return "", fmt.Errorf("failed matching regex")
	}

	sitePackagesPath := strings.TrimSuffix(pipSitePackages, "pip")

	return sitePackagesPath, nil
}

func GetVersion(pipVersionOutput string) string {
	versionWithSuffix := strings.TrimPrefix(pipVersionOutput, "pip ") // it contains a new line
	spaceIndex := strings.Index(versionWithSuffix, " ")
	version := versionWithSuffix[:spaceIndex]
	return version

}

func GetMetadata(pipVersionOutput string) (string, string, error) {
	version := GetVersion(pipVersionOutput)
	sitePackages, err := GetSitePackages(pipVersionOutput)
	return version, sitePackages, err
}
