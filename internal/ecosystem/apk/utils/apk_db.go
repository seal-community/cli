package utils

import (
	"fmt"
	"log/slog"
	"strings"
)

const ApkDBPath = "/lib/apk/db/installed" // Works for all alpine versions
const NamePrefix = "P:"
const ProvidesPrefix = "p:"
const VersionPrefix = "V:"

type PackageInfoEntry struct {
	Value     string
	LineIndex int
}

type PackageInfo struct {
	Name     PackageInfoEntry
	Version  PackageInfoEntry
	Provides PackageInfoEntry
}

func parseAPKDBLine(packageInfo *PackageInfo, line string, lineIndex int) {
	var entry *PackageInfoEntry
	prefix, value := line[:2], line[2:]

	switch prefix {
	case NamePrefix:
		entry = &packageInfo.Name
	case ProvidesPrefix:
		entry = &packageInfo.Provides
	case VersionPrefix:
		entry = &packageInfo.Version
	default:
		return
	}

	entry.Value = value
	entry.LineIndex = lineIndex
}

func parseAPKDB(db string) map[string]PackageInfo {
	// Based on https://wiki.alpinelinux.org/wiki/Apk_spec#APKINDEX_Format
	var packageMap map[string]PackageInfo = make(map[string]PackageInfo)
	var currPackage PackageInfo

	lines := strings.Split(db, "\n")
	for i, line := range lines {
		// Empty line means end of package info
		if line != "" {
			parseAPKDBLine(&currPackage, line, i)
			continue
		}

		if currPackage.Name.Value != "" {
			packageMap[currPackage.Name.Value] = currPackage
			currPackage = PackageInfo{}
		}
	}

	return packageMap
}

func changeLineInFile(fileContent string, lineIndex int, newLine string) string {
	newContent := ""
	lines := strings.Split(fileContent, "\n")
	for i, line := range lines {
		if i == lineIndex {
			newContent += newLine + "\n"
		} else {
			newContent += line + "\n"
		}
	}

	return newContent[:len(newContent)-1] // remove last newline
}

func modifyApkDBContentForSilence(dbContent string, packageInfo PackageInfo, newPackageName string) string {
	selfProvides := fmt.Sprintf("%s=%s", packageInfo.Name.Value, packageInfo.Version.Value)
	newNameLine := fmt.Sprintf("P:%s", newPackageName)

	if packageInfo.Provides.Value == "" {
		slog.Debug("package does not provide anything, adding new provides entry", "package", packageInfo.Name.Value, "added", selfProvides)
		newNameLine += fmt.Sprintf("\np:%s", selfProvides) // add new provides line after the package name line
	} else {
		slog.Debug("adding to existing provides entry", "package", packageInfo.Name.Value, "added", selfProvides)
		newProvide := fmt.Sprintf("p:%s %s", packageInfo.Provides.Value, selfProvides)
		dbContent = changeLineInFile(dbContent, packageInfo.Provides.LineIndex, newProvide)
	}

	return changeLineInFile(dbContent, packageInfo.Name.LineIndex, newNameLine)
}
