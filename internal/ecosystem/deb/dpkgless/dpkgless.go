package dpkgless

import (
	"archive/tar"
	"bufio"
	"cli/internal/common"
	"compress/gzip"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/blakesmith/ar"
	"github.com/xi2/xz"
)

// the path where the dpkg status files are located (control files and hashes files)
// unlike other distros, in distroless, every package has seperate files in this directory
// https://github.com/GoogleContainerTools/distroless
const dpkgStatusPath = "/var/lib/dpkg/status.d"

const hashesFileSuffix = ".md5sums"

// the status is preset here because it doesn't exist in distroless
const controlFileQueryTemplate = "%s %s %s install ok installed\n"

// This function parses the control file and returns the package name, version and architecture
func parseControlFile(fileContent []byte) (string, string, string, error) {
	lines := strings.Split(string(fileContent), "\n")
	packageName := ""
	version := ""
	arch := ""

	for _, line := range lines {
		if strings.HasPrefix(line, "Package: ") {
			packageName = strings.TrimSpace(strings.TrimPrefix(line, "Package: "))
		} else if strings.HasPrefix(line, "Version: ") {
			version = strings.TrimSpace(strings.TrimPrefix(line, "Version: "))
		} else if strings.HasPrefix(line, "Architecture: ") {
			arch = strings.TrimSpace(strings.TrimPrefix(line, "Architecture: "))
		}
	}

	if packageName == "" || version == "" || arch == "" {
		return "", "", "", fmt.Errorf("failed to parse control file")
	}

	return packageName, version, arch, nil
}

// returns a formatted line for the control file the same way dpkg_manager does (`dpkgQueryFormat`)
func getControlFormattedLine(controlFileContent []byte) (string, error) {
	packageName, version, arch, err := parseControlFile(controlFileContent)
	if err != nil {
		slog.Error("failed to parse control file", "err", err)
		return "", err
	}

	return fmt.Sprintf(controlFileQueryTemplate, packageName, version, arch), nil
}

// This function goes over all the files in the /var/lib/dpkg/status.d directory
// it then parses these files and returns a formatted string with the package name, version, architecture
// and a fake status since in this case there is no status present and we reuse the dpkg code
func ListPackagesFromFilesystem() (string, error) {
	if _, err := os.Stat(dpkgStatusPath); os.IsNotExist(err) {
		slog.Error("dpkg status directory does not exist", "path", dpkgStatusPath)
		return "", err
	}

	result := ""

	// go over all the items in the directory and read them
	items, err := os.ReadDir(dpkgStatusPath)
	if err != nil {
		slog.Error("failed to read directory", "path", dpkgStatusPath, "err", err)
		return "", err
	}

	for _, item := range items {
		itemName := item.Name()
		if item.IsDir() || strings.HasSuffix(itemName, hashesFileSuffix) {
			slog.Debug("skipping path", "path", itemName)
			continue
		}

		// read the file
		filePath := filepath.Join(dpkgStatusPath, itemName)
		fileContent, err := os.ReadFile(filePath)
		if err != nil {
			slog.Error("failed to read file", "path", filePath, "err", err)
			return "", err
		}

		parsedLine, err := getControlFormattedLine(fileContent)
		if err != nil {
			slog.Error("failed to parse control file, skipping", "path", filePath, "err", err)
			continue
		}

		result += parsedLine
	}

	return result, nil
}

// goes over the hashes file and returns a list of the files listed in it with absolute path
func getFilesListFromHashesFile(reader io.Reader) ([]string, error) {
	filesList := make([]string, 0)
	fileScanner := bufio.NewScanner(reader)
	for fileScanner.Scan() {
		line := fileScanner.Text()
		fields := strings.Fields(line)

		if len(fields) < 2 {
			slog.Warn("Got invalid file line", "line", line)
			continue
		}

		filesList = append(filesList, filepath.Join("/", fields[1]))
	}

	return filesList, nil
}

