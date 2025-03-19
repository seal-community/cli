//go:build !windows

package common

import (
	"log/slog"
	"os"
	"syscall"
)

func GetFileStats(path string) (*UnixStat, error) {
	fileInfo, err := os.Stat(path)
	if err != nil {
		slog.Error("failed getting file stats", "err", err, "path", path)
		return nil, err
	}

	stat, ok := fileInfo.Sys().(*syscall.Stat_t)
	if !ok {
		slog.Error("failed getting file stats", "path", path)
		return nil, NewPrintableError("failed getting file stats")
	}

	unixStat := &UnixStat{
		Uid: stat.Uid,
		Gid: stat.Gid,
	}

	return unixStat, nil
}
