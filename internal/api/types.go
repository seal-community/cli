package api

import (
	"cli/internal/common"
	"cli/internal/ecosystem/mappings"
	"fmt"
)

type Page[T interface{}] struct {
	Items  []T `json:"items"`
	Total  int `json:"total"`
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

type PublicPackage struct {
	Name           string `json:"library_name"`
	Version        string `json:"library_version"`
	PackageManager string `json:"library_package_manager"`
}

type Vulnerability struct {
	CVE              string          `json:"cve,omitempty"`
	MaliciousID      string          `json:"malicious_id,omitempty"`
	SnykID           string          `json:"snyk_id,omitempty"`
	GitHubAdvisoryID string          `json:"github_advisory_id,omitempty"`
	UnifiedScore     float32         `json:"unified_score,omitempty"`
	EmbeddedVia      []PublicPackage `json:"embedded_via,omitempty"`
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
	Name           string `json:"escaped_name"`    // using escaped name to correctly match packages that resolve to the same value (e.g. for pip)
	NormalizedName string `json:"normalized_name"` // the names are normalized differently according to the package manager
	PackageManager string `json:"package_manager"`
	Id             string `json:"id"`
}

type PackageVersion struct {
	// this struct has much more fields, but we only need these
	VersionId                       string          `json:"id"` // internal uuid
	Library                         Package         `json:"library"`
	Version                         string          `json:"version"`
	RecommendedLibraryVersionId     string          `json:"recommended_library_version_id,omitempty"`
	RecommendedLibraryVersionString string          `json:"recommended_library_version,omitempty"`
	OpenVulnerabilities             []Vulnerability `json:"open_vulnerabilities"`
	SealedVulnerabilities           []Vulnerability `json:"sealed_vulnerabilities"`
	OriginVersionString             string          `json:"origin_version"`
	OriginVersionId                 string          `json:"origin_version_id"`
}

type Metadata map[string]interface{}

func (p *PackageVersion) CanBeFixed() bool {
	return p.RecommendedLibraryVersionId != ""
}

// is this packages a sealed version (could have a recommended version e.g. sp1)
func (p *PackageVersion) IsSealed() bool {
	return p.OriginVersionId != ""
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
	return common.DependencyId(p.Library.PackageManager, p.Library.NormalizedName, p.Version)
}

func (p *PackageVersion) RecommendedId() string {
	// in future we should have a recommendedName field / entire new object for it (in case we have a completely different package name)
	return common.DependencyId(p.Library.PackageManager, p.Library.NormalizedName, p.RecommendedLibraryVersionString)
}

func (p *PackageVersion) OriginId() string {
	if p.OriginVersionString == "" {
		// if this is an origin version it does not have an origin version set, return itself for matching in dictionaries
		return p.Id()
	}

	return common.DependencyId(p.Library.PackageManager, p.Library.NormalizedName, p.OriginVersionString)
}

func (p *PackageVersion) Descriptor() string {
	return fmt.Sprintf("%s@%s", p.Library.Name, p.Version)
}

func (p *PackageVersion) RecommendedDescriptor() string {
	return fmt.Sprintf("%s@%s", p.Library.Name, p.RecommendedLibraryVersionString)
}

func (p *PackageVersion) Ecosystem() string {
	return mappings.BackendManagerToEcosystem(p.Library.PackageManager)
}

func (p *PackageVersion) PublicPackage() PublicPackage {
	return PublicPackage{
		Name:           p.Library.Name,
		Version:        p.Version,
		PackageManager: p.Library.PackageManager,
	}
}

// Vulnerabilities are equivalent if they have the same CVE, MaliciousID, SnykID and GitHubAdvisoryID
func (v *Vulnerability) Equivalent(other Vulnerability) bool {
	return v.CVE == other.CVE && v.MaliciousID == other.MaliciousID && v.SnykID == other.SnykID && v.GitHubAdvisoryID == other.GitHubAdvisoryID
}
