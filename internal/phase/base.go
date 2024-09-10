package phase

import (
	"cli/internal/api"
	"cli/internal/common"
	"cli/internal/config"
	"cli/internal/ecosystem/dotnet"
	"cli/internal/ecosystem/golang"
	"cli/internal/ecosystem/java"
	"cli/internal/ecosystem/node"
	"cli/internal/ecosystem/python"
	"cli/internal/ecosystem/shared"
	"cli/internal/project"
	"cli/internal/repository"
	"fmt"
	"log/slog"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/schollz/progressbar/v3"
	"golang.org/x/term"
)

func getProjectDirAbs(p string) string {
	if p == "" {
		return common.CliCWD
	}

	return common.GetAbsDirPath(p)
}

func getTargetFileAbs(p string) string {
	if p == "" {
		return ""
	}

	abs, _ := filepath.Abs(p) // ignoring err, propagated from internal call to os.Cwd

	f, err := os.Stat(abs)
	if err != nil || f.IsDir() {
		slog.Debug("target path is not a file", "err", err, "path", abs) // ignoring error case here, same logic
		return ""
	}

	// strip input from file component
	return abs
}

type basePhase struct {
	Project project.ProjectInfo

	BaseDir    string
	TargetFile string

	Workdir string // .seal internal work dir
	Server  api.Server
	Config  *config.Config
	Bar     *progressbar.ProgressBar
	showBar bool // required because can't access progressbar unexported state
	Manager shared.PackageManager
	Fixer   shared.DependencyFixer

	CanAuthenticate bool
}

func findPackageManager(configDir *config.Config, projectDir string, target string) (shared.PackageManager, error) {
	nodeManager, nodeErr := node.GetPackageManager(configDir, projectDir, target)
	pythonManager, pythonErr := python.GetPackageManager(configDir, projectDir, target)
	javaManager, javaErr := java.GetPackageManager(configDir, projectDir, target)
	dotnetManager, dotnetErr := dotnet.GetPackageManager(configDir, projectDir, target)
	golangManager, golangErr := golang.GetPackageManager(configDir, projectDir, target)

	availableManagers := []struct {
		manager shared.PackageManager
		err     error
	}{
		{nodeManager, nodeErr},
		{pythonManager, pythonErr},
		{javaManager, javaErr},
		{golangManager, golangErr},
		// dotnet should be last for now since its current implementation searches
		// recursively which can lead to a false positive identification
		{dotnetManager, dotnetErr},
	}

	// choose first manager without error
	var manager shared.PackageManager
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

	// to have pretty message on the input file / project directory
	actualTarget := projectDir
	if target != "" {
		actualTarget = target
	}

	slog.Error("no package manager found in the project directory", "errs", []error{nodeErr, pythonErr, dotnetErr})
	return nil, common.NewPrintableError("failed to find a supported package manager for %s", actualTarget)
}

func (p *basePhase) findTargetFileFromManager() (string, error) {
	targets := p.Manager.GetScanTargets() // no need to normalize, as paths are found from this run
	if len(targets) == 0 {
		slog.Error("failed finding any scan targets")
		return "", fmt.Errorf("no target file found for %s", p.Manager.Name())
	}

	if len(targets) > 1 {
		slog.Warn("unsupported multiple scan targets; using the first", "target", targets, "manager", p.Manager.Name())
	}

	f := targets[0]

	// manager should return it as abs path, but just in case
	// make sure this returns abs path to the file
	// to conform to the normal flow
	absFile := getTargetFileAbs(f)
	if absFile == "" {
		// must exist
		slog.Error("failed finding scan target from manager")
		return "", fmt.Errorf("failed finding scan target file")
	}

	return absFile, nil
}

// inits project id, prints warning message if generates new id
// assumes we already have target file
func (p *basePhase) initLocalProject(projectDir string, targetFile string) error {

	relTarget, err := filepath.Rel(projectDir, targetFile)
	// must be a subpath within project dir, so not allowed to have relative dir traversal; unlikely
	if err != nil || strings.Contains(relTarget, "..") {
		slog.Error("failed getting relative path for target file ", "err", err, "dir", projectDir, "file", targetFile)
		return common.NewPrintableError("cannot use file %s and target dir %s", targetFile, projectDir)
	}

	remoteUrl, err := repository.FindGitRemoteUrl(projectDir) // remote url can be empty string if not relevant to target dir
	if err != nil {
		// continue best-effort in case remote url repo logic failed
		slog.Warn("error finding remote url - continuing to use fallback", "err", err)
	}

	projId, found, err := project.ChooseProjectId(p.Manager, projectDir, relTarget, p.Config.Project, p.Config.ProjectMap, remoteUrl)
	if err != nil {
		return err
	}

	slog.Info("using project", "id", projId, "found", found)

	p.Project.Tag = projId
	p.Project.FoundLocally = found

	slog.Info("generating display name candidate")
	p.Project.NameCandidate = project.GenerateProjectDisplayName(p.Manager, projectDir)

	slog.Debug("initialized project", "project", p.Project)

	return nil
}

// used to print message despite having a progress bar running
// will used \r to start from the beginning of the line
// and overwrite the existing line, padding until the terminal's width to clear remaining
// parts of the progress bar
// do not use \n in the message itself
//
// this adds \n after the message, so when the progress bar continues it does not remove the printed line
func printMsgDespiteProgressBar(msg string, args ...any) {
	baseMsgL := fmt.Sprintf(msg, args...)
	fd := int(os.Stdout.Fd())
	if term.IsTerminal(fd) {
		width, _, err := term.GetSize(fd)
		if err == nil {
			fmt.Printf("\r%s%s\n", baseMsgL, strings.Repeat(" ", int(math.Max(0, float64(width-len(baseMsgL))))))
			return
		}
		slog.Warn("could not get size of terminal", "err", err)
	}

	// fallback just print, will duplicate progress bar
	fmt.Printf("%s\n", baseMsgL)
}

