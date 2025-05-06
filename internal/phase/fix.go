package phase

import (
	"cli/internal/actions"
	"cli/internal/api"
	"cli/internal/common"
	"cli/internal/ecosystem/shared"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"runtime/debug"

	"golang.org/x/sync/errgroup"
)

const ConcurrentDownloadCount = 10
const FixSteps = 2

type fixPhase struct {
	*scanPhase
}

type FixMode string

const (
	FixModeLocal  FixMode = "local"  // use actions file
	FixModeRemote FixMode = "remote" // use remotely configured rules
	FixModeAll    FixMode = "all"    // install all available fixes
)

type PostFixRunner interface {
	HandleAppliedFixes(projectDir string, fixes []shared.DependencyDescriptor, vulnerable []api.PackageVersion) error
	ShouldSkip() bool
	GetStepDescription() string
}

func NewFixPhase(target string, targetType common.TargetType, configPath string, showProgress bool) (*fixPhase, error) {
	sp, err := NewScanPhase(target, targetType, configPath, showProgress)
	if err != nil {
		return nil, err
	}

	sp.addToMax(FixSteps) // increase max to accommodate fix logic in progress bar
	fp := &fixPhase{
		scanPhase: sp,
	}

	return fp, nil
}

func packageDownloadWorker(ctx context.Context, artifactServer api.ArtifactServer, manager shared.PackageManager, downloadJobsChannel chan shared.DependencyDescriptor, downloadResultsChannel chan shared.PackageDownload) (err error) {
	defer func() {
		if panicObj := recover(); panicObj != nil {
			slog.Error("panic caught", "err", panicObj, "trace", string(debug.Stack()))
			err = fmt.Errorf("panic caught: %v", panicObj)
		}
	}()

	for {
		select {
		case <-ctx.Done():
			slog.Debug("download worker cancelled")
			return nil

		case descriptor, more := <-downloadJobsChannel:
			if !more {
				slog.Debug("download worker finished")
				return nil
			}

			fixedPackage := *descriptor.AvailableFix
			slog.Debug("downloading package", "id", fixedPackage.Id())
			data, name, err := manager.DownloadPackage(artifactServer, descriptor)
			if err != nil {
				slog.Error("failed downloading package", "err", err)
				return common.NewPrintableError("failed downloading package %s", fixedPackage.Descriptor())
			}

			slog.Debug("finished downloading package", "id", fixedPackage.Id())
			downloadResultsChannel <- shared.PackageDownload{Entry: descriptor, Data: data, ArtifactFileName: name}
		}
	}
}
func cleanWorkdir(fixer shared.DependencyFixer, err *error) {
	if *err == nil {
		slog.Debug("cleaning up original folders")
		cleanupOk := fixer.Cleanup()
		if !cleanupOk {
			// keep on fs for troubleshooting
			slog.Warn("cleanup failed")
		}
	} else {
		slog.Warn("rolling back installed fixes due to failure")
		if !fixer.Rollback() {
			slog.Error("failed rollback")
		}

	}

	// all sub folders should be restored due to rollback or by cleanup
}

func shouldSkipPackage(entry shared.DependencyDescriptor) bool {
	p := entry.VulnerablePackage
	packageId := p.Id()
	if len(p.OpenVulnerabilities) == 0 {
		slog.Warn("skipping, package has no open vulnerabilities", "id", packageId)
		return true
	}

	if !p.CanBeFixed() {
		slog.Debug("skipping, no fix available for package", "id", packageId)
		return true
	}

	if len(entry.Locations) == 0 {
		slog.Warn("skipping, package not found in discovered deps", "package", p)
		return true
	}

	return false
}

