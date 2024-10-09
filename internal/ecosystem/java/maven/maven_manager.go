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

	version_parse "github.com/hashicorp/go-version"
)

const sealCacheName = ".seal-m2"
const mavenIndicator = "pom.xml"
const mavenManagerName = "maven"
const mavenConfigName = ".mvn/maven.config"
const m2CacheFlag = "-Dmaven.repo.local"
const minimumMavenVersion = "3.3.1"

type MavenPackageManager struct {
	Config         *config.Config
	targetDir      string
	javaTargetFile string
	mavenVersion   string
	cacheDir       string
}

func NewMavenManager(config *config.Config, javaFile string, targetDir string) *MavenPackageManager {
	cacheDir := config.Maven.CachePath
	if cacheDir == "" {
		slog.Debug("maven seal cache path is not set, setting to default value")
		cacheDir = filepath.Join(targetDir, sealCacheName)
	}

	m := &MavenPackageManager{Config: config, javaTargetFile: javaFile, targetDir: targetDir, cacheDir: cacheDir}
	return m
}

func (m *MavenPackageManager) Name() string {
	return mavenManagerName
}

func (m *MavenPackageManager) GetVersion() string {
	if m.mavenVersion != "" {
		return m.mavenVersion
	}
	version := utils.GetVersion(m.targetDir)
	if version == "" {
		slog.Error("failed getting maven version")
		return ""
	}
	m.mavenVersion = version
	return m.mavenVersion
}

func (m *MavenPackageManager) IsVersionSupported(version string) bool {
	if version == "" {
		slog.Error("maven version is empty")
		return false
	}

	v, err1 := version_parse.NewVersion(version)
	sv, err2 := version_parse.NewVersion(minimumMavenVersion)

	if err1 != nil || err2 != nil {
		slog.Warn("failed parsing maven version", "version", version)
		return false
	}

	if v.LessThan(sv) {
		slog.Warn("maven version is not supported", "version", version)
		return false
	}

	return true
}

func IsMavenIndicatorFile(path string) bool {
	return strings.HasSuffix(path, mavenIndicator)
}

// works on dir
func GetJavaIndicatorFile(path string) (string, error) {
	packageFile := filepath.Join(path, mavenIndicator)
	exists, err := common.PathExists(packageFile)
	if err != nil {
		slog.Error("failed checking file exists", "file", mavenIndicator, "err", err)
		return "", err
	}

	if exists {
		slog.Info("found maven indicator file", "file", mavenIndicator, "path", packageFile)
		return packageFile, nil
	}

	slog.Debug("no maven indicator file found")
	return "", nil
}

func (m *MavenPackageManager) ListDependencies() (common.DependencyMap, error) {
	result, ok := listPackages(m.targetDir)
	if !ok {
		slog.Error("failed running package manager in the current dir", "name", m.Name())
		return nil, shared.ManagerProcessFailed
	}

	parser := &dependencyParser{config: m.Config, cacheDir: m.cacheDir, normalizer: m}
	dependencyMap, err := parser.Parse(result.Stdout, m.targetDir)
	if err != nil {
		slog.Error("failed parsing package manager output", "err", err, "code", result.Code, "stderr", result.Stderr)
		slog.Debug("manager output", "stdout", result.Stdout) // useful for debugging its output
		return nil, shared.FailedParsingManagerOutput
	}

	return dependencyMap, nil
}

func (m *MavenPackageManager) GetProjectName() string {
	args := []string{"help:evaluate", "-Dexpression=project.name", "-q", "-DforceStdout"}
	listResult, err := common.RunCmdWithArgs(m.targetDir, utils.MavenExeName, args...)
	if err != nil || listResult.Code != 0 {
		slog.Warn("failed to get maven project name", "err", err, "exitcode", listResult.Code)
		return ""
	}

	slog.Info("maven project name: ", "name", listResult.Stdout)
	return listResult.Stdout
}

func (m *MavenPackageManager) GetFixer(workdir string) shared.DependencyFixer {
	return utils.NewFixer(m.targetDir, filepath.Join(workdir, ".m2"), m.cacheDir)
}

func (m *MavenPackageManager) GetEcosystem() string {
	return mappings.JavaEcosystem
}

func (m *MavenPackageManager) GetScanTargets() []string {
	return []string{m.javaTargetFile}
}

func (m *MavenPackageManager) DownloadPackage(server api.ArtifactServer, descriptor shared.DependnecyDescriptor) ([]byte, error) {
	return utils.DownloadMavenPackage(server, descriptor.AvailableFix.Library.Name, descriptor.AvailableFix.Version)
}

// Overwrites the jar file in diskPath to a new jar containing the sealed names
func changeToSealedName(packageName, packageOriginalVersion, diskPath string) error {
	groupId, artifactId, err := utils.SplitJavaPackageName(packageName)
	if err != nil {
		slog.Error("failed getting package name for dependency", "err", err, "path", packageName)
		return common.NewPrintableError("failed getting package name for dependency %s", packageName)
	}

	newJarPath, err := utils.CreateSealedNameJar(diskPath, groupId, artifactId, packageOriginalVersion)
	if err != nil {
		slog.Error("failed changing to sealed name", "err", err, "path", diskPath)
		return common.NewPrintableError("failed changing package %s to sealed name", packageName)
	}

	if err = os.Rename(newJarPath, diskPath); err != nil {
		slog.Error("failed renaming sealed file", "err", err, "from", newJarPath, "to", diskPath)
		return err
	}

	return nil
}

