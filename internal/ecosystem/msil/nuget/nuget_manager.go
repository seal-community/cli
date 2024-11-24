package nuget

import (
	"bufio"
	"cli/internal/api"
	"cli/internal/common"
	"cli/internal/config"
	"cli/internal/ecosystem/mappings"
	"cli/internal/ecosystem/msil/utils"
	"cli/internal/ecosystem/shared"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
)

const nugetExeName = "nuget"
const nugetManagerName = "nuget"

// this version tested to work with add/update operations;
// despite it being supported from an earlier version - those didn't work
const minimalRequiredNugetVersion = "3.5.0"

type NugetManager struct {
	Config        *config.Config
	targetDir     string
	targetFile    string
	auxFile       string // additional file we use for scan / edit - currently only packages.config is supported
	packagesDir   string
	projectFormat utils.ProjectFormat
	nugetVersion  string
	solutionDir   string
}

func getDefaultAuxFileForFormat(format utils.ProjectFormat) string {
	switch format {
	case utils.FormatLegacyPackagesConfig:
		return utils.DefaultPackagesConfigFile
	}

	return ""
}

func getPackagesDirForFormat(format utils.ProjectFormat) string {
	switch format {
	case utils.FormatLegacyPackagesConfig:
		return utils.DefaultPackagesDirName
	}

	return ""
}

// will look up the tree until sln is found
func findSolutionDir(projFile string) string {
	const limitParentDir = 10
	d := filepath.Dir(projFile)

	for i := 1; i <= limitParentDir; i++ {
		slog.Debug("searching for .sln files in folder", "path", d)
		ptrn := filepath.Join(d, "*.sln")
		matches, err := filepath.Glob(ptrn)
		if err != nil {
			slog.Error("failed running glob for solution files in path", "path", d)
			break
		}

		if len(matches) > 0 {
			slog.Info("found solution matches", "files", matches)
			return d
		}

		old := d
		d = filepath.Join(d, "..")
		if d == old {
			slog.Info("reached root of filesystem without finding sln files")
			break
		}
	}

	slog.Warn("did not find any .sln files")
	return ""
}

func NewNugetManager(config *config.Config, targetDir string, targetFile string, frmt utils.ProjectFormat, auxFile string, packagesDir string) (*NugetManager, error) {
	if auxFile == "" {
		filename := getDefaultAuxFileForFormat(frmt)
		if filename == "" {
			slog.Error("unsupported project format for aux file", "format", frmt)
			return nil, fmt.Errorf("unsupported project format")
		}

		auxFile = filepath.Join(targetDir, filename)
		slog.Info("will use default auxilary file", "name", auxFile)
	}

	if exists, err := common.PathExists(auxFile); !exists || err != nil {
		slog.Error("aux file does not exist", "path", auxFile, "err", err)
		return nil, fmt.Errorf("could not find packages config file")
	}

	slnFolder := findSolutionDir(targetFile)
	if slnFolder == "" {
		slog.Error("cannot find solution file")
		return nil, fmt.Errorf("could not find sln file")
	}

	if packagesDir == "" {
		// find the packages dir. should be next to solution
		packagesFolderName := getPackagesDirForFormat(frmt)
		if packagesFolderName == "" {
			slog.Error("unsupported project format for packages dir", "format", frmt)
			return nil, fmt.Errorf("unknown packages dir")
		}

		packagesDir = filepath.Join(slnFolder, packagesFolderName)
	}

	if exists, err := common.DirExists(packagesDir); !exists || err != nil {
		slog.Error("could not find packages dir", "err", err, "exists", exists, "path", packagesDir)
		return nil, fmt.Errorf("could not find packages dir")
	}

	slog.Info("using packages dir", "path", packagesDir)

	m := &NugetManager{Config: config,
		targetDir:     targetDir,
		targetFile:    targetFile,
		projectFormat: frmt,
		auxFile:       auxFile,
		packagesDir:   packagesDir,
		solutionDir:   slnFolder,
	}

	return m, nil
}

func (m *NugetManager) Name() string {
	return nugetManagerName
}

func (m *NugetManager) GetVersion() string {
	if m.nugetVersion != "" {
		return m.nugetVersion
	}

	m.nugetVersion = getVersion(m.solutionDir) // running `nuget` from solution dir for good practice
	return m.nugetVersion
}

func parseVersionOutput(stdout string) string {

	s := bufio.NewScanner(strings.NewReader(stdout))
	s.Scan()
	line := s.Text()
	prefix := "nuget version: "
	if !strings.HasPrefix(strings.ToLower(line), prefix) {
		slog.Error("unexpeted stdout", "line", line)
		return ""
	}

	parts := strings.Split(line, ": ")
	if len(parts) != 2 {
		slog.Error("unexpeted stdout parts", "parts", parts)
		return ""
	}

	return parts[1]
}

func getVersion(runFromDir string) string {
	result, err := common.RunCmdWithArgs(runFromDir, nugetExeName)
	if err != nil {
		slog.Error("failed running dotnet version", "err", err)
		return ""
	}

	if result.Code != 0 {
		slog.Error("running dotnet version returned non-zero", "result", result)
		return ""
	}

	return parseVersionOutput(result.Stdout)
}

func (m *NugetManager) IsVersionSupported(version string) bool {

	// nuget version could contain a fourth element 'revision', so not semver compliant
	// https://learn.microsoft.com/en-us/nuget/concepts/package-versioning?tabs=semver20sort#where-nugetversion-diverges-from-semantic-versioning
	supported, _ := common.VersionAtLeast(version, minimalRequiredNugetVersion)
	return supported
}

func (m *NugetManager) ListDependencies() (common.DependencyMap, error) {
	f, err := common.OpenFile(m.auxFile)
	if err != nil {
		slog.Error("failed opening auxilary file", "path", m.auxFile)
		return nil, err
	}
	defer f.Close()

	return parsePackagesConfig(f, m.packagesDir)
}

func (m *NugetManager) GetProjectName() string {
	return filepath.Base(m.targetFile)
}

func (m *NugetManager) GetFixer(workdir string) shared.DependencyFixer {
	// if we add support for project.json, we could check format
	return newFixer(workdir, m.packagesDir, m.auxFile, m.solutionDir)
}

func (m *NugetManager) GetEcosystem() string {
	return mappings.DotnetEcosystem
}

func (m *NugetManager) GetScanTargets() []string {
	// currently not adding the aux file to the scan targets array
	// it is not fully supported when locating project id & actions file;
	// might want to add a new array for aux files in the future
	return []string{m.targetFile}
}

func (m *NugetManager) DownloadPackage(server api.ArtifactServer, descriptor shared.DependencyDescriptor) ([]byte, string, error) {
	return utils.DownloadNugetPackage(server, descriptor.AvailableFix.Library.NormalizedName, descriptor.AvailableFix.Version)
}

func (m *NugetManager) HandleFixes(fixes []shared.DependencyDescriptor) error {
	if m.Config.UseSealedNames {
		slog.Warn("using sealed names in nuget is not supported yet")
	}

	// not required

	return nil
}

func (m *NugetManager) NormalizePackageName(name string) string {
	return utils.NormalizeName(name)
}

func (m *NugetManager) SilencePackages(silenceArray []string, allDependencies common.DependencyMap) (map[string][]string, error) {
	slog.Warn("Silencing packages is not support for nuget")
	return nil, nil
}

func (m *NugetManager) ConsolidateVulnerabilities(vulnerablePackages *[]api.PackageVersion, allDependencies common.DependencyMap) (*[]api.PackageVersion, error) {
	return vulnerablePackages, nil
}
