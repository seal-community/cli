package utils

import (
	"github.com/klauspost/compress/zip"
	"io"
	"log/slog"
	"path/filepath"
)

func GetZipFilePathsSet(file []*zip.File) map[string]bool {
	zipFilePaths := make(map[string]bool, 0)
	for _, zipItem := range file {
		zipFilePaths[filepath.ToSlash(zipItem.Name)] = true
	}

	return zipFilePaths
}

func AddFileToZip(zipWriter *zip.Writer, filePath string, file io.ReadCloser) error {
	targetItem, err := zipWriter.Create(filePath)
	if err != nil {
		slog.Error("failed creating zip item", "err", err, "path", filePath)
		return err
	}

	_, err = io.Copy(targetItem, file)
	if err != nil {
		slog.Error("failed copying zip item", "err", err, "path", filePath)
		return err
	}

	return nil
}
