package output

import (
	"cli/internal/api"
	"cli/internal/common"
	"cli/internal/snyk"
	"log/slog"
)

// adds new rules to snyk policy file (unless no changes found)
// currently there is no support for removing old entries
func EditSnykPolicyFile(policyFilePath string, vulnerable []api.PackageVersion, fixed []api.PackageVersion) (bool, error) {
	slog.Info("working on snyk policy file", "path", policyFilePath)
	recommendedToVulnerable := make(map[string]api.PackageVersion)
	addedRules := false

	// building map of fixes to their respective vulnerable package
	for _, vulnPackage := range vulnerable {
		if vulnPackage.RecommendedLibraryVersionString == "" {
			// could happen if we have 'fake' packages specificed in the actions file
			slog.Warn("empty recommended version, skipping", "vulnpackage", vulnPackage)
			continue
		}

		if _, exists := recommendedToVulnerable[vulnPackage.RecommendedId()]; exists {
			// should not happen as the inputs are already deduped
			slog.Warn("dup recommended version, skipping", "vulnpackage", vulnPackage)
			continue
		}

		recommendedToVulnerable[vulnPackage.RecommendedId()] = vulnPackage
	}

	f, err := common.OpenFile(policyFilePath)
	if err != nil {
		return false, common.WrapWithPrintable(err, "failed to open existing .snyk file %s", policyFilePath)
	}

	var pf *snyk.PolicyFile
	if f != nil {
		pf, err = snyk.LoadPolicy(f) // if the file is empty this will fail validation
		f.Close()                    // we want to write to it
	} else {
		pf, err = snyk.NewPolicy()
	}

	if err != nil {
		return false, common.FallbackPrintableMsg(err, "faild loading .snyk file")
	}

	for _, fixedPackage := range fixed {
		linkedVulnPackage, exist := recommendedToVulnerable[fixedPackage.Id()]
		if !exist {
			slog.Warn("fixed version not found in vulnerable", "id", fixedPackage.Id())
			continue
		}

		for _, vuln := range fixedPackage.SealedVulnerabilities {
			if vuln.SnykID != "" {
				slog.Debug("adding fixed vulerability to snyk policy", "issue", vuln.SnykID, "package", fixedPackage.Library.Name, "recommended version", fixedPackage.RecommendedLibraryVersionString)
				// IMPORTANT: using original version since snyk does not detect our fix
				if pf.AddRule(vuln.SnykID, linkedVulnPackage.Library.Name, linkedVulnPackage.Version) { // using the vuln package for the rule since snyk is not aware of our fixed changes on disk
					addedRules = true // will not edit the file if we have nothing to change
				}
			}
		}
	}

	if addedRules {
		slog.Info("have new rules for snyk policy file")
		f, err = common.CreateFile(policyFilePath)
		if err != nil {
			return false, common.WrapWithPrintable(err, "failed to create .snyk file: %s", policyFilePath)
		}

		defer f.Close()

		if err = snyk.SavePolicy(pf, f); err != nil {
			slog.Error("failed dumping updated policy", "err", err)
			return false, common.NewPrintableError("failed to save .snyk file")
		}
	}

	return addedRules, nil
}
