package utils

import (
	"archive/zip"
	"bytes"
	"cli/internal/common"
	"cli/internal/ecosystem/shared"
	"encoding/csv"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

const recordFilename = "RECORD"

var distRecordPath = fmt.Sprintf(".dist-info/%s", recordFilename)

type fixer struct {
	rollback   map[string]string // original-dependency-path -> tmp-location
	projectDir string
	workdir    string
}

func unzipFile(file *zip.File, sitePackages string) error {
	target := filepath.Join(sitePackages, file.Name)
	if err := os.MkdirAll(filepath.Dir(target), os.ModePerm); err != nil {
		slog.Error("failed creating target dir", "err", err, "target", target)
		return err
	}

	targetFile, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
	if err != nil {
		slog.Error("failed creating file", "err", err, "file", target)
		return err
	}
	defer targetFile.Close()

	rc, err := file.Open()
	if err != nil {
		slog.Error("failed opening file", "err", err, "file", file.Name)
		return err
	}
	defer rc.Close()

	if _, err := io.Copy(targetFile, rc); err != nil {
		slog.Error("failed writing file", "err", err, "file", target)
		return err
	}

	slog.Debug("extracted file", "file", target)

	return nil
}

// Extracts the payload to the site-packages directory
// Returns the path to the .dist-info directory in site-packages
//
// The payload is a .whl file, which is actually a zip file.
// It's content should be places directly under site-packages.
// To rollback easily, we return the dist-info path, which should look like: `.../site-packages/<name>-<version>.dist-info`
func (f *fixer) extractPackage(sitePackagesPath string, payload []byte) (string, error) {
	// Open zipfile in memory
	r, err := zip.NewReader(bytes.NewReader(payload), int64(len(payload)))
	if err != nil {
		slog.Error("failed reading package", "err", err, "payloadLen", len(payload), "start", string(payload[:100]))
		return "", err
	}

	distInfoPath := ""
	for _, file := range r.File {
		slog.Debug("extracting file", "file", file.Name)

		err = unzipFile(file, sitePackagesPath)
		if err != nil {
			return "", err
		}

		if strings.HasSuffix(file.Name, distRecordPath) {
			if distInfoPath != "" {
				slog.Warn("multiple dist-info directories found", "path", distInfoPath, "new", file.Name)
			}

			distInfoPath = filepath.Dir(file.Name)
		}
	}

	return distInfoPath, nil
}

func parseRecordFile(recordFile io.Reader) ([]string, error) {
	csvReader := csv.NewReader(recordFile)

	files := make([]string, 0)
	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			slog.Error("failed reading RECORD file", "err", err)
			return nil, err
		}

		files = append(files, record[0])
	}

	return files, nil
}

// We use RECORD file to know what to move back when rolling back
// It is a CSV file where the first column includes the file path
// It includes all the files in the .whl package, including the .dist-info directory
func readRecordFile(path string) ([]string, error) {
	recordFile, err := os.Open(filepath.Join(path, recordFilename))
	if err != nil {
		slog.Error("failed reading RECORD file", "err", err)
		return nil, err
	}
	defer recordFile.Close()

	return parseRecordFile(recordFile)
}

func backupDependency(dep common.Dependency, src string, dst string, files []string) error {
	// Move the dependency to the temporary directory
	dirs := make([]string, 0)
	for _, file := range files {
		orig := filepath.Join(src, file)
		tmp := filepath.Join(dst, file)
		origDir := filepath.Dir(orig)

		if !slices.Contains(dirs, origDir) {
			dirs = append(dirs, origDir)
		}

		err := os.MkdirAll(filepath.Dir(tmp), os.ModePerm)
		if err != nil {
			slog.Error("failed creating target dir", "err", err, "target", tmp)
			return err
		}

		if err := os.Rename(orig, tmp); err != nil {
			slog.Error("failed moving original to temp dir", "err", err, "original", dep.DiskPath, "tmp-path", dst)
			return fmt.Errorf("failed backing up package %s", dep.PrintableName())
		}
	}

	// Remove directories from site-packages, since os.Rename for files did not remove them
	for _, dir := range dirs {
		slog.Debug("removing dir", "dir", dir)
		if err := os.RemoveAll(dir); err != nil {
			slog.Error("failed removing dir", "err", err, "dir", dir)
			return fmt.Errorf("failed removing directory %s", dir)
		}
	}

	return nil
}

// Will fix the dependency, assuming payload is a .whl file
func (f *fixer) Fix(dep *common.Dependency, payload []byte) (bool, error) {
	files, err := readRecordFile(dep.DiskPath)
	if err != nil {
		slog.Error("failed reading RECORD file", "err", err)
		return false, fmt.Errorf("failed reading RECORD file for package %s", dep.PrintableName())
	}

	// Create a temporary directory for the dependency
	tmpName := filepath.Join(f.workdir, "site-packages", dep.Name)
	err = os.MkdirAll(tmpName, os.ModePerm)
	if err != nil {
		slog.Error("failed creating tmp dir", "err", err)
		return false, fmt.Errorf("failed creating backup directory for package %s", dep.PrintableName())
	}

	sitePackages := filepath.Dir(dep.DiskPath)
	err = backupDependency(*dep, sitePackages, tmpName, files)
	if err != nil {
		return false, err
	}

	f.rollback[dep.DiskPath] = tmpName

	distInfoPath, err := f.extractPackage(sitePackages, payload)
	if err != nil {
		slog.Error("failed extracting package", "err", err, "target", sitePackages, "payloadLen", len(payload))
		return false, fmt.Errorf("failed applying fix for package %s", dep.PrintableName())
	}

	// Update diskPath so that fix summary will show a real path
	if distInfoPath != "" {
		dep.DiskPath = filepath.Join(sitePackages, distInfoPath)
	} else {
		dep.DiskPath = sitePackages
	}

	slog.Info("fixed package instance", "path", dep.DiskPath)
	return true, nil
}

func (f *fixer) Rollback() bool {
	finishedOk := true
	for orig, tmpName := range f.rollback {
		if err := f.rollbackDependecy(tmpName, orig); err != nil {
			finishedOk = false
		}
	}

	return finishedOk
}

func (f *fixer) rollbackDependecy(from string, to string) error {
	slog.Debug("rolling back", "from", from, "to", to)

	// move each dir under `to` to `from`
	dirs, err := os.ReadDir(from)
	if err != nil {
		slog.Error("failed reading dir", "err", err, "dir", from)
		return err
	}

	sitePackages := filepath.Dir(to)
	for _, d := range dirs {
		fromDir := filepath.Join(from, d.Name())
		toDir := filepath.Join(sitePackages, d.Name())

		if err := os.Rename(fromDir, toDir); err != nil {
			slog.Error("failed rollback", "err", err, "from", fromDir, "to", toDir)
			// greedy try to restore as much as possible
			return err
		}
	}

	return nil
}

func (f *fixer) Cleanup() bool {
	finishedOk := true
	for orig, tmpName := range f.rollback {
		if err := os.RemoveAll(tmpName); err != nil {
			slog.Error("failed removing tmp dir", "orig", orig, "tmp", tmpName)
			finishedOk = false
		}
	}

	return finishedOk
}

func NewFixer(projectDir string, workdir string) shared.DependencyFixer {
	return &fixer{
		projectDir: projectDir,
		workdir:    workdir,
		rollback:   make(map[string]string, 100),
	}
}