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
	rollback       map[string]string // original-dependency-path -> tmp-location
	rollbackRemove []string          // sp-version dist-info paths
	projectDir     string
	workdir        string
}

// Extracts the payload to the site-packages directory
// Returns the path to the .dist-info directory in site-packages
//
// The payload is a .whl file, which is actually a zip file.
// It's content should be places directly under site-packages.
// To rollback easily, we return the dist-info path, which should look like: `.../site-packages/<name>-<version>.dist-info`
//
// The function will append any dot-dot paths found in the original RECORD file to the new RECORD file
// dot-dot paths in the original RECORD are paths created during installation of the original package, such that their content can't be overriden by the sealed package
// They should be kept as is
func (f *fixer) extractPackage(sitePackagesPath string, payload []byte, dotdotPaths []string) (string, error) {
	// Open zipfile in memory
	r, err := zip.NewReader(bytes.NewReader(payload), int64(len(payload)))
	if err != nil {
		slog.Error("failed reading package", "err", err, "payloadLen", len(payload), "start", string(payload[:100]))
		return "", err
	}

	distInfoPath := ""
	for _, file := range r.File {
		common.Trace("extracting file", "file", file.Name)

		err = common.UnzipFile(file, sitePackagesPath)
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

	if distInfoPath == "" {
		slog.Warn("no dist-info directory found")
		return "", nil
	}

	f.rollbackRemove = append(f.rollbackRemove, filepath.Join(sitePackagesPath, distInfoPath))

	if len(dotdotPaths) == 0 {
		return distInfoPath, nil
	}

	// Append dot-dot paths to the new RECORD file
	recordFile, err := os.OpenFile(filepath.Join(sitePackagesPath, distInfoPath, recordFilename), os.O_APPEND|os.O_WRONLY, os.ModePerm)
	if err != nil {
		slog.Error("failed reading RECORD file while appending dot-dot paths", "err", err)
		return distInfoPath, err
	}
	defer recordFile.Close()

	// append dot-dot paths to the new RECORD file
	for _, p := range dotdotPaths {
		if _, err := recordFile.WriteString(p + ",,\r\n"); err != nil {
			slog.Error("failed writing to RECORD file", "err", err)
			return distInfoPath, err
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
			slog.Error("failed parsing RECORD file", "err", err)
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

		if filepath.Base(file) != file && !slices.Contains(dirs, origDir) {
			dirs = append(dirs, origDir)
		}

		err := os.MkdirAll(filepath.Dir(tmp), os.ModePerm)
		if err != nil {
			slog.Error("failed creating target dir while backing up", "err", err, "target", tmp)
			return err
		}

		if err := os.Rename(orig, tmp); err != nil {
			slog.Error("failed moving original to temp dir", "err", err, "original", dep.DiskPath, "tmp-path", dst)
			return fmt.Errorf("failed backing up package %s", dep.PrintableName())
		}
	}

	// Remove directories from site-packages, since os.Rename for files did not remove them
	for _, dir := range dirs {
		common.Trace("removing dir", "dir", dir)
		if err := os.RemoveAll(dir); err != nil {
			slog.Error("failed removing dir", "err", err, "dir", dir)
			return fmt.Errorf("failed removing directory %s", dir)
		}
	}

	return nil
}

func splitDotdotPaths(paths []string) ([]string, []string) {
	dotdotPaths := make([]string, 0)
	absPaths := make([]string, 0)

	for _, p := range paths {
		if strings.HasPrefix(p, "..") {
			dotdotPaths = append(dotdotPaths, p)
		} else {
			absPaths = append(absPaths, p)
		}
	}

	return absPaths, dotdotPaths
}

// Will fix the dependency, assuming payload is a .whl file
func (f *fixer) Fix(entry shared.DependnecyDescriptor, dep *common.Dependency, packageData []byte) (bool, error) {
	recordPaths, err := readRecordFile(dep.DiskPath)
	if err != nil {
		return false, fmt.Errorf("failed reading RECORD file for package %s", dep.PrintableName())
	}

	recordPaths, dotdotPaths := splitDotdotPaths(recordPaths)

	// Create a temporary directory for the dependency
	tmpName := filepath.Join(f.workdir, "site-packages", dep.Name)
	err = os.MkdirAll(tmpName, os.ModePerm)
	if err != nil {
		slog.Error("failed creating tmp dir", "err", err)
		return false, fmt.Errorf("failed creating backup directory for package %s", dep.PrintableName())
	}

	sitePackages := filepath.Dir(dep.DiskPath)
	err = backupDependency(*dep, sitePackages, tmpName, recordPaths)
	if err != nil {
		return false, err
	}

	f.rollback[dep.DiskPath] = tmpName

	distInfoPath, err := f.extractPackage(sitePackages, packageData, dotdotPaths)
	if err != nil {
		slog.Error("failed extracting package", "err", err, "target", sitePackages, "payloadLen", len(packageData))
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

	for _, distInfoPath := range f.rollbackRemove {
		if err := os.RemoveAll(distInfoPath); err != nil {
			slog.Error("failed removing dist-info", "err", err, "path", distInfoPath)
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

		// remove the target dir, ignore if doesn't exist
		_ = os.RemoveAll(toDir)

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
