package scanners

import (
	"cli/internal/api"
	"log/slog"
)

// Builds map of fixes to their respective vulnerable package
func buildRecommendedToVulnMap(vulnerable []api.PackageVersion) map[string]api.PackageVersion {
	recommendedToVulnerable := make(map[string]api.PackageVersion)

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

	return recommendedToVulnerable
}
