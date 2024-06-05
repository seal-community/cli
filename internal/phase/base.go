package phase

import (
	"cli/internal/api"
	"cli/internal/common"
	"cli/internal/config"
	"cli/internal/ecosystem/dotnet"
	"cli/internal/ecosystem/java"
	"cli/internal/ecosystem/node"
	"cli/internal/ecosystem/python"
	"cli/internal/ecosystem/shared"
	"fmt"
	"log/slog"
	"path/filepath"

	"github.com/schollz/progressbar/v3"
)

type basePhase struct {
	ProjectDir string
	TargetFile string // optional as part of the commandline input
	Workdir    string
	Server     api.Server
	Config     *config.Config
	Bar        *progressbar.ProgressBar
	showBar    bool // required because can't access progressbar unexported state
	Manager    shared.PackageManager
	Fixer      shared.DependencyFixer
}

func findPackageManager(configDir *config.Config, projectDir string, target string) (shared.PackageManager, error) {
	nodeManager, nodeErr := node.GetPackageManager(configDir, projectDir, target)
	pythonManager, pythonErr := python.GetPackageManager(configDir, projectDir, target)
	javaManager, javaErr := java.GetPackageManager(configDir, projectDir, target)
	dotnetManager, dotnetErr := dotnet.GetPackageManager(configDir, projectDir, target)

	availableManagers := []struct {
		manager shared.PackageManager
		err     error
	}{
		{nodeManager, nodeErr},
		{pythonManager, pythonErr},
		{javaManager, javaErr},
		// dotnet should be last for now since its current implementation searches
		// recursively which can lead to a false positive identification
		{dotnetManager, dotnetErr},
	}

	manager := shared.PackageManager(nil)

	for _, m := range availableManagers {
		if m.err == nil {
			if manager != nil {
				slog.Warn("multiple package managers found, defaulting to", "manager", manager.Name())
				return manager, nil
			}
			manager = m.manager
		}
	}

	if manager != nil {
		slog.Info("found package manager", "manager", manager.Name())
		return manager, nil
	}

	slog.Error("no package manager found in the project directory", "errs", []error{nodeErr, pythonErr, dotnetErr})
	return nil, common.NewPrintableError("failed to find a supported package manager in the project directory")
}

func findProjectId(projMap map[string]config.ProjectInfo, projectDir string, targetFile string) (string, error) {

	relTarget, err := filepath.Rel(projectDir, targetFile)
	if err != nil {
		slog.Error("failed getting relative path for target file ", "err", err, "dir", projectDir, "file", targetFile)
		return "", common.NewPrintableError("could not locate target file %s in target dir %s", targetFile, projectDir)
	}

	for pid, pi := range projMap {
		for _, target := range pi.Targets {
			// case-sensitive exact comparison just in case it matters
			if target == relTarget {
				slog.Debug("found project id in config project map", "target", relTarget, "id", pid)
				return pid, nil
			}
		}
	}

	slog.Debug("target does not appear in config project map", "target", relTarget)
	return "", nil
}

func calculateProjectId(manager shared.PackageManager, projectDir string, targetFile string) (string, error) {
	// not implemented yet- should not reach here, return err that signifies the actual reason
	return "", common.NewPrintableError("config does not contain the provided target %s", targetFile)
}

func getProjectId(conf *config.Config, manager shared.PackageManager, projectDir string, targetFile string) (string, error) {
	if len(conf.ProjectMap) != 0 && targetFile != "" {
		slog.Debug("looking for project id in new config format", "target", targetFile)
		// this only works when we were provided a target file
		projId, err := findProjectId(conf.ProjectMap, projectDir, targetFile)
		if err != nil {
			return "", err
		}

		if projId != "" { // found the target file
			return projId, nil
		}

		slog.Warn("did not find manifest file in config, generating id")
		projId, err = calculateProjectId(manager, projectDir, targetFile)
		if err != nil || projId == "" {
			slog.Error("failed generating project id", "err", err, "projectId", projId)
			return "", common.FallbackPrintableMsg(err, "failed calculating project id")
		}

		// IMPORTANT: can technically print here, as it is part of the init of the phase that comes before the progress bar is initialized
		fmt.Printf("warning: using newly generated project-id: %s\n", projId)
		return projId, nil

	} else {
		// legacy project name
		// perform best effort to find a project name if it was not configured;
		slog.Info("project name not configured, using manager value", "manager", manager.Name())
		projId := manager.GetProjectName(projectDir)
		if projId == "" {
			slog.Warn("manager project name not viable, using folder name")
			projId = filepath.Base(projectDir)
		}

		return projId, nil
	}
}

