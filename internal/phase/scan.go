package phase

import (
	"cli/internal/api"
	"cli/internal/common"
	"log/slog"
)

const ScanSteps = 3

type scanPhase struct {
	*basePhase
}

func NewScanPhase(target string, configPath string, showProgress bool) (*scanPhase, error) {
	c := basePhase{}
	if err := c.init(target, configPath, showProgress); err != nil {
		return nil, err
	}
	c.addToMax(ScanSteps)

	return &scanPhase{
		basePhase: &c,
	}, nil

}

type PackageManagerMetadata struct {
	Version string `json:"version"`
	Name    string `json:"name"`
}

func (sp *scanPhase) Collect() (common.DependencyMap, error) {
	defer common.ExecutionTimer().Log()
	packageManager := sp.Manager

	slog.Info("collecting packages", "manager_version", packageManager.GetVersion())

	dependencyMap, err := packageManager.ListDependencies()
	if err != nil {
		slog.Error("failed parsing package manager output", "err", err)
		// general error, might be caused due to return code
		return nil, common.FallbackPrintableMsg(err, "failed parsing project dependencies")

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

// will query vulnerable packages in db
// uses authenticated endpoint if possible
// will generate activity for the scanned items if instructed so
func (sp *scanPhase) checkVulnerabilitiesInPackages(deps []common.Dependency, metadata api.Metadata, generateActivity bool) (*[]api.PackageVersion, error) {
	progressCb := func([]api.PackageVersion, int) {
		// for each 'unexpected' step (i.e. chunk) increase max by one
		sp.addFinishedStep()
	}

	if generateActivity && !sp.CanAuthenticate {
		slog.Warn("bad input for generating scan acitivty", "project", sp.Project.Tag)
		return nil, common.NewPrintableError("uploading scan results requires a valid token and project")
	}

	if sp.CanAuthenticate {
		// if generateActivity is true we will store the vulnerable packages as activity
		// authentication check should have happend before hand
		return fetchPackagesInfoAuth(sp.Backend, deps, metadata, api.OnlyVulnerable, progressCb, generateActivity)
	} else {
		slog.Debug("using unauth package query")
		return fetchPackagesInfo(sp.Backend, deps, metadata, api.OnlyVulnerable, progressCb)
	}
}

func (sp *scanPhase) Scan(generateActivity bool) (*ScanResult, error) {
	slog.Info("starting scan", "target", sp.BaseDir)

	sp.Bar.Describe("Checking metadata")
	_ = sp.Bar.RenderBlank() // draw without progress to show the description

	metadata, err := gatherMetadata(sp.Manager)
	if err != nil {
		slog.Error("failed collecting metadata", "err", err)
		return nil, common.FallbackPrintableMsg(err, "failed checking metadata")
	}

	sp.advanceStep("Scanning local dependencies")

	dependencyMap, err := sp.Collect()
	slog.Debug("done collecting dependencies")
	if err != nil {
		return nil, common.FallbackPrintableMsg(err, "failed collecting dependencies")
	}

	if len(dependencyMap) == 0 {
		slog.Warn("no dependencies found", "target", sp.BaseDir)
		// return "No dependencies found", true
		return &ScanResult{}, nil // empty result
	}

	slog.Info("finished local dependency gathering", "count", len(dependencyMap))
	sp.advanceStep("Searching for vulnerabilities")

	vulnerable, err := sp.checkVulnerabilitiesInPackages(reduceToUniqueDeps(dependencyMap), metadata, generateActivity)

	if err != nil || vulnerable == nil {
		slog.Error("failed getting vulnerabilities", "err", err)
		return nil, common.FallbackPrintableMsg(err, "server error")
	}

	vulnerable, err = sp.Manager.ConsolidateVulnerabilities(vulnerable, dependencyMap)
	if err != nil {
		slog.Error("failed consolidating vulnerabilities", "err", err)
		return nil, err
	}

	result := &ScanResult{
		Vulnerable:      *vulnerable,
		AllDependencies: dependencyMap,
	}

	slog.Info("got vulnerable packages from server", "count", len(result.Vulnerable))
	sp.advanceStep("") // final step

	return result, nil
}
