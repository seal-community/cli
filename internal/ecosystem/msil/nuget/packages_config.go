package nuget

import (
	"cli/internal/common"
	"cli/internal/ecosystem/mappings"
	"cli/internal/ecosystem/msil/utils"
	"encoding/xml"
	"fmt"
	"io"
	"log/slog"
	"path/filepath"
)

// format: https://learn.microsoft.com/en-us/nuget/reference/packages-config
type packageDescriptor struct {
	Library   string `xml:"id,attr"`
	Version   string `xml:"version,attr"`
	Framework string `xml:"targetFramework,attr"`
	Dev       bool   `xml:"developmentDependency,attr"` // only relevant if parent project is a package
}

type root struct {
	Packages []packageDescriptor `xml:"package"`
}

// format of packages folder in `packages` dir, which differs from the global cache as doesn't lower case
func formatDependencyDiskPath(packagesDir string, library string, version string) string {
	return filepath.Join(packagesDir, fmt.Sprintf("%s.%s", library, version))
}

var NoPackagesFoundError = common.NewPrintableError("no dependencies found in packages folder")

// returns dependencies, even if non of them were found in disk
// in that case error will be returned alongside the data
func parsePackagesConfig(r io.Reader, packagesDir string) (common.DependencyMap, error) {
	c := root{}
	if err := xml.NewDecoder(r).Decode(&c); err != nil {
		slog.Error("failed decoding packages config", "err", err)
		return nil, common.WrapWithPrintable(err, "unsupported or malformed packages.config file")
	}

	deps := make(common.DependencyMap)
	depsMissing := 0
	for _, p := range c.Packages {
		diskPath := formatDependencyDiskPath(packagesDir, p.Library, p.Version)
		if exists, err := common.DirExists(diskPath); !exists || err != nil {
			slog.Warn("could not find dependency on disk; ignoring", "err", err, "found", exists, "path", diskPath)
			depsMissing++
		}

		common.Trace("discovered package info", "info", p)
		newDep := &common.Dependency{
			Name:           p.Library,
			NormalizedName: utils.NormalizeName(p.Library),
			Version:        p.Version,
			PackageManager: mappings.NugetManager,
			DiskPath:       diskPath,
			Dev:            p.Dev,
		}

		key := newDep.Id()
		if _, ok := deps[key]; !ok {
			deps[key] = make([]*common.Dependency, 0, 1)
		}
		deps[key] = append(deps[key], newDep)
	}

	if depsMissing > 0 {
		slog.Warn("missing dependencies in disk, but found in config", "count", depsMissing, "total", len(deps))
		if depsMissing == len(deps) {
			// probably wrong packages dir
			slog.Error("could not find any of the packages", "path", packagesDir)
			return deps, NoPackagesFoundError
		}
	}

	return deps, nil
}
