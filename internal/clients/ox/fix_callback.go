package ox

import (
	"cli/internal/api"
	"cli/internal/common"
	"cli/internal/config"
	"cli/internal/ecosystem/shared"
	"fmt"
	"log/slog"
	"strings"
)

const (
	defaultLimit  = 100
	maxIssueCount = 100000
)

type vulnerabilityMapping map[string]string
type OxCallback struct {
	Config *config.Config
}

func parseKey(library string, version string, cve string) string {
	// don't support filtering by language/ecosystem yet
	return strings.ToLower(strings.Join([]string{library, version, cve}, "/"))
}

// Creates a mapping where each key is a vulnerability identifier and each value is the corresponding fixed version
// Example output:
//
//	{
//	  "lodash/4.17.15/cve-2021-23337": "4.17.15+sp1",
//	  "axios/0.21.1/cve-2021-3749": "0.21.1+sp1",
//	  "log4j/2.14.1/cve-2021-44228": "2.17.1+sp2"
//	}
func buildSealedVulnerabilitiesMapping(fixes []shared.DependencyDescriptor) vulnerabilityMapping {
	mapping := make(vulnerabilityMapping)
	for _, entry := range fixes {
		fix := entry.AvailableFix
		for _, vuln := range fix.SealedVulnerabilities {
			// Filter out non-CVE vulnerabilities (ox only support cve)
			if vuln.CVE == "" {
				common.Trace("Skipping non-CVE vulnerability", "fix", fix)
				continue
			}
			key := parseKey(fix.Library.Name, fix.OriginVersionString, vuln.CVE)
			mapping[key] = fix.Version
		}
	}

	return mapping
}

func getIssuesForVulnerableLibs(c *OxClient, vulnerableLibs []string) ([]Issue, error) {
	var allIssues []Issue
	offset := 0

	for {
		if len(allIssues) >= maxIssueCount {
			return nil, fmt.Errorf("exceeded maximum issue count of %d", maxIssueCount)
		}

		input := GetIssuesInput{
			Offset: offset,
			Limit:  defaultLimit,
			ConditionalFilters: []GetIssuesFilter{
				{
					Condition: "OR",
					FieldName: "uniqueLibs",
					Values:    vulnerableLibs,
				},
				{
					Condition: "AND",
					FieldName: "apps",
					Values:    []string{c.Application},
				},
			},
		}

		batchIssues, err := c.GetIssues(input)
		if err != nil {
			slog.Error("Failed to get application issues", "err", err, "offset", offset)
			return nil, fmt.Errorf("failed to get issues at offset %d: %w", offset, err)
		}

		issues := batchIssues.Data.GetIssues.Issues
		allIssues = append(allIssues, issues...)

		slog.Debug("Fetched issues page", "count", len(issues), "offset", offset, "total_so_far", len(allIssues))
		if len(issues) < defaultLimit {
			common.Trace("Reached end of issues", "offset", offset, "total_so_far", len(allIssues))
			break
		}
		offset += defaultLimit
	}

	return allIssues, nil
}

// builds a list of vulnerable libraries in the format "libraryName@version"
func buildVulnerableLibrariesList(fixes []shared.DependencyDescriptor) []string {
	vulnerableLibsList := make([]string, 0, len(fixes))
	for _, fix := range fixes {
		// don't support filtering by language/ecosystem yet
		libIdentifier := fmt.Sprintf("%s@%s", fix.VulnerablePackage.Library.Name, fix.VulnerablePackage.Version)
		vulnerableLibsList = append(vulnerableLibsList, libIdentifier)
	}

	return vulnerableLibsList
}

