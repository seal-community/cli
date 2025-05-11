package gradle

import (
	"cli/internal/actions"
	"cli/internal/api"
	"cli/internal/common"
	"cli/internal/config"
	"cli/internal/ecosystem/java/utils"
	"cli/internal/ecosystem/mappings"
	"cli/internal/ecosystem/shared"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const privateHomeDirName = ".seal-gradle"
const gradleBuildFile = "gradle.build" // not required
const gradleLockFileName = "gradle.lock"
const gradleManagerName = "gradle"
const minimumGradleVersion = "7.6.4"

type GradlePackageManager struct {
	Config        *config.Config
	targetDir     string
	targetFile    string
	gradleVersion string

	workdir  string            // e.g. ./.seal/.gradle
	rollback map[string]string // original version path -> tmp-location

	// when we start, there's a home dir for gradle - by default in ~/.gradle;
	// if we're running after a previous fix or user itself modified it to be somewhere else
	originalHomeDir string

	// after we fix we create a new custom home dir and cache, this is the new home dir
	// this is stored outside of workdir for persistency
	privateHomeDir string

	runner     *gradleRunner
	useGradlew bool
}

func gradlewExists(path string) bool {
	p := filepath.Join(path, projectGradlewExe)
	exists, err := common.PathExists(p)
	if err != nil {
		slog.Error("failed checking gradlew exists", "err", err)
	}

	return exists
}

func NewGradleManager(config *config.Config, targetFile string, targetDir string) *GradlePackageManager {
	useGradlew := true                   // currently only support gradle wrapper
	newHomeDir := config.Gradle.HomePath // to allow specifying home dir outside the project's dir - must be absolute

	if newHomeDir == "" {
		newHomeDir = filepath.Join(targetDir, privateHomeDirName)
	}

	// we expect the target to be the root of the project, and not a subproject inside. therefore gradle wrapper is expected
	if !gradlewExists(targetDir) {
		slog.Error("gradlew could not be found", "file", targetFile, "dir", targetFile)
		return nil
	}

	runner := NewGradleRunner(targetDir, useGradlew)
	originalHomeDir := getHomeDir(runner)
	slog.Info("original home dir", "original-home", originalHomeDir, "new-home", newHomeDir)

	lockFile, _ := loadLockFile(targetDir) // ignoring error for best effort
	if lockFile != "" {
		// this would only denote lock file existing for the root project, there could be lockfiles for sub project in their respective folders
		slog.Info("found lock file", "path", targetDir)
		slog.Debug("lockfile data", "data", lockFile)
	}

	m := &GradlePackageManager{Config: config,
		targetFile:      targetFile,
		targetDir:       targetDir,
		privateHomeDir:  newHomeDir,
		originalHomeDir: originalHomeDir,
		runner:          runner,
		useGradlew:      useGradlew,
		rollback:        make(map[string]string, 100),
	}

	return m
}

// lockfiles are present in `gradle.lockfile“ in the projects dir (can also be in root)
// see https://docs.gradle.org/current/userguide/dependency_locking.html
// can return the content if exists, error will be nil otherwise
func loadLockFile(projDir string) (string, error) {
	lockPath := filepath.Join(projDir, gradleLockFileName)
	if exists, err := common.PathExists(lockPath); !exists || err != nil {
		slog.Info("no lock file or failed reading it", "exits", exists, "err", err, "path", lockPath)
		return "", err
	}

	data, err := os.ReadFile(lockPath)
	if err != nil {
		slog.Error("no lock file or failed reading it", "path", lockPath, "err", err)
		return "", err
	}

	slog.Debug("loaded lock file", "path", lockPath)

	return string(data), nil
}

// used when initializing the mangaer, as well as later
func getHomeDir(runner *gradleRunner) string {
	output := runner.Status()
	if output == "" {
		slog.Error("failed getting status command from gradle")
		return ""
	}

	homeDir := parseHomeDir(output)
	if homeDir == "" {
		slog.Error("failed parsing status output from gradle")
		return ""
	}

	return homeDir
}

func (m *GradlePackageManager) Name() string {
	return gradleManagerName
}

func (m *GradlePackageManager) Class() actions.ManagerClass {
	return actions.ManifestManager
}

func (m *GradlePackageManager) GetVersion() string {
	if m.gradleVersion != "" {
		return m.gradleVersion
	}

	versionStdout := m.runner.Version()
	if versionStdout == "" {
		slog.Error("failed getting gradle version", "dir", m.targetDir)
		return ""
	}

	version := parseVersionOutput(versionStdout)
	if version == "" {
		slog.Error("failed parsing gradle version", "dir", m.targetDir, "stdout", versionStdout)
		return ""
	}

	m.gradleVersion = version
	return m.gradleVersion
}

func (m *GradlePackageManager) IsVersionSupported(version string) bool {
	supported, _ := common.VersionAtLeast(version, minimumGradleVersion)

	if !supported {
		slog.Error("unsupported gradle version", "found", version, "minimal", minimumGradleVersion)
		return false
	}

	return true
}

func IsGradleIndicatorFile(path string) bool {
	return strings.HasSuffix(path, projectGradlewExe) || strings.HasSuffix(path, gradleBuildFile)
}

// works on dir
func GetGradleIndicatorFile(path string) (string, error) {
	packageFile := filepath.Join(path, projectGradlewExe)
	exists, err := common.PathExists(packageFile)
	if err != nil {
		slog.Error("failed checking file exists", "file", projectGradlewExe, "err", err)
		return "", err
	}

	if exists {
		slog.Info("found gradle indicator file", "file", projectGradlewExe, "path", packageFile)
		return packageFile, nil
	}

	slog.Debug("no gradle indicator file found")
	return "", nil
}

func (m *GradlePackageManager) getPackages() []utils.JavaPackageInfo {
	allPackages := make([]utils.JavaPackageInfo, 0, 100)

	projectsStdout := m.runner.Projects()
	projects := parseProjectsOutput(projectsStdout)
	slog.Info("got gradle projects", "count", len(projects), "projects", projects)

	scope := CompileClasspath
	if !m.Config.Gradle.ProdOnlyDeps {
		slog.Warn("only supporting prod dependencies for gradle")
	}

	for _, p := range projects {
		depsOutput := m.runner.Dependencies(p, scope) // using only prod deps for now, others not supported
		if depsOutput == "" {
			slog.Info("no dependencies can be found for project", "name", p)
			continue
		}

		projPackages := parsePackages(depsOutput, scope)
		slog.Debug("found packages for project", "count", len(projPackages), "projName", p)
		allPackages = append(allPackages, projPackages...)
	}

	return allPackages
}

func getSealMetadata(artifactPath string) (*shared.SealPackageMetadata, error) {
	folder := filepath.Dir(artifactPath)
	metadataPath := filepath.Join(folder, shared.SealMetadataFileName)
	return shared.LoadPackageSealMetadata(metadataPath)
}

// there could be several locations of the same jar file
// structure (for 8.14) is:
//
//	{homedir}
//		caches
//			modules-2
//				files-2.1
//					{org-id}
//						{artifact-id}
//							{version}
//								{sha1sum}
//									{artifact-file-name}
func findGradleDepednecyLocations(homeDirPath string, pi utils.JavaPackageInfo) ([]string, error) {
	fileName := utils.GetPackageFileName(pi.ArtifactName, pi.Version)
	ptrn := filepath.Join(homeDirPath, "caches",
		"modules-2", "files-2.1", // these are versioned as well, modules-2 / files-2.1 are used since gradle 6.1 (see https://docs.gradle.org/current/userguide/dependency_caching.html#sec:cache-copy)
		pi.OrgName, pi.ArtifactName, pi.Version, "*", fileName)

	slog.Debug("looking for jar locations in path", "ptrb", ptrn, "package", pi)
	matches, err := filepath.Glob(ptrn)
	if err != nil {
		slog.Error("bad glob pattern for gradle dep location", "err", err, "home", homeDirPath, "package", pi)
		return nil, err
	}

	slog.Debug("found locations for package info", "matches", matches)
	return matches, nil
}

func (m *GradlePackageManager) NormalizePackageName(name string) string {
	return utils.NormalizePackageName(name)
}

func (m *GradlePackageManager) ListDependencies(be api.Backend) (common.DependencyMap, error) {
	if runtime.GOOS == "windows" {
		slog.Error("unsupported os for gradle")
		// using this function to return a printable error until support is implemented
		// doing so in the 'constructor' won't be shown to user
		return nil, common.NewPrintableError("gradle on Windows is not yet supported")
	}

	packages := m.getPackages()
	depMap := make(common.DependencyMap)
	manager := mappings.MavenManager
	locsFound := 0
	for _, pi := range packages {
		packageName := utils.FormatJavaPackageName(pi.OrgName, pi.ArtifactName)
		id := common.DependencyId(manager, packageName, pi.Version)

		if depMap[id] != nil {
			slog.Debug("skipping dup depdency", "id", id, "package", pi)
			continue
		}

		deps := make([]*common.Dependency, 0, 1)
		locs, err := findGradleDepednecyLocations(m.originalHomeDir, pi)
		if err != nil {
			// should not happen unless glob patter modified / OS not supported
			return nil, err
		}

		if len(locs) == 0 {
			// this could happen if `build` / `build --dry-run` was not run prior to running the CLI (8.14 works with dry run)
			slog.Debug("no jar files were found for package", "homedir", m.originalHomeDir, "package", pi)
			continue
		}

		locsFound += len(locs)

		for _, l := range locs {
			version := pi.Version
			metadata, err := getSealMetadata(l)
			if err != nil {
				slog.Error("failed loading seal metadata", "path", l, "package", pi)
				return nil, err
			}

			if metadata != nil {
				version = metadata.SealedVersion
				slog.Info("found sealed package", "version", version)
			}

			newDep := &common.Dependency{
				Name:           packageName,
				NormalizedName: m.NormalizePackageName(packageName),
				Version:        version,
				PackageManager: manager,
				DiskPath:       l,
			}

			slog.Debug("adding gradle dependnecy", "dep", newDep)
			// IMPORTANT: unsupported shaded dependency discovery

			deps = append(deps, newDep)
		}

		depMap[id] = deps
	}

	if locsFound == 0 && len(packages) > 0 {
		// means we should find at least 1 in disk, but non were found
		// happens if user doesn't download its dependencies before running us
		return nil, common.NewPrintableError("could not find artifacts in cache; make sure to install your dependencies")
	}
	return depMap, nil
}

func (m *GradlePackageManager) GetProjectName() string {
	// since there could be multiple projects let the caller decide fallback
	return ""
}

func (m *GradlePackageManager) GetFixer(workdir string) shared.DependencyFixer {
	m.workdir = filepath.Join(workdir, ".gradle")
	return m
}

func (m *GradlePackageManager) GetEcosystem() string {
	return mappings.JavaEcosystem
}

func (m *GradlePackageManager) GetScanTargets() []string {
	return []string{m.targetFile}
}

func (m *GradlePackageManager) DownloadPackage(server api.ArtifactServer, descriptor shared.DependencyDescriptor) ([]byte, string, error) {
	return utils.DownloadMavenPackage(server, descriptor.AvailableFix.Library.Name, descriptor.AvailableFix.Version)
}

func (m *GradlePackageManager) HandleFixes(fixes []shared.DependencyDescriptor) error {
	for _, fix := range fixes {
		for _, loc := range fix.FixedLocations {
			metadata := shared.SealPackageMetadata{SealedVersion: fix.AvailableFix.Version}
			packagePath := strings.Replace(loc, m.originalHomeDir, m.privateHomeDir, 1)
			packageDirPath := filepath.Dir(packagePath)
			metadataFilePath := filepath.Join(packageDirPath, shared.SealMetadataFileName)

			err := shared.SavePackageMetadata(metadata, metadataFilePath)
			if err != nil {
				return err
			}
		}
	}

	err := m.patchGradleWrapper()
	if err != nil {
		slog.Error("failed patching gradlew", "err", err)
		return err
	}

	return nil
}

func (m *GradlePackageManager) SilencePackages(silenceArray []api.SilenceRule, allDependencies common.DependencyMap) (map[string][]string, error) {
	slog.Warn("Silencing packages is not support for gradle")
	return nil, nil
}

func (m *GradlePackageManager) ConsolidateVulnerabilities(vulnerablePackages *[]api.PackageVersion, allDependencies common.DependencyMap) (*[]api.PackageVersion, error) {
	return utils.ConsolidateVulnerabilities(vulnerablePackages, allDependencies)
}