// HandleFixes will create a metadata file for each package in the fixes map to indicate it was fixed
func (m *MavenPackageManager) HandleFixes(fixes []shared.DependnecyDescriptor) error {
	for _, fix := range fixes {
		metadata := shared.SealPackageMetadata{SealedVersion: fix.AvailableFix.Version}
		packageDirPath := utils.GetJavaPackagePath(m.cacheDir, fix.VulnerablePackage.Library.Name, fix.AvailableFix.OriginVersionString)
		metadataFilePath := filepath.Join(packageDirPath, shared.SealMetadataFileName)

		err := shared.SavePackageMetadata(metadata, metadataFilePath)
		if err != nil {
			return err
		}

		if m.Config.UseSealedNames {
			for _, diskPath := range fix.FixedLocations {
				slog.Info("changing package to sealed name", "id", fix.VulnerablePackage.Library.Name, "path", diskPath)
				if err := changeToSealedName(fix.VulnerablePackage.Library.Name, fix.AvailableFix.OriginVersionString, diskPath); err != nil {
					return common.FallbackPrintableMsg(err, "failed changing %s to sealed name", fix.VulnerablePackage.Library.Name)
				}
			}
		}
	}

	currCacheDir := utils.GetCacheDir(m.targetDir)
	if currCacheDir == "" {
		slog.Warn("failed getting maven cache dir")
		return common.NewPrintableError("failed getting maven cache dir")
	}

	if currCacheDir == m.cacheDir {
		slog.Debug("maven cache dir is already set")
		return nil
	}

	slog.Info("setting maven cache dir", "dir", m.cacheDir)
	err := setCacheDir(m.targetDir, m.cacheDir)
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
		// maven outputs the error to stdout
		slog.Error("failed running maven dependency:tree", "err", listResult.Stderr, "out", listResult.Stdout)
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

	fi, err := file.Stat()
	if err != nil {
		slog.Error("failed getting file info", "err", err)
		return common.NewPrintableError("failed getting file info for %s", mavenConfigName)
	}

	// maven version 3.3.1 doesn't support an empty line in the start of the file
	if fi.Size() != 0 {
		// ignoring error because the same operation is checked in the next block
		_, _ = file.WriteString("\n")
	}

	_, err = file.WriteString(fmt.Sprintf("%s=%s", m2CacheFlag, newCacheDir))
	if err != nil {
		slog.Error("failed writing to maven.config file", "err", err)
		return common.NewPrintableError("failed writing to %s file", mavenConfigName)
	}

	return nil
}

// all maven packages are supposed to be lower case according to
// https://docs.oracle.com/javase/tutorial/java/package/namingpkgs.html
// However, there are some packages that doesn't follow this rule and the current behavior is case sensitive
func (m *MavenPackageManager) NormalizePackageName(name string) string {
	return name
}

func parseSilenceInput(silenceEntry string) (string, string) {
	silenceParts := strings.Split(silenceEntry, "@")
	if len(silenceParts) != 2 {
		return "", ""
	}

	return silenceParts[0], silenceParts[1]
}

func (m *MavenPackageManager) SilencePackages(silenceArray []string, allDependencies common.DependencyMap) ([]common.Dependency, error) {
	// make sure the seal-m2 folder exists and initialized
	df := m.GetFixer(m.targetDir)
	if err := df.Prepare(); err != nil {
		slog.Error("failed preparing folders", err)
		return nil, err
	}

	silenced := make([]common.Dependency, 0, 1)
	for _, silenceEntry := range silenceArray {
		packageName, packageVersion := parseSilenceInput(silenceEntry)
		if packageName == "" || packageVersion == "" {
			slog.Warn("failed parsing silence entry", "entry", silenceEntry)
			return nil, common.NewPrintableError("failed parsing silence entry %s", silenceEntry)
		}

		entryId := common.DependencyId(mappings.MavenManager, m.NormalizePackageName(packageName), packageVersion)

		if _, ok := allDependencies[entryId]; !ok {
			slog.Warn("failed silencing package, package not found", "entry", silenceEntry)
			continue
		}

		for _, dep := range allDependencies[entryId] {
			jarPath := dep.DiskPath
			if err := common.ConvertSymLinkToFile(jarPath); err != nil {
				slog.Warn("failed converting symlink to file", "path", jarPath, "err", err)
				return nil, common.NewPrintableError("failed converting symlink to file, path: %s", jarPath)
			}

			if err := changeToSealedName(dep.Name, dep.Version, jarPath); err != nil {
				slog.Warn("failed changing to sealed name", "path", jarPath, "err", err)
				return nil, common.NewPrintableError("failed changing %s to sealed name", silenceEntry)
			}
			silenced = append(silenced, *dep)
		}
	}
	return silenced, nil
}
