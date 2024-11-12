package output

import (
	"cli/internal/api"
	"cli/internal/common"
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
)

const falseCanSealValue = "X "
const maliciousWarningSign = "⚠ "

var trueCanSealValue string

func init() {
	// using a space to put them in the center
	if runtime.GOOS == "windows" {
		trueCanSealValue = "v "
	} else {
		trueCanSealValue = "✔ "
	}
}

type ConsolePrinter struct{}

func formatVuln(vulnerability api.Vulnerability) string {
	id := vulnerability.PreferredId()
	score := vulnerability.UnifiedScore
	var color common.AnsiColor

	formattedScore := ""

	if score != 0 {
		formattedScore = fmt.Sprintf("(%.1f)", score)
	}

	switch {
	case score < 5:
		color = common.AnsiLightGrey
	case score < 9:
		color = common.AnsiOrange
	default:
		color = common.AnsiLightRed
	}

	return fmt.Sprintf("%s %s", common.Colorize(id, color), formattedScore)
}

func (p ConsolePrinter) Handle(vulnerablePackages []api.PackageVersion, allDeps common.DependencyMap) error {
	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)

	t.AppendHeader(table.Row{
		headerLibrary,
		headerVersion,
		headerEcosystem,
		headerVulnerabilities,
		headerCanSeal,
		headerRecommendedVersion,
	})

	t.SetColumnConfigs([]table.ColumnConfig{
		{Align: text.AlignCenter, Name: "Can Seal"},
	})

	for _, vulnPackage := range vulnerablePackages {
		common.Trace("vulnerable package", "package", vulnPackage)
		if len(vulnPackage.OpenVulnerabilities) == 0 {
			// should not have been returned
			slog.Warn("skipping package, no open vulnerabilities",
				"manager", vulnPackage.Library.PackageManager,
				"library", vulnPackage.Library.Name,
				"version", vulnPackage.Version,
			)
			continue
		}

		var hasSealed string
		if vulnPackage.CanBeFixed() {
			hasSealed = common.Colorize(trueCanSealValue, common.AnsiBrightGreen)
		} else {
			hasSealed = common.Colorize(falseCanSealValue, common.AnsiNiceRed)
		}

		var maliciousSign string = ""
		if vulnPackage.IsMalicious() {
			maliciousSign = common.Colorize(maliciousWarningSign, common.AnsiNiceRed)
		}

		t.AppendRow([]interface{}{
			maliciousSign + vulnPackage.Library.Name,
			vulnPackage.Version,
			strings.Title(vulnPackage.Ecosystem()),
			formatVuln(vulnPackage.OpenVulnerabilities[0]),
			hasSealed,
			vulnPackage.RecommendedLibraryVersionString,
		})

		// IMPORTANT: we might want to limit the max numver of CVEs shown in the future
		for _, vulnerability := range vulnPackage.OpenVulnerabilities[1:] {
			t.AppendRow([]interface{}{
				"",
				"",
				"",
				formatVuln(vulnerability),
				"",
				"",
			})
		}

		t.AppendSeparator()
	}

	t.Render() // prints to stdout

	return nil
}