func (fp *fixPhase) fixPackage(downloadedPackage shared.PackageDownload, fixer shared.DependencyFixer) (error, []string) {
	var err error

	entry := downloadedPackage.Entry
	packageId := entry.VulnerablePackage.Id()
	packageDesc := entry.VulnerablePackage.Descriptor()

	fp.advanceStep(fmt.Sprintf("Fixing %s", packageDesc))

	fixedLocations := make([]string, 0, len(entry.Locations))
	for _, depInstance := range downloadedPackage.Entry.Locations {
		slog.Debug("fixing dependency instance", "id", packageId, "path", depInstance.DiskPath)

		var fixedPath string // can be empty; e.g. yum
		var fixed bool
		if fixed, fixedPath, err = fixer.Fix(entry, &depInstance, downloadedPackage.Data, downloadedPackage.ArtifactFileName); err != nil {
			return common.FallbackPrintableMsg(err, "failed applying fix to %s", packageDesc), nil
		}

		if fixed {
			slog.Info("fixed dependency instance", "id", packageId, "fixed-path", fixedPath, "orig-path", depInstance.DiskPath)
			fixedLocations = append(fixedLocations, fixedPath)
		}
	}

	return nil, fixedLocations
}

func (fp *fixPhase) HandleCallbacks(fixes []shared.DependencyDescriptor, vulnerable []api.PackageVersion, callbacks ...PostFixRunner) {
	defer fp.advanceStep("") // must mirror the minimum steps count for this command
	if len(callbacks) == 0 {
		slog.Debug("no callbacks to run")
		return
	}

	fp.addToMax(len(callbacks)) // increase max to accommodate fix logic in progress bar

	for _, callback := range callbacks {
		step := callback.GetStepDescription()
		fp.advanceStep(step)

		if callback.ShouldSkip() {
			slog.Debug("Skipping callback", "step", step)
			continue
		}

		slog.Debug("Running callback")
		if err := callback.HandleAppliedFixes(fp.BaseDir, fixes, vulnerable); err != nil {
			slog.Warn("callback failed", "err", err) // Failings here should show a warning, and not stop the process
		}
	}
}

func buildRemoteOverrideQuery(vulnerablePackages []api.PackageVersion) []api.RemoteOverrideQuery {
	queries := make([]api.RemoteOverrideQuery, 0, len(vulnerablePackages))
	for _, pkg := range vulnerablePackages {
		originId := pkg.OriginVersionId
		if originId == "" {
			originId = pkg.VersionId // this is an origin verison
		}

		recommendedId := pkg.RecommendedLibraryVersionId // using local to not reuse pointer
		if recommendedId == "" {
			// should always have a recommended version if a fix is applicable
			slog.Info("ignoring vulnerable package without recommendation", "origin", pkg.OriginVersionId, "version", pkg.Version, "library", pkg.Library.Name)
			continue
		}

		query := api.RemoteOverrideQuery{
			LibraryId:            pkg.Library.Id,
			OriginVersionId:      originId,
			RecommendedVersionId: &recommendedId,
		}

		queries = append(queries, query)
	}

	return queries
}

// query the BE for the recommended versions specified in the input vulnerable packages
func (fp *fixPhase) queryRemoteConfigPackages(vulnerablePackages []api.PackageVersion, project string) ([]api.PackageVersion, error) {
	queries := buildRemoteOverrideQuery(vulnerablePackages)

	fixes, err := fetchOverriddenPackagesInfo(fp.Backend, queries, nil)
	if err != nil {
		if errors.Is(err, api.RemoteOverrideDisabledError) {
			return []api.PackageVersion{}, err
		}
		slog.Error("failed getting fixed versions per remote config", "err", err, "project", fp.Project.Tag)
		return nil, common.FallbackPrintableMsg(err, "failed querying remote config")
	}

	slog.Debug("got fixes info", "count", len(*fixes))
	return *fixes, nil
}

// query the BE for silence rules, return a list of packageName:version
func (fp *fixPhase) QuerySilenceRules() ([]api.SilenceRule, error) {
	rules, err := fp.Backend.QuerySilenceRules()
	if err != nil {
		slog.Error("failed querying silence rules", "err", err)
		return nil, common.NewPrintableError("failed querying silence rules")
	}

	slog.Debug("got silence rules", "count", len(rules))
	return rules, nil
}

