package maven

import (
	"cli/internal/api"
	"cli/internal/common"
	"cli/internal/config"
	"cli/internal/ecosystem/mappings"
	"fmt"
	"path/filepath"
	"testing"
)

func TestIsVersionSupported(t *testing.T) {
	var m *MavenPackageManager
	if m.IsVersionSupported("3.3.0") {
		t.Fatal("should not support version")
	}

	if m.IsVersionSupported("") {
		t.Fatal("should not support empty version")
	}

	if !m.IsVersionSupported(minimumMavenVersion) {
		t.Fatal("should support version 3.3.1")
	}

	if !m.IsVersionSupported("1003.3.1") {
		t.Fatal("should support newer version")
	}

}
func TestIndicatorMatches(t *testing.T) {
	ps := []string{
		`/b/pom.xml`,
		`C:\pom.xml`,
		`../pom.xml`,
		`..\pom.xml`,
		`./abc/../pom.xml`,
		`.\abc\..\pom.xml`,
	}

	for i, p := range ps {
		t.Run(fmt.Sprintf("pth_%d", i), func(t *testing.T) {
			if !IsMavenIndicatorFile(p) {
				t.Fatalf("failed to detect indicator path `%s`", p)
			}
		})
	}
}

func TestIndicatorDoesNotMatchOtherXml(t *testing.T) {
	// as it is intended to be handled by dir
	ps := []string{
		`/b/package.xml`,
		`C:\package.xml`,
		`../package.xml`,
		`..\package.xml`,
		`./abc/../package.xml`,
		`.\abc\..\package.xml`,
	}

	for i, p := range ps {
		t.Run(fmt.Sprintf("pth_%d", i), func(t *testing.T) {
			if IsMavenIndicatorFile(p) {
				t.Fatalf("failed to detect indicator path `%s`", p)
			}
		})
	}
}

func TestNormalizePackageNames(t *testing.T) {
	c, _ := config.New(nil)
	manager := NewMavenManager(c, "", "")
	names := []string{
		"aaaaa",
		"aaAAa",
		"AAAAA",
		"AAa_a",
	}
	for i, n := range names {
		t.Run(fmt.Sprintf("name_%d", i), func(t *testing.T) {
			if manager.NormalizePackageName(n) != n {
				t.Fatalf("failed to normalize `%s`", n)
			}
		})
	}
}

func TestGetJavaIndicatorFileAbsPath(t *testing.T) {
	tmp := t.TempDir()
	dst := filepath.Join(tmp, "pom.xml")
	fi, err := common.CreateFile(dst)
	if fi == nil || err != nil {
		t.Fatalf("faile: %v %v", fi, err)
	}
	defer fi.Close()

	p, err := GetJavaIndicatorFile(tmp)
	if err != nil {
		t.Fatalf("failed getting indicator %v", err)
	}

	if p != dst {
		t.Fatalf("excepted %s; got %s", dst, p)
	}
}

func TestConsolidateVulnerabilitiesBackendInfoNoEmbeddings(t *testing.T) {
	vulns := []api.PackageVersion{
		{
			Version: "1.2.3",
			Library: api.Package{
				Name:           "lodash",
				NormalizedName: "lodash",
				PackageManager: mappings.MavenManager,
			},
			RecommendedLibraryVersionId:     "123123",
			RecommendedLibraryVersionString: "1.2.3-sp1",
		},
		{
			Version: "1.2.3",
			Library: api.Package{
				Name:           "notlodash",
				NormalizedName: "notlodash",
				PackageManager: mappings.MavenManager,
			},
			RecommendedLibraryVersionId:     "123123",
			RecommendedLibraryVersionString: "1.2.3-sp1",
		},
	}

	result := consolidateVulnerabilitiesBackendInfo(vulns, make(map[string]bool, 0))
	if len(result) != 2 {
		t.Fatalf("failed to consolidate")
	}
}

