package common

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/otiai10/copy"
)

var CliCWD string // the working directory the cli started running from

func init() {
	var err error
	CliCWD, err = os.Getwd()
	if err != nil {
		// could happen in edge case, like https://stackoverflow.com/questions/13614048/go-why-does-os-getwd-sometimes-fail-with-eof
		panic("failed getting current working directory")
	}
}

func CreateFile(p string) (*os.File, error) {
	if f, err := os.Stat(p); err == nil {
		if f.IsDir() {
			slog.Error("path is a directory", "path", p)
			return nil, NewPrintableError("file path is a directory %s", p)
		}

		slog.Warn("existing file will be overwritten", "path", p)
	} else if !os.IsNotExist(err) {
		slog.Error("stat failed", "err", err)
		return nil, NewPrintableError("failed checking file exists %s", p)
	}

	f, err := os.Create(p)
	if err != nil {
		slog.Error("create file failed", "err", err)
		return nil, NewPrintableError("failed creating file %s", p)
	}
	return f, nil
}

func OpenFile(p string) (*os.File, error) {
	f, err := os.Open(p)
	if err != nil {
		if !os.IsNotExist(err) {
			slog.Info("failed opening file", "err", err, "path", p)
			return nil, NewPrintableError("could not open file in %s", p)
		}

		slog.Info("file not found", "path", p)
		return nil, nil

	}

	return f, err
}

func DumpBytes(path string, data []byte) error {
	f, err := CreateFile(path)
	if err != nil {
		slog.Error("failed creating file", "path", path, "err", err)
		return err
	}
	defer f.Close()

	_, err = f.Write(data)
	return err
}

func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}

	if os.IsNotExist(err) {
		return false, nil
	}

	return false, err
}

func FindPathsWithSuffix(path string, suffix string) ([]string, error) {
	paths := []string{}
	err := filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			slog.Error("walk failed", "err", err)
			return err
		}

		if strings.HasSuffix(strings.ToLower(info.Name()), strings.ToLower(suffix)) {
			paths = append(paths, p)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return paths, nil
}

func DirExists(path string) (bool, error) {
	fi, err := os.Stat(path)
	if err == nil {
		return fi.IsDir(), nil
	}

	if os.IsNotExist(err) {
		return false, nil
	}

	return false, err
}

// returns the absolute path for input's directory, if file will strip the file element
func GetAbsDirPath(p string) string {
	targetDir, _ := filepath.Abs(p) // ignoring err, propagated from internal call to os.Cwd

	f, err := os.Stat(targetDir)
	if err != nil {
		slog.Error("bad target dir", "err", err, "path", targetDir)
		return ""
	}

	if f.IsDir() {
		return targetDir
	}

	// strip input from file component
	return filepath.Dir(targetDir)
}

func IsDirEmpty(path string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer f.Close()

	_, err = f.Readdir(1)
	if err == io.EOF {
		return true, nil
	}
	return false, err
}

// converts a symlinked file to be a file by deleting the symlink and copying the file it points to
// returns error if part of the path besides the file itself is a symlink
// if the file is not a symlink, it will return without erroring
func ConvertSymLinkToFile(path string) error {
	slog.Info("converting symlink to file", "path", path)

	parentDir := filepath.Dir(path)
	resolvedParentDir, err := filepath.EvalSymlinks(parentDir)
	if err != nil {
		slog.Error("failed resolving symlink", "err", err, "path", parentDir)
		return err
	}

	if filepath.Clean(parentDir) != resolvedParentDir {
		slog.Error("parent directory path include symlinks", "path", parentDir, "resolved", resolvedParentDir)
		return NewPrintableError("failed converting symlink, parent directory is behind a symlink: %s", parentDir)
	}

	resolvedPath, err := filepath.EvalSymlinks(path)
	if err != nil {
		slog.Error("failed resolving symlink", "err", err, "path", path)
		return err
	}

	if filepath.Clean(path) == resolvedPath {
		slog.Info("path is not a symlink", "path", path)
		return nil
	}

	opts := copy.Options{
		PreserveTimes: true,
		PreserveOwner: true,
	}

	if err := os.Remove(path); err != nil {
		slog.Error("failed removing symlink", "err", err, "path", path)
		return err
	}

	if err := copy.Copy(resolvedPath, path, opts); err != nil {
		slog.Error("failed converting symlink to file", "err", err, "path", resolvedPath)
		return err
	}

	return nil
}
