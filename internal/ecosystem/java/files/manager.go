package files

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
)

const javaFilesManagerName = "java_files"

type JavaFilesPackageManager struct {
	Config    *config.Config
	targetDir string
}

func (m *JavaFilesPackageManager) Name() string {
	return javaFilesManagerName
}

func (m *JavaFilesPackageManager) Class() actions.ManagerClass {
	return actions.FilesManager
}

func (m *JavaFilesPackageManager) GetVersion() string {
	// no version for java files
	// using underscore as a placeholder since can't be empty
	return "_"
}

func (m *JavaFilesPackageManager) IsVersionSupported(version string) bool {
	// since there's no version, all versions are supported
	return true
}

func supportedJavaFile(path string) bool {
	ext := filepath.Ext(path)

	switch ext {
	case ".jar":
		return true
	default:
		return false
	}
}

// go over all files in the target directory and create dependencies for each jar file
// does this in a best-effort manner, skipping jars that can't be parsed
// Handles symlinks too.
func (m *JavaFilesPackageManager) ListDependencies() (common.DependencyMap, error) {
	dependencies := make(common.DependencyMap, 0)
	err := filepath.WalkDir(m.targetDir, func(path string, info os.DirEntry, err error) error {
		if err != nil {
			slog.Warn("failed walking path", "err", err, "path", path)
			return filepath.SkipDir
		}

		if info.IsDir() {
			return nil
		}

		// for symbolic links, we need to evaluate the real path
		if info.Type()&os.ModeSymlink != 0 {
			slog.Debug("evaluating symlink", "path", path)
			realPath, err := filepath.EvalSymlinks(path)
			if err != nil {
				slog.Error("failed evaluating symlink", "err", err, "path", path)
				return nil
			}

			path = realPath
		}

		if !supportedJavaFile(path) {
			return nil
		}

		// get dependencies from jar, skipping it if there are any errors
		deps, err := getFileDependencies(path, m)
		if err != nil {
			slog.Warn("failed getting dependencies from jar", "err", err)
			return nil
		}

		for _, dep := range deps {
			depId := dep.Id()
			slog.Debug("adding dependency", "dep", dep)
			dependencies[depId] = append(dependencies[depId], dep)
		}

		return nil
	})

	if err != nil {
		slog.Error("failed listing dependencies", "err", err)
		return nil, err
	}

	return dependencies, nil
}

func (m *JavaFilesPackageManager) GetProjectName() string {
	// no project for java files; user must provide the project
	return ""
}

func (m *JavaFilesPackageManager) GetFixer(workdir string) shared.DependencyFixer {
	return newFixer(m.targetDir, workdir)
}

func (m *JavaFilesPackageManager) GetEcosystem() string {
	return mappings.JavaEcosystem
}

func (m *JavaFilesPackageManager) GetScanTargets() []string {
	return []string{m.targetDir}
}

func (m *JavaFilesPackageManager) DownloadPackage(server api.ArtifactServer, descriptor shared.DependencyDescriptor) ([]byte, string, error) {
	return utils.DownloadMavenPackage(server, descriptor.AvailableFix.Library.Name, descriptor.AvailableFix.Version)
}

func (m *JavaFilesPackageManager) HandleFixes(fixes []shared.DependencyDescriptor) error {
	if m.Config.UseSealedNames {
		slog.Debug("using sealed names")
		for _, fix := range fixes {
			err := utils.SealJarName(fix)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (m *JavaFilesPackageManager) NormalizePackageName(name string) string {
	return utils.NormalizePackageName(name)
}

func (m *JavaFilesPackageManager) SilencePackages(silenceArray []api.SilenceRule, allDependencies common.DependencyMap) (map[string][]string, error) {
	return utils.SilencePackages(silenceArray, allDependencies, m)
}

func (m *JavaFilesPackageManager) ConsolidateVulnerabilities(vulnerablePackages *[]api.PackageVersion, allDependencies common.DependencyMap) (*[]api.PackageVersion, error) {
	return utils.ConsolidateVulnerabilities(vulnerablePackages, allDependencies)
}

func GetPackageManager(config *config.Config, targetDir string, targetFile string) (shared.PackageManager, error) {
	m := &JavaFilesPackageManager{Config: config, targetDir: targetDir}
	return m, nil
}