func TestConsolidateVulnerabilitiesBackendInfoSanity(t *testing.T) {
	vulns := []api.PackageVersion{
		{
			Version: "1.2.3",
			Library: api.Package{
				Name:           "lodash",
				NormalizedName: "lodash",
				PackageManager: mappings.MavenManager,
			},
			RecommendedLibraryVersionId:     "123123",
			RecommendedLibraryVersionString: "1.2.3-sp1",
			OpenVulnerabilities: []api.Vulnerability{
				{
					CVE: "CVE-2024-1234",
					EmbeddedVia: []api.PublicPackage{
						{
							Name:           "notlodash",
							Version:        "1.2.3",
							PackageManager: mappings.MavenManager,
						},
					},
				},
			},
		},
		{ // Should be removed as it is embedded
			Version: "1.2.3",
			Library: api.Package{
				Name:           "notlodash",
				NormalizedName: "notlodash",
				PackageManager: mappings.MavenManager,
			},
			RecommendedLibraryVersionId:     "123123",
			RecommendedLibraryVersionString: "1.2.3-sp1",
			OpenVulnerabilities: []api.Vulnerability{
				{
					CVE: "CVE-2024-1234",
				},
			},
		},
		{
			Version: "1.2.3",
			Library: api.Package{
				Name:           "notlodash2",
				NormalizedName: "notlodash2",
				PackageManager: mappings.MavenManager,
			},
			RecommendedLibraryVersionId:     "123123",
			RecommendedLibraryVersionString: "1.2.3-sp1",
			OpenVulnerabilities: []api.Vulnerability{
				{
					CVE: "CVE-2024-5678",
				},
			},
		},
	}

	result := consolidateVulnerabilitiesBackendInfo(vulns, make(map[string]bool, 0))
	if len(result) != 2 {
		t.Fatalf("failed to consolidate")
	}

	for _, r := range result {
		if r.Library.Name == "notlodash" {
			t.Fatalf("failed to consolidate")
		}
	}
}

func TestConsolidateVulnerabilitiesBackendInfoMultiple(t *testing.T) {
	vulns := []api.PackageVersion{
		{
			Version: "1.2.3",
			Library: api.Package{
				Name:           "lodash",
				NormalizedName: "lodash",
				PackageManager: mappings.MavenManager,
			},
			RecommendedLibraryVersionId:     "123123",
			RecommendedLibraryVersionString: "1.2.3-sp1",
			OpenVulnerabilities: []api.Vulnerability{
				{
					CVE: "CVE-2024-1234",
					EmbeddedVia: []api.PublicPackage{
						{
							Name:           "notlodash",
							Version:        "1.2.3",
							PackageManager: mappings.MavenManager,
						},
					},
				},
				{
					CVE: "CVE-2024-5678",
					EmbeddedVia: []api.PublicPackage{
						{
							Name:           "notlodash2",
							Version:        "1.2.3",
							PackageManager: mappings.MavenManager,
						},
					},
				},
			},
		},
		{ // Should be removed as it is embedded
			Version: "1.2.3",
			Library: api.Package{
				Name:           "notlodash",
				NormalizedName: "notlodash",
				PackageManager: mappings.MavenManager,
			},
			RecommendedLibraryVersionId:     "123123",
			RecommendedLibraryVersionString: "1.2.3-sp1",
			OpenVulnerabilities: []api.Vulnerability{
				{
					CVE: "CVE-2024-1234",
				},
			},
		},
		{
			Version: "1.2.3",
			Library: api.Package{
				Name:           "notlodash2",
				NormalizedName: "notlodash2",
				PackageManager: mappings.MavenManager,
			},
			RecommendedLibraryVersionId:     "123123",
			RecommendedLibraryVersionString: "1.2.3-sp1",
			OpenVulnerabilities: []api.Vulnerability{
				{
					CVE: "CVE-2024-5678",
				},
			},
		},
	}

	result := consolidateVulnerabilitiesBackendInfo(vulns, make(map[string]bool, 0))
	if len(result) != 1 {
		t.Fatalf("failed to consolidate")
	}

	if result[0].Library.Name != "lodash" {
		t.Fatalf("failed to consolidate")
	}
}

