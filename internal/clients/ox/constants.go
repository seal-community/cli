package ox

const (
	CommentHeader                  = "Seal bot remediation notes:\n"
	CommentAllVulnerabilitiesFixed = "\nThe Seal bot excluded the issue since all existing vulnerabilities were remediated"
	CommentHighCriticalFixed       = "\nThe Seal bot excluded the issue since all high and critical vulnerabilities were remediated"
	CommentVulnerabilityFixed      = "%s was fixed in package %s by using version %s\n"
)
