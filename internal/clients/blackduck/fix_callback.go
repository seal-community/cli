package blackduck

import (
	"cli/internal/api"
	"cli/internal/common"
	"cli/internal/config"
	"cli/internal/ecosystem/shared"
	"context"
	"log/slog"
	"strings"

	"golang.org/x/sync/errgroup"
)

const patchedStatus = "PATCHED"
const newStatus = "NEW"
const patchComment = "vulnerability patched by seal-security"

type vulnerabilityMapping map[string]bool

func parseKey(vals []string) string {
	return strings.ToLower(strings.Join(vals, "/")) // Has to be '/' because this is what BlackDuck using in the componentVersionOriginId field
}

func buildSealedVulnerabilitiesMapping(fixResults []api.PackageVersion) vulnerabilityMapping {
	mapping := make(vulnerabilityMapping)
	for _, fix := range fixResults {
		for _, vuln := range fix.SealedVulnerabilities {
			v := vuln.PreferredId()
			key := parseKey([]string{fix.Library.PackageManager, fix.Library.Name, fix.OriginVersion, v})
			mapping[key] = true
		}
	}

	slog.Debug("built sealed vulnerabilities mapping", "mapping", mapping)
	return mapping
}

func patchVulnInBlackDuck(c *BlackDuckClient, bdVuln bdVulnerableBOMComponent, fixMapping vulnerabilityMapping) error {
	pkgManager := bdVuln.ComponentVersionOriginName
	packageFullName := bdVuln.ComponentVersionOriginId
	vuln := bdVuln.VulnerabilityWithRemediation.VulnerabilityName
	slog.Debug("processing vulnerability", "packageManager", pkgManager, "packageFullName", packageFullName, "vuln", vuln)

	key := parseKey([]string{pkgManager, packageFullName, vuln})
	slog.Debug("checking if vulnerability is sealed", "key", key)
	if _, ok := fixMapping[key]; ok {
		// Patch the vulnerability signed by seal
		url := bdVuln.Meta.Href
		update := bdUpdateBOMComponentVulnerabilityRemediation{
			RemediationStatus: patchedStatus,
			Comment:           patchComment,
		}

		slog.Debug("patching vulnerability", "url", url, "update", update, "pkgManager", pkgManager, "packageFullName", packageFullName, "vuln", vuln)
		err := c.updateVuln(url, &update)
		if err != nil {
			return common.NewPrintableError("failed to update BlackDuck that %s was sealed for %s", bdVuln.ComponentVersionOriginId, vuln)
		}

		return nil
	}

	slog.Debug("vulnerability is not sealed", "pkgManager", pkgManager, "packageFullName", packageFullName, "vuln", vuln)
	if bdVuln.VulnerabilityWithRemediation.RemediationStatus == patchedStatus && bdVuln.VulnerabilityWithRemediation.Description == patchComment {
		// If the vulnerability signed by seal is not found in the fixMapping, unpatch the vulnerability
		url := bdVuln.Meta.Href
		update := bdUpdateBOMComponentVulnerabilityRemediation{
			RemediationStatus: newStatus,
			Comment:           "",
		}
		slog.Debug("unpatching vulnerability", "url", url, "update", update, "pkgManager", pkgManager, "packageFullName", packageFullName, "vuln", vuln)
		err := c.updateVuln(url, &update)
		if err != nil {
			return common.NewPrintableError("failed to update BlackDuck that %s is not sealed for %s", bdVuln.ComponentVersionOriginId, vuln)
		}

		return nil
	}

	return nil
}

func updateVulnerabilityWorker(ctx context.Context, c *BlackDuckClient, vulnerabilitiesChannel chan bdVulnerableBOMComponent, fixMapping vulnerabilityMapping) error {
	for {
		select {
		case <-ctx.Done():
			slog.Debug("update vulnerability worker cancelled")
			return nil
		case bdVuln, more := <-vulnerabilitiesChannel:
			if !more {
				slog.Debug("no more vulnerabilities to update")
				return nil
			}
			err := patchVulnInBlackDuck(c, bdVuln, fixMapping)
			if err != nil {
				slog.Warn("failed updating vulnerability", "err", err)
			}
		}
	}
}

type BlackDuckCallback struct {
	Config *config.Config
}

func handleAppliedFixes(bdProject string, c *BlackDuckClient, fixResults []api.PackageVersion) error {
	project, err := c.getProjectByName(bdProject)
	if err != nil {
		slog.Error("failed getting project", "err", err)
		return common.FallbackPrintableMsg(err, "Failed to update BlackDuck")
	}

	fixMapping := buildSealedVulnerabilitiesMapping(fixResults)
	vulnerabilitiesChannel := make(chan bdVulnerableBOMComponent, 10)
	g, ctx := errgroup.WithContext(context.Background())
	g.Go(func() error {
		return updateVulnerabilityWorker(ctx, c, vulnerabilitiesChannel, fixMapping)
	})

	err = c.getAllVulnsInProject(project, vulnerabilitiesChannel)
	if err != nil {
		return err
	}

	close(vulnerabilitiesChannel)

	if err := g.Wait(); err != nil {
		slog.Error("failed updating vulnerabilities", "err", err)
		return common.NewPrintableError("failed to update BlackDuck")
	}

	slog.Info("successfully updated BlackDuck")
	return nil
}

func (b *BlackDuckCallback) HandleAppliedFixes(projectDir string, fixes shared.FixMap, fixResults []api.PackageVersion) error {
	bdConfg := b.Config.BlackDuck
	c := NewClient(bdConfg)
	return handleAppliedFixes(bdConfg.Project, c, fixResults)
}

func (b *BlackDuckCallback) ShouldSkip() bool {
	bdConfg := b.Config.BlackDuck

	if bdConfg.Url == "" {
		slog.Debug("BlackDuck URL is not set in the configuration file")
		return true
	}

	if bdConfg.Token == "" {
		slog.Debug("BlackDuck token is not set in the configuration file")
		return true
	}

	if bdConfg.Project == "" {
		slog.Debug("BlackDuck project is not set in the configuration file")
		return true
	}

	if bdConfg.VersionName == "" {
		slog.Debug("BlackDuck version is not set in the configuration file")
		return true
	}

	return false
}

func (b *BlackDuckCallback) GetStepDescription() string {
	return "Updating BlackDuck"
}
