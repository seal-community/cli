package common

import (
	"archive/zip"
	"io"
	"log/slog"
	"os"
	"path/filepath"
)

func UnzipFile(file *zip.File, location string) error {
	Trace("extracting file", "file", file.Name, "location", location, "location+file", filepath.Join(location, file.Name))
	target := filepath.Join(location, file.Name)
	if file.FileInfo().IsDir() {
		if err := os.MkdirAll(target, os.ModePerm); err != nil {
			slog.Error("failed creating dir for dir zip record", "err", err, "target", file.Name)
			return err
		}

		return nil
	}

	if err := os.MkdirAll(filepath.Dir(target), os.ModePerm); err != nil {
		slog.Error("failed creating target dir while extracting", "err", err, "target", target)
		return err
	}

	targetFile, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
	if err != nil {
		slog.Error("failed creating file", "err", err, "file", target)
		return err
	}

	defer targetFile.Close()

	rc, err := file.Open()
	if err != nil {
		slog.Error("failed opening file", "err", err, "file", file.Name)
		return err
	}

	defer rc.Close()

	if _, err := io.Copy(targetFile, rc); err != nil {
		slog.Error("failed writing file", "err", err, "file", target)
		return err
	}

	Trace("extracted file", "file", target)

	return nil
}

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
		slog.Error("failed creating zip item header", "err", err, "path", filePath)
		return err
	}

	_, err = io.Copy(targetItem, file)
	if err != nil {
		slog.Error("failed copying zip item", "err", err, "path", filePath)
		return err
	}

	return nil
}
