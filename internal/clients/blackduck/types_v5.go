package blackduck

type bdVersions struct {
	TotalCount int         `json:"totalCount"`
	Items      []bdVersion `json:"items"`
	Meta       bdMeta      `json:"_meta"`
}

type bdVersion struct {
	VersionName           string    `json:"versionName"`           // (Required) The name of the version
	Nickname              string    `json:"nickname"`              // An alternative commonly used name or alias for the project version
	ReleaseComments       string    `json:"releaseComments"`       // Pertinent comments or notes associated with the version
	ReleasedOn            string    `json:"releasedOn"`            // Time/Date the version was released
	Phase                 string    `json:"phase"`                 // (Required) Current phase of development of the version. Possible Values: [PLANNING, DEVELOPMENT, RELEASED, DEPRECATED, ARCHIVED, PRERELEASE]
	Distribution          string    `json:"distribution"`          // (Required) The distribution of the version. Possible Values: [EXTERNAL, SAAS, INTERNAL, OPENSOURCE]
	License               bdLicense `json:"license"`               // The combination of licenses to assign to the component version
	ProtectedFromDeletion bool      `json:"protectedFromDeletion"` // Whether or not this project version is protected from automatic deletion
	Branch                string    `json:"branch"`                // The repository branch name of the version
	CreatedAt             string    `json:"createdAt"`             // The date/time when the version was created
	CreatedBy             string    `json:"createdBy"`             // The username who created the version
	CreatedByUser         string    `json:"createdByUser"`         // URL for the resource representing the version creator
	SettingUpdatedAt      string    `json:"settingUpdatedAt"`      // Time/Date the version was last updated
	SettingUpdatedBy      string    `json:"settingUpdatedBy"`      // The username who last updated the version
	SettingUpdatedByUser  string    `json:"settingUpdatedByUser"`  // URL for the resource representing the user who last updated the version
	ScheduledDeletionDate string    `json:"scheduledDeletionDate"` // The date after which the project version is scheduled to be deleted automatically
	Source                string    `json:"source"`                // (Required) The source type of the version
	Meta                  bdMeta    `json:"_meta"`                 // (Required) Meta data associated with the representation
}
