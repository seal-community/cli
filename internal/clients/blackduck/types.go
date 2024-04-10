package blackduck

type bdVulnerabilityWithRemediation struct {
	VulnerabilityName          string   `json:"vulnerabilityName"`          // The general identifier of the vulnerability
	Description                string   `json:"description"`                // The description of the vulnerability
	VulnerabilityPublishedDate string   `json:"vulnerabilityPublishedDate"` // The date the vulnerability was published
	VulnerabilityUpdatedDate   string   `json:"vulnerabilityUpdatedDate"`   // The date/time the vulnerability was updated
	BaseScore                  float64  `json:"baseScore"`                  // Score summarizing the overall risk presented by the vulnerability
	OverallScore               float64  `json:"overallScore"`               // The overall score for the vulnerability, considering base and temporal scores
	ExploitabilitySubscore     float64  `json:"exploitabilitySubscore"`     // Sub-score detailing the risk presented by current exploit techniques or exploit availability
	ImpactSubscore             float64  `json:"impactSubscore"`             // Sub-score detailing the data exposure that occurs if the vulnerability is successfully exploited
	Source                     string   `json:"source"`                     // The vulnerability database/reporting authority this vulnerability originates from. Possible Values: [NVD, BDSA]
	Severity                   string   `json:"severity"`                   // The general level of risk severity presented by the vulnerability. Possible Values: [CRITICAL, HIGH, MEDIUM, LOW]
	RemediationStatus          string   `json:"remediationStatus"`          // The remediation status of the BOM component. Possible Values: [DUPLICATE, IGNORED, MITIGATED, NEEDS_REVIEW, NEW, PATCHED, REMEDIATION_COMPLETE, REMEDIATION_REQUIRED]
	CweId                      string   `json:"cweId"`                      // Identifier of the common weakness enumeration
	RemediationTargetAt        string   `json:"remediationTargetAt"`        // The targeted remediation date/time for the vulnerability
	RemediationActualAt        string   `json:"remediationActualAt"`        // The date/time the vulnerability was actually remediated
	RemediationCreatedAt       string   `json:"remediationCreatedAt"`       // The date/time the vulnerability remediation was created
	RemediationUpdatedAt       string   `json:"remediationUpdatedAt"`       // The date/time the vulnerability remediation was updated
	RemediationCreatedBy       string   `json:"remediationCreatedBy"`       // The username of the user who created the vulnerability remediation
	RemediationUpdatedBy       string   `json:"remediationUpdatedBy"`       // The username of the user who updated the vulnerability remediation
	RelatedVulnerability       string   `json:"relatedVulnerability"`       // The related vulnerability of the vulnerability found
	BdsaTags                   []string `json:"bdsaTags"`                   // The BDSA tags of the vulnerability found
}

type bdUpdateBOMComponentVulnerabilityRemediation struct {
	Comment           string `json:"comment"`           //  Pertinent comments associated with the vulnerability remediation
	RemediationStatus string `json:"remediationStatus"` // (Required) The remediation status of the BOM component. Possible Values: [DUPLICATE, IGNORED, MITIGATED, NEEDS_REVIEW, NEW, PATCHED, REMEDIATION_COMPLETE, REMEDIATION_REQUIRED]
}

type bdMeta struct {
	Href  string   `json:"href"`
	Allow []string `json:"allow"`
	Links []bdLink `json:"links"`
}

type bdLink struct {
	Rel  string `json:"rel"`
	Href string `json:"href"`
}

type bdLicense struct {
	Type     string `json:"type"` // The relation of contained licenses. Possible Values: [CONJUNCTIVE, DISJUNCTIVE]
	Licenses []struct {
		License              string        `json:"license"`  // URL to the representation of the license to assign
		Licenses             []interface{} `json:"licenses"` // Related licenses which follow the same structure as the license field
		Name                 string        `json:"name"`     // The name of the license
		LicenseFamilySummary struct {
			Name string `json:"name"` // The name of the license family
			Href string `json:"href"` // URL to the representation of the license family
		} `json:"licenseFamilySummary"` // A summary of the license family
	} `json:"licenses"` // Related licenses which follow the same structure as the license field
	LicenseDisplay string `json:"licenseDisplay"` // The display status of the collection of licenses
}
