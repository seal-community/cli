package ox

type GraphQLRequest struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables"`
}

type GraphQLError struct {
	Message    string                 `json:"message"`
	Locations  []GraphQLErrorLocation `json:"locations"`
	Path       []interface{}          `json:"path"`
	Extensions map[string]interface{} `json:"extensions"`
}
type GraphQLErrorLocation struct {
	Line   int `json:"line"`
	Column int `json:"column"`
}

type GetIssuesFilter struct {
	Condition string   `json:"condition"` // e.g., "AND"
	FieldName string   `json:"fieldName"` // e.g., "uniqueLibs" or "apps"
	Values    []string `json:"values"`    // e.g., ["axios@1.6.0"]
}

type GetIssuesInput struct {
	Offset             int               `json:"offset"`
	Limit              int               `json:"limit"`
	ConditionalFilters []GetIssuesFilter `json:"conditionalFilters,omitempty"`
}

type Severity string

const (
	severityLow      Severity = "1"
	severityMedium   Severity = "2"
	severityHigh     Severity = "3"
	severityCritical Severity = "4"
)

type ScaVulnerability struct {
	Cve        string   `json:"cve"`
	OxSeverity Severity `json:"oxSeverity"` // e.g., "1-4" LOW,MEDIUM,HIGH,CRITICAL
	LibName    string   `json:"libName"`
	LibVersion string   `json:"libVersion"`
}

type Issue struct {
	ID        string `json:"id"`
	IssueID   string `json:"issueId"`
	MainTitle string `json:"mainTitle"`
	Severity  string `json:"severity"`
	App       struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"app"`
	Category struct {
		Name       string `json:"name"`
		CategoryID int    `json:"categoryId"`
	} `json:"category"`
	ScaVulnerabilities []ScaVulnerability `json:"scaVulnerabilities"`
	Comment            string             `json:"comment"`
}

type GetIssuesResponse struct {
	Data struct {
		GetIssues struct {
			Issues              []Issue `json:"issues"`
			TotalIssues         int     `json:"totalIssues"`
			TotalFilteredIssues int     `json:"totalFilteredIssues"`
		} `json:"getIssues"`
	} `json:"data"`
	Errors []GraphQLError `json:"errors,omitempty"`
}

type ExcludeAlertInput struct {
	OxIssueID string `json:"oxIssueId"`
	Comment   string `json:"comment"`
}

type ExcludeBulkAlertsResponse struct {
	Data struct {
		ExcludeBulkAlerts []struct {
			TotalExclusions int `json:"totalExclusions"`
		} `json:"excludeBulkAlerts"`
	} `json:"data"`
	Errors []GraphQLError `json:"errors,omitempty"`
}

type VulnerabilityStats struct {
	TotalVulns    int
	FixedVulns    int
	TotalHighCrit int
	FixedHighCrit int
}

type ExcludedIssue struct {
	Issue  Issue
	Reason string
}

type AddCommentToIssueInput struct {
	IssueID string `json:"issueId"`
	Comment string `json:"comment"`
}

type AddCommentToIssueResponse struct {
	Data struct {
		AddCommentToIssue bool `json:"addCommentToIssue"`
	} `json:"data"`
	Errors []GraphQLError `json:"errors,omitempty"`
}
