package utils

import (
	"cli/internal/api"
	"cli/internal/common"
	"cli/internal/ecosystem/mappings"
	"cli/internal/ecosystem/shared"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

type PackageNotFoundError struct {
	Path        string
	PackageName string
}

func (e *PackageNotFoundError) Error() string {
	return fmt.Sprintf("Package %s status could not be determined in path: %s", e.PackageName, e.Path)
}

func NewPackageNotFoundError(path string, packageName string) error {
	return &PackageNotFoundError{Path: path, PackageName: packageName}
}

const SealPrefix = "seal-"

func groupPackagesByName(packages []StatusFilePackage) map[string][]*StatusFilePackage {
	// On debian the same package can be installed for different architectures
	packageMap := make(map[string][]*StatusFilePackage)

	for i := range packages {
		packageMap[packages[i].Package] = append(packageMap[packages[i].Package], &packages[i])
	}

	return packageMap
}

func isValidPackageFileSuffix(fileNameSuffix string) bool {
	return fileNameSuffix == "" || (len(fileNameSuffix) > 0 && (fileNameSuffix[0] == '.' || fileNameSuffix[0] == ':'))
}

func findPackageFiles(infoFilesPath string, packageName string) ([]string, error) {
	var matchedFiles []string

	err := filepath.Walk(infoFilesPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			baseName := filepath.Base(path)
			if strings.HasPrefix(baseName, packageName) {
				suffix := strings.TrimPrefix(baseName, packageName)
				if isValidPackageFileSuffix(suffix) {
					matchedFiles = append(matchedFiles, baseName)
				}
			}
		}
		return nil
	})

	return matchedFiles, err
}

func getInfoFilePath(infoFilesDirPath string, packageFileName string) string {
	return fmt.Sprintf("%s/%s", infoFilesDirPath, packageFileName)
}

func getNewInfoFilePath(infoFilesDirPath string, packageFileName string) string {
	new_file_name := SealPrefix + packageFileName
	return fmt.Sprintf("%s/%s", infoFilesDirPath, new_file_name)
}

func renameInfoFile(packageFileName string, infoFilesDirPath string) error {
	current_file_path := getInfoFilePath(infoFilesDirPath, packageFileName)
	new_file_path := getNewInfoFilePath(infoFilesDirPath, packageFileName)

	err := common.Move(current_file_path, new_file_path)
	if err != nil {
		slog.Error("failed renaming file", "current_path", current_file_path, "new_path", new_file_path)
		return err
	}

	return nil
}

func renameInfoFiles(rule api.SilenceRule, infoFilesDirPath string) error {
	packageFileNames, err := findPackageFiles(infoFilesDirPath, rule.Library)
	if err != nil {
		slog.Error("failed to fetch package info files", "err", err, "rule", rule)
		return err
	}
	slog.Debug("renaming package info files", "package", rule.Library, "files", packageFileNames)

	for _, packageFN := range packageFileNames {
		err := renameInfoFile(packageFN, infoFilesDirPath)
		if err != nil {
			slog.Error("failed renaming package info file", "package", rule.Library, "file", packageFN, "err", err)
			return err
		}
	}

	return nil
}

func modifyPackageStatusForSilence(pkg *StatusFilePackage) {
	packageRequirement := fmt.Sprintf("%s (<= %s)", pkg.Package, pkg.Version)

	pkg.Provides = fmt.Sprintf("%s (= %s)", pkg.Package, pkg.Version)
	pkg.Package = SealPrefix + pkg.Package
	pkg.Source = SealPrefix + pkg.Source

	requirementFields := []*string{&pkg.Conflicts, &pkg.Breaks, &pkg.Replaces}

	for _, field := range requirementFields {
		if *field == "" {
			*field = packageRequirement
		} else {
			*field += ", " + packageRequirement
		}
	}
}

