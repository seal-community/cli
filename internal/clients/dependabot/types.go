package dependabot

type dependabotDependencyPackageComponent struct {
	Ecosystem string `json:"ecosystem"`
	Name      string `json:"name"`
}

type dependabotDependencyComponent struct {
	Package      dependabotDependencyPackageComponent `json:"package"`
	ManifestPath string                               `json:"manifest_path"`
}

type dependabotSecurityAdvisory struct {
	GHASId string `json:"ghsa_id"`
	CVEId  string `json:"cve_id"`
}

type dependabotVulnerableComponent struct {
	State            string                        `json:"state"`
	Dependency       dependabotDependencyComponent `json:"dependency"`
	SecurityAdvisory dependabotSecurityAdvisory    `json:"security_advisory"`
	Url              string                        `json:"url"`
	DismissedComment *string                       `json:"dismissed_comment"`
}

type dependabotVulnerableComponents []dependabotVulnerableComponent

type dependabotUpdateComponentVulnerabilityRemediation struct {
	State            string `json:"state"`                      // One of [open, dismissed]
	DismissedReason  string `json:"dismissed_reason,omitempty"` // One of [fix_started, inaccurate, no_bandwidth, not_used, tolerable_risk]
	DismissedComment string `json:"dismissed_comment,omitempty"`
}
