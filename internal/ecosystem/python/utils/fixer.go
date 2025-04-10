package utils

import (
	"archive/zip"
	"bufio"
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
	"sort"
	"strings"
)

const recordFilename = "RECORD"
const installedFilesFilename = "installed-files.txt"

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
// dot-dot paths in the original RECORD are paths created during installation of the original package, such that their content can't be overridden by the sealed package
// They should be kept as is
func (f *fixer) extractWhlPackage(sitePackagesPath string, payload []byte, dotdotPaths []string) (string, error) {
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

	for _, p := range dotdotPaths {
		if _, err := recordFile.WriteString(p + ",,\r\n"); err != nil {
			slog.Error("failed writing to RECORD file", "err", err)
			return distInfoPath, err
		}
	}

	return distInfoPath, nil
}

func writePipOutput(output string) (string, error) {
	// write the pip output to a file and do not remove it, so user can inspect it
	pipOutput, err := os.CreateTemp("", "seal-pip-output-*.log")
	if err != nil {
		slog.Error("failed creating pip output file", "err", err)
		return "", err
	}
	defer pipOutput.Close()

	err = os.WriteFile(pipOutput.Name(), []byte(output), os.ModePerm)
	if err != nil {
		slog.Error("failed writing pip output to file", "err", err)
		return "", err
	}
	slog.Error("pip failed to install source package", "err", err, "pipOutput", pipOutput.Name())

	return pipOutput.Name(), nil
}

func (f *fixer) extractSourcePackage(sitePackagesPath string, dep common.Dependency, payload []byte, fileName string) (string, error) {
	srcName := strings.TrimSuffix(fileName, ".tar.gz")
	slog.Debug("source package name", "name", srcName)

	tmpDir, err := os.MkdirTemp("", "seal-source-package-*")
	if err != nil {
		slog.Error("failed creating temp dir", "err", err)
		return "", err
	}
	defer os.RemoveAll(tmpDir)

	tmpPath := filepath.Join(tmpDir, fileName)
	err = os.WriteFile(tmpPath, payload, os.ModePerm)
	if err != nil {
		slog.Error("failed writing source package to temp file", "err", err)
		return "", err
	}
	defer os.Remove(tmpPath)

	// run pip install on the temporary file
	slog.Info("installing source package via pip", "path", tmpPath)
	pr, err := common.RunCmdWithArgsCombinedOutput(f.workdir, "pip", "install", tmpPath)
	if err != nil {
		slog.Error("failed to run pip install", "err", err)
		return "", err
	}

	common.Trace("pip source installed", "output", pr.Stdout)
	if pr.Code != 0 {
		slog.Error("pip failed installing source package", "exitcode", pr.Code, "path", tmpPath)

		pipOutput, err := writePipOutput(pr.Stdout)
		if err != nil {
			// log the error to not lose it in case of double failure
			slog.Error("failed writing pip output", "err", err, "output", pr.Stdout)
			return "", err
		}

		return "", common.NewPrintableError("failed installing %s from source, this is probably an issue with pip, check its output at %s", srcName, pipOutput)
	}

	f.rollbackRemove = append(f.rollbackRemove, dep.DiskPath)

	distInfoPath := filepath.Join(sitePackagesPath, fmt.Sprintf("%s.dist-info", srcName))
	return distInfoPath, nil
}