func TestConsolidateVulnerabilitiesBackendBothShadedAndNot(t *testing.T) {
	both := api.PackageVersion{

		Version: "1.2.3",
		Library: api.Package{
			Name:           "notlodash",
			NormalizedName: "notlodash",
			PackageManager: mappings.MavenManager,
		},
		RecommendedLibraryVersionId:     "123123",
		RecommendedLibraryVersionString: "1.2.3-sp1",
		OpenVulnerabilities: []api.Vulnerability{
			{
				CVE: "CVE-2024-1234",
			},
		},
	}
	vulns := []api.PackageVersion{
		{
			Version: "1.2.3",
			Library: api.Package{
				Name:           "lodash",
				NormalizedName: "lodash",
				PackageManager: mappings.MavenManager,
			},
			RecommendedLibraryVersionId:     "123123",
			RecommendedLibraryVersionString: "1.2.3-sp1",
			OpenVulnerabilities: []api.Vulnerability{
				{
					CVE: "CVE-2024-1234",
					EmbeddedVia: []api.PublicPackage{
						{
							Name:           "notlodash",
							Version:        "1.2.3",
							PackageManager: mappings.MavenManager,
						},
					},
				},
				{
					CVE: "CVE-2024-5678",
					EmbeddedVia: []api.PublicPackage{
						{
							Name:           "notlodash2",
							Version:        "1.2.3",
							PackageManager: mappings.MavenManager,
						},
					},
				},
			},
		},
		both, // Should not be removed as it is embedded but also a parent
	}

	parents := map[string]bool{both.Id(): true}

	result := consolidateVulnerabilitiesBackendInfo(vulns, parents)
	if len(result) != 2 {
		t.Fatalf("failed to consolidate")
	}

	if result[0].Library.Name != "lodash" {
		t.Fatalf("failed to consolidate")
	}

	if result[1].Library.Name != "notlodash" {
		t.Fatalf("failed to consolidate")
	}
}

func TestConsolidateVulnerabilitiesCliInfoNoEmbeddings(t *testing.T) {
	vulns := []api.PackageVersion{
		{
			Version: "1.2.3",
			Library: api.Package{
				Name:           "lodash",
				NormalizedName: "lodash",
				PackageManager: mappings.MavenManager,
			},
			RecommendedLibraryVersionId:     "123123",
			RecommendedLibraryVersionString: "1.2.3-sp1",
		},
		{
			Version: "1.2.3",
			Library: api.Package{
				Name:           "notlodash",
				NormalizedName: "notlodash",
				PackageManager: mappings.MavenManager,
			},
			RecommendedLibraryVersionId:     "123123",
			RecommendedLibraryVersionString: "1.2.3-sp1",
		},
	}

	dep1 := &common.Dependency{
		Name:           "lodash",
		NormalizedName: "lodash",
		Version:        "1.2.3",
		PackageManager: mappings.MavenManager,
	}
	dep2 := &common.Dependency{
		Name:           "notlodash",
		NormalizedName: "notlodash",
		Version:        "1.2.3",
		PackageManager: mappings.MavenManager,
	}

	deps := common.DependencyMap{
		"lodash":    {dep1},
		"notlodash": {dep2},
	}

	result := consolidateVulnerabilitiesCliInfo(vulns, deps, map[string]bool{dep1.Id(): true, dep2.Id(): true})
	if len(result) != 2 {
		t.Fatalf("failed to consolidate")
	}
}

func TestConsolidateVulnerabilitiesCliInfoSanity(t *testing.T) {
	vulns := []api.PackageVersion{
		{
			Version: "1.2.3",
			Library: api.Package{
				Name:           "lodash",
				NormalizedName: "lodash",
				PackageManager: mappings.MavenManager,
			},
			RecommendedLibraryVersionId:     "123123",
			RecommendedLibraryVersionString: "1.2.3-sp1",
		},
		{
			Version: "1.2.3",
			Library: api.Package{
				Name:           "notlodash",
				NormalizedName: "notlodash",
				PackageManager: mappings.MavenManager,
			},
			RecommendedLibraryVersionId:     "123123",
			RecommendedLibraryVersionString: "1.2.3-sp1",
		},
	}

	parent := &common.Dependency{
		Name:           "lodash",
		NormalizedName: "lodash",
		Version:        "1.2.3",
		PackageManager: mappings.MavenManager,
	}
	deps := common.DependencyMap{
		"lodash": {
			parent,
		},
		"notlodash": {
			&common.Dependency{
				Name:           "notlodash",
				NormalizedName: "notlodash",
				Version:        "1.2.3",
				PackageManager: mappings.MavenManager,
				Parent:         parent,
				IsShaded:       true,
			},
		},
	}

	result := consolidateVulnerabilitiesCliInfo(vulns, deps, map[string]bool{parent.Id(): true})
	if len(result) != 1 {
		t.Fatalf("failed to consolidate")
	}

	if result[0].Library.Name != "lodash" {
		t.Fatalf("failed to consolidate")
	}
}

