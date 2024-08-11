package common

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	cp "github.com/otiai10/copy"
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

func CopyDir(src string, dst string) error {
	err := cp.Copy(src, dst, cp.Options{PreserveTimes: true, PreserveOwner: true})
	if err != nil {
		slog.Error("copy failed", "err", err)
		return NewPrintableError("failed copying directory %s to %s", src, dst)
	}
	return nil
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