// combine fix + vulnerable + dependency information for same package
func buildDescriptorsForFixes(scanResult ScanResult, fixedPackages []api.PackageVersion, overrideMethod shared.OverriddenMethod, managerClass actions.ManagerClass) ([]shared.DependencyDescriptor, error) {
	// use a map from origin id to the new dependency descriptor struct, so we can update it with the server response
	descs := make(map[string][]*shared.DependencyDescriptor)
	for i := range scanResult.Vulnerable { // index since going to use pointer to the struct
		vulnerable := scanResult.Vulnerable[i]
		// When fixing OS managers, a dependency may require a fix in multiple archs,
		// so we duplicate the entries by architecture.
		// This is irrelevant for non-OS managers."
		if managerClass == actions.OsManager {
			populateDescriptorsWithDifferentArch(scanResult, vulnerable, descs)
		} else {
			populateDescriptorsWithDifferentLocation(scanResult, vulnerable, descs)
		}
	}
	availableFixes := make([]shared.DependencyDescriptor, 0, len(fixedPackages))
	for i := range fixedPackages { // index since going to use pointer to the struct
		pkg := fixedPackages[i]
		descArray, exists := descs[pkg.OriginId()]
		if !exists {
			slog.Warn("fixed package origin id was not found in vulnerable ids map", "origin", pkg.OriginId())
			continue
		}
		for _, desc := range descArray {
			desc.AvailableFix = &pkg
			desc.OverrideMethod = overrideMethod
			availableFixes = append(availableFixes, *desc)
		}
	}
	return availableFixes, nil
}

func populateDescriptorsWithDifferentLocation(scanResult ScanResult, vulnerable api.PackageVersion, descs map[string][]*shared.DependencyDescriptor) {
	locations := scanResult.AllDependencies[vulnerable.Id()]
	descriptor := shared.DependencyDescriptor{
		VulnerablePackage: &vulnerable,
		Locations:         make(map[string]common.Dependency),
		FixedLocations:    make([]string, 0, len(locations)),
		AvailableFix:      nil,
	}
	for _, loc := range locations {
		descriptor.Locations[loc.DiskPath] = *loc
	}
	descs[vulnerable.OriginId()] = append(descs[vulnerable.OriginId()], &descriptor)
}

func populateDescriptorsWithDifferentArch(scanResult ScanResult, vulnerable api.PackageVersion, descs map[string][]*shared.DependencyDescriptor) {
	for _, deps := range scanResult.AllDependencies[vulnerable.Id()] {
		descriptor := shared.DependencyDescriptor{
			VulnerablePackage: &vulnerable,
			Locations:         make(map[string]common.Dependency),
			FixedLocations:    make([]string, 0, 1),
			AvailableFix:      nil,
		}
		descriptor.Locations[deps.DiskPath] = *deps
		descs[vulnerable.OriginId()] = append(descs[vulnerable.OriginId()], &descriptor)
	}
}

// fetches the available fixes according the the fix mode
// either the recommended ones in the scan result(all is from server, local was patched to contain the actions file values), or remote config
func (fp *fixPhase) GetAvailableFixes(scanResult *ScanResult, mode FixMode) ([]shared.DependencyDescriptor, error) {

	var err error
	var fixedPackages []api.PackageVersion

	fp.advanceStep("Querying available fixes")
	slog.Info("getting fixes for discovered packages", "vulnerableCount", len(scanResult.Vulnerable))

	overrideMethod := shared.NotOverridden
	switch mode {
	case FixModeLocal:
		overrideMethod = shared.OverriddenFromLocal
		fallthrough // perform same request as no override
	case FixModeAll:
		fixedPackages, err = fp.QueryRecommendedPackages(scanResult.Vulnerable)
	case FixModeRemote:
		overrideMethod = shared.OverriddenFromRemote
		// fetch packages according to scan result's recommend
		// if local was used the scan result should already be updated
		fixedPackages, err = fp.queryRemoteConfigPackages(scanResult.Vulnerable, fp.Project.Tag)
	}

	if err != nil {
		slog.Error("failed querying fixes", "err", err)
		return nil, common.FallbackPrintableMsg(err, "failed querying fixes")
	}
	return buildDescriptorsForFixes(*scanResult, fixedPackages, overrideMethod, fp.Manager.Class())
}

