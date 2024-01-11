package api

import (
	"cli/internal/common"
	"fmt"
)

type Page[T interface{}] struct {
	Items  []T `json:"items"`
	Total  int `json:"total"`
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

type Vulnerability struct {
	CVE              string  `json:"cve,omitempty"`
	MaliciousID      string  `json:"malicious_id,omitempty"`
	SnykID           string  `json:"snyk_id,omitempty"`
	GitHubAdvisoryID string  `json:"github_advisory_id,omitempty"`
	UnifiedScore     float32 `json:"unified_score,omitempty"`
}

func (VulnerabilityObject Vulnerability) PreferredId() string {
	if VulnerabilityObject.MaliciousID != "" {
		return VulnerabilityObject.MaliciousID
	}
	if VulnerabilityObject.CVE != "" {
		return VulnerabilityObject.CVE
	}
	if VulnerabilityObject.GitHubAdvisoryID != "" {
		return VulnerabilityObject.GitHubAdvisoryID
	}
	if VulnerabilityObject.SnykID != "" {
		return VulnerabilityObject.SnykID
	}
	panic("no vulnerability id exists") // bad server data
}

type Package struct {
	Name           string `json:"name"`
	PackageManager string `json:"package_manager"`
}

type PackageVersion struct {
	// this struct has much more fields, but we only need these
	Version                         string          `json:"version"`
	Library                         Package         `json:"library"`
	RecommendedLibraryVersionId     string          `json:"recommended_library_version_id,omitempty"`
	RecommendedLibraryVersionString string          `json:"recommended_library_version,omitempty"`
	OpenVulnerabilities             []Vulnerability `json:"open_vulnerabilities"`
}

type Metadata map[string]interface{}

type BulkCheckRequest struct {
	Entries  []common.Dependency    `json:"entries"`
	Metadata map[string]interface{} `json:"metadata"`
}

func (p *PackageVersion) CanBeFixed() bool {
	return p.RecommendedLibraryVersionId != ""
}

func (p *PackageVersion) IsMalicious() bool {
	for _, vuln := range p.OpenVulnerabilities {
		if vuln.MaliciousID != "" {
			return true
		}
	}
	return false
}

func (p *PackageVersion) Id() string {
	return common.DependencyId(p.Library.PackageManager, p.Library.Name, p.Version)
}

func (p *PackageVersion) RecommendedId() string {
	// in future we should have a recommendedName field / entire new object for it (in case we have a completely different package name)
	return common.DependencyId(p.Library.PackageManager, p.Library.Name, p.RecommendedLibraryVersionString)
}

func (p *PackageVersion) Descriptor() string {
	return fmt.Sprintf("%s@%s", p.Library.Name, p.Version)
}
func (p *PackageVersion) RecommendedDescriptor() string {
	return fmt.Sprintf("%s@%s", p.Library.Name, p.RecommendedLibraryVersionString)
}