func TestConsolidateVulnerabilitiesCliInfoCopiesOpenVulns(t *testing.T) {
	cve := "CVE-2024-1234"
	vulns := []api.PackageVersion{
		{
			Version: "1.2.3",
			Library: api.Package{
				Name:           "lodash",
				NormalizedName: "lodash",
				PackageManager: mappings.MavenManager,
			},
			RecommendedLibraryVersionId:     "123123",
			RecommendedLibraryVersionString: "1.2.3-sp1",
			OpenVulnerabilities:             []api.Vulnerability{},
		},
		{
			Version: "1.2.3",
			Library: api.Package{
				Name:           "notlodash",
				NormalizedName: "notlodash",
				PackageManager: mappings.MavenManager,
			},
			RecommendedLibraryVersionId:     "123123",
			RecommendedLibraryVersionString: "1.2.3-sp1",
			OpenVulnerabilities: []api.Vulnerability{
				{
					CVE: cve,
				},
			},
		},
	}

	parent := &common.Dependency{
		Name:           "lodash",
		NormalizedName: "lodash",
		Version:        "1.2.3",
		PackageManager: mappings.MavenManager,
	}
	deps := common.DependencyMap{
		"lodash": {
			parent,
		},
		"notlodash": {
			&common.Dependency{
				Name:           "notlodash",
				NormalizedName: "notlodash",
				Version:        "1.2.3",
				PackageManager: mappings.MavenManager,
				Parent:         parent,
				IsShaded:       true,
			},
		},
	}

	result := consolidateVulnerabilitiesCliInfo(vulns, deps, map[string]bool{parent.Id(): true})
	if len(result) != 1 {
		t.Fatalf("failed to consolidate")
	}

	if result[0].Library.Name != "lodash" {
		t.Fatalf("failed to consolidate")
	}

	if len(result[0].OpenVulnerabilities) != 1 {
		t.Fatalf("failed to consolidate, expected 1 vuln but got %d", len(result[0].OpenVulnerabilities))
	}

	if result[0].OpenVulnerabilities[0].CVE != cve {
		t.Fatalf("failed to consolidate")
	}

	if result[0].OpenVulnerabilities[0].EmbeddedVia[0].Name != "notlodash" {
		t.Fatalf("failed to consolidate")
	}
}

func TestConsolidateVulnerabilitiesCliInfoConsolidatesOpenVulns(t *testing.T) {
	cve := "CVE-2024-1234"
	vulns := []api.PackageVersion{
		{
			Version: "1.2.3",
			Library: api.Package{
				Name:           "lodash",
				NormalizedName: "lodash",
				PackageManager: mappings.MavenManager,
			},
			RecommendedLibraryVersionId:     "123123",
			RecommendedLibraryVersionString: "1.2.3-sp1",
			OpenVulnerabilities: []api.Vulnerability{
				{
					CVE: cve,
				},
			},
		},
		{
			Version: "1.2.3",
			Library: api.Package{
				Name:           "notlodash",
				NormalizedName: "notlodash",
				PackageManager: mappings.MavenManager,
			},
			RecommendedLibraryVersionId:     "123123",
			RecommendedLibraryVersionString: "1.2.3-sp1",
			OpenVulnerabilities: []api.Vulnerability{
				{
					CVE: cve,
				},
			},
		},
	}

	parent := &common.Dependency{
		Name:           "lodash",
		NormalizedName: "lodash",
		Version:        "1.2.3",
		PackageManager: mappings.MavenManager,
	}
	deps := common.DependencyMap{
		"lodash": {
			parent,
		},
		"notlodash": {
			&common.Dependency{
				Name:           "notlodash",
				NormalizedName: "notlodash",
				Version:        "1.2.3",
				PackageManager: mappings.MavenManager,
				Parent:         parent,
				IsShaded:       true,
			},
		},
	}

	result := consolidateVulnerabilitiesCliInfo(vulns, deps, map[string]bool{parent.Id(): true})
	if len(result) != 1 {
		t.Fatalf("failed to consolidate")
	}

	if result[0].Library.Name != "lodash" {
		t.Fatalf("failed to consolidate")
	}

	if len(result[0].OpenVulnerabilities) != 1 {
		t.Fatalf("failed to consolidate, expected 1 vuln but got %d", len(result[0].OpenVulnerabilities))
	}

	if result[0].OpenVulnerabilities[0].CVE != cve {
		t.Fatalf("failed to consolidate")
	}

	if result[0].OpenVulnerabilities[0].EmbeddedVia[0].Name != "notlodash" {
		t.Fatalf("failed to consolidate")
	}
}