// Uninstalls a deb package by going over the hashes file and moving each file to the backup directory
// the backed up files will be saved in the backup folder including their path
// for example, /var/lib/dpkg/status.d/zlib1g will be saved in
// {backupDir}/var/lib/dpkg/status.d/zlib1g
// this way we can restore the files easily
func UninstallDebPackage(packageName string, backupDirPath string) error {
	hashesFilePath := filepath.Join(dpkgStatusPath, packageName+hashesFileSuffix)
	controlFilePath := filepath.Join(dpkgStatusPath, packageName)

	if _, err := os.Stat(hashesFilePath); os.IsNotExist(err) {
		slog.Error("hashes file does not exist", "path", hashesFilePath)
		return err
	}

	hashesFile, err := os.Open(hashesFilePath)
	if err != nil {
		slog.Error("failed to open hashes file", "path", hashesFilePath, "err", err)
		return err
	}
	defer hashesFile.Close()

	filesList, err := getFilesListFromHashesFile(hashesFile)
	if err != nil {
		slog.Error("failed to get files list from hashes file", "path", hashesFilePath, "err", err)
		return err
	}

	// add the control file to the list of files to backup and remove
	filesList = append(filesList, hashesFilePath, controlFilePath)

	for _, file := range filesList {
		// make sure the backup folder path exists
		backupParentDirPath := filepath.Dir(filepath.Join(backupDirPath, file))
		if err := os.MkdirAll(backupParentDirPath, os.ModePerm); err != nil {
			slog.Error("failed to create backup path", "path", backupDirPath, "err", err)
			return err
		}

		err = common.Move(file, filepath.Join(backupDirPath, file))
		if err != nil {
			slog.Error("failed to move file", "file", file, "err", err)
			return err
		}
	}

	return nil
}

func getPackageNameFromControlFile(controlFileContent []byte) string {
	name, _, _, err := parseControlFile(controlFileContent)
	if err != nil {
		slog.Error("failed to parse control file", "err", err)
		return ""
	}

	return name
}

// reads the control and hashes file, gets the package name from the control file
// and then saves the control file to /var/lib/dpkg/status.d/{packageName}
// and the hashes file to /var/lib/dpkg/status.d/{packageName}.md5sums
func extractControlTar(controlReader *tar.Reader) error {
	controlFileContent := []byte{}
	var controlFileHeader *tar.Header
	hashesFileContent := []byte{}
	var hashesFileHeader *tar.Header

	for {
		header, err := controlReader.Next()
		if err == io.EOF {
			break
		}

		if err != nil {
			slog.Error("failed to read control tar", "err", err)
			return err
		}

		if header.Name == "./control" {
			controlFileContent, err = io.ReadAll(controlReader)
			controlFileHeader = header
		} else if header.Name == "./md5sums" {
			hashesFileContent, err = io.ReadAll(controlReader)
			hashesFileHeader = header
		}

		if err != nil {
			slog.Error("failed to read file", "name", header.Name, "err", err)
			return err
		}
	}

	if len(controlFileContent) == 0 || len(hashesFileContent) == 0 {
		return fmt.Errorf("failed to find control or hashes file in control tar")
	}

	// get the package name from the control file
	packageName := getPackageNameFromControlFile(controlFileContent)
	if packageName == "" {
		slog.Error("failed to get package name from control file")
		return fmt.Errorf("failed to get package name from control file")
	}

	// write the control file to the filesystem
	controlFilePath := filepath.Join(dpkgStatusPath, packageName)
	if err := os.WriteFile(controlFilePath, controlFileContent, controlFileHeader.FileInfo().Mode()); err != nil {
		slog.Error("failed to write control file", "path", controlFilePath, "err", err)
		return err
	}

	// write the hashes file to the filesystem
	hashesFilePath := filepath.Join(dpkgStatusPath, packageName+hashesFileSuffix)
	if err := os.WriteFile(hashesFilePath, hashesFileContent, hashesFileHeader.FileInfo().Mode()); err != nil {
		slog.Error("failed to write hashes file", "path", hashesFilePath, "err", err)
		return err
	}

	return nil
}

