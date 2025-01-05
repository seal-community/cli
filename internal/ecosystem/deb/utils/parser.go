package utils

import (
	"cli/internal/common"
	"cli/internal/ecosystem/mappings"
	"log/slog"
	"slices"
	"strings"
)

var InstalledStatuses = []string{"install ok installed", "install ok half-installed", "failed-config"}

func ParseDpkgVersion(dpkgVersionOutput string) string {
	lines := strings.Split(dpkgVersionOutput, "\n")
	versionLine := strings.TrimSpace(lines[0])
	// example: Debian 'dpkg' package management program version 1.20.13 (amd64).
	versionLineParts := strings.Fields(versionLine)
	return versionLineParts[len(versionLineParts)-2]
}

// Use dpkg to list installed packages
func ParseDpkgQueryInstalled(dpkgList string) (common.DependencyMap, error) {
	deps := make(common.DependencyMap)
	lines := strings.Split(dpkgList, "\n")

	for _, line := range lines {
		if len(line) == 0 {
			continue
		}
		// each installed line looks like this:
		// zlib1g 1:1.2.13.dfsg-1 amd64 install ok installed
		fields := strings.Fields(line)
		if len(fields) < 4 {
			slog.Warn("Got invalid package line", "line", line)
			continue
		}

		name := fields[0]
		isInstalled, status := isStatusInstalled(fields)
		if !isInstalled {
			slog.Debug("ignoring package with non-installed status", "name", name, "status", status)
			continue
		}

		newDep := &common.Dependency{
			Name:           name,
			NormalizedName: name, // no normalization needed for dpkg packages
			Version:        fields[1],
			PackageManager: mappings.DebManager,
			Arch:           fields[2],
		}

		key := newDep.Id()
		if _, ok := deps[key]; !ok {
			deps[key] = make([]*common.Dependency, 0, 1)
		}

		deps[key] = append(deps[key], newDep)
	}

	return deps, nil
}

func isStatusInstalled(fields []string) (bool, string) {
	status := strings.Join(fields[3:], " ")
	return slices.Contains(InstalledStatuses, status), status
}
