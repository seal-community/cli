package blackduck

import (
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
	return strings.ToLower(strings.Join(vals, "|"))
}

func buildSealedVulnerabilitiesMapping(fixes shared.FixMap) vulnerabilityMapping {
	mapping := make(vulnerabilityMapping)
	for _, fix := range fixes {
		packageName := fix.Package.Library.Name
		packageVersion := fix.Package.Version
		packageManager := fix.Package.Library.PackageManager

		for _, vuln := range fix.Package.SealedVulnerabilities {
			v := vuln.PreferredId()
			key := parseKey([]string{packageName, packageVersion, packageManager, v})
			mapping[key] = true
		}
	}

	return mapping
}

func patchVulnInBlackDuck(c *BlackDuckClient, bdVuln bdVulnerableBOMComponent, fixMapping vulnerabilityMapping) error {
	pkgManager := bdVuln.ComponentVersionOriginName
	pkgName := bdVuln.ComponentName
	pkgVersion := bdVuln.ComponentVersionName
	vuln := bdVuln.VulnerabilityWithRemediation.VulnerabilityName

	key := parseKey([]string{pkgName, pkgVersion, pkgManager, vuln})

	if _, ok := fixMapping[key]; ok {
		// Patch the vulnerability signed by seal
		url := bdVuln.Meta.Href
		update := bdUpdateBOMComponentVulnerabilityRemediation{
			RemediationStatus: patchedStatus,
			Comment:           patchComment,
		}

		slog.Debug("patching vulnerability", "url", url, "update", update, "packageManager", pkgManager, "packageName", pkgName, "packageVersion", pkgVersion, "vulnerability", vuln)
		err := c.updateVuln(url, update)
		if err != nil {
			return common.NewPrintableError("failed to update BlackDuck that %s was sealed for %s", bdVuln.ComponentVersionOriginId, vuln)
		}

		return nil
	}

	if bdVuln.VulnerabilityWithRemediation.RemediationStatus == patchedStatus && bdVuln.VulnerabilityWithRemediation.Description == patchComment {
		// If the vulnerability signed by seal is not found in the fixMapping, unpatch the vulnerability
		url := bdVuln.Meta.Href
		update := bdUpdateBOMComponentVulnerabilityRemediation{
			RemediationStatus: newStatus,
			Comment:           "",
		}
		slog.Debug("unpatching vulnerability", "url", url, "update", update, "packageManager", pkgManager, "packageName", pkgName, "packageVersion", pkgVersion, "vulnerability", vuln)
		err := c.updateVuln(url, update)
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

func handleAppliedFixes(bdProject string, c *BlackDuckClient, fixes shared.FixMap) error {
	project, err := c.getProjectByName(bdProject)
	if err != nil {
		slog.Error("failed getting project", "err", err)
		return common.FallbackPrintableMsg(err, "Failed to update BlackDuck")
	}

	fixMapping := buildSealedVulnerabilitiesMapping(fixes)
	vulnerabilitiesChannel := make(chan bdVulnerableBOMComponent, 10)
	g, ctx := errgroup.WithContext(context.Background())
	for i := 0; i < 10; i++ {
		g.Go(func() error {
			return updateVulnerabilityWorker(ctx, c, vulnerabilitiesChannel, fixMapping)
		})
	}

	err = c.getAllVulnsInProject(project, vulnerabilitiesChannel)
	if err != nil {
		return err
	}

	close(vulnerabilitiesChannel)

	if err := g.Wait(); err != nil {
		slog.Error("failed updating vulnerabilities", "err", err)
		return common.NewPrintableError("failed to update BlackDuck")
	}

	return nil
}

func (b *BlackDuckCallback) HandleAppliedFixes(projectDir string, fixes shared.FixMap) error {
	bdConfg := b.Config.BlackDuck
	c := NewClient(bdConfg.Url, bdConfg.Token)
	return handleAppliedFixes(bdConfg.Project, c, fixes)
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

	return false
}

func (b *BlackDuckCallback) GetStepDescription() string {
	return "Updating BlackDuck"
}