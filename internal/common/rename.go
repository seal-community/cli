package common

import (
	"errors"
	"github.com/otiai10/copy"
	"log/slog"
	"os"
	"syscall"
)

// MoveFile safely moves a file, handling cross-device link errors.
func MoveFile(source, destination string) error {
	// Attempt to use os.Rename().
	err := os.Rename(source, destination)
	if err == nil {
		return nil
	}

	// Check if the error is of type *os.LinkError
	var linkErr *os.LinkError
	if !errors.As(err, &linkErr) {
		slog.Error("rename failed:", "err", err)
		return err
	}

	// Check the underlying syscall error
	errno, ok := linkErr.Err.(syscall.Errno)
	if !ok || errno != syscall.EXDEV {
		slog.Error("rename failed:", "err", err, "message", errno.Error())
		return err
	}

	slog.Debug("Cross-device link detected (EXDEV).")
	// Handle cross-device move logic here (e.g., copy and delete)

	// Copy the file.
	if err := copy.Copy(source, destination); err != nil {
		slog.Error("copy failed:", "err", err)
		return err
	}

	// Remove the original file.
	if err := os.RemoveAll(source); err != nil {
		slog.Error("remove failed:", "err", err)
		return err
	}

	return nil
}
