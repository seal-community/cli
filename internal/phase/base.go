package phase

import (
	"cli/internal/api"
	"cli/internal/common"
	"cli/internal/config"
	"cli/internal/ecosystem/node"
	"cli/internal/ecosystem/shared"
	"log/slog"
	"path/filepath"

	"github.com/schollz/progressbar/v3"
)

type basePhase struct {
	ProjectDir string
	Workdir    string
	Server     api.Server
	Config     *config.Config
	Bar        *progressbar.ProgressBar
	showBar    bool // required because can't access progressbar unexported state
	Manager    shared.PackageManager
	Fixer      shared.DependencyFixer
}

func (p *basePhase) init(path string, showProgress bool) error {
	var err error
	p.ProjectDir = getProjectDir(path)
	if p.ProjectDir == "" {
		return common.NewPrintableError("bad project directory path: %s", path)
	}

	confFilePath := filepath.Join(p.ProjectDir, config.ConfigFileName)

	p.Config, err = InitConfiguration(confFilePath)
	if err != nil {
		return err
	}

	p.Manager, err = node.GetPackageManager(p.Config, p.ProjectDir) // currently only node ecosystem managers are supported
	if err != nil {
		return err
	}

	if p.Config.Project == "" {
		// perform best effort to find a project name if it was not configured;
		slog.Info("project name not configured, using package.json value")
		projName := p.Manager.GetProjectName(p.ProjectDir)
		if projName == "" {
			slog.Warn("package.json name not viable, using folder name")
			projName = filepath.Base(p.ProjectDir)
		}

		p.Config.Project = projName
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

func (p *basePhase) QueryFixesForPackages(vulnerablePackages []api.PackageVersion) ([]api.PackageVersion, error) {

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

	available, err := p.Server.GetFixedPackages(deps, nil, nil)
	if err != nil {
		slog.Error("failed getting fixed versions info", "err", err)
		return nil, common.NewPrintableError("server error")
	}

	slog.Debug("got fixes info", "count", len(*available))
	return *available, nil
}
