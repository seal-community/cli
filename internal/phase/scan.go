package phase

import (
	"cli/internal/api"
	"cli/internal/common"
	"errors"
	"fmt"
	"log/slog"
)

const ScanSteps = 3

type scanPhase struct {
	*basePhase
	// hold scan results
}

func NewScanPhase(projectDir string, showProgress bool) (*scanPhase, error) {
	c := basePhase{}
	if err := c.init(projectDir, showProgress); err != nil {
		return nil, err
	}
	c.addToMax(ScanSteps)

	return &scanPhase{
		basePhase: &c,
	}, nil

}

var ManagerProcessFailed = common.NewPrintableError("failed running package manager")
var FailedParsingManagerOutput = common.NewPrintableError("failed parsing package manager output")

type PackageManagerMetadata struct {
	Version string `json:"version"`
	Name    string `json:"name"`
}

func (sp *scanPhase) metadata() (*PackageManagerMetadata, error) {
	packageManager := sp.Manager

	name := packageManager.Name()
	version := packageManager.GetVersion(sp.ProjectDir)
	if version == "" {
		slog.Error("failed getting version of manager", "name", name)
		// IMPORTANT: in future we might want to return printable error here
		return nil, fmt.Errorf("failed getting package manager version")
	}

	slog.Info("package manager version", "version", version, "name", name)

	return &PackageManagerMetadata{Version: version, Name: name}, nil
}

func (sp *scanPhase) Collect() (common.DependencyMap, error) {
	defer common.ExecutionTimer().Log()
	packageManager := sp.Manager
	targetDir := sp.ProjectDir

	slog.Info("collecting npm packages", "manager_version", packageManager.GetVersion(targetDir))

	result, ok := packageManager.ListDependencies(targetDir)
	if !ok {
		slog.Error("failed running package manager in the current dir", "name", packageManager.Name())
		// propagate error message
		return nil, ManagerProcessFailed
	}

	slog.Debug("going to parse output", "code", result.Code, "stderr", result.Stderr)
	parser := packageManager.GetParser()
	dependencyMap, err := parser.Parse(result.Stdout, targetDir)

	if err != nil {
		slog.Error("failed parsing package manager output", "err", err, "code", result.Code, "stderr", result.Stderr)
		slog.Debug("manager output", "stdout", result.Stdout) // useful for debugging its output
		// general error, might be caused due to return code
		return nil, errors.Join(err, FailedParsingManagerOutput)

	}

	return dependencyMap, nil
}

type ScanResult struct {
	Vulnerable      []api.PackageVersion
	AllDependencies common.DependencyMap
}

func reduceToUniqueDeps(dependencyMap common.DependencyMap) []common.Dependency {
	// will return the first instance of every dep for now
	dependencies := make([]common.Dependency, 0, len(dependencyMap))
	for _, val := range dependencyMap {
		if len(val) > 0 {
			dependencies = append(dependencies, *val[0])
		}
	}

	return dependencies
}

func (sp *scanPhase) Scan() (*ScanResult, error) {
	slog.Info("starting scan", "target", sp.ProjectDir)

	metadata := sp.cliMetadata()

	sp.Bar.Describe("Checking metadata")
	_ = sp.Bar.RenderBlank() // draw without progress to show the description

	nodeMetadata, err := sp.metadata()
	if err != nil {
		return nil, common.FallbackPrintableMsg(err, "failed checking metadata")
	}

	if nodeMetadata != nil {
		slog.Debug("done collecting npm metadata")
		metadata["npm"] = nodeMetadata
	}

	sp.advanceStep("Scanning local dependencies")

	dependencyMap, err := sp.Collect()
	slog.Debug("done collecting npm dependencies")
	if err != nil {
		return nil, common.FallbackPrintableMsg(err, "failed collecting dependencies")
	}

	if len(dependencyMap) == 0 {
		slog.Warn("no dependencies found", "target", sp.ProjectDir)
		// return "No dependencies found", true
		return &ScanResult{}, nil // empty result
	}

	slog.Info("finished local dependency gathering", "count", len(dependencyMap))
	sp.advanceStep("Searching for vulnerabilities")

	vulnerable, err := sp.Server.CheckVulnerablePackages(reduceToUniqueDeps(dependencyMap), metadata, func(chunk []api.PackageVersion, idx, total int) {
		// for each 'unexpected' step (i.e. chunk) increase max by one
		sp.addFinishedStep()
	})

	if err != nil {
		slog.Error("failed getting vulnerabilities", "err", err)
		return nil, common.NewPrintableError("server error")
	}

	result := &ScanResult{
		Vulnerable:      *vulnerable,
		AllDependencies: dependencyMap,
	}

	slog.Info("got packages from server", "count", len(result.Vulnerable))
	sp.advanceStep("") // final step

	return result, nil
}
