package golang

import (
	"cli/internal/common"
	"cli/internal/ecosystem/mappings"
	"log/slog"
	"os"
	"strings"

	"golang.org/x/mod/modfile"
	"golang.org/x/mod/module"
)

func ParseGoVersion(goVersionOutput string) string {
	version := strings.Split(goVersionOutput, " ")[2]
	version = strings.TrimPrefix(version, "go")

	return version
}

func ParseGoModFile(goModPath string) (*modfile.File, error) {
	goModContents, err := os.ReadFile(goModPath)
	if err != nil {
		slog.Error("failed reading go.mod file", "err", err)
		return nil, err
	}

	goMod, err := modfile.Parse(goModFilename, goModContents, nil)
	if err != nil {
		slog.Error("failed parsing go.mod file", "err", err)
		return nil, err
	}

	return goMod, err
}

func BuildDependencyMap(goMod *modfile.File) common.DependencyMap {
	replaces := make(map[module.Version]module.Version)
	for _, replace := range goMod.Replace {
		replaces[replace.Old] = replace.New
	}

	deps := make(common.DependencyMap)
	for _, required := range goMod.Require {
		versionlessMod := module.Version{
			Path:    required.Mod.Path,
			Version: "",
		}
		// replace specific version
		replaced, ok := replaces[required.Mod]
		if ok {
			required.Mod = replaced
		}
		// replace module with any version
		replaced, ok = replaces[versionlessMod]
		if ok {
			required.Mod = replaced
		}

		// local path modules don't have a version, ignore them
		if required.Mod.Version == "" {
			continue
		}

		version := strings.TrimPrefix(required.Mod.Version, "v")
		newDep := &common.Dependency{
			Name:           required.Mod.Path,
			NormalizedName: NormalizePackageName(required.Mod.Path),
			Version:        version,
			PackageManager: mappings.GolangManager,
		}

		key := newDep.Id()
		deps[key] = []*common.Dependency{newDep}
	}

	return deps
}
