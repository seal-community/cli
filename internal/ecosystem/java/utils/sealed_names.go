package utils

import (
	"archive/zip"
	"bufio"
	"cli/internal/common"
	"cli/internal/ecosystem/mappings"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

const symbolicName = "Bundle-SymbolicName"
const impVendorId = "Implementation-Vendor-Id"
const sealGroupId = "seal"
const pomXMLFileName = "pom.xml"
const pomPropertiesFileName = "pom.properties"
const manifestFileName = "MANIFEST.MF"

// Creates a new jar with the following changes:
//  1. META-INF/MANIFEST.MF - changing the symbolic name
//  2. META-INF/maven/<groupId>/<artifactId>/pom.properties - changing the version to be the original one
//     The artifactId to be the original one, and the groupId to be "seal"
//  3. META-INF/maven/<groupId>/<artifactId>/pom.xml - changing the groupId to be "seal"
func CreateSealedNameJar(jarPath, groupId, artifactId, originalVersion string) (string, error) {
	slog.Info("Creating new sealed name jar", "jarPath", jarPath)

	pomPropertiesFilePath := filepath.ToSlash(filepath.Join("META-INF/maven", groupId, artifactId, pomPropertiesFileName))
	pomXMLFilePath := filepath.ToSlash(filepath.Join("META-INF/maven", groupId, artifactId, pomXMLFileName))
	manifestFilePath := filepath.ToSlash(filepath.Join("META-INF", manifestFileName))

	origReader, err := zip.OpenReader(jarPath)
	if err != nil {
		slog.Error("failed reading package", "err", err, "path", jarPath)
		return "", err
	}
	defer origReader.Close()

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
		switch currFileName := filepath.ToSlash(header.Name); currFileName {
		case pomPropertiesFilePath:
			pomProp := PomProperties{GroupId: sealGroupId, ArtifactId: artifactId, Version: originalVersion}
			itemReader = pomProp.GetAsReader()
		case manifestFilePath:
			itemReader = getSilencedManifest(zipItemReader, artifactId)
		case pomXMLFilePath:
			itemReader = getSilencedPomXML(zipItemReader)
		}

		if itemReader == nil {
			return "", fmt.Errorf("failed to create new item reader for %s", zipItem.Name)
		}

		_, err = io.Copy(targetItem, itemReader)
		if err != nil {
			slog.Error("failed copying zip item", "err", err, "path", zipItem.Name)
			return "", err
		}
	}

	return sealedNameFilePath, nil
}

// Change the groupId value in the pom.xml to be `seal` for both the project
// and it's parent if it exists
func getSilencedPomXML(pomXMLReader io.Reader) io.ReadCloser {
	pom := ReadPomXMLFromFile(pomXMLReader)
	if pom == nil {
		return nil
	}

	err := pom.Silence()
	if err != nil {
		slog.Error("failed sealing pom.xml", "err", err)
		return nil
	}

	return pom.GetAsReader()
}

