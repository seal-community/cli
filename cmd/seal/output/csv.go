package output

import (
	"cli/internal/api"
	"cli/internal/common"
	"encoding/csv"
	"fmt"
	"io"
	"log/slog"
	"strings"
)

type CsvExporter struct {
	Writer io.Writer
}

const csvSubTextSeparator = "|" // used to separate list within a csv field
const csvEmbedTestSeparator = "&"
const csvCanSealTrueValue = "TRUE"
const csvCanSealFalseValue = "FALSE"

func formatVulnCsv(vuln api.Vulnerability) string {
	if len(vuln.EmbeddedVia) == 0 {
		return vuln.PreferredId()
	}

	names := make([]string, 0, len(vuln.EmbeddedVia))
	for _, p := range vuln.EmbeddedVia {
		names = append(names, p.Name)
	}

	embeddingVerb := "shaded"

	vulnFormat := "%s(via %s %s)"

	return fmt.Sprintf(vulnFormat, vuln.PreferredId(), embeddingVerb, strings.Join(names, csvEmbedTestSeparator))

}

func (e CsvExporter) Handle(vulnerablePackages []api.PackageVersion, allDeps common.DependencyMap) error {
	w := csv.NewWriter(e.Writer)
	defer w.Flush()

	// write header
	headerParts := []string{
		headerLibrary,
		headerVersion,
		headerEcosystem,
		headerVulnerabilities,
		headerCanSeal,
		headerRecommendedVersion,
	}

	if err := w.Write(headerParts); err != nil {
		slog.Error("failed writing csv header line", "err", err)
		return err
	}

	for i, vulnPackage := range vulnerablePackages {
		if len(vulnPackage.OpenVulnerabilities) == 0 {
			// should not have been returned
			slog.Warn("skipping package, no open vulnerabilities",
				"manager", vulnPackage.Library.PackageManager,
				"library", vulnPackage.Library.Name,
				"version", vulnPackage.Version,
			)
			continue
		}

		hasSealed := csvCanSealFalseValue
		if vulnPackage.CanBeFixed() {
			hasSealed = csvCanSealTrueValue
		}

		// in csv we don't care how many vulnerability ids we have
		combinedIds := formatVulnCsv(vulnPackage.OpenVulnerabilities[0])
		// must have at least 1, so okay to slice
		for _, vulnerability := range vulnPackage.OpenVulnerabilities[1:] {
			combinedIds = strings.Join([]string{combinedIds, formatVulnCsv(vulnerability)}, csvSubTextSeparator)
		}

		lineParts := []string{
			vulnPackage.Library.Name,
			vulnPackage.Version,
			vulnPackage.Library.PackageManager,
			combinedIds,
			hasSealed,
			vulnPackage.RecommendedLibraryVersionString,
		}

		if err := w.Write(lineParts); err != nil {
			slog.Error("failed writing csv line", "line_idx", i, "err", err)
			return err
		}
	}

	return nil
}
