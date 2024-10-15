package yum

import (
	"cli/internal/common"
	"cli/internal/ecosystem/mappings"
	"fmt"
	"log/slog"
	"strings"
)

func parseYumVersion(goVersionOutput string) string {
	lines := strings.Split(goVersionOutput, "\n")
	version := strings.TrimSpace(lines[0])

	return version
}

func parseNameArch(nameArch string) (string, string) {
	parts := strings.Split(nameArch, ".")
	if len(parts) < 2 {
		slog.Warn("failed splitting name and arch", "nameArch", nameArch)
		return "", ""
	}

	name := strings.Join(parts[:len(parts)-1], ".")
	return name, parts[len(parts)-1]
}

func isWrapped(line string) bool {
	// if line is empty, it was not wrapped
	if strings.TrimSpace(line) == "" {
		return false
	}

	// if line is shorter than 80 characters, it was wrapped
	if len(line) < 80 {
		return true
	}

	// if line does not include any space, it was wrapped
	if !strings.Contains(line, " ") {
		return true
	}

	return false
}

// yum wraps lines to 80 characters, so we unwrap them
func unwrapYum(lines []string) []string {
	unwrapped := make([]string, 0, len(lines))
	i := 0

	for i < len(lines) {
		currentLine := ""
		line := lines[i]
		for isWrapped(line) {
			currentLine += line
			i++
			line = lines[i]
		}

		currentLine += line
		unwrapped = append(unwrapped, currentLine)
		i++
	}
	return unwrapped
}

// Use yum to list installed packages
// we don't use rpm/dnf/etc.. here to avoid discrepancies with yum
func parseYumListInstalled(yli string, normalizer *YumPackageManager) (common.DependencyMap, error) {
	deps := make(common.DependencyMap)
	lines := strings.Split(yli, "\n")

	// Find index of start line
	startIndex := -1
	for i, line := range lines {
		if strings.HasPrefix(line, "Installed Packages") {
			startIndex = i
			break
		}
	}
	if startIndex == -1 {
		slog.Error("failed finding installed packages", "result", yli)
		return nil, fmt.Errorf("failed finding installed packages indicator")
	}

	lines = unwrapYum(lines[startIndex+1:])

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) != 3 {
			slog.Warn("failed splitting to fields", "line", line)
			continue
		}

		name, arch := parseNameArch(fields[0])
		if name == "" || arch == "" {
			slog.Warn("failed parsing name and arch", "line", line)
			continue
		}

		version := fields[1]

		newDep := &common.Dependency{
			Name:           name,
			NormalizedName: normalizer.NormalizePackageName(name),
			Version:        version,
			PackageManager: mappings.RpmManager,
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
