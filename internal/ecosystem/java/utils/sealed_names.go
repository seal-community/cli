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

	pomChanged := false
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
			pomChanged = true
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

	// If the pom.properties file was not found, create a new one
	if !pomChanged {
		pomProp := PomProperties{GroupId: sealGroupId, ArtifactId: artifactId, Version: originalVersion}
		itemReader := pomProp.GetAsReader()
		if itemReader == nil {
			slog.Error("failed to create new item reader for pom.properties")
			return "", fmt.Errorf("failed to create new item reader for %s", pomPropertiesFileName)
		}

		err := common.AddFileToZip(sealedZipWriter, pomPropertiesFilePath, itemReader)
		if err != nil {
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

func ShouldSilence(dependency common.Dependency, packagesToSilence map[string]bool) (bool, error) {
	jarPath := dependency.DiskPath
	slog.Info("Checking if silence needed", "jarPath", jarPath)

	if s, ok := packagesToSilence[dependency.Id()]; ok && s {
		slog.Debug("package in silence list", "package", dependency.Id())
		return true, nil
	}

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
func getSilencedJar(dep common.Dependency, packagesToSilence map[string]bool, silenceMainManifest bool) (string, map[string]bool, error) {
	manifestFilePath := filepath.ToSlash(filepath.Join("META-INF", manifestFileName))

	jarPath := dep.DiskPath
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

	zipFilePathsList := common.GetZipFilePathsSet(origReader.File)

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
				return "", nil, fmt.Errorf("failed parsing pom.xml file in %s", jarPath)
			}

			if v, ok := packagesToSilence[pom.GetPackageId()]; ok && v {
				silencedPackagesMap[pom.GetPackageId()] = true
				err := pom.Silence()
				if err != nil {
					return "", nil, fmt.Errorf("failed sealing pom.xml file in %s", jarPath)
				}

				pomPropertiesFilePath := filepath.ToSlash(filepath.Join(filepath.Dir(currFilePath), pomPropertiesFileName))
				if ok := zipFilePathsList[pomPropertiesFilePath]; !ok {
					// If the pom.properties file does not exist, create a new one
					pomProp := pom.GetPomProperties()

					pomReader := pomProp.GetAsReader()
					if pomReader == nil {
						slog.Error("failed to create new item reader for pom.properties")
						return "", nil, fmt.Errorf("failed to create new item reader for %s", pomPropertiesFileName)
					}

					err := common.AddFileToZip(sealedZipWriter, pomPropertiesFilePath, pomReader)
					if err != nil {
						return "", nil, err
					}

					slog.Info("created new pom.properties file in path", "path", pomPropertiesFilePath)
				}
			}

			itemReader = pom.GetAsReader()
		} else if currFileName == pomPropertiesFileName {
			pomProperties := ReadPomPropertiesFromFile(zipItemReader)
			if pomProperties == nil {
				return "", nil, fmt.Errorf("failed parsing pom.properties file in %s", jarPath)
			}

			if v, ok := packagesToSilence[pomProperties.GetPackageId()]; ok && v {
				slog.Debug("changing groupId in pom.properties")
				silencedPackagesMap[pomProperties.GetPackageId()] = true
				pomProperties.GroupId = sealGroupId
			}

			itemReader = pomProperties.GetAsReader()
		} else if currFilePath == manifestFilePath && silenceMainManifest {
			// an huristic to find the artifactId since parsing it from the manifest is not trivial
			_, artifactId, err := SplitJavaPackageName(dep.NormalizedName)
			if err != nil {
				slog.Error("failed parsing artifactId from package name", "err", err, "package", dep.Name)
				return "", nil, err
			}

			itemReader = getSilencedManifest(zipItemReader, artifactId)
			silencedPackagesMap[dep.Id()] = true
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

	// If the pom.properties file of the jar itself does not exist,
	// and the jar needs to be silenced, create a new silenced pom.properties file
	if silenceMainManifest {
		groupId, artifactId, err := SplitJavaPackageName(dep.NormalizedName)
		if err != nil {
			slog.Error("failed parsing artifactId from package name", "err", err, "package", dep.Name)
			return "", nil, err
		}

		mainPomPropertiesFilePath := filepath.ToSlash(filepath.Join("META-INF/maven", groupId, artifactId, pomPropertiesFileName))
		if _, ok := zipFilePathsList[mainPomPropertiesFilePath]; !ok {
			pomProp := PomProperties{GroupId: sealGroupId, ArtifactId: artifactId, Version: dep.Version}
			itemReader := pomProp.GetAsReader()
			if itemReader == nil {
				slog.Error("failed to create new item reader for pom.properties")
				return "", nil, fmt.Errorf("failed to create new item reader for %s", pomPropertiesFileName)
			}

			err := common.AddFileToZip(sealedZipWriter, mainPomPropertiesFilePath, itemReader)
			if err != nil {
				return "", nil, err
			}

			slog.Info("created new pom.properties file in path", "path", mainPomPropertiesFilePath)
		}
	}

	return sealedNameFilePath, silencedPackagesMap, nil
}

// Silences the provided jar by changing the groupIds in the pom.xml, pom.properties
// of the provided packages to silence given as a map (acting as a set for convenience).
func SilenceJar(dep common.Dependency, packagesToSilence map[string]bool, silenceMainManifest bool) ([]string, error) {
	jarPath := dep.DiskPath
	slog.Info("Silencing jar", "jarPath", jarPath)

	sealedNameFilePath, silencedPackagesMap, err := getSilencedJar(dep, packagesToSilence, silenceMainManifest)
	if err != nil {
		slog.Error("failed silencing jar", "jarPath", jarPath, "err", err)
		return nil, err
	}

	if err := common.ConvertSymLinkToFile(jarPath); err != nil {
		slog.Warn("failed converting symlink to file", "path", jarPath, "err", err)
		return nil, fmt.Errorf("failed converting symlink to file, path: %s", jarPath)
	}

	if err = common.MoveFile(sealedNameFilePath, jarPath); err != nil {
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