func TestConsolidateVulnerabilitiesCliInfoAddsOpenVulns(t *testing.T) {
	cve1 := "CVE-2024-1234"
	cve2 := "CVE-2024-5678"
	vulns := []api.PackageVersion{
		{
			Version: "1.2.3",
			Library: api.Package{
				Name:           "lodash",
				NormalizedName: "lodash",
				PackageManager: mappings.MavenManager,
			},
			RecommendedLibraryVersionId:     "123123",
			RecommendedLibraryVersionString: "1.2.3-sp1",
			OpenVulnerabilities: []api.Vulnerability{
				{
					CVE: cve1,
				},
			},
		},
		{
			Version: "1.2.3",
			Library: api.Package{
				Name:           "notlodash",
				NormalizedName: "notlodash",
				PackageManager: mappings.MavenManager,
			},
			RecommendedLibraryVersionId:     "123123",
			RecommendedLibraryVersionString: "1.2.3-sp1",
			OpenVulnerabilities: []api.Vulnerability{
				{
					CVE: cve2,
				},
			},
		},
	}

	parent := &common.Dependency{
		Name:           "lodash",
		NormalizedName: "lodash",
		Version:        "1.2.3",
		PackageManager: mappings.MavenManager,
	}
	deps := common.DependencyMap{
		"lodash": {
			parent,
		},
		"notlodash": {
			&common.Dependency{
				Name:           "notlodash",
				NormalizedName: "notlodash",
				Version:        "1.2.3",
				PackageManager: mappings.MavenManager,
				Parent:         parent,
				IsShaded:       true,
			},
		},
	}

	result := consolidateVulnerabilitiesCliInfo(vulns, deps, map[string]bool{parent.Id(): true})
	if len(result) != 1 {
		t.Fatalf("failed to consolidate")
	}

	if result[0].Library.Name != "lodash" {
		t.Fatalf("failed to consolidate")
	}

	if len(result[0].OpenVulnerabilities) != 2 {
		t.Fatalf("failed to consolidate, expected 1 vuln but got %d", len(result[0].OpenVulnerabilities))
	}

	if result[0].OpenVulnerabilities[0].CVE != cve1 {
		t.Fatalf("failed to consolidate")
	}

	if len(result[0].OpenVulnerabilities[0].EmbeddedVia) != 0 {
		t.Fatalf("failed to consolidate")
	}

	if result[0].OpenVulnerabilities[1].CVE != cve2 {
		t.Fatalf("failed to consolidate")
	}

	if result[0].OpenVulnerabilities[1].EmbeddedVia[0].Name != "notlodash" {
		t.Fatalf("failed to consolidate")
	}
}