// Change the symbolic name value in the manifest to be `seal.<artifactId>`
// Returns a ReadCloser object created from the new manifest string
func getSilencedManifest(manifestReader io.Reader, artifactId string) io.ReadCloser {
	newManifest := ""
	newSymbolicName := fmt.Sprintf("%s: %s.%s\n", symbolicName, sealGroupId, artifactId)
	newImpVendorId := fmt.Sprintf("%s: %s\n", impVendorId, sealGroupId)
	changed := false
	scanner := bufio.NewScanner(manifestReader)

	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, symbolicName) {
			newManifest += newSymbolicName
			changed = true
		} else if strings.HasPrefix(line, impVendorId) {
			newManifest += newImpVendorId
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

func packageDependencyId(groupId, artifactId, version string) string {
	packageName := fmt.Sprintf("%s:%s", groupId, artifactId)
	return common.DependencyId(mappings.MavenManager, packageName, version)
}

func ShouldSilence(jarPath string, packagesToSilence map[string]bool) (bool, error) {
	slog.Info("Checking if silence needed", "jarPath", jarPath)
	jarReader, err := zip.OpenReader(jarPath)
	if err != nil {
		slog.Warn("failed reading package", "err", err, "path", jarPath)
		return false, nil
	}
	defer jarReader.Close()

	for _, zipItem := range jarReader.File {
		switch currFileName := filepath.Base(zipItem.Name); currFileName {
		case pomXMLFileName:
			zipItemReader, err := zipItem.Open()
			if err != nil {
				slog.Error("failed opening zip item", "err", err, "path", zipItem.Name)
				return false, err
			}
			defer zipItemReader.Close()

			pom := ReadPomXMLFromFile(zipItemReader)
			if pom == nil {
				return false, fmt.Errorf("failed creating new pom xml object for %s", zipItem.Name)
			}

			packageId := pom.GetPackageId()
			if _, ok := packagesToSilence[packageId]; ok {
				return true, nil
			}
		case pomPropertiesFileName:
			zipItemReader, err := zipItem.Open()
			if err != nil {
				slog.Error("failed opening zip item", "err", err, "path", zipItem.Name)
				return false, err
			}
			defer zipItemReader.Close()

			pomProperties := ReadPomPropertiesFromFile(zipItemReader)
			if pomProperties == nil {
				return false, fmt.Errorf("failed creating new pom properties object for %s", zipItem.Name)
			}

			packageId := pomProperties.GetPackageId()

			if _, ok := packagesToSilence[packageId]; ok {
				return true, nil
			}
		default:
			continue
		}
	}
	return false, nil
}

// Silences the provided jar by changing the groupIds in the pom.xml, pom.properties
// if the package id is in the packagesToSilence map (acting as a set for convenience).
// If silenceMainManifest is true, it will also change the groupId in the main manifest file.
// Returns the path to the new jar and a map (set) of the silenced packages.
func getSilencedJar(jarPath string, packagesToSilence map[string]bool, silenceMainManifest bool) (string, map[string]bool, error) {
	manifestFilePath := filepath.ToSlash(filepath.Join("META-INF", manifestFileName))

	origReader, err := zip.OpenReader(jarPath)
	if err != nil {
		slog.Error("failed reading package", "err", err, "path", jarPath)
		return "", nil, err
	}
	defer origReader.Close()

	sealedNameFile, err := os.CreateTemp("./.seal", "tmp_jar")
	if err != nil {
		slog.Error("failed creating sealed file", "err", err, "path", jarPath)
		return "", nil, err
	}
	defer sealedNameFile.Close()

	sealedNameFilePath := sealedNameFile.Name()

	sealedZipWriter := zip.NewWriter(sealedNameFile)
	defer sealedZipWriter.Close()

	silencedPackagesMap := make(map[string]bool, 0)

	for _, zipItem := range origReader.File {
		zipItemReader, err := zipItem.Open()
		if err != nil {
			slog.Error("failed opening zip item", "err", err, "path", zipItem.Name)
			return "", nil, err
		}

		header := zipItem.FileHeader
		targetItem, err := sealedZipWriter.CreateHeader(&header)
		if err != nil {
			slog.Error("failed creating zip item header", "err", err, "path", zipItem.Name)
			return "", nil, err
		}

		itemReader := zipItemReader
		currFilePath := filepath.ToSlash(header.Name)
		currFileName := filepath.Base(header.Name)
		if currFileName == pomXMLFileName {
			pom := ReadPomXMLFromFile(zipItemReader)
			if pom == nil {
				return "", nil, common.NewPrintableError("failed parsing pom.xml file in %s", jarPath)
			}

			if v, ok := packagesToSilence[pom.GetPackageId()]; ok && v {
				silencedPackagesMap[pom.GetPackageId()] = true
				err := pom.Silence()
				if err != nil {
					return "", nil, common.NewPrintableError("failed sealing pom.xml file in %s", jarPath)
				}
			}

			itemReader = pom.GetAsReader()
		} else if currFileName == pomPropertiesFileName {
			pomProperties := ReadPomPropertiesFromFile(zipItemReader)
			if pomProperties == nil {
				return "", nil, common.NewPrintableError("failed parsing pom.properties file in %s", jarPath)
			}

			if v, ok := packagesToSilence[pomProperties.GetPackageId()]; ok && v {
				silencedPackagesMap[pomProperties.GetPackageId()] = true
				pomProperties.GroupId = sealGroupId
			}

			itemReader = pomProperties.GetAsReader()
		} else if currFilePath == manifestFilePath && silenceMainManifest {
			// an huristic to find the artifactId since parsing it from the manifest is not trivial
			itemReader = getSilencedManifest(zipItemReader, filepath.Base(filepath.Dir(filepath.Dir(jarPath))))
		}

		if itemReader == nil {
			return "", nil, fmt.Errorf("failed to create new item reader for %s", zipItem.Name)
		}

		_, err = io.Copy(targetItem, itemReader)
		if err != nil {
			slog.Error("failed copying zip item", "err", err, "path", zipItem.Name)
			return "", nil, err
		}
	}
	return sealedNameFilePath, silencedPackagesMap, nil
}

// Silences the provided jar by changing the groupIds in the pom.xml, pom.properties
// of the provided packages to silence given as a map (acting as a set for convenience).
func SilenceJar(jarPath string, packagesToSilence map[string]bool, silenceMainManifest bool) ([]string, error) {
	slog.Info("Silencing jar", "jarPath", jarPath)

	sealedNameFilePath, silencedPackagesMap, err := getSilencedJar(jarPath, packagesToSilence, silenceMainManifest)
	if err != nil {
		slog.Error("failed silencing jar", "jarPath", jarPath, "err", err)
		return nil, err
	}

	if err := common.ConvertSymLinkToFile(jarPath); err != nil {
		slog.Warn("failed converting symlink to file", "path", jarPath, "err", err)
		return nil, common.NewPrintableError("failed converting symlink to file, path: %s", jarPath)
	}

	if err = os.Rename(sealedNameFilePath, jarPath); err != nil {
		slog.Error("failed renaming sealed file", "err", err, "from", sealedNameFilePath, "to", jarPath)
		return nil, err
	}

	// return as an array instead of set
	silencedPackages := make([]string, 0)
	for k := range silencedPackagesMap {
		silencedPackages = append(silencedPackages, k)
	}

	return silencedPackages, nil
}