func (fp *fixPhase) Fix(availableFixes []shared.DependencyDescriptor, skipSignChecks bool) (_ []shared.DependencyDescriptor, err error) {
	// assumes running from the directory of the project
	// 		relies on dependencies being installed beforehand (e.g. `npm install`)
	// returns a list of the fixed descriptors
	fixer := fp.Manager.GetFixer(fp.Workdir)
	if fixer == nil {
		// should not happend
		mn := fp.Manager.Name()
		slog.Error("manager returned nil fixer", "manager", mn)
		return nil, common.FallbackPrintableMsg(err, "fixing is not supported for %s", mn)
	}

	defer cleanWorkdir(fixer, &err) // will rollback if encountered error

	downloadResultsChannel := make(chan shared.PackageDownload, len(availableFixes))
	downloadJobsChannel := make(chan shared.DependencyDescriptor, len(availableFixes))
	g, ctx := errgroup.WithContext(context.Background())

	// start workers
	for i := 0; i < ConcurrentDownloadCount; i++ {
		g.Go(func() (err error) {
			return packageDownloadWorker(ctx, fp.ArtifactServer, fp.Manager, downloadJobsChannel, downloadResultsChannel)
		})
	}

	jobCount := 0
	// send download jobs
	for _, entry := range availableFixes {
		if shouldSkipPackage(entry) {
			continue
		}

		jobCount++
		downloadJobsChannel <- entry
	}

	close(downloadJobsChannel) // to signal workers to stop
	go func() {
		// wait for all workers to finish, then close the results channel to signal the main thread
		common.Trace("starting wait on downloader group")
		err := g.Wait()
		common.Trace("downloader group finished", "err", err)
		close(downloadResultsChannel)
	}()

	fp.Bar.Describe("Downloading packages")
	fp.addToMax(jobCount) // add steps here to bump the progress bar once

	// copy the results to an array
	results := make([]shared.PackageDownload, 0, len(availableFixes))
	for result := range downloadResultsChannel {
		results = append(results, result)
	}

	// verify seal signatures on the artifacts
	if !skipSignChecks {
		err := verifyPackagesSingatures(fp.Backend, results, fp.Config.UseSealedNames)
		if err != nil {
			slog.Error("failed validating signatures", "err", err)
			return nil, common.FallbackPrintableMsg(err, "failed validating package signatures")
		}
	}

	common.Trace("prepare phase started")
	if err := fixer.Prepare(); err != nil {
		slog.Error("failed preparing fixer", "err", err)
		return nil, common.FallbackPrintableMsg(err, "failed preparing environment")
	}
	common.Trace("prepare phase done")

	// Fix packages one at a time
	fixed := make([]shared.DependencyDescriptor, 0, len(availableFixes))
	for _, downloadedPackage := range results {
		err, fixedLocations := fp.fixPackage(downloadedPackage, fixer)
		if err != nil {
			slog.Error("failed fixing package", "err", err)
			return nil, err
		}

		if len(fixedLocations) > 0 {
			// update entry with fixed locations
			entry := downloadedPackage.Entry
			entry.FixedLocations = append(entry.FixedLocations, fixedLocations...)
			fixed = append(fixed, entry)
		}
	}

	// Handle errors from download workers
	if err := g.Wait(); err != nil {
		slog.Error("failed waiting for downloader group", "err", err)
		return nil, common.FallbackPrintableMsg(err, "failed downloading packages")
	}

	slog.Debug("finished downloading packages", "count", len(fixed))

	// handle fixes only after all packages are downloaded
	// this is to avoid partial fixes in case of failure
	if len(fixed) > 0 {
		slog.Debug("letting manager handle post fixes")
		if err := fp.Manager.HandleFixes(fixed); err != nil {
			slog.Error("manager failed to handle fixes", "err", err)
			return nil, err
		}
	}

	return fixed, nil
}