func parseStatusFile(statusFilePath string) ([]StatusFilePackage, error) {
	slog.Debug("parsing status file.", "path", statusFilePath)
	fd, err := common.OpenFile(statusFilePath)
	if fd == nil {
		slog.Error("status file does not exist", "path", statusFilePath)
		return nil, NewPackageNotFoundError(statusFilePath, "")
	}
	if err != nil {
		slog.Error("failed opening status file for parsing", "err", err)
		return nil, err
	}

	parser := NewParser(fd)
	packages, err := parser.Parse()
	if err != nil {
		slog.Error("failed parsing status file.", "err", err)
		return nil, err
	}

	return packages, nil
}

func dumpStatusFile(packages []StatusFilePackage, statusFilePath string) error {
	newStatusFileContents := DumpPackages(packages)

	return common.DumpBytes(statusFilePath, []byte(newStatusFileContents))
}

func RenamePackage(
	silenceRule api.SilenceRule,
	statusFilePath string,
	infoFilesDirPath string,
	dependencyId string,
) error {
	slog.Debug("silencing package", "id", dependencyId)
	packages, err := parseStatusFile(statusFilePath)
	if err != nil {
		slog.Error("failed parsing status file", "path", statusFilePath, "err", err)
		return err
	}

	nameToStatusFilePackages := groupPackagesByName(packages)
	ruleStatusFilePackages, exists := nameToStatusFilePackages[silenceRule.Library]
	if !exists {
		slog.Error("could not find package in status file.", "package", silenceRule.Library, "status_file_packages", packages)
		return NewPackageNotFoundError(statusFilePath, silenceRule.Library)
	}
	for _, pkg := range ruleStatusFilePackages {
		modifyPackageStatusForSilence(pkg)
	}
	err = dumpStatusFile(packages, statusFilePath)
	if err != nil {
		slog.Error("failed dumping status file", "path", statusFilePath, "err", err)
		return err
	}

	err = renameInfoFiles(silenceRule, infoFilesDirPath)
	if err != nil {
		slog.Error("failed renaming info files", "info_files_dir", infoFilesDirPath, "err", err)
		return err
	}

	return nil
}

func SilencePackage(
	rule api.SilenceRule,
	allDependencies common.DependencyMap,
	statusFilePath string,
	infoFilesDirPath string,
) (string, []string, error) {
	ruleDependencyId := common.DependencyId(mappings.DebManager, rule.Library, rule.Version)
	if _, exists := allDependencies[ruleDependencyId]; !exists {
		slog.Error("target dependency doesn't exist for rule", "rule", rule, "dependencies", allDependencies)
		return ruleDependencyId, nil, NewPackageNotFoundError("", rule.Library)
	}

	err := RenamePackage(
		rule,
		statusFilePath,
		infoFilesDirPath,
		ruleDependencyId,
	)
	if err != nil {
		slog.Error("failed silencing package", "rule", rule, "err", err)
		return ruleDependencyId, nil, err
	}

	silencedPaths := []string{}

	for _, dep := range allDependencies[ruleDependencyId] {
		silencedPaths = append(silencedPaths, dep.DiskPath)
	}

	return ruleDependencyId, silencedPaths, err
}

func RenameFix(
	fix shared.DependencyDescriptor,
	statusFilePath string,
	InfoFilesDirPath string,
) error {
	slog.Debug("renaming fixed package", "fix", fix)
	if len(fix.Locations) == 0 {
		slog.Error("no locations found for fix", "fix", fix)
		return fmt.Errorf("fix has no locations")
	}

	dependenciesToRename := []common.Dependency{}
	for _, dep := range fix.Locations {
		dependenciesToRename = append(dependenciesToRename, dep)
	}

	silenceRule := api.SilenceRule{
		Manager: mappings.DebManager,
		Library: dependenciesToRename[0].Name,
		Version: dependenciesToRename[0].Version,
	}

	return RenamePackage(
		silenceRule,
		statusFilePath,
		InfoFilesDirPath,
		common.DependencyId(mappings.DebManager, dependenciesToRename[0].Name, dependenciesToRename[0].Version),
	)
}
