package common

import (
	"errors"
	"log/slog"
	"os"
	"syscall"

	"github.com/otiai10/copy"
)

// tries to use os.Rename first
// handles cross-device link errors by copying to destination and removing source
// in this case, will only shallow-copy links
func Move(source, destination string) error {
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
	opts := copy.Options{
		PreserveTimes: true,
		PreserveOwner: true,
	}

	if err := copy.Copy(source, destination, opts); err != nil {
		slog.Error("copy failed", "err", err, "src", source, "dst", destination)
		if rmErr := os.RemoveAll(destination); rmErr != nil {
			// attempting to clean if copy failed midway, nothing we can do if this fails
			slog.Warn("attempted removal of destination", "err", rmErr)
		}

		return err
	}

	// Remove the original file.
	if err := os.RemoveAll(source); err != nil {
		slog.Error("remove failed:", "err", err)
		return err
	}

	return nil
}