// extracts the data tar and writes the files to the filesystem
// this tar contains the files that need to be installed
func extractDataTar(dataReader *tar.Reader) error {
	for {
		header, err := dataReader.Next()
		if err == io.EOF {
			break
		}

		if err != nil {
			slog.Error("failed to read data tar", "err", err)
			return err
		}

		if header.Name == "./" {
			continue
		}

		targetPath := filepath.Join("/", header.Name)

		if header.FileInfo().Mode()&os.ModeSymlink == os.ModeSymlink { // handle symlinks
			slog.Debug("creating symlink", "name", targetPath, "linkname", header.Linkname)

			if _, err := os.Lstat(targetPath); err == nil {
				slog.Debug("symlink already exists, removing", "path", targetPath)
				os.Remove(targetPath)
			}

			if err := os.Symlink(header.Linkname, targetPath); err != nil {
				slog.Error("failed to create symlink", "path", header.Name, "err", err)
				return err
			}
		} else if header.FileInfo().IsDir() {
			// skip, we'll create directories when we create files
			continue
		} else { // handle regular files
			// make sure the folder path exists
			if err := os.MkdirAll(filepath.Dir(targetPath), os.ModePerm); err != nil {
				slog.Error("failed to create directory", "path", targetPath, "err", err)
				return err
			}

			// create the file
			file, err := os.Create(targetPath)
			if err != nil {
				slog.Error("failed to create file", "path", targetPath, "err", err)
				return err
			}
			defer file.Close()

			// copy the data to it
			if _, err := io.Copy(file, dataReader); err != nil {
				slog.Error("failed to copy file", "path", targetPath, "err", err)
				return err
			}
		}
	}

	return nil
}

// gets the file reader and the file name, and returns a tar reader for the file
func getTarReader(inReader io.Reader, fileName string) (*tar.Reader, error) {
	reader := inReader
	var err error
	if strings.HasSuffix(fileName, ".tar.xz") {
		reader, err = xz.NewReader(inReader, 0)
		if err != nil {
			slog.Error("failed to create xz reader", "err", err)
			return nil, err
		}
	} else if strings.HasSuffix(fileName, ".tar.gz") {
		reader, err = gzip.NewReader(inReader)
		if err != nil {
			slog.Error("failed to create gzip reader", "err", err)
			return nil, err
		}
	}

	return tar.NewReader(reader), nil
}

// reads the deb package and extracts the control and data tar files to mimic a dpkg installation
// the control tar contains the control file and the hashes file
// * the control file contains data about the package like it's name, version and architecture
// * the hashes file contains the list of files that are part of the package and their hashes
// the data tar contains the files that need to be installed (their hashes are in the hashes file)
func InstallDebPackage(debFileReader io.Reader) error {
	debReader := ar.NewReader(debFileReader)

	handledControlFile := false
	handledDataFile := false

	for {
		header, err := debReader.Next()
		if err == io.EOF {
			break
		}

		if err != nil {
			slog.Error("failed to read deb package", "err", err)
			return err
		}

		fileName := filepath.Base(header.Name)
		if strings.HasPrefix(fileName, "control.tar") {
			reader, err := getTarReader(debReader, fileName)
			if err != nil {
				slog.Error("failed to create tar reader", "err", err)
				return err
			}

			err = extractControlTar(reader)
			if err != nil {
				slog.Error("failed to extract control file", "err", err)
				return err
			}

			handledControlFile = true
		} else if strings.HasPrefix(fileName, "data.tar") {
			reader, err := getTarReader(debReader, fileName)
			if err != nil {
				slog.Error("failed to create tar reader", "err", err)
				return err
			}

			err = extractDataTar(reader)
			if err != nil {
				slog.Error("failed to extract data file", "err", err)
				return err
			}

			handledDataFile = true
		}

	}

	if !(handledControlFile && handledDataFile) {
		slog.Error("failed to find control or data file in deb package", "control", handledControlFile, "data", handledDataFile)
		return fmt.Errorf("failed to find control or data file in deb package")
	}

	return nil
}
