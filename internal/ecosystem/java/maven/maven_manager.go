package maven

import (
	"cli/internal/api"
	"cli/internal/common"
	"cli/internal/config"
	"cli/internal/ecosystem/java/utils"
	"cli/internal/ecosystem/mappings"
	"cli/internal/ecosystem/shared"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

const sealCacheName = ".seal-m2"
const mavenIndicator = "pom.xml"
const mavenManagerName = "maven"
const mavenConfigName = ".mvn/maven.config"
const m2CacheFlag = "-Dmaven.repo.local"

type MavenPackageManager struct {
	Config         *config.Config
	workDir        string
	javaTargetFile string
	mavenVersion   string
	cacheDir       string
}

func (m *MavenPackageManager) Name() string {
	return mavenManagerName
}

func (m *MavenPackageManager) GetVersion(targetDir string) string {
	if m.mavenVersion != "" {
		return m.mavenVersion
	}
	version := utils.GetVersion(targetDir)
	if version == "" {
		slog.Error("failed getting maven version")
		return ""
	}
	m.mavenVersion = version
	return m.mavenVersion
}

func IsMavenIndicatorFile(path string) bool {
	return strings.HasSuffix(path, mavenIndicator)
}

func GetJavaIndicatorFile(path string) (string, error) {
	packageFile := filepath.Join(path, mavenIndicator)
	exists, err := common.PathExists(packageFile)
	if err != nil {
		slog.Error("failed checking file exists", "file", mavenIndicator, "err", err)
		return "", err
	}

	if exists {
		slog.Info("found maven indicator file", "file", mavenIndicator, "path", packageFile)
		return mavenIndicator, nil
	}

	slog.Debug("no maven indicator file found")
	return "", nil
}

func NewMavenManager(config *config.Config, javaFile string, targetDir string) *MavenPackageManager {
	cacheDir := config.Maven.CachePath
	if cacheDir == "" {
		slog.Debug("maven seal cache path is not set, setting to default value")
		cacheDir = filepath.Join(targetDir, sealCacheName)
	}

	m := &MavenPackageManager{Config: config, javaTargetFile: javaFile, workDir: targetDir, cacheDir: cacheDir}
	return m
}

func (m *MavenPackageManager) ListDependencies(targetDir string) (*common.ProcessResult, bool) {
	return listPackages(targetDir)
}

func (m *MavenPackageManager) GetParser() shared.ResultParser {
	return &dependencyParser{config: m.Config, cacheDir: m.cacheDir}
}

func (m *MavenPackageManager) GetProjectName(dir string) string {
	args := []string{"help:evaluate", "-Dexpression=project.name", "-q", "-DforceStdout"}
	listResult, err := common.RunCmdWithArgs(dir, utils.MavenExeName, args...)
	if err != nil || listResult.Code != 0 {
		slog.Warn("failed to get maven project name")
		return ""
	}
	slog.Info("maven project name: ", "name", listResult.Stdout)
	return listResult.Stdout
}

func (m *MavenPackageManager) GetFixer(projectDir string, workdir string) shared.DependencyFixer {
	return utils.NewFixer(projectDir, filepath.Join(workdir, ".m2"), m.cacheDir)
}

func (m *MavenPackageManager) GetEcosystem() string {
	return mappings.JavaEcosystem
}

func (m *MavenPackageManager) GetScanTargets() []string {
	return []string{m.javaTargetFile}
}

func (m *MavenPackageManager) DownloadPackage(server api.Server, descriptor shared.DependnecyDescriptor) ([]byte, error) {
	return utils.DownloadMavenPackage(server, descriptor.AvailableFix.Library.Name, descriptor.AvailableFix.Version)
}

// HandleFixes will create a metadata file for each package in the fixes map to indicate it was fixed
func (m *MavenPackageManager) HandleFixes(projectDir string, fixes []shared.DependnecyDescriptor) error {
	for _, fix := range fixes {
		metadata := shared.SealPackageMetadata{SealedVersion: fix.AvailableFix.Version}
		packageDirPath := utils.GetJavaPackagePath(m.cacheDir, fix.VulnerablePackage.Library.Name, fix.VulnerablePackage.Version)
		metadataFilePath := filepath.Join(packageDirPath, shared.SealMetadataFileName)

		slog.Info("creating metadata file", "path", metadataFilePath)
		w, err := common.CreateFile(metadataFilePath)
		if err != nil {
			return err
		}

		err = shared.SavePackageMetadata(metadata, w)
		if err != nil {
			slog.Error("failed saving metadata file", "path", metadataFilePath)
			return fmt.Errorf("failed saving %s", metadataFilePath)
		}
	}

	currCacheDir := utils.GetCacheDir(projectDir)
	if currCacheDir == "" {
		slog.Warn("failed getting maven cache dir")
		return common.NewPrintableError("failed getting maven cache dir")
	}
	
	if currCacheDir == m.cacheDir {
		slog.Debug("maven cache dir is already set")
		return nil
	}

	slog.Info("setting maven cache dir", "dir", m.cacheDir)
	err := setCacheDir(projectDir, currCacheDir)
	if err != nil {
		return common.NewPrintableError("failed setting maven cache dir")
	}

	return nil
}

// runs maven's dependency:tree command and returns the output
// using the -DoutputType=dot flag to get the output in dot format
// using the -DoutputFile flag to write the output to a temp file
// then read the file and output the result
func listPackages(targetDir string) (*common.ProcessResult, bool) {
	tmpfile, err := os.CreateTemp("", "maven-dependency-tree-output")
	if err != nil {
		slog.Error("failed creating temp file", "err", err)
		return nil, false
	}
	defer os.Remove(tmpfile.Name())

	args := []string{"dependency:tree", "-DoutputType=dot", "-DoutputFile=" + tmpfile.Name(), "-DappendOutput=true"}
	listResult, err := common.RunCmdWithArgs(targetDir, utils.MavenExeName, args...)
	if err != nil {
		return nil, false
	}

	if listResult.Code != 0 {
		slog.Error("failed running maven dependency:tree", "err", listResult.Stderr)
		return listResult, false
	}

	data, err := io.ReadAll(tmpfile)
	if err != nil {
		slog.Error("failed reading temp file", "err", err)
		return nil, false
	}

	result := &common.ProcessResult{
		Stdout: string(data),
		Stderr: "",
		Code:   0,
	}
	return result, true
}

// setCacheDir sets the maven cache directory in the maven.config file using the -Dmaven.repo.local flag
// The latest flag use is the deciding one, so appending will always work
// The maven.config file is created if it does not exist
func setCacheDir(projectDir string, newCacheDir string) error {
	mvnConfigDir := filepath.Join(projectDir, ".mvn")
	exists, err := common.DirExists(mvnConfigDir)
	if err != nil {
		return err
	}
	if !exists {
		if err := os.Mkdir(mvnConfigDir, 0755); err != nil {
			slog.Error("mkdir failed", "err", err)
			return common.NewPrintableError("failed creating new cache directory %s", mvnConfigDir)
		}
	}

	mvnConfigFile := filepath.Join(projectDir, mavenConfigName)
	file, err := os.OpenFile(mvnConfigFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		slog.Error("failed opening maven.config file", "err", err)
		return common.NewPrintableError("failed opening %s file", mavenConfigName)
	}
	defer file.Close()

	_, err = file.WriteString(fmt.Sprintf("\n%s=%s", m2CacheFlag, newCacheDir))
	if err != nil {
		slog.Error("failed writing to maven.config file", "err", err)
		return common.NewPrintableError("failed writing to %s file", mavenConfigName)
	}

	return nil
}
