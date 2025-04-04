package dependabot

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

const dismissedStatus = "dismissed"
const dismissedReason = "fix_started"
const openStatus = "open"
const dismissedComment = "vulnerability patched by seal-security"
const defaultGitHubUrl = "https://api.github.com"

type openVulnerabilityMapping map[string]map[string]bool // openVulnerabilities in a stucture of { "getOpenVulnerabilityKey()": { "version": true } }
type vulnerabilityMapping map[string]bool
type DependabotCallback struct {
	Config *config.Config
}

func parseKey(vals []string) string {
	return strings.ToLower(strings.Join(vals, "/"))
}

func getOpenVulnerabilityKey(vulnerability api.Vulnerability, library api.PackageVersion) string {
	return vulnerability.GitHubAdvisoryID + library.Library.Name
}

func getOpenAndSealedMap(fixes []shared.DependencyDescriptor, vulnerable []api.PackageVersion) openVulnerabilityMapping {
	// If there's a sealed vulnerability that is opened, we shouldn't dismiss the Dependabot alert
	// We use both GHSA (since Dependabot using GitHub Advisory) and Package name in case there are multiple packages with the same ID (rare)
	// Get all open vulns as true and sealed vulns as false, to be able to seal/unseal accordingly
	openVulnerabilitiesMap := make(openVulnerabilityMapping)
	for _, openEntry := range vulnerable {
		for _, openPackageVuln := range openEntry.OpenVulnerabilities {
			key := getOpenVulnerabilityKey(openPackageVuln, openEntry)
			if _, exists := openVulnerabilitiesMap[key]; !exists {
				openVulnerabilitiesMap[key] = make(map[string]bool)
			}
			slog.Debug("Adding to open vulns map", "package", openEntry.Library.Name, "ID", openPackageVuln.GitHubAdvisoryID, "version", openEntry.Version)
			openVulnerabilitiesMap[key][openEntry.Version] = true
		}
	}
	for _, fixEntry := range fixes {
		if fixEntry.AvailableFix != nil {
			for _, sealedPackageVuln := range fixEntry.AvailableFix.SealedVulnerabilities {
				slog.Info("Adding to sealed vulns map", "package", fixEntry.VulnerablePackage.Library.Name, "ID", sealedPackageVuln.GitHubAdvisoryID, "version", fixEntry.VulnerablePackage.Version)
				key := getOpenVulnerabilityKey(sealedPackageVuln, *fixEntry.VulnerablePackage)
				openVulnerabilitiesMap[key][fixEntry.VulnerablePackage.Version] = false
			}
		}
	}
	slog.Debug("All open & sealed vulns", "vulns", openVulnerabilitiesMap)
	return openVulnerabilitiesMap
}

func buildSealedVulnerabilitiesMapping(fixes []shared.DependencyDescriptor, vulnerable []api.PackageVersion) vulnerabilityMapping {
	// Creating a map of sealed vulnerabilities to later check if they are not found anymore and should be unsealed
	sealedVulnerabilitiesMapping := make(vulnerabilityMapping)
	openVulnerabilitiesMap := getOpenAndSealedMap(fixes, vulnerable)

	for _, entry := range fixes {
		if entry.AvailableFix != nil {
			fix := entry.AvailableFix
			for _, vuln := range fix.SealedVulnerabilities {
				if vuln.GitHubAdvisoryID != "" {
					vulnId := vuln.GitHubAdvisoryID
					update := true
					slog.Debug("Checking if vuln is open in openVulnerabilitiesMap", "ID", vulnId)
					key := getOpenVulnerabilityKey(vuln, *entry.VulnerablePackage)
					for _, openVersion := range openVulnerabilitiesMap[key] {
						if openVersion {
							update = false
							break
						}
					}
					if update {
						slog.Debug("Adding GitHub vulnerability to sealed packages map", "ID", vulnId)
						key := parseKey([]string{fix.Library.PackageManager, fix.Library.Name, vulnId})
						sealedVulnerabilitiesMapping[key] = true
					} else {
						slog.Debug("GitHub vulnerability is still open in another version. Will not close alert", "ID", vulnId)
					}
				} else {
					slog.Debug("GitHub vulnerability ID not found. Found only (", vuln.PreferredId(), "). Will skip from Dependabot update")
				}
			}
		}
	}
	slog.Debug("built sealed vulnerabilities mapping", "mapping", sealedVulnerabilitiesMapping)
	return sealedVulnerabilitiesMapping
}

