package apk

import (
	"cli/internal/common"
	"cli/internal/ecosystem/mappings"
	"log/slog"
	"regexp"
	"strings"
)

func parseAPKVersion(goVersionOutput string) string {
	splitLine := strings.Split(goVersionOutput, " ")
	version := strings.TrimSpace(splitLine[1])
	return strings.TrimRight(version, ",")
}

func parseNameVersion(filename string) (name string, version string) {
	// Regex to match APK package format: name-version-release.apk
	re := regexp.MustCompile(`^(.+)-(\d[\d\.]*[^-]*)-r(\d+)$`)

	matches := re.FindStringSubmatch(filename)
	if len(matches) != 4 {
		return "", ""
	}

	name = matches[1]
	version = matches[2] + "-r" + matches[3] // Full version including release
	return name, version
}

func parseAPKListInstalled(ali string) (common.DependencyMap, error) {
	deps := make(common.DependencyMap)
	lines := strings.Split(ali, "\n")

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		fields := strings.Fields(line)
		// Should be at least 5 because license can have spaces (e.g. Apache-2.0 or GPL-2.0)
		if len(fields) < 5 {
			slog.Warn("failed splitting to fields", "line", line)
			continue
		}

		name, version := parseNameVersion(fields[0])
		if name == "" || version == "" {
			slog.Warn("failed parsing name and arch", "line", line)
			continue
		}

		arch := fields[1]

		newDep := &common.Dependency{
			Name:           name,
			NormalizedName: name,
			Version:        version,
			PackageManager: mappings.ApkManager,
			Arch:           arch,
		}

		key := newDep.Id()
		if _, ok := deps[key]; !ok {
			deps[key] = make([]*common.Dependency, 0, 1)
		}

		deps[key] = append(deps[key], newDep)
	}
	return deps, nil
}