// prints warning to console
func (p *basePhase) InitRemoteProject() error {
	if err := p.ValidateAuth(); err != nil {
		slog.Error("auth failed", "err", err)
		return common.FallbackPrintableMsg(err, "authentication issue")
	}

	p.Bar.Describe("Getting project information")

	projDesc, err := p.Server.InitializeProject(p.Project.Tag, p.Project.NameCandidate)
	if err != nil {
		slog.Error("failed initializing project", "err", err, "tag", p.Project.Tag, "name-candidate", p.Project.NameCandidate)
		return common.FallbackPrintableMsg(err, "failed querying project from server")
	}

	if p.Project.Tag != projDesc.Tag {
		slog.Error("project tag mismatch", "remote", projDesc.Tag, "local", p.Project.Tag, "remote-name", projDesc.Name, "name-candidate", p.Project.NameCandidate, "is-new", projDesc.New)
		return common.NewPrintableError("wrong project id found on server: %s", projDesc.Tag)
	}

	msg := ""
	if projDesc.New {
		msg = "created new project"
	} else {
		msg = "project name"
	}

	// strong-arming the progress bar and printing the message anyway
	printMsgDespiteProgressBar("%s: %s", common.Colorize(msg, common.AnsiDarkGrey), projDesc.Name)
	printMsgDespiteProgressBar("")

	p.Project.New = projDesc.New
	p.Project.RemoteName = projDesc.Name

	slog.Info("received remote project information", "tag", projDesc.Tag, "name", projDesc.Name, "is-new", projDesc.New)

	p.addFinishedStep() // since this was unexpected in scan flow

	return nil
}

// prints warning to console if this is a new project
func (p *basePhase) init(targetPath string, configPath string, showProgress bool) error {
	var err error

	// using locals until we initialize the manager, then we can use the Phase.Project struct
	projectDir := getProjectDirAbs(targetPath)
	targetFile := getTargetFileAbs(targetPath) // will be empty if a directory was provided

	if projectDir == "" {
		return common.NewPrintableError("bad project directory path: %s", targetPath)
	}

	p.BaseDir = projectDir
	p.TargetFile = targetFile

	confFilePath := configPath
	if confFilePath == "" {
		slog.Debug("loading config from project folder", "dir", projectDir)
		confFilePath = filepath.Join(projectDir, config.ConfigFileName)
	}

	slog.Info("initialized project paths", "project-dir", projectDir, "target", targetFile, "provided-path", targetPath, "config", confFilePath)

	p.Config, err = InitConfiguration(confFilePath)
	if err != nil {
		return err
	}

	slog.Info("initiated config", "has-token", p.Config.Token != "")

	p.Manager, err = findPackageManager(p.Config, projectDir, targetFile)
	if err != nil {
		return err
	}

	if targetFile == "" {
		// reaching here means we already found an indicator and have a package manager associated with the project dir
		// use target file according to manager until scanning directory is deprecated
		slog.Warn("looking up indicator in project dir since target file not provided", "project-dir", projectDir)

		target, err := p.findTargetFileFromManager()
		if err != nil || target == "" {
			// unlikely as we already found an indicator file in the manager
			slog.Error("failed finding target file using the manager", "err", err, "target", target)
			return common.NewPrintableError("could not find a scannable target in %s", projectDir)
		}

		targetFile = target
	}

	// use p.Project.{} after here
	if err := p.initLocalProject(projectDir, targetFile); err != nil {
		return err
	}

	// validate project, regardless of phase
	if reason := project.ValidateProjectId(p.Project.Tag); reason != "" {
		slog.Error("invalid projcet name", "name", p.Project.Tag, "project-dir", p.BaseDir)
		return common.NewPrintableError("invalid project name `%s` - %s", p.Project.Tag, reason)
	}

	if !p.Project.FoundLocally {
		// IMPORTANT: can technically print here, as it is part of the init of the phase that comes before the progress bar is initialized
		slog.Info("generated project id is new", "tag", p.Project.Tag, "display-name", p.Project.NameCandidate)
		fmt.Printf("\n%s: %s\n", common.Colorize("using project-id", common.AnsiDarkGrey), p.Project.Tag)
	}

	p.Workdir, err = createInternalSealFolder(p.BaseDir)
	if err != nil {
		slog.Error("failed creating seal temp dir in project", "project-path", p.BaseDir)
		return common.NewPrintableError("failed creating temporary folder under %s", p.BaseDir)
	}

	if p.Config.Token != "" {
		p.CanAuthenticate = true
	}

	p.Server = api.Server{AuthToken: buildAuthToken(p.Config.Token, p.Project.Tag)}

	p.Bar = common.NewProgressBar(showProgress, 0) // no steps, should be configured by actual phase
	p.showBar = showProgress                       // bar should not be changed directly

	slog.Info("initialized", "project-id", p.Project.Tag, "manager", p.Manager.Name(), "project-dir", p.BaseDir, "target", p.TargetFile, "tmp-workdir", p.Workdir)

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
			NormalizedName: vulnerable.Library.NormalizedName,
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

// check authentication according to api.Server's configured values
// relevant for authenticated flows
func (p *basePhase) ValidateAuth() error {
	p.Bar.Describe("Checking authentication")
	if !p.CanAuthenticate {
		slog.Error("no auth token")
		return common.NewPrintableError("missing authentication token")
	}

	err := p.Server.CheckAuthenticationValid()
	p.addFinishedStep() // treating this as unexpected step (scan flow)

	return err
}