func patchVulnInDependabot(c *DependabotClient, dependabotVuln dependabotVulnerableComponent, fixMapping vulnerabilityMapping) error {
	pkgManager := dependabotVuln.Dependency.Package.Ecosystem
	packageFullName := dependabotVuln.Dependency.Package.Name
	cveId := dependabotVuln.SecurityAdvisory.CVEId
	ghsaId := dependabotVuln.SecurityAdvisory.GHASId
	slog.Debug("processing vulnerability", "packageManager", pkgManager, "packageFullName", packageFullName, "vuln GitHub ID", ghsaId, "CVE ID", cveId)

	key := parseKey([]string{pkgManager, packageFullName, ghsaId})
	slog.Debug("checking if vulnerability is sealed", "key", key)
	if _, ok := fixMapping[key]; ok {
		// Patch the vulnerability signed by seal
		url := dependabotVuln.Url
		update := dependabotUpdateComponentVulnerabilityRemediation{
			State:            dismissedStatus,
			DismissedReason:  dismissedReason,
			DismissedComment: dismissedComment,
		}

		slog.Debug("patching vulnerability", "url", url, "update", update, "pkgManager", pkgManager, "packageFullName", packageFullName, "vuln GitHub ID", ghsaId, "CVE ID", cveId)
		err := c.updateVuln(url, &update)
		if err != nil {
			return common.NewPrintableError("failed to update Dependabot that %s was sealed for %s", packageFullName, ghsaId)
		}

		return nil
	}

	slog.Debug("vulnerability is not sealed", "pkgManager", pkgManager, "packageFullName", packageFullName, "vuln GitHub ID", ghsaId, "CVE ID", cveId)
	if dependabotVuln.State == dismissedStatus && dependabotVuln.DismissedComment != nil && *dependabotVuln.DismissedComment == dismissedComment {
		// If the vulnerability signed by seal is not found in the fixMapping, unpatch the vulnerability
		url := dependabotVuln.Url
		update := dependabotUpdateComponentVulnerabilityRemediation{
			State: openStatus,
		}
		slog.Debug("unpatching vulnerability", "url", url, "update", update, "pkgManager", pkgManager, "packageFullName", packageFullName, "vuln GitHub ID", ghsaId, "CVE ID", cveId)
		err := c.updateVuln(url, &update)
		if err != nil {
			return common.NewPrintableError("failed to update Dependabot that %s is not sealed for %s", packageFullName, ghsaId)
		}

		return nil
	}

	return nil
}

func updateVulnerabilityWorker(ctx context.Context, c *DependabotClient, vulnerabilitiesChannel chan dependabotVulnerableComponent, fixMapping vulnerabilityMapping) error {
	for {
		select {
		case <-ctx.Done():
			slog.Debug("update vulnerability worker cancelled")
			return nil
		case dependabotVuln, more := <-vulnerabilitiesChannel:
			if !more {
				slog.Debug("no more vulnerabilities to update")
				return nil
			}
			err := patchVulnInDependabot(c, dependabotVuln, fixMapping)
			if err != nil {
				slog.Warn("failed updating vulnerability", "err", err)
			}
		}
	}
}

func handleAppliedFixes(c *DependabotClient, fixes []shared.DependencyDescriptor, vulnerable []api.PackageVersion) error {
	fixMapping := buildSealedVulnerabilitiesMapping(fixes, vulnerable)
	vulnerabilitiesChannel := make(chan dependabotVulnerableComponent, 10)
	g, ctx := errgroup.WithContext(context.Background())
	g.Go(func() error {
		return updateVulnerabilityWorker(ctx, c, vulnerabilitiesChannel, fixMapping)
	})

	err := c.getAllVulnsInProject(vulnerabilitiesChannel)
	if err != nil {
		return err
	}

	close(vulnerabilitiesChannel)

	if err := g.Wait(); err != nil {
		slog.Error("failed updating vulnerabilities", "err", err)
		return common.NewPrintableError("failed to update Dependabot")
	}

	slog.Info("successfully updated Dependabot")
	return nil
}

func (b *DependabotCallback) HandleAppliedFixes(projectDir string, fixes []shared.DependencyDescriptor, vulnerable []api.PackageVersion) error {
	dependabotConfig := b.Config.Dependabot
	c := NewClient(dependabotConfig)
	return handleAppliedFixes(c, fixes, vulnerable)
}

func (b *DependabotCallback) ShouldSkip() bool {
	dependabotConfig := b.Config.Dependabot

	if dependabotConfig.Token == "" {
		slog.Debug("skipping dependabot", "reason", "Dependabot token not set")
		return true
	}

	if dependabotConfig.Owner == "" {
		slog.Debug("skipping dependabot", "reason", "Dependabot project not set")
		return true
	}

	if dependabotConfig.Repo == "" {
		slog.Debug("skipping dependabot", "reason", "Dependabot version not set")
		return true
	}

	return false
}

func (b *DependabotCallback) GetStepDescription() string {
	return "Updating Dependabot"
}