func (f *fixer) extractPackage(sitePackagesPath string, dep common.Dependency, payload []byte, dotdotPaths []string, fileName string) (string, error) {
	switch {
	case strings.HasSuffix(fileName, ".whl"):
		return f.extractWhlPackage(sitePackagesPath, payload, dotdotPaths)
	case strings.HasSuffix(fileName, ".tar.gz"):
		return f.extractSourcePackage(sitePackagesPath, dep, payload, fileName)
	}

	return "", fmt.Errorf("unsupported package type %s", fileName)
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

// installed-files.txt is a simple text file where each line is a file path
// that the package installed.
// We need to convert these paths to relative paths to the the
// site-packages directory like the RECORD file.
// The function receives the installed-files.txt io.Reader
// and basePath which is the install location (egg-info directory)
// since the paths in the file are originally relative to it.
// It returns the list of relative paths to the site-packages directory
func parseInstalledFilesFile(installedFilesFile io.Reader, basePath string) ([]string, error) {
	files := make([]string, 0)
	scanner := bufio.NewScanner(installedFilesFile)

	for scanner.Scan() {
		path := scanner.Text()
		absPath, err := filepath.Abs(filepath.Join(basePath, path))
		if err != nil {
			slog.Error("failed converting to absolute path", "err", err, "path", path)
			return nil, err
		}

		relPath, err := filepath.Rel(filepath.Dir(basePath), absPath)
		if err != nil {
			slog.Error("failed converting to relative path", "err", err, "path", absPath)
			return nil, err
		}
		files = append(files, relPath)
	}

	return files, nil
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

		if err := common.Move(orig, tmp); err != nil {
			slog.Error("failed moving original to temp dir", "err", err, "original", orig, "tmp-path", dst)
			return fmt.Errorf("failed backing up package %s", dep.PrintableName())
		}
	}

	// Remove directories from site-packages, since os.Rename for files did not remove them
	// we need to sort by length, since we want to remove the deepest directories first
	// to avoid removing parent directories that are not empty
	sort.Slice(dirs, func(i, j int) bool {
		return len(dirs[i]) > len(dirs[j])
	})

	for _, dir := range dirs {
		isEmpty, err := common.IsDirEmpty(dir)
		if err != nil {
			slog.Error("failed checking if dir is empty", "err", err, "dir", dir)
			return fmt.Errorf("failed checking if directory %s is empty", dir)
		}

		if !isEmpty {
			slog.Debug("dir not empty, skipping removal", "dir", dir)
			continue
		}
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

func (f *fixer) Prepare() error {
	return nil
}

// We use RECORD/installed-files file to know what to move back when rolling back
// RECORD is a CSV file where the first column includes the file path
// installed-files is a txt file where each line is a file path
// both includes all the files in the .whl package
func getBackupPaths(path string) ([]string, error) {
	recordFile, err := os.Open(filepath.Join(path, recordFilename))
	if err == nil {
		defer recordFile.Close()
		slog.Info("reading RECORD file", "path", path)
		return parseRecordFile(recordFile)
	}

	slog.Info("failed to find RECORD file, trying installed-files.txt file", "err", err)

	installedFiles, err := os.Open(filepath.Join(path, installedFilesFilename))
	if err == nil {
		defer installedFiles.Close()
		slog.Info("reading installed-files.txt file", "path", path)
		return parseInstalledFilesFile(installedFiles, path)
	}

	slog.Error("failed reading RECORD and installed-files.txt files for path", "path", path, "err", err)
	return nil, fmt.Errorf("failed reading RECORD and installed-files.txt files for path %s", path)
}

func (f *fixer) Fix(entry shared.DependencyDescriptor, dep *common.Dependency, packageData []byte, fileName string) (bool, string, error) {
	backupPaths, err := getBackupPaths(dep.DiskPath)
	if err != nil {
		slog.Error("failed getting backup paths", "err", err)
		return false, "", err
	}

	backupPaths, dotdotPaths := splitDotdotPaths(backupPaths)

	// Create a temporary directory for the dependency
	tmpName := filepath.Join(f.workdir, "site-packages", dep.Name)
	err = os.MkdirAll(tmpName, os.ModePerm)
	if err != nil {
		slog.Error("failed creating tmp dir", "err", err)
		return false, "", fmt.Errorf("failed creating backup directory for package %s", dep.PrintableName())
	}

	sitePackages := filepath.Dir(dep.DiskPath)
	err = backupDependency(*dep, sitePackages, tmpName, backupPaths)
	if err != nil {
		return false, "", err
	}

	f.rollback[dep.DiskPath] = tmpName

	distInfoPath, err := f.extractPackage(sitePackages, *dep, packageData, dotdotPaths, fileName)
	if err != nil {
		slog.Error("failed extracting package", "err", err, "target", sitePackages, "payloadLen", len(packageData))
		return false, "", common.FallbackPrintableMsg(err, "failed applying fix for package %s", dep.PrintableName())
	}

	fixedLocation := sitePackages
	if distInfoPath != "" {
		fixedLocation = filepath.Join(sitePackages, distInfoPath)
	}

	slog.Info("fixed package instance", "path", dep.DiskPath, "fixedPath", fixedLocation)
	return true, fixedLocation, nil
}

func (f *fixer) Rollback() bool {
	finishedOk := true
	for orig, tmpName := range f.rollback {
		if err := f.rollbackDependency(tmpName, orig); err != nil {
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

func (f *fixer) rollbackDependency(from string, to string) error {
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
			slog.Error("failed removing tmp dir", "orig", orig, "tmp", tmpName, "err", err)
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
