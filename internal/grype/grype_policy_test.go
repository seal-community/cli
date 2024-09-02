package grype

import (
	"cli/internal/ecosystem/mappings"
	"fmt"
	"strings"
	"testing"
)

func TestLoadPolicyWithIgnore(t *testing.T) {
	var tests = []struct {
		content string
	}{
		{
			`ignore:
  - vulnerability: GHSA-jwhx-xcg6-8xhj
    package:
      name: aiohttp
      version: 3.9.5
      type: python
`},
		{
			`# This configuration file will be used to track CVEs that we can ignore for the
# latest release of Dangerzone, and offer our analysis.

  ignore:
    # GHSA-jwhx-xcg6-8xhj
    # =============
    - vulnerability: GHSA-jwhx-xcg6-8xhj
`},
		{
			`ignore:
- vulnerability: GHSA-jwhx-xcg6-8xhj
  package:
    name: aiohttp
    version: 3.9.5
    type: python
add-cpes-if-none: true
`},
	}

	for _, tt := range tests {
		r := strings.NewReader(tt.content)
		pf, err := LoadPolicy(r)
		if err != nil {
			t.Fatal(err)
		}

		if len(pf.ignore.Content) != 1 {
			t.Fatalf("expected 1 ignore rule, got %d", len(pf.ignore.Content))
		}

		if ok := pf.existingVulns["GHSA-jwhx-xcg6-8xhj"]; !ok {
			t.Fatalf("expected to find GHSA-jwhx-xcg6-8xhj in existing vulns")
		}
	}
}

func TestLoadPolicyWithoutIgnore(t *testing.T) {
	content := `add-cpes-if-none: true
fail-on-severity: high`
	r := strings.NewReader(content)
	pf, err := LoadPolicy(r)
	if err != nil {
		t.Fatal(err)
	}

	if len(pf.ignore.Content) != 0 {
		t.Fatalf("expected 1 ignore rule, got %d", len(pf.ignore.Content))
	}

	if len(pf.existingVulns) != 0 {
		t.Fatalf("expected 0 existing vulns, got %d", len(pf.existingVulns))
	}
}

func TestNewPolicy(t *testing.T) {
	pf, err := NewPolicy()
	if err != nil {
		t.Fatal(err)
	}

	if len(pf.ignore.Content) != 0 {
		t.Fatalf("expected 0 rules, got %d", len(pf.ignore.Content))
	}
}

func TestSavePolicy(t *testing.T) {
	tests := []struct {
		content string
	}{
		{
			`ignore:
  - vulnerability: GHSA-jwhx-xcg6-8xhj
    package:
      name: aiohttp
      version: 3.9.5
      type: python
`},
		{
			`# This configuration file will be used to track CVEs that we can ignore for the
# latest release of Dangerzone, and offer our analysis.
ignore:
  # CVE-2024-5171
  # =============
  - vulnerability: CVE-2024-5171
`},
	}

	for _, tt := range tests {
		r := strings.NewReader(tt.content)
		pf, err := LoadPolicy(r)
		if err != nil {
			t.Fatal(err)
		}

		b := &strings.Builder{}
		err = SavePolicy(pf, b)
		if err != nil {
			t.Fatal(err)
		}

		if b.String() != tt.content {
			t.Fatalf("expected %s, got %s", tt.content, b.String())
		}
	}

}

func TestAddRule(t *testing.T) {
	vulnId := "GHSA-jwhx-xcg6-8xhj"
	pkg := "aiohttp"
	version := "3.9.5"
	pkgManager := "python"
	bePkgManager := "PyPI"
	expected := fmt.Sprintf(`ignore:
  - vulnerability: %s
    reason: Fixed by Seal Security
    package:
      name: %s
      version: %s
      type: %s
`, vulnId, pkg, version, pkgManager)

	pf, err := NewPolicy()
	if err != nil {
		t.Fatal(err)
	}

	if !pf.AddRule(vulnId, pkg, version, bePkgManager) {
		t.Fatal("expected true")
	}

	if pf.AddRule(vulnId, pkg, version, bePkgManager) {
		t.Fatal("expected false")
	}

	b := &strings.Builder{}
	err = SavePolicy(pf, b)
	if err != nil {
		t.Fatal(err)
	}

	if b.String() != expected {
		t.Fatalf("expected %s, got %s", expected, b.String())
	}
}

func TestAddRuleWithExisting(t *testing.T) {
	vulnId := "GHSA-jwhx-xcg6-8xhj"
	pkg := "aiohttp"
	version := "3.9.5"
	pkgManager := "python"
	bePkgManager := "PyPI"
	expected := fmt.Sprintf(`ignore:
  - vulnerability: %s
    reason: Fixed by Seal Security
    package:
      name: %s
      version: %s
      type: %s
`, vulnId, pkg, version, pkgManager)

	pf, err := LoadPolicy(strings.NewReader(expected))
	if err != nil {
		t.Fatal(err)
	}

	if pf.AddRule(vulnId, pkg, version, bePkgManager) {
		t.Fatal("expected true")
	}

	if pf.AddRule(vulnId, "some", "other", "npm") {
		t.Fatal("expected true")
	}

	b := &strings.Builder{}
	err = SavePolicy(pf, b)
	if err != nil {
		t.Fatal(err)
	}

	if b.String() != expected {
		t.Fatalf("expected %s, got %s", expected, b.String())
	}
}

func TestAddRulePreserveOtherConfigs(t *testing.T) {
	vulnId := "GHSA-jwhx-xcg6-8xhj"
	pkg := "aiohttp"
	version := "3.9.5"
	pkgManager := "python"
	bePkgManager := "PyPI"
	before := `add-cpes-if-none: true
fail-on-severity: high
`
	expected := fmt.Sprintf(`%signore:
  - vulnerability: %s
    reason: Fixed by Seal Security
    package:
      name: %s
      version: %s
      type: %s
`, before, vulnId, pkg, version, pkgManager)

	pf, err := LoadPolicy(strings.NewReader(before))
	if err != nil {
		t.Fatal(err)
	}

	if !pf.AddRule(vulnId, pkg, version, bePkgManager) {
		t.Fatal("expected true")
	}

	b := &strings.Builder{}
	err = SavePolicy(pf, b)
	if err != nil {
		t.Fatal(err)
	}

	if b.String() != expected {
		t.Fatalf("expected %s, got %s", expected, b.String())
	}
}

func TestGrypePackageManager(t *testing.T) {
	maps := [][]string{
		{mappings.NpmManager, "npm"},
		{mappings.PythonManager, "python"},
		{mappings.NugetManager, "dotnet"},
		{mappings.MavenManger, "java-archive"},
		{mappings.GolangManager, "go-module"},
		{"asdasdasda", ""},
	}

	for i, m := range maps {
		t.Run(fmt.Sprintf("map_%d", i), func(t *testing.T) {
			given := m[0]
			expected := m[1]
			if result := grypePackageManager(given); result != expected {
				t.Fatalf("wrong manager, expected: `%s` got: `%s`", expected, result)
			}
		})
	}
}
