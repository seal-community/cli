package utils

import (
	"archive/tar"
	"cli/internal/common"
	"compress/gzip"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

const directoryPermissions = 0755

func createDir(target string) error {
	if err := os.MkdirAll(target, directoryPermissions); err != nil && !os.IsExist(err) {
		return err
	}

	return nil
}

func writeFile(target string, src io.Reader, entry *tar.Header) error {
	mode := entry.FileInfo().Mode()

	outFile, err := os.OpenFile(target, os.O_RDWR|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	
	defer outFile.Close()

	bytesWritten, err := io.Copy(outFile, src)
	if err != nil {
		slog.Error("failed writing to targer path", "err", err, "path", target)
		return err
	}

	if bytesWritten != entry.Size {
		slog.Error("did not write entire file", "written", bytesWritten, "total", entry.Size)
		return fmt.Errorf("file not written to disk entirely")
	}

	return nil
}

func isIllegalPath(path string) bool {
	// minimal validation
	if filepath.IsAbs(path) || filepath.VolumeName(path) != "" || strings.HasPrefix(path, `\`) {
		return true
	}

	for _, forbidden := range []string{`../`, `..\`} {
		if strings.Contains(path, forbidden) {
			return true
		}
	}

	return false
}

func getTargetPathForNpm(outputDir string, relativePath string) string {
	// extract contents of `package/...` directly
	entryPath := filepath.FromSlash(relativePath)
	prefix := "package" + string(filepath.Separator)
	entryPath = strings.TrimPrefix(entryPath, prefix)
	return filepath.Join(outputDir, entryPath)
}

func UntarNpmPackage(r io.Reader, outputDir string) error {
	gzReader, err := gzip.NewReader(r)
	if err != nil {
		slog.Error("failed to create gzip reader", "err", err)
		return err
	}

	tr := tar.NewReader(gzReader)
	entry, err := tr.Next()
	for ; err == nil; entry, err = tr.Next() {
		common.Trace("parsing tar entry", "path", entry.Name)
		if entry.Name == "" {
			slog.Error("empty path for entry", "path", entry.Name)
			return fmt.Errorf("bad tar entry path empty")
		}

		if isIllegalPath(entry.Name) {
			slog.Error("path for entry contains illegal substr", "path", entry.Name)
			return fmt.Errorf("bad tar entry path")
		}

		target := getTargetPathForNpm(outputDir, entry.Name)
		common.Trace("target path for entry", "path", target, "type", entry.Typeflag)

		switch entry.Typeflag {
		case tar.TypeSymlink:
			slog.Warn("symlink type unsupported", "entry", entry.Name)
			return fmt.Errorf("entry is symlink")
		case tar.TypeReg:

			if entry.FileInfo().Mode()&fs.ModeType != 0 {
				// catch non regular files like symlinks etc
				slog.Error("file entry unsupported flags", "path", target, "mode", entry.FileInfo().Mode())
				return fmt.Errorf("file entry is not regular")
			}

			dir := filepath.Dir(target)
			if err := createDir(dir); err != nil {
				slog.Error("failed creating dir for file entry", "path", target)
				return err
			}
			if err := writeFile(target, tr, entry); err != nil {
				slog.Error("failed writing file entry", "path", target)
				return err
			}

		case tar.TypeDir:
			if err := createDir(target); err != nil {
				slog.Error("failed creating dir for dir entry", "path", target)
				return err
			}
		default:
			slog.Warn("unsupported type in untar", "type", entry.Typeflag)
		}
	}

	if err != io.EOF {
		slog.Error("failed getting next tar entry")
		return err
	}

	return nil
}
