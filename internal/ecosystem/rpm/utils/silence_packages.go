package utils

import (
	"cli/internal/api"
	"cli/internal/common"
	"cli/internal/ecosystem/mappings"
	"fmt"
	"log/slog"
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

var SealPrefix = []byte("seal-")

const packageNameTag = 1000
const sourcePackageTag = 1044
const minSignatureTag = 200
const maxSignatureTag = 300

func addSealPrefixInBlob(blob []byte) ([]byte, error) {
	header := createHeaderBlob(blob)
	if header == nil {
		return nil, fmt.Errorf("failed parsing header blob")
	}

	slog.Debug("adding seal prefix in blob", "header", header)
	header.modifyTagContent(packageNameTag, append(SealPrefix, header.getEntry(packageNameTag).Content...))
	if header.hasEntry(sourcePackageTag) {
		slog.Debug("adding seal prefix in source package tag")
		header.modifyTagContent(sourcePackageTag, append(SealPrefix, header.getEntry(sourcePackageTag).Content...))
	}

	slog.Debug("removing signature tags")
	for _, entry := range header.iterateValues() {
		if minSignatureTag <= entry.Tag && entry.Tag < maxSignatureTag {
			header.removeEntry(entry.Tag)
		}
	}

	return header.dumpBytes(), nil
}

func renamePackage(silenceRule api.SilenceRule, dependencyId string) error {
	slog.Debug("silencing package", "id", dependencyId)
	db, err := connectToRpmSQLiteDB()
	if err != nil {
		slog.Error("failed connecting to rpm db", "err", err)
		return err
	}
	defer db.Close()

	hnum, blob, err := getRpmSQLiteDBPackageData(db, silenceRule.Library)
	if err != nil {
		slog.Error("failed getting rpm db package data", "err", err)
		return err
	}

	blob, err = addSealPrefixInBlob(blob)
	if err != nil {
		slog.Error("failed adding seal prefix in blob", "err", err)
		return err
	}

	sealedName := string(SealPrefix) + silenceRule.Library
	slog.Debug("updating package in rpm DB", "sealedName", sealedName)
	err = updatePackageSQLite(db, hnum, blob, sealedName)
	if err != nil {
		slog.Error("failed updating package", "err", err)
		return err
	}

	slog.Debug("package silenced successfully")
	return nil
}

func SilencePackage(rule api.SilenceRule, allDependencies common.DependencyMap) (string, []string, error) {
	ruleDependencyId := common.DependencyId(mappings.RpmManager, rule.Library, rule.Version)
	if _, exists := allDependencies[ruleDependencyId]; !exists {
		slog.Error("target dependency doesn't exist for rule", "rule", rule, "dependencies", allDependencies)
		return ruleDependencyId, nil, NewPackageNotFoundError("", rule.Library)
	}

	err := renamePackage(rule, ruleDependencyId)
	if err != nil {
		slog.Error("failed silencing package", "rule", rule, "err", err)
		return ruleDependencyId, nil, err
	}

	silencedPaths := []string{}
	for _, dep := range allDependencies[ruleDependencyId] {
		silencedPaths = append(silencedPaths, dep.DiskPath)
	}

	return ruleDependencyId, silencedPaths, nil
}
