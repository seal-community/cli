package phase

import (
	"cli/internal/api"
	"cli/internal/common"
	"cli/internal/ecosystem/shared"
	"context"
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

func NewFixPhase(projectDir string, showProgress bool) (*fixPhase, error) {
	sp, err := NewScanPhase(projectDir, showProgress)
	if err != nil {
		return nil, err
	}

	sp.addToMax(FixSteps) // increase max to accomodate fix logic in progress bar
	fp := &fixPhase{
		scanPhase: sp,
	}

	// this phase requires authentication - must have valid project name
	proj := fp.Config.Project
	if reason := validateProjectName(proj); reason != "" {
		slog.Error("invalid projcet name", "name", proj, "project-dir", projectDir)
		return nil, common.NewPrintableError("invalid project name `%s` - %s", proj, reason)
	}

	return fp, nil
}

type PackageDownload struct {
	packageVersion *api.PackageVersion
	data           []byte
}

type FixReporter interface {
	Report(shared.FixMap)
}

func packageDownloadWorker(ctx context.Context, server api.Server, manager shared.PackageManager, downloadJobsChannel chan api.PackageVersion, downloadResultsChannel chan PackageDownload) (err error) {
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

		case toDownload, more := <-downloadJobsChannel:
			if !more {
				slog.Debug("download worker finished")
				return nil
			}

			data, err := manager.DownloadPackage(server, toDownload)
			if err != nil {
				slog.Error("failed downloading package", "err", err)
				return common.NewPrintableError("failed downloading package %s", toDownload.RecommendedDescriptor())
			}

			downloadResultsChannel <- PackageDownload{packageVersion: &toDownload, data: data}
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

func (fp *fixPhase) Authenticate() error {
	fp.Bar.Describe("Checking authentication")
	err := fp.Server.CheckAuthenticationValid()
	_ = fp.Bar.Add(1)

	return err
}

func addFixToMap(summary shared.FixMap, p *api.PackageVersion, localPath string) {
	fixKey := shared.FormatFixKey(p)
	entry, exists := summary[fixKey]
	if !exists {
		entry = &shared.FixedEntry{Package: p, Paths: make(map[string]bool)}
		summary[fixKey] = entry
	}

	entry.Paths[localPath] = true
}

func shouldSkipPackage(p api.PackageVersion, allDeps common.DependencyMap) bool {
	packageId := p.Id()
	if len(p.OpenVulnerabilities) == 0 {
		slog.Warn("package has no open vulnerabilities", "id", packageId)
		return true
	}

	if !p.CanBeFixed() {
		slog.Debug("no fix available for package", "id", packageId)
		return true
	}

	if _, ok := allDeps[packageId]; !ok {
		slog.Warn("package not found in discovered deps", "package", p)
		return true
	}

	return false
}

func (fp *fixPhase) fixPackage(summary shared.FixMap, downloadedPackage PackageDownload, allDeps common.DependencyMap, fixer shared.DependencyFixer) error {
	var err error
	packageId := downloadedPackage.packageVersion.Id()
	packageDesc := downloadedPackage.packageVersion.Descriptor()
	fp.advanceStep(fmt.Sprintf("Fixing %s", packageDesc))

	for _, depInstance := range allDeps[packageId] {
		slog.Debug("fixing dependency instance", "id", packageId, "path", depInstance.DiskPath)

		var done bool
		if done, err = fixer.Fix(depInstance, downloadedPackage.data); err != nil {
			return common.FallbackPrintableMsg(err, "failed applying fix to %s", packageDesc)
		}

		if done {
			// mapping between downloaded-fixed to dependencies works now since they have the same Id, if we have a 'new' name for the fixed version this needs to be updated
			addFixToMap(summary, downloadedPackage.packageVersion, depInstance.DiskPath)
			slog.Info("finished fixing instance", "id", packageId, "path", depInstance.DiskPath)
		}
	}

	return nil
}

func (fp *fixPhase) Fix(scanResult *ScanResult) (_ shared.FixMap, err error) {
	// currently will only work for node, and assumes running from the directory of the project
	// 		relies on dependencies being installed beforehand (e.g. `npm install`)
	fixer := fp.Manager.GetFixer(fp.ProjectDir, fp.Workdir)
	defer cleanWorkdir(fixer, &err) // will rollback if encountered error

	vulnerablePackages := scanResult.Vulnerable
	allDeps := scanResult.AllDependencies

	downloadResultsChannel := make(chan PackageDownload, len(vulnerablePackages))
	downloadJobsChannel := make(chan api.PackageVersion, len(vulnerablePackages))
	g, ctx := errgroup.WithContext(context.Background())

	// start workers
	for i := 0; i < ConcurrentDownloadCount; i++ {
		g.Go(func() (err error) {
			return packageDownloadWorker(ctx, fp.Server, fp.Manager, downloadJobsChannel, downloadResultsChannel)
		})
	}

	jobCount := 0
	// send download jobs
	for _, vulnPackage := range vulnerablePackages {
		if shouldSkipPackage(vulnPackage, allDeps) {
			continue
		}

		jobCount++
		downloadJobsChannel <- vulnPackage
	}

	close(downloadJobsChannel) // to signal workers to stop
	go func() {
		// wait for all workers to finish, then close the results channel to signal the main thread
		_ = g.Wait()
		close(downloadResultsChannel)
	}()

	fp.Bar.Describe("Downloading packages")
	fp.addToMax(jobCount) // add steps here to bump the progress bar once

	// Fix packages one at a time
	summary := make(shared.FixMap)
	for downloadedPackage := range downloadResultsChannel {
		if err = fp.fixPackage(summary, downloadedPackage, allDeps, fixer); err != nil {
			return nil, err
		}
	}

	if len(summary) > 0 {
		if err := fp.Manager.HandleFixes(fp.ProjectDir, summary); err != nil {
			slog.Error("manager failed to handle fixes", "err", err)
			return nil, err
		}
	}

	slog.Debug("finished downloading packages")

	// Handle errors from download workers
	if err := g.Wait(); err != nil {
		return nil, common.FallbackPrintableMsg(err, "failed downloading packages")
	}

	fp.advanceStep("") // must mirror the minimum steps count for this command
	return summary, nil
}
