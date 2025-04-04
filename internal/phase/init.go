package phase

import (
	"cli/internal/api"
	"cli/internal/common"
	"cli/internal/config"
	java_files "cli/internal/ecosystem/java/files"
	"cli/internal/ecosystem/mappings"
	"cli/internal/ecosystem/shared"
	"cli/internal/project"
	"cli/internal/repository"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const SealInternalFolderName = ".seal"

func InitConfiguration(path string) (*config.Config, error) {

	var confFile *os.File
	var confReader io.Reader
	confReader, err := os.Open(path)
	if err != nil {
		if !os.IsNotExist(err) {
			slog.Error("failed opening conf file", "err", err, "path", path)
			return nil, common.NewPrintableError("could not open config file in %s", path)
		}
		slog.Warn("initializing without config file")
		confReader = strings.NewReader("")
	} else {
		defer confFile.Close()
	}

	configuration, err := config.Load(confReader, nil)
	if err != nil {
		slog.Error("failed opening conf file", "err", err, "path", path)
		return nil, common.FallbackPrintableMsg(err, "failed parsing config file")
	}

	slog.Info("config loaded", "data", configuration) // will not log sensitive strings

	return configuration, nil
}

func createInternalSealFolder(projectDir string) (string, error) {
	p := filepath.Join(projectDir, SealInternalFolderName)
	err := os.RemoveAll(p)
	if err != nil {
		return "", err
	}

	slog.Debug("creating tmp folder", "path", p)

	err = os.MkdirAll(p, os.ModePerm) // will allow it if exists
	if err != nil {
		return "", err
	}

	return p, nil
}

func getPackageManager(targetType common.TargetType, config *config.Config, projectDir string, targetFile string) (manager shared.PackageManager, err error) {
	slog.Debug("checking for manager", "type", targetType, "dir", projectDir, "file", targetFile)
	switch targetType {
	case common.OsTarget:
		manager, err = findOSPackageManager(config, projectDir)
	case common.JavaFilesTarget:
		manager, err = java_files.GetPackageManager(config, projectDir, targetFile)
	case common.ManifestTarget:
		manager, err = findManifestPackageManager(config, projectDir, targetFile)
	default:
		slog.Error("unsupported target type", "type", targetType, "dir", projectDir, "file", targetFile)
		return nil, common.NewPrintableError("failed to find a supported package manager for %s", projectDir)
	}

	return manager, err

}

// prints warning to console if this is a new project / some misconfiguration might have happened
// IMPORTANT: can technically print here, as it is part of the init of the phase that comes before the progress bar is initialized
func (p *basePhase) init(targetPath string, configPath string, showProgress bool) error {
	var err error
	fmt.Printf("\n") // using this to print a new line before start of output

	// using locals until we initialize the manager, then we can use the Phase.Project struct
	if p.TargetType == common.OsTarget ||
		(p.TargetType == common.JavaFilesTarget && targetPath == "") {
		// os target or java files target with no target file should use the CWD and override the target path
		targetPath = common.CliCWD
		slog.Info("overriding target path - will use CWD", "targetPath", targetPath, "targetType", p.TargetType)
	}

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

	slog.Info("loaded config", "has-token", p.Config.Token != "")

	p.Manager, err = getPackageManager(p.TargetType, p.Config, projectDir, targetFile)
	if err != nil {
		return err
	}

	version := p.Manager.GetVersion()
	if !p.Manager.IsVersionSupported(version) {
		slog.Error("unsupported package manager version", "version", version)
		return common.NewPrintableError("unsupported package manager version %s", version)
	}

	if targetFile == "" && p.TargetType == common.ManifestTarget {
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

	p.Workdir, err = createInternalSealFolder(p.BaseDir)
	if err != nil {
		slog.Error("failed creating seal temp dir in project", "project-path", p.BaseDir)
		return common.NewPrintableError("failed creating temporary folder under %s", p.BaseDir)
	}

	// add to cleanup list so if the user chose to clean up, we will remove the folder
	common.AddPathToClean(common.RemoveTypeWd, p.Workdir)

	if err := p.initServers(); err != nil {
		slog.Error("failed initializing servers", "err", err)
		return err
	}

	if !p.Project.FoundLocally {
		slog.Info("generated project id is new", "tag", p.Project.Tag, "display-name", p.Project.NameCandidate)
		fmt.Printf("%s: %s\n", common.Colorize("using project-id", common.AnsiDarkGrey), p.Project.Tag)
	}

	p.Bar = common.NewProgressBar(showProgress, 0) // no steps, should be configured by actual phase
	p.showBar = showProgress                       // bar should not be changed directly

	slog.Info("initialized", "project-id", p.Project.Tag, "manager", p.Manager.Name(), "project-dir", p.BaseDir, "target", p.TargetFile, "tmp-workdir", p.Workdir)

	return nil
}

// returns url according to the manager's ecosystem
func getArtifactServerUrl(manager shared.PackageManager, conf *config.Config) string {
	ecosystem := manager.GetEcosystem()

	switch ecosystem {
	case mappings.PythonEcosystem:
		return api.PypiServer

	case mappings.NodeEcosystem:
		return api.NpmServer

	case mappings.DotnetEcosystem:
		return api.NugetServer

	case mappings.JavaEcosystem:
		return api.MavenServer

	case mappings.GolangEcosystem:
		return api.GolangServer

	case mappings.PhpEcosystem:
		return api.PackagistServer

	case mappings.RpmEcosystem:
		return api.RpmServer

	case mappings.DebEcosystem:
		return api.DebServer

	case mappings.ApkEcosystem:
		return api.ApkServer
	}

	slog.Error("could not match artifact server to manager", "ecosystem", ecosystem, "manager", manager.Name())

	return ""
}

// inits project id, prints warning message if generates new id
// assumes we already have target file
func (p *basePhase) initLocalProject(projectDir string, targetFile string) (err error) {

	// non-manifest targets require explicit project passed
	if p.TargetType != common.ManifestTarget {
		if p.Config.Project == "" && p.Config.Token != "" {
			slog.Error("project ID missing while token is present")
			return common.NewPrintableError("project ID missing")
		}

		p.Project.Tag = p.Config.Project
		p.Project.FoundLocally = true // we are in OS mode, so we must have a project id given by the user
		p.Project.NameCandidate = p.Config.Project
		return nil
	}

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

func (p *basePhase) initDefaultServers(client http.Client) error {

	token := p.Config.Token.Value()
	server := api.NewCliServer(token, p.Project.Tag, client)
	if server == nil {
		return common.NewPrintableError("failed setting up server")
	}

	baseUrl := getArtifactServerUrl(p.Manager, p.Config)
	if baseUrl == "" {
		return common.NewPrintableError("unsupported ecosystem")
	}

	artifactServer := api.NewArtifactServer(baseUrl, token, p.Project.Tag, client)
	if artifactServer == nil {
		return common.NewPrintableError("failed setting up artifact server")
	}

	if token != "" {
		p.CanAuthenticate = true
		slog.Debug("token configured, will use authenticated flow")
	}

	p.Backend = server
	p.ArtifactServer = artifactServer

	return nil
}

func (p *basePhase) initJfrogServers(client http.Client) error {
	jfrogToken := p.Config.JFrog.Token.Value()

	if p.Config.JFrog.Host == "" || jfrogToken == "" {
		// warn that it might be misconfigured
		slog.Error("partial JFrog configuration", "host", p.Config.JFrog.Host, "has-token", jfrogToken == "")
		return common.NewPrintableError("using JFrog requires both host and token")
	}

	cliBaseUrl, err := getJfrogCliServerUrl(p.Config)
	if err != nil {
		return err
	}

	cliBackend := api.NewCliJfrogServer(
		client,
		p.Project.Tag,
		jfrogToken,
		cliBaseUrl,
	)

	artifactServerBaseUrl, err := getJfrogArtifactServerUrl(p.Config, p.Manager)
	if err != nil {
		return err
	}

	artifactServer := api.NewJFrogArtifactServer(client,
		p.Project.Tag,
		jfrogToken,
		artifactServerBaseUrl,
	)

	p.CanAuthenticate = true
	p.Backend = cliBackend
	p.ArtifactServer = artifactServer

	return nil
}

// depends on project being initialized
// prints to screen if uses jfrog setver
func (p *basePhase) initServers() error {
	httpClient := http.Client{}

	if p.Config.JFrog.Enabled {
		slog.Info("initializing jfrog servers")
		if err := p.initJfrogServers(httpClient); err != nil {
			slog.Error("failed initialize jfrog", "err", err)
			return common.FallbackPrintableMsg(err, "failed to initialize JFrog servers")
		}

		fmt.Println(common.Colorize("using JFrog artifactory", common.AnsiDarkGrey))
		return nil
	}

	slog.Info("initializing default servers")
	return p.initDefaultServers(httpClient)
}
