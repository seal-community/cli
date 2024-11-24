package utils

import (
	"cli/internal/common"
	"fmt"
	"log/slog"
	"path/filepath"

	"github.com/beevik/etree"
)

type ProjectFormat int

// identifying format https://learn.microsoft.com/en-us/nuget/resources/check-project-format
const (
	FormatUnknown = iota

	FormatLegacy               = iota
	FormatLegacyPackagesConfig = iota // oldest, deprecated formats; see https://learn.microsoft.com/en-us/nuget/reference/packages-config
	FormatLegacyProjectJson    = iota // older, also supports lockfile 'project.lock.json'; see https://learn.microsoft.com/en-us/nuget/archive/project-json

	FormatSupportedByDotnet = iota

	FormatMigrated = iota // Migrated, contains PackagesReference but no Sdk attribute
	FormatSdk      = iota
)

const DefaultPackagesDirName = "packages"
const DefaultPackagesConfigFile = "packages.config"
const DefaultProjectJsonFile = "project.json"

func inspectProjectFormat(doc *etree.Document) (ProjectFormat, error) {
	projElem := doc.SelectElement("Project")
	if projElem == nil {
		slog.Error("could not find Project element in project")
		return FormatUnknown, fmt.Errorf("not Project tag in project file")
	}

	sdkAttr := projElem.SelectAttr("Sdk")
	if sdkAttr != nil {
		slog.Debug("project has sdk attribute - new format", "value", sdkAttr.Value)
		return FormatSdk, nil
	}

	toolsVersionAttr := projElem.SelectAttr("ToolsVersion")
	if toolsVersionAttr == nil {
		slog.Error("unknown format")
		return FormatUnknown, fmt.Errorf("unknown format - not ToolsVersion and no Sdk attributes found")
	}

	// existence of ToolsVersion attribute can either be migrated, or legacy
	pkgRefs := doc.FindElements("//PackageReference")
	if len(pkgRefs) != 0 {
		slog.Debug("found PackageReference element - migrated project", "count", len(pkgRefs))
		return FormatMigrated, nil
	}

	slog.Debug("inspected as legacy format")
	return FormatLegacy, nil
}

func DetectProjectFormat(projPath string) (ProjectFormat, error) {
	f, err := common.OpenFile(projPath)
	if err != nil {
		slog.Error("failed opening project file", "path", projPath)
		return FormatUnknown, err
	}
	defer f.Close()

	doc := etree.NewDocument() // might want to enable `Permissive` on document's ReadSettings if we encounter errors
	if _, err := doc.ReadFrom(f); err != nil {
		slog.Error("failed reading project file", "err", err)
		return FormatUnknown, err
	}

	format, err := inspectProjectFormat(doc)  // we could further detect the format by checking existance of packages.config / project.json files
	if err != nil || format != FormatLegacy { // either error or detected exact format, return
		return format, err
	}

	pkgsPath := filepath.Join(filepath.Dir(projPath), DefaultPackagesConfigFile)
	projJson := filepath.Join(filepath.Dir(projPath), DefaultProjectJsonFile)

	slog.Info("inspecting folder to detect exact legacy format", "packages", pkgsPath, "projectjson", projJson)
	exists, err := common.PathExists(pkgsPath)
	if err != nil {
		slog.Error("failed checking path exists", "err", err)
		return FormatUnknown, err
	}

	if exists {
		slog.Info("detected packages.config file", "path", pkgsPath)
		return FormatLegacyPackagesConfig, nil
	}

	exists, err = common.PathExists(projJson)
	if err != nil {
		slog.Error("failed checking path exists", "err", err)
		return FormatUnknown, err
	}

	if exists {
		slog.Info("detected project.json file", "path", projJson)
		return FormatLegacyProjectJson, nil
	}

	slog.Error("could not infer exact legacy format", "file", projPath)
	return FormatUnknown, nil
}
