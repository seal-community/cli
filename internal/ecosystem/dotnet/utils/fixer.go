package utils

import (
	"archive/zip"
	"bytes"
	"cli/internal/common"
	"cli/internal/ecosystem/shared"
	"crypto/sha512"
	"encoding/base64"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

// Files and directories to ignore when extracting the nuget package
var ignore = []string{"[Content_Types].xml", "_rels", "package"}

type fixer struct {
	rollback   map[string]string // original-dependency-path -> tmp-location
	projectDir string
	workdir    string
}

func (*fixer) Cleanup() bool {
	return true // No cleanup needed
}

func savePackageFiles(location, packageName string, nupkgData []byte) error {
	// Remove dir if exists, so we could download the most recent version, and so there will be no issues
	slog.Debug("removing directory", "location", location, "packageName", packageName)
	if err := os.RemoveAll(location); err != nil {
		slog.Error("failed removing directory", "err", err, "location", location)
		return err
	}

	if err := os.MkdirAll(location, os.ModePerm); err != nil {
		slog.Error("failed creating directory", "err", err, "location", location)
		return err
	}

	libraryPath := filepath.Join(location, packageName)
	if err := os.WriteFile(libraryPath, nupkgData, 0644); err != nil {
		slog.Error("failed writing nuget package to disk", "err", err, "path", libraryPath)
		return err
	}

	if err := ExtractPackage(location, nupkgData, ignore); err != nil {
		slog.Error("failed extracting nuget package", "err", err, "path", libraryPath)
		return err
	}
	if err := os.WriteFile(strings.ToLower(libraryPath+".sha512"), []byte(hashData(nupkgData)), 0644); err != nil {
		slog.Error("failed writing sha512 file", "err", err, "path", libraryPath)
		return err
	}

	files, err := os.ReadDir(location)
	if err != nil {
		slog.Error("failed reading directory", "err", err, "location", location)
		return err
	}
	for _, file := range files {
		for _, suffix := range []string{".nupkg", ".nuspec"} {
			if strings.HasSuffix(file.Name(), suffix) {
				slog.Debug("renaming file", "file", file.Name())
				currentPath := filepath.Join(location, file.Name())
				newPath := filepath.Join(location, strings.ToLower(file.Name()))
				if err := os.Rename(currentPath, newPath); err != nil {
					slog.Error("failed renaming file", "err", err, "file", file)
					return err
				}
			}
		}
	}

	return nil
}

func hashData(data []byte) string {
	hasher := sha512.New()
	hasher.Write(data)
	hash := hasher.Sum(nil)
	return base64.StdEncoding.EncodeToString(hash)
}

func ExtractPackage(location string, payload []byte, filesToSkip []string) error {
	r, err := zip.NewReader(bytes.NewReader(payload), int64(len(payload)))
	if err != nil {
		slog.Error("failed reading package", "err", err, "payloadLen", len(payload), "start", string(payload[:100]))
		return err
	}

	for _, file := range r.File {
		skip := false
		for _, skipFile := range filesToSkip {
			if strings.HasPrefix(file.Name, skipFile) {
				skip = true
				break
			}
		}

		if skip {
			slog.Debug("skipping file", "file", file.Name)
			continue
		}

		slog.Debug("extracting file", "file", file.Name)

		err = common.UnzipFile(file, location)
		if err != nil {
			return err
		}
	}

	return nil
}

// extract the data to the <HOME>/.nuget/packages/<Package>/<Version> cache folder
func (f *fixer) Fix(dep *common.Dependency, packageDownload shared.PackageDownload) (bool, error) {
	sealedVersion := packageDownload.PackageVersion.RecommendedLibraryVersionString
	location := filepath.Join(GetGlobalPackagesCachePath(), dep.Name, sealedVersion)
	packageName := fmt.Sprintf("%s.%s.nupkg", dep.Name, sealedVersion)
	err := savePackageFiles(location, packageName, packageDownload.Data)
	return err == nil, err
}

func (*fixer) Rollback() bool {
	return true // No need to rollback files, we dont modify the source version
}

func NewFixer(projectDir string, workdir string) shared.DependencyFixer {
	return &fixer{
		projectDir: projectDir,
		workdir:    workdir,
		rollback:   make(map[string]string, 100),
	}
}
