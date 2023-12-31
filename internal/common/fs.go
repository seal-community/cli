package common

import (
	"log/slog"
	"os"
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
