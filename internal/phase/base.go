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
	Config  *config.Config
	Workdir string // .seal internal work dir

	Project project.ProjectInfo

	BaseDir    string
	TargetFile string

	ArtifactServer api.ArtifactServer
	Backend        api.Backend
	Manager        shared.PackageManager

	Bar     *progressbar.ProgressBar
	showBar bool // required because can't access progressbar unexported state

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

	if !filepath.IsAbs(f) {
		// manager should return it as abs path; but make sure
		// abs will use CWD if the path is not absolute
		slog.Debug("manager passed non-abs path", "target", f)
		f = filepath.Join(p.BaseDir, f)
	}

	if exists, _ := common.PathExists(f); !exists {
		// must exist
		slog.Error("failed finding scan target from manager", "final-path", f, "scan-target", targets[0])
		return "", fmt.Errorf("failed finding scan target file")
	}

	return f, nil
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

	projDesc, err := p.InitializeProject()
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

	available, err := fetchPackagesInfo(p.Backend, deps, nil, api.OnlyFixed, nil)
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

	err := p.Backend.CheckAuthenticationValid()
	p.addFinishedStep() // treating this as unexpected step (scan flow)

	return err
}

func (p basePhase) InitializeProject() (*api.ProjectDescriptor, error) {
	return p.Backend.InitializeProject(p.Project.NameCandidate)
}

func fetchPackagesInfo(server api.Backend, deps []common.Dependency, metadata api.Metadata, queryType api.PackageQueryType, chunkDone api.ChunkDownloadedCallback) (*[]api.PackageVersion, error) {
	allVersions := make([]api.PackageVersion, 0, len(deps))
	chunkSize := server.GetPackageChunkSize()

	err := common.ConcurrentChunks(deps, chunkSize,
		func(chunk []common.Dependency, chunkIdx int) (*api.Page[api.PackageVersion], error) {
			return server.QueryPackages(&api.BulkCheckRequest{
				Metadata: metadata,
				Entries:  chunk,
			}, queryType)
		},
		func(data *api.Page[api.PackageVersion], chunkIdx int) error {
			// safe to perform, run from inside mutex
			allVersions = append(allVersions, data.Items...)
			if chunkDone != nil {
				chunkDone(data.Items, chunkIdx)
			}
			return nil
		})

	return &allVersions, err
}

func fetchPackagesInfoAuth(server api.Backend, deps []common.Dependency, metadata api.Metadata, queryType api.PackageQueryType, chunkDone api.ChunkDownloadedCallback, generateActivity bool) (*[]api.PackageVersion, error) {
	allVersions := make([]api.PackageVersion, 0, len(deps))
	chunkSize := server.GetPackageChunkSize()

	err := common.ConcurrentChunks(deps, chunkSize,
		func(chunk []common.Dependency, chunkIdx int) (*api.Page[api.PackageVersion], error) {
			return server.QueryPackagesAuth(&api.BulkCheckRequest{
				Metadata: metadata,
				Entries:  chunk,
			}, queryType, generateActivity)
		},
		func(data *api.Page[api.PackageVersion], chunkIdx int) error {
			// safe to perform, run from inside mutex
			allVersions = append(allVersions, data.Items...)
			if chunkDone != nil {
				chunkDone(data.Items, chunkIdx)
			}
			return nil
		})

	return &allVersions, err
}

func fetchOverriddenPackagesInfo(server api.Backend, query []api.RemoteOverrideQuery, chunkDone api.ChunkDownloadedCallback) (*[]api.PackageVersion, error) {
	allVersions := make([]api.PackageVersion, 0, len(query))
	chunkSize := server.GetRemoteConfigChunkSize()

	err := common.ConcurrentChunks(query, chunkSize,
		func(chunk []api.RemoteOverrideQuery, chunkIdx int) (*api.Page[api.PackageVersion], error) {
			return server.QueryRemoteConfig(chunk)
		},
		func(data *api.Page[api.PackageVersion], chunkIdx int) error {
			// safe to perform, run from inside mutex
			allVersions = append(allVersions, data.Items...)
			if chunkDone != nil {
				chunkDone(data.Items, chunkIdx)
			}
			return nil
		})

	return &allVersions, err
}