func TestConsolidateVulnerabilitiesCliInfoConsolidatesOpenVulnsTwice(t *testing.T) {
	cve := "CVE-2024-1234"
	vulns := []api.PackageVersion{
		{
			Version: "1.2.3",
			Library: api.Package{
				Name:           "lodash",
				NormalizedName: "lodash",
				PackageManager: mappings.MavenManager,
			},
			RecommendedLibraryVersionId:     "123123",
			RecommendedLibraryVersionString: "1.2.3-sp1",
			OpenVulnerabilities: []api.Vulnerability{
				{
					CVE: cve,
				},
			},
		},
		{
			Version: "1.2.3",
			Library: api.Package{
				Name:           "notlodash",
				NormalizedName: "notlodash",
				PackageManager: mappings.MavenManager,
			},
			RecommendedLibraryVersionId:     "123123",
			RecommendedLibraryVersionString: "1.2.3-sp1",
			OpenVulnerabilities: []api.Vulnerability{
				{
					CVE: cve,
				},
			},
		},
		{
			Version: "1.2.3",
			Library: api.Package{
				Name:           "notlodash2",
				NormalizedName: "notlodash2",
				PackageManager: mappings.MavenManager,
			},
			RecommendedLibraryVersionId:     "123123",
			RecommendedLibraryVersionString: "1.2.3-sp1",
			OpenVulnerabilities: []api.Vulnerability{
				{
					CVE: cve,
				},
			},
		},
	}

	parent := &common.Dependency{
		Name:           "lodash",
		NormalizedName: "lodash",
		Version:        "1.2.3",
		PackageManager: mappings.MavenManager,
	}
	deps := common.DependencyMap{
		"lodash": {
			parent,
		},
		"notlodash": {
			&common.Dependency{
				Name:           "notlodash",
				NormalizedName: "notlodash",
				Version:        "1.2.3",
				PackageManager: mappings.MavenManager,
				Parent:         parent,
				IsShaded:       true,
			},
		},
		"notlodash2": {
			&common.Dependency{
				Name:           "notlodash2",
				NormalizedName: "notlodash2",
				Version:        "1.2.3",
				PackageManager: mappings.MavenManager,
				Parent:         parent,
				IsShaded:       true,
			},
		},
	}

	result := consolidateVulnerabilitiesCliInfo(vulns, deps, map[string]bool{parent.Id(): true})
	if len(result) != 1 {
		t.Fatalf("failed to consolidate")
	}

	if result[0].Library.Name != "lodash" {
		t.Fatalf("failed to consolidate")
	}

	if len(result[0].OpenVulnerabilities) != 1 {
		t.Fatalf("failed to consolidate, expected 1 vuln but got %d", len(result[0].OpenVulnerabilities))
	}

	if result[0].OpenVulnerabilities[0].CVE != cve {
		t.Fatalf("failed to consolidate")
	}

	if len(result[0].OpenVulnerabilities[0].EmbeddedVia) != 2 {
		t.Fatalf("failed to consolidate")
	}

	if result[0].OpenVulnerabilities[0].EmbeddedVia[0].Name != "notlodash" {
		t.Fatalf("failed to consolidate")
	}

	if result[0].OpenVulnerabilities[0].EmbeddedVia[1].Name != "notlodash2" {
		t.Fatalf("failed to consolidate")
	}
}

// If the shading library doesn't have vulns, we ignore it
// On the next scan, the BE will already know it because it updates shading relations
func TestConsolidateVulnerabilitiesCliInfoOnlyShadedHasLibraryVersion(t *testing.T) {
	cve2 := "CVE-2024-5678"
	vulns := []api.PackageVersion{
		{
			Version: "1.2.3",
			Library: api.Package{
				Name:           "notlodash",
				NormalizedName: "notlodash",
				PackageManager: mappings.MavenManager,
			},
			RecommendedLibraryVersionId:     "123123",
			RecommendedLibraryVersionString: "1.2.3-sp1",
			OpenVulnerabilities: []api.Vulnerability{
				{
					CVE: cve2,
				},
			},
		},
	}

	parent := &common.Dependency{
		Name:           "lodash",
		NormalizedName: "lodash",
		Version:        "1.2.3",
		PackageManager: mappings.MavenManager,
	}
	deps := common.DependencyMap{
		"lodash": {
			parent,
		},
		"notlodash": {
			&common.Dependency{
				Name:           "notlodash",
				NormalizedName: "notlodash",
				Version:        "1.2.3",
				PackageManager: mappings.MavenManager,
				Parent:         parent,
				IsShaded:       true,
			},
		},
	}

	result := consolidateVulnerabilitiesCliInfo(vulns, deps, map[string]bool{parent.Id(): true})
	if len(result) != 0 {
		t.Fatalf("failed to consolidate, expected 0 but got %d", len(result))
	}
}

