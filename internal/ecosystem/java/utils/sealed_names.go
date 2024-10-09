package utils

import (
	"archive/zip"
	"bufio"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

const pomPropertiesTemplate = `artifactId=%s
groupId=seal
version=%s
`

const symbolicName = "Bundle-SymbolicName"

// Creates a new jar with the following changes:
//  1. META-INF/MANIFEST.MF - changing the symbolic name
//  2. META-INF/maven/<groupId>/<artifactId>/pom.properties - changing the version to be the original one
//     The artifactId to be the original one, and the groupId to be "seal"
func CreateSealedNameJar(jarPath, groupId, artifactId, originalVersion string) (string, error) {
	slog.Info("Creating new sealed name jar", "jarPath", jarPath)

	pomPropertiesFilePath := filepath.ToSlash(filepath.Join("META-INF/maven", groupId, artifactId, "pom.properties"))
	manifestFilePath := filepath.ToSlash(filepath.Join("META-INF", "MANIFEST.MF"))

	origReader, err := zip.OpenReader(jarPath)
	defer origReader.Close()
	if err != nil {
		slog.Error("failed reading package", "err", err, "path", jarPath)
		return "", err
	}

	sealedNameFile, err := os.CreateTemp("./.seal", "tmp_jar")
	if err != nil {
		slog.Error("failed creating sealed file", "err", err, "path", jarPath)
		return "", err
	}
	defer sealedNameFile.Close()

	sealedNameFilePath := sealedNameFile.Name()

	sealedZipWriter := zip.NewWriter(sealedNameFile)
	defer sealedZipWriter.Close()

	for _, zipItem := range origReader.File {
		zipItemReader, err := zipItem.Open()
		if err != nil {
			slog.Error("failed opening zip item", "err", err, "path", zipItem.Name)
			return "", err
		}

		header := zipItem.FileHeader
		targetItem, err := sealedZipWriter.CreateHeader(&header)
		if err != nil {
			slog.Error("failed creating zip item header", "err", err, "path", zipItem.Name)
			return "", err
		}

		itemReader := zipItemReader
		if filepath.ToSlash(header.Name) == pomPropertiesFilePath {
			itemReader = getSealedPomProperties(artifactId, originalVersion)
		} else if filepath.ToSlash(header.Name) == manifestFilePath {
			itemReader = getSealedManifest(zipItemReader, artifactId)
		}

		_, err = io.Copy(targetItem, itemReader)
		if err != nil {
			slog.Error("failed copying zip item", "err", err, "path", zipItem.Name)
			return "", err
		}
	}

	return sealedNameFilePath, nil
}

func getSealedPomProperties(artifactId, version string) io.ReadCloser {
	return io.NopCloser(strings.NewReader(fmt.Sprintf(pomPropertiesTemplate, artifactId, version)))
}

// Change the symbolic name value in the manifest to be `seal.<artifactId>`
// Returns a ReadCloser object created from the new manifest string
func getSealedManifest(manifestReader io.Reader, artifactId string) io.ReadCloser {
	newManifest := ""
	newSymbolicName := fmt.Sprintf("%s: %s\n", symbolicName, "seal."+artifactId)
	changed := false
	scanner := bufio.NewScanner(manifestReader)

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, symbolicName) {
			newManifest += newSymbolicName
			changed = true
		} else {
			newManifest += line + "\n"
		}
	}

	if !changed {
		slog.Debug("manifest does not contain symbolic name, adding it", "symbolicName", symbolicName, "artifactId", artifactId)
		newManifest = newSymbolicName + newManifest
	}

	return io.NopCloser(strings.NewReader(newManifest))
}
