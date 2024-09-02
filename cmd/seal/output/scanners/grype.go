package scanners

import (
	"cli/internal/api"
	"cli/internal/common"
	"cli/internal/grype"
	"log/slog"
)

// adds new rules to grype policy file (unless no changes found)
// currently there is no support for removing old entries
// entries reference the origin version of the package, since that's what grype sees
func EditGrypePolicyFile(policyFilePath string, vulnerable []api.PackageVersion, fixed []api.PackageVersion) (bool, error) {
	slog.Info("working on grype policy file", "path", policyFilePath)
	recommendedToVulnerable := buildRecommendedToVulnMap(vulnerable)
	addedRules := false

	f, err := common.OpenFile(policyFilePath)
	if err != nil {
		return false, common.WrapWithPrintable(err, "failed to open existing file %s", policyFilePath)
	}

	var pf *grype.PolicyFile
	if f != nil {
		pf, err = grype.LoadPolicy(f) // if the file is empty this will fail validation
		f.Close()                     // we want to write to it
	} else {
		pf, err = grype.NewPolicy()
	}

	if err != nil {
		return false, common.FallbackPrintableMsg(err, "faild loading file %s", policyFilePath)
	}

	for _, fixedPackage := range fixed {
		linkedVulnPackage, exist := recommendedToVulnerable[fixedPackage.Id()]
		if !exist {
			slog.Warn("fixed version not found in vulnerable", "id", fixedPackage.Id())
			continue
		}

		for _, vuln := range fixedPackage.SealedVulnerabilities {
			// add entry for each id supported by grype
			// grype doesn't match CVE to GHSA, so we need to add both
			grypeSupportedIds := []string{vuln.CVE, vuln.GitHubAdvisoryID}
			for _, vulnId := range grypeSupportedIds {
				if vulnId != "" {
					slog.Debug("adding fixed vulerability to grype policy", "vuln", vulnId, "package", fixedPackage.Library.Name, "recommended version", fixedPackage.RecommendedLibraryVersionString)
					if pf.AddRule(vulnId, linkedVulnPackage.Library.Name, linkedVulnPackage.Version, linkedVulnPackage.Library.PackageManager) {
						addedRules = true
					}
				}
			}
		}
	}

	if addedRules {
		slog.Info("have new rules for grype policy file")
		f, err = common.CreateFile(policyFilePath)
		if err != nil {
			return false, common.WrapWithPrintable(err, "failed to create file: %s", policyFilePath)
		}

		defer f.Close()

		if err = grype.SavePolicy(pf, f); err != nil {
			slog.Error("failed dumping updated policy", "err", err)
			return false, common.NewPrintableError("failed to save file %s", policyFilePath)
		}
	}

	return addedRules, nil
}
