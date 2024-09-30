package grype

import (
	"cli/internal/ecosystem/mappings"
	"fmt"
	"strings"
	"testing"
)

func TestLoadPolicyWithIgnore(t *testing.T) {
	var tests = []struct {
		content     string
		expectedKey ExistingVulnsKey
	}{
		{
			`ignore:
  - vulnerability: GHSA-jwhx-xcg6-8xhj
    package:
      name: aiohttp
      version: 3.9.5
      type: python
`, ExistingVulnsKey{
				vulnId:         "GHSA-jwhx-xcg6-8xhj",
				packageName:    "aiohttp",
				packageVersion: "3.9.5",
				packageManager: "python",
			}},
		{
			`# This configuration file will be used to track CVEs that we can ignore for the
# latest release of Dangerzone, and offer our analysis.

  ignore:
    # GHSA-jwhx-xcg6-8xhj
    # =============
    - vulnerability: GHSA-jwhx-xcg6-8xhj
`, ExistingVulnsKey{
				vulnId:         "GHSA-jwhx-xcg6-8xhj",
				packageName:    "",
				packageVersion: "",
				packageManager: "",
			}},
		{
			`ignore:
- vulnerability: GHSA-jwhx-xcg6-8xhj
  package:
    name: aiohttp
    version: 3.9.5
    type: python
add-cpes-if-none: true
`, ExistingVulnsKey{
				vulnId:         "GHSA-jwhx-xcg6-8xhj",
				packageName:    "aiohttp",
				packageVersion: "3.9.5",
				packageManager: "python",
			}},
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

		var vulnKey ExistingVulnsKey
		for key := range pf.existingVulns {
			vulnKey = key
			break
		}
		if vulnKey != tt.expectedKey {
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

func TestAddMavenRuleDropsGroupName(t *testing.T) {
	vulnId := "GHSA-rgv9-q543-rqg4"
	groupName := "com.fasterxml.jackson.core"
	artifactId := "jackson-databind"
	pkg := fmt.Sprintf("%s:%s", groupName, artifactId)
	version := "2.10.5.1"
	pkgManager := "java-archive"
	bePkgManager := "Maven"
	expected := fmt.Sprintf(`ignore:
  - vulnerability: %s
    reason: Fixed by Seal Security
    package:
      name: %s
      version: %s
      type: %s
`, vulnId, artifactId, version, pkgManager)

	pf, err := NewPolicy()
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

func Test(t *testing.T) {
	expected := `ignore:
  - vulnerability: CVE-2022-29217
    reason: Fixed by Seal Security
    package:
      name: pyjwt
      version: 1.7.1
      type: python`
	r := strings.NewReader(expected)
	pf, err := LoadPolicy(r)
	if err != nil {
		t.Fatal(err)
	}

	if len(pf.ignore.Content) != 1 {
		t.Fatalf("expected 1 ignore rule, got %d", len(pf.ignore.Content))
	}

	pf.AddRule("CVE-2022-29217", "pyjwt", "1.7.1", "python")
}

func TestExtractExistingVulnsKey(t *testing.T) {
	tests := []struct {
		vulnIndex   int
		name        string
		content     string
		expectedKey ExistingVulnsKey
		expectError bool
	}{
		{
			vulnIndex: 0,
			name:      "Valid vulnerability entry",
			content: `ignore:
- vulnerability: CVE-2021-1234
  package:
    name: package1
    version: 1.0.0
    type: python
`,
			expectedKey: ExistingVulnsKey{
				vulnId:         "CVE-2021-1234",
				packageName:    "package1",
				packageVersion: "1.0.0",
				packageManager: "python",
			},
			expectError: false,
		},
		{
			vulnIndex: 0,
			name:      "Vulnerability entry with extra fields",
			content: `ignore:
- randomkey: randomvalue
  vulnerability: CVE-2021-5678
  package:
    name: package2
    version: 2.0.0
    type: rpm
`,
			expectedKey: ExistingVulnsKey{
				vulnId:         "CVE-2021-5678",
				packageName:    "package2",
				packageVersion: "2.0.0",
				packageManager: "rpm",
			},
			expectError: false,
		},
		{
			vulnIndex: 0,
			name:      "Vulnerability entry with missing package details",
			content: `ignore:
- vulnerability: CVE-2021-9999
  package:
    name: 
`,
			expectedKey: ExistingVulnsKey{
				vulnId:         "CVE-2021-9999",
				packageName:    "",
				packageVersion: "",
				packageManager: "",
			},
			expectError: false,
		},
		{
			vulnIndex: 0,
			name:      "Malformed package field",
			content: `ignore:
- vulnerability: CVE-2021-1234
  package: badformat
`,
			expectedKey: ExistingVulnsKey{},
			expectError: true,
		},
		{
			vulnIndex: 0,
			name:      "Vulnerability entry with additional random keys",
			content: `ignore:
- vulnerability: CVE-2022-3456
  package:
    name: package4
    version: 4.0.0
    type: tar
  random: value
  extra:
    more: values
`,
			expectedKey: ExistingVulnsKey{
				vulnId:         "CVE-2022-3456",
				packageName:    "package4",
				packageVersion: "4.0.0",
				packageManager: "tar",
			},
			expectError: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			pf, err := LoadPolicy(strings.NewReader(test.content))

			if test.expectError && err == nil {
				t.Fatal("Expected error but got none")
			}
			if !test.expectError && err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if !test.expectError {
				if len(pf.existingVulns) != 1 {
					t.Fatalf("Expected one vulnerability in existing vulns")
				}
				var vulnKey ExistingVulnsKey
				for key := range pf.existingVulns {
					vulnKey = key
					break
				}
				if vulnKey != test.expectedKey {
					t.Fatalf("Expected %#v, got %#v", test.expectedKey, vulnKey)
				}
			}
		})
	}
}
