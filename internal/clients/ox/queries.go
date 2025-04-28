package ox

const (
	GetIssuesQuery = `
	query GetIssues($getIssuesInput: IssuesInput) {
		getIssues(getIssuesInput: $getIssuesInput) {
			issues {
				id
				issueId
				mainTitle
				severity
				app { id name }
				category { name categoryId }
				scaVulnerabilities {
					cve
					oxSeverity
					libName
					libVersion
				}
				comment
			}
			totalIssues
			totalFilteredIssues
		}
	}`

	ExcludeBulkAlertsMutation = `
	mutation ExcludeBulkAlerts($input: [ExcludeAlertInput!]!) {
		excludeBulkAlerts(input: $input) {
			totalExclusions
		}
	}`

	AddCommentToIssueMutation = `
	mutation AddCommentToIssue($input: addCommentToIssueInput!) {
		addCommentToIssue(input: $input)
	}`
)