func TestGetEmbeddedVulnerablePackageSanity(t *testing.T) {
	cve := "CVE-2024-5678"
	embeddedPackage := api.PackageVersion{
		Version: "1.2.3",
		Library: api.Package{
			Name:           "notlodash",
			NormalizedName: "notlodash",
			PackageManager: mappings.MavenManager,
		},
		RecommendedLibraryVersionId:     "123123",
		RecommendedLibraryVersionString: "1.2.3-sp1",
		OpenVulnerabilities: []api.Vulnerability{
			{
				CVE: cve,
			},
		},
	}
	embeddingPackage := api.PackageVersion{
		Version: "1.2.3",
		Library: api.Package{
			Name:           "lodash",
			NormalizedName: "lodash",
			PackageManager: mappings.MavenManager,
		},
		RecommendedLibraryVersionId:     "123123",
		RecommendedLibraryVersionString: "1.2.3-sp1",
		OpenVulnerabilities: []api.Vulnerability{
			{
				CVE: cve,
			},
		},
	}

	vulns := []api.PackageVersion{embeddedPackage, embeddingPackage}

	parent := &common.Dependency{
		Name:           "lodash",
		NormalizedName: "lodash",
		Version:        "1.2.3",
		PackageManager: mappings.MavenManager,
	}
	deps := common.DependencyMap{
		"lodash": {
			parent,
		},
		"notlodash": {
			&common.Dependency{
				Name:           "notlodash",
				NormalizedName: "notlodash",
				Version:        "1.2.3",
				PackageManager: mappings.MavenManager,
				Parent:         parent,
				IsShaded:       true,
			},
		},
	}

	embedded := getEmbeddedVulnerablePackage(embeddingPackage, vulns, deps)
	if len(embedded) != 1 {
		t.Fatalf("failed to consolidate, expected 0 but got %d", len(embedded))
	}

	if embedded[0].Library.Name != "notlodash" {
		t.Fatalf("failed to consolidate")
	}
}

func TestGetEmbeddedVulnerablePackageAppearsTwice(t *testing.T) {
	cve := "CVE-2024-5678"
	embeddedPackage := api.PackageVersion{
		Version: "1.2.3",
		Library: api.Package{
			Name:           "notlodash",
			NormalizedName: "notlodash",
			PackageManager: mappings.MavenManager,
		},
		RecommendedLibraryVersionId:     "123123",
		RecommendedLibraryVersionString: "1.2.3-sp1",
		OpenVulnerabilities: []api.Vulnerability{
			{
				CVE: cve,
			},
		},
	}
	embeddingPackage := api.PackageVersion{
		Version: "1.2.3",
		Library: api.Package{
			Name:           "lodash",
			NormalizedName: "lodash",
			PackageManager: mappings.MavenManager,
		},
		RecommendedLibraryVersionId:     "123123",
		RecommendedLibraryVersionString: "1.2.3-sp1",
		OpenVulnerabilities: []api.Vulnerability{
			{
				CVE: cve,
			},
		},
	}

	vulns := []api.PackageVersion{embeddedPackage, embeddingPackage}

	parent := &common.Dependency{
		Name:           "lodash",
		NormalizedName: "lodash",
		Version:        "1.2.3",
		PackageManager: mappings.MavenManager,
	}
	deps := common.DependencyMap{
		"lodash": {
			parent,
		},
		"notlodash": {
			&common.Dependency{
				Name:           "notlodash",
				NormalizedName: "notlodash",
				Version:        "1.2.3",
				PackageManager: mappings.MavenManager,
				Parent:         parent,
				IsShaded:       true,
			},
			&common.Dependency{
				Name:           "notlodash",
				NormalizedName: "notlodash",
				Version:        "1.2.3",
				PackageManager: mappings.MavenManager,
				Parent:         nil,
				IsShaded:       false,
			},
		},
	}

	embedded := getEmbeddedVulnerablePackage(embeddingPackage, vulns, deps)
	if len(embedded) != 1 {
		t.Fatalf("failed to consolidate, expected 0 but got %d", len(embedded))
	}

	if embedded[0].Library.Name != "notlodash" {
		t.Fatalf("failed to consolidate")
	}
}
