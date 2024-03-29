package common

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"
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
		slog.Info("actions file not found", "path", p)
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

func FindFileWithSuffix(path string, suffix string) (string, error) {
	suffix = strings.ToLower(suffix)
	var found string
	err := filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			slog.Error("walk failed", "err", err)
			return err
		}
		if info.IsDir() {
			return nil
		}
		if strings.HasSuffix(strings.ToLower(info.Name()), suffix) {
			found = p
			return filepath.SkipDir
		}
		return nil
	})
	if err != nil {
		return "", err
	}

	if found == "" {
		return "", NewPrintableError("no file found with suffix %s in %s", suffix, path)
	}

	return found, nil
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