// processIssueVulnerabilities analyzes an issue's vulnerabilities and builds a comment
func processIssueVulnerabilities(issue Issue, fixedVulns vulnerabilityMapping, excludeHighCritical bool) (isIssueToExclude bool, comment string) {
	var commentBuilder strings.Builder
	commentBuilder.WriteString(CommentHeader)

	allFixed := true
	allHighCriticalFixed := true

	for _, vuln := range issue.ScaVulnerabilities {
		key := parseKey(vuln.LibName, vuln.LibVersion, vuln.Cve)
		isHighCritical := vuln.OxSeverity == severityHigh || vuln.OxSeverity == severityCritical
		fixedVersion := fixedVulns[key]

		if fixedVersion == "" {
			allFixed = false
			if isHighCritical {
				allHighCriticalFixed = false
			}
		} else {
			commentBuilder.WriteString(fmt.Sprintf(CommentVulnerabilityFixed, vuln.Cve, vuln.LibName, fixedVersion))
		}
	}

	if allFixed {
		commentBuilder.WriteString(CommentAllVulnerabilitiesFixed)
		slog.Debug("Issue marked for exclusion - all vulnerabilities fixed",
			"issueId", issue.IssueID)
	} else if allHighCriticalFixed && excludeHighCritical {
		commentBuilder.WriteString(CommentHighCriticalFixed)
		slog.Debug("Issue marked for exclusion - high/critical vulnerabilities fixed",
			"issueId", issue.IssueID)
	}

	isIssueToExclude = allFixed || (excludeHighCritical && allHighCriticalFixed)
	return isIssueToExclude, commentBuilder.String()
}

// handles the processing and exclusion of issues based on fixed vulnerabilities
func processAndExcludeIssues(c *OxClient, issues []Issue, fixedVulns vulnerabilityMapping, excludeHighCritical bool) []ExcludedIssue {
	var issuesToExclude []ExcludedIssue

	for _, issue := range issues {
		if len(issue.ScaVulnerabilities) == 0 {
			continue
		}

		isIssueToExclude, comment := processIssueVulnerabilities(issue, fixedVulns, excludeHighCritical)

		if isIssueToExclude {
			issuesToExclude = append(issuesToExclude, ExcludedIssue{
				Issue:  issue,
				Reason: comment,
			})
		} else if len(comment) > 0 {
			if err := c.AddCommentToIssue(issue, comment); err != nil {
				slog.Error("Failed to add comment to issue", "err", err, "issueId", issue.IssueID)
			}
		}
	}

	return issuesToExclude
}

func handleAppliedFixes(c *OxClient, fixes []shared.DependencyDescriptor, _ []api.PackageVersion) error {
	vulnerableLibs := buildVulnerableLibrariesList(fixes)
	fixedVulns := buildSealedVulnerabilitiesMapping(fixes)

	issues, err := getIssuesForVulnerableLibs(c, vulnerableLibs)
	if err != nil {
		return fmt.Errorf("failed to get relevant issues: %w", err)
	}
	slog.Info("Retrieved issues from OX", "count", len(issues))

	issuesToExclude := processAndExcludeIssues(c, issues, fixedVulns, c.ExcludeWhenHighCriticalFixed)

	if len(issuesToExclude) > 0 {
		slog.Info("Excluding resolved issues", "count", len(issuesToExclude))
		if err := c.ExcludeIssues(issuesToExclude); err != nil {
			return fmt.Errorf("failed to exclude resolved issues: %w", err)
		}
	}

	return nil
}

func (b *OxCallback) HandleAppliedFixes(projectDir string, fixes []shared.DependencyDescriptor, vulnerable []api.PackageVersion) error {
	oxConfg := b.Config.Ox
	c := NewClient(oxConfg)
	return handleAppliedFixes(c, fixes, vulnerable)
}

func (b *OxCallback) ShouldSkip() bool {
	oxConfg := b.Config.Ox

	if oxConfg.Url == "" {
		slog.Warn("skipping ox", "reason", "OX URL not set")
		return true
	}

	if oxConfg.Token == "" {
		slog.Warn("skipping ox", "reason", "OX token not set")
		return true
	}

	if oxConfg.Application == "" {
		slog.Warn("skipping ox", "reason", "OX application not set")
		return true
	}

	return false
}

func (b *OxCallback) GetStepDescription() string {
	return "Updating OX"
}