func (p *basePhase) init(targetPath string, configPath string, showProgress bool) error {
	var err error
	p.ProjectDir = getProjectDir(targetPath)
	p.TargetFile = getTargetFile(targetPath) // will be empty if a directory was provided

	if p.ProjectDir == "" {
		return common.NewPrintableError("bad project directory path: %s", targetPath)
	}

	confFilePath := configPath
	if confFilePath == "" {
		slog.Debug("loading config from project folder", "dir", p.ProjectDir)
		confFilePath = filepath.Join(p.ProjectDir, config.ConfigFileName)
	}

	slog.Info("initialized project paths", "project-dir", p.ProjectDir, "target", p.TargetFile, "provided-path", targetPath, "config", confFilePath)

	p.Config, err = InitConfiguration(confFilePath)
	if err != nil {
		return err
	}

	p.Manager, err = findPackageManager(p.Config, p.ProjectDir, p.TargetFile)
	if err != nil {
		return err
	}

	if p.Config.Project == "" {
		// was not set as env override (or was set using legacy config)
		// try finding matching project id from project map
		slog.Debug("project id not set, trying to find it")
		projectId, err := getProjectId(p.Config, p.Manager, p.ProjectDir, p.TargetFile)
		if err != nil {
			return common.FallbackPrintableMsg(err, "failed finding project id")
		}

		p.Config.Project = projectId
	}

	p.Workdir, err = createInternalSealFolder(p.ProjectDir)
	if err != nil {
		slog.Error("failed creating seal temp dir in project", "project-path", p.ProjectDir)
		return common.NewPrintableError("failed creating temporary folder under %s", p.ProjectDir)
	}

	p.Server = api.Server{AuthToken: buildAuthToken(p.Config)}

	p.Bar = common.NewProgressBar(showProgress, 0) // no steps, should be configured by actual phase
	p.showBar = showProgress                       // bar should not be changed directly

	slog.Info("initialized", "conf-project", p.Config.Project, "project-dir", p.ProjectDir, "manager", p.Manager.Name())

	return nil
}

func (p *basePhase) cliMetadata() map[string]interface{} {
	return map[string]interface{}{
		"version": common.CliVersion,
	}
}

func (p *basePhase) HideProgress() {
	if !p.Bar.IsFinished() && p.showBar {
		slog.Warn("progress bar is not finished", "max", p.Bar.GetMax())
	}

	p.Bar.Finish()
}

// finish current step and change desc
// empty string will not change the current message
func (p *basePhase) advanceStep(desc string) {
	_ = p.Bar.Add(1)

	if desc != "" {
		p.Bar.Describe(desc)
	}
}

// for unknown steps that have finished
func (p *basePhase) addFinishedStep() {
	p.Bar.ChangeMax(p.Bar.GetMax() + 1)
	_ = p.Bar.Add(1)
}

func (p *basePhase) addToMax(amount int) {
	p.Bar.ChangeMax(p.Bar.GetMax() + amount)
}

// query the BE for the recommended versions specified in the input vulnerable packages
func (p *basePhase) QueryRecommendedPackages(vulnerablePackages []api.PackageVersion) ([]api.PackageVersion, error) {
	// uses the recommended fields to create a new common.Dependency instance and query the BE about it

	slog.Info("grabbing information about available fixes", "vulnerableCount", len(vulnerablePackages))
	// building array of 'deps' using the recommended fixed version
	deps := make([]common.Dependency, 0, len(vulnerablePackages))
	for _, vulnerable := range vulnerablePackages {
		if vulnerable.RecommendedLibraryVersionString == "" {
			slog.Info("ignoring vulnerable without recommendation")
			continue
		}

		deps = append(deps, common.Dependency{Name: vulnerable.Library.Name,
			Version:        vulnerable.RecommendedLibraryVersionString,
			PackageManager: vulnerable.Library.PackageManager,
		})
	}

	available, err := p.Server.FetchPackagesInfo(deps, nil, api.OnlyFixed, nil)
	if err != nil {
		slog.Error("failed getting fixed versions info", "err", err)
		return nil, common.NewPrintableError("failed querying recommended fixes")
	}

	slog.Debug("got fixes info", "count", len(*available))
	return *available, nil
}
