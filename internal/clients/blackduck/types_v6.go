package blackduck

type bdProjects struct {
	TotalCount     int           `json:"totalCount"`
	Items          []bdProject   `json:"items"`
	AppliedFilters []interface{} `json:"appliedFilters"`
	Meta           bdMeta        `json:"_meta"`
}

type bdProject struct {
	Name                    string   `json:"name"`                    // (Required) The general identifier of the project
	Description             string   `json:"description"`             // (Required) Summary of what the project represents in terms of functionality and use
	ProjectOwner            string   `json:"projectOwner"`            // URL for the resource representing the project owner
	ProjectTier             int      `json:"projectTier"`             // (Required) The level of exposure the project has to potential attackers. Possible Values: [0, 1, 2, 3, 4, 5]
	ProjectLevelAdjustments bool     `json:"projectLevelAdjustments"` // (Required) Whether BOM level adjustments are applied at the project level (to all releases)
	CloneCategories         []string `json:"cloneCategories"`         // The set of categories of data to clone when creating a new project version. Possible Values: [COMPONENT_DATA, VULN_DATA, LICENSE_TERM_FULFILLMENT, CUSTOM_FIELD_DATA, VERSION_SETTINGS, DEEP_LICENSE]
	CustomSignatureEnabled  bool     `json:"customSignatureEnabled"`  // If the project should be matched against for future scans

	// CustomSignatureDepth is int in the docs and string at runtime
	CustomSignatureDepth          int    `json:"customSignatureDepth,string"`   // The match depth when custom signature is enabled for the project
	SnippetAdjustmentApplied      bool   `json:"snippetAdjustmentApplied"`      // (Required) Whether snippet adjustment for partial snippet match is applied to full file snippet match
	ProjectGroup                  string `json:"projectGroup"`                  // The project group link for the project. If no group is provided, the project will be assigned to the root project group
	Repository                    string `json:"repository"`                    // The URL to this project SCM repository
	RepositoryName                string `json:"repositoryName"`                // The name of this project SCM repository
	RepositoryGroupId             string `json:"repositoryGroupId"`             // The Repository GroupId of this project SCM repository
	UnmatchedFileRetentionEnabled bool   `json:"unmatchedFileRetentionEnabled"` // Whether unmatched files are retained for all releases of the project. If unspecified, the global setting is used.
	PurgeUnmatchedFilesEnabled    string `json:"purgeUnmatchedFilesEnabled"`    // Whether existing unmatched files are purged for all releases of the project. If unspecified, the global setting is used.
	CreatedAt                     string `json:"createdAt"`                     // (Required) The date/time when the project was created
	CreatedBy                     string `json:"createdBy"`                     // (Required) The username who created the project
	CreatedByUser                 string `json:"createdByUser"`                 // (Required) URL for the resource representing the project creator
	UpdatedAt                     string `json:"updatedAt"`                     // (Required) The date/time when the project settings were last updated
	UpdatedBy                     string `json:"updatedBy"`                     // (Required) The username who updated the project setting
	UpdatedByUser                 string `json:"updatedByUser"`                 // (Required) URL for the resource representing the project last editor
	Source                        string `json:"source"`
	Meta                          bdMeta `json:"_meta"` // (Required) Meta data associated with the representation
}

type bdVulnerableBOMComponents struct {
	TotalCount     int                        `json:"totalCount"`
	Items          []bdVulnerableBOMComponent `json:"items"`
	AppliedFilters []interface{}              `json:"appliedFilters"`
	Meta           bdMeta                     `json:"_meta"`
}

type bdVulnerableBOMComponent struct {
	ComponentVersion             string                         `json:"componentVersion"`             // URL to the representation of the component version
	ComponentName                string                         `json:"componentName"`                // The name of the component
	ComponentVersionName         string                         `json:"componentVersionName"`         // The name of the component version
	ComponentVersionOriginName   string                         `json:"componentVersionOriginName"`   // The name of the component origin
	ComponentVersionOriginId     string                         `json:"componentVersionOriginId"`     // The ID of the component origin
	Ignored                      bool                           `json:"ignored"`                      // True if the component is ignored from the bill of materials, false otherwise
	License                      bdLicense                      `json:"license"`                      // The combination of licenses assigned to the component version
	VulnerabilityWithRemediation bdVulnerabilityWithRemediation `json:"vulnerabilityWithRemediation"` // The vulnerability associated with the BOM component
	PackageUrl                   string                         `json:"packageUrl"`                   // The Package URL, also known as PURL, of the component.
	Meta                         bdMeta                         `json:"_meta"`                        // Meta data associated with the representation
}
