package ox

import (
	"cli/internal/api"
	"cli/internal/config"
	"cli/internal/ecosystem/mappings"
	"cli/internal/ecosystem/shared"
	"net/http"
	"strings"
	"testing"
)

func getFixAndScanResult() []shared.DependencyDescriptor {
	scannedSmolToml := api.PackageVersion{
		Version:                         "1.3.0",
		Library:                         api.Package{NormalizedName: "smol-toml", Name: "smol-toml", PackageManager: mappings.NpmManager},
		RecommendedLibraryVersionId:     "11111",
		RecommendedLibraryVersionString: "1.3.0+sp1",
		OpenVulnerabilities: []api.Vulnerability{
			{GitHubAdvisoryID: "GHSA-pqhp-25j4-6hq9"},
		},
		OriginVersionString: "1.3.0",
	}
	fixedSmolToml := api.PackageVersion{
		Version:             "1.3.0+sp1",
		Library:             api.Package{NormalizedName: "smol-toml", Name: "smol-toml", PackageManager: mappings.NpmManager},
		OpenVulnerabilities: []api.Vulnerability{},
		SealedVulnerabilities: []api.Vulnerability{
			{GitHubAdvisoryID: "GHSA-pqhp-25j4-6hq9"},
		},
		OriginVersionString: "1.3.0",
	}

	scannedEjs := api.PackageVersion{
		Version:                         "3.1.10",
		Library:                         api.Package{NormalizedName: "ejs", Name: "ejs", PackageManager: mappings.NpmManager},
		RecommendedLibraryVersionId:     "2222222",
		RecommendedLibraryVersionString: "3.1.10+sp1",
		OpenVulnerabilities: []api.Vulnerability{
			{CVE: "CVE-2024-33883", GitHubAdvisoryID: "GHSA-ghr5-ch3p-vcr6"},
			{CVE: "CVE-dummy-open-vuln", GitHubAdvisoryID: "GHSA-dummy-open-vuln"},
		},
		OriginVersionString: "3.1.10",
	}
	fixedEjs := api.PackageVersion{
		Version:             "3.1.10+sp1",
		Library:             api.Package{NormalizedName: "ejs", Name: "ejs", PackageManager: mappings.NpmManager},
		OpenVulnerabilities: []api.Vulnerability{},
		SealedVulnerabilities: []api.Vulnerability{
			{CVE: "CVE-2024-33883", GitHubAdvisoryID: "GHSA-ghr5-ch3p-vcr6"},
		},
		OriginVersionString: "3.1.10",
	}

	scannedEjsAnotherVersion := api.PackageVersion{
		Version:                         "3.1.9",
		Library:                         api.Package{NormalizedName: "ejs", Name: "ejs", PackageManager: mappings.NpmManager},
		RecommendedLibraryVersionId:     "3333333",
		RecommendedLibraryVersionString: "3.1.9+sp1",
		OpenVulnerabilities: []api.Vulnerability{
			{CVE: "CVE-2024-33883", GitHubAdvisoryID: "GHSA-ghr5-ch3p-vcr6"},
			{CVE: "CVE-dummy-open-vuln", GitHubAdvisoryID: "GHSA-dummy-open-vuln"},
		},
		OriginVersionString: "3.1.9",
	}
	fixedEjsAnotherVersion := api.PackageVersion{
		Version: "3.1.9+sp1",
		Library: api.Package{NormalizedName: "ejs", Name: "ejs", PackageManager: mappings.NpmManager},
		OpenVulnerabilities: []api.Vulnerability{
			{CVE: "CVE-2024-33883", GitHubAdvisoryID: "GHSA-ghr5-ch3p-vcr6"},
		},
		SealedVulnerabilities: []api.Vulnerability{},
		OriginVersionString:   "3.1.9",
	}

	return []shared.DependencyDescriptor{
		{VulnerablePackage: &scannedSmolToml, AvailableFix: &fixedSmolToml, Locations: nil, FixedLocations: nil},
		{VulnerablePackage: &scannedEjs, AvailableFix: &fixedEjs, Locations: nil, FixedLocations: nil},
		{VulnerablePackage: &scannedEjsAnotherVersion, AvailableFix: &fixedEjsAnotherVersion, Locations: nil, FixedLocations: nil},
	}
}

func getVulnerableResults() []api.PackageVersion {
	vulnerableSmolToml := api.PackageVersion{
		Version: "1.3.0",
		Library: api.Package{NormalizedName: "smol-toml", Name: "smol-toml", PackageManager: mappings.NpmManager},
		OpenVulnerabilities: []api.Vulnerability{
			{GitHubAdvisoryID: "GHSA-pqhp-25j4-6hq9"},
		},
	}
	vulnerableEjs := api.PackageVersion{
		Version: "3.1.10",
		Library: api.Package{NormalizedName: "ejs", Name: "ejs", PackageManager: mappings.NpmManager},
		OpenVulnerabilities: []api.Vulnerability{
			{CVE: "CVE-2024-33883", GitHubAdvisoryID: "GHSA-ghr5-ch3p-vcr6"},
			{CVE: "CVE-dummy-open-vuln", GitHubAdvisoryID: "GHSA-dummy-open-vuln"},
		},
	}
	vulnerableEjsAnotherVersion := api.PackageVersion{
		Version: "3.1.9",
		Library: api.Package{NormalizedName: "ejs", Name: "ejs", PackageManager: mappings.NpmManager},
		OpenVulnerabilities: []api.Vulnerability{
			{CVE: "CVE-2024-33883", GitHubAdvisoryID: "GHSA-ghr5-ch3p-vcr6"},
			{CVE: "CVE-dummy-open-vuln", GitHubAdvisoryID: "GHSA-dummy-open-vuln"},
		},
	}
	return []api.PackageVersion{
		vulnerableSmolToml,
		vulnerableEjs,
		vulnerableEjsAnotherVersion,
	}
}

func TestHandleAppliedFixes(t *testing.T) {
	tests := []struct {
		name          string
		fixes         []shared.DependencyDescriptor
		vulnerable    []api.PackageVersion
		config        config.OxConfig
		expectedError string
		expectedCalls map[string]int
		responseBody  string
		statusCode    int
	}{
		{
			name:       "successful exclusion of all vulnerabilities",
			fixes:      getFixAndScanResult(),
			vulnerable: getVulnerableResults(),
			config: config.OxConfig{
				Url:                          "https://test.ox.security",
				Token:                        config.SensitiveString("test-token"),
				Application:                  "test-app",
				ExcludeWhenHighCriticalFixed: true,
			},
			expectedError: "",
			expectedCalls: map[string]int{
				"https://test.ox.security": 2,
			},
			responseBody: `{
				"data": {
					"getIssues": {
						"issues": [
							{
								"id": "1",
								"issueId": "issue-1",
								"mainTitle": "Test Issue",
								"severity": "HIGH",
								"app": {
									"id": "app-1",
									"name": "test-app"
								},
								"category": {
									"name": "Security",
									"categoryId": 1
								},
								"scaVulnerabilities": [
									{
										"cve": "CVE-2023-1234",
										"oxSeverity": "HIGH",
										"libName": "smol-toml",
										"libVersion": "1.3.0"
									}
								],
								"comment": "Seal bot remediation notes:\nCVE-2023-1234 was fixed in package smol-toml by using version 1.3.0+sp1\n\nThe Seal bot excluded the issue since all existing vulnerabilities were remediated"
							}
						],
						"totalIssues": 1,
						"totalFilteredIssues": 1
					}
				}
			}`,
			statusCode: 200,
		},
		{
			name:       "api error when getting issues",
			fixes:      getFixAndScanResult(),
			vulnerable: getVulnerableResults(),
			config: config.OxConfig{
				Url:                          "https://test.ox.security",
				Token:                        config.SensitiveString("test-token"),
				Application:                  "test-app",
				ExcludeWhenHighCriticalFixed: false,
			},
			expectedError: "failed to get relevant issues",
			expectedCalls: map[string]int{
				"https://test.ox.security": 1,
			},
			responseBody: `{"error": "Internal Server Error"}`,
			statusCode:   500,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			transport := &fakeRoundTripper{
				statusCode: map[string]int{
					tt.config.Url: tt.statusCode,
				},
				jsonContent: map[string]string{
					tt.config.Url: tt.responseBody,
				},
				Validator: func(req *http.Request) {
					if req.Header.Get("Authorization") != string(tt.config.Token) {
						t.Errorf("expected Authorization header '%s', got %s", tt.config.Token, req.Header.Get("Authorization"))
					}
					if req.Header.Get("Content-Type") != "application/json" {
						t.Errorf("expected Content-Type header 'application/json', got %s", req.Header.Get("Content-Type"))
					}
				},
			}

			httpClient := http.Client{Transport: transport}
			oxClient := OxClient{
				Client:                       httpClient,
				Url:                          tt.config.Url,
				Token:                        tt.config.Token.Value(),
				Application:                  tt.config.Application,
				ExcludeWhenHighCriticalFixed: tt.config.ExcludeWhenHighCriticalFixed,
			}

			err := handleAppliedFixes(&oxClient, tt.fixes, tt.vulnerable)
			if tt.expectedError != "" {
				if err == nil {
					t.Fatalf("expected error containing '%s', got nil", tt.expectedError)
				}
				if !strings.Contains(err.Error(), tt.expectedError) {
					t.Errorf("expected error to contain '%s', got '%s'", tt.expectedError, err.Error())
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			}

			transport.CheckUrlCounter(t, tt.expectedCalls)
		})
	}
}

func TestProcessIssueVulnerabilities(t *testing.T) {
	tests := []struct {
		name                string
		issue               Issue
		fixedVulns          vulnerabilityMapping
		excludeHighCritical bool
		expectedExclude     bool
		expectedComment     string
	}{
		{
			name: "all vulnerabilities fixed",
			issue: Issue{
				ScaVulnerabilities: []ScaVulnerability{
					{
						Cve:        "CVE-2024-1234",
						OxSeverity: severityHigh,
						LibName:    "test-lib",
						LibVersion: "1.0.0",
					},
					{
						Cve:        "CVE-2024-5678",
						OxSeverity: severityCritical,
						LibName:    "test-lib",
						LibVersion: "1.0.0",
					},
				},
			},
			fixedVulns: vulnerabilityMapping{
				"test-lib/1.0.0/cve-2024-1234": "1.0.0+sp1",
				"test-lib/1.0.0/cve-2024-5678": "1.0.0+sp1",
			},
			excludeHighCritical: true,
			expectedExclude:     true,
			expectedComment:     "Seal bot remediation notes:\nCVE-2024-1234 was fixed in package test-lib by using version 1.0.0+sp1\nCVE-2024-5678 was fixed in package test-lib by using version 1.0.0+sp1\n\nThe Seal bot excluded the issue since all existing vulnerabilities were remediated",
		},
		{
			name: "some vulnerabilities fixed",
			issue: Issue{
				ScaVulnerabilities: []ScaVulnerability{
					{
						Cve:        "CVE-2024-1234",
						OxSeverity: severityHigh,
						LibName:    "test-lib",
						LibVersion: "1.0.0",
					},
					{
						Cve:        "CVE-2024-5678",
						OxSeverity: severityCritical,
						LibName:    "test-lib",
						LibVersion: "1.0.0",
					},
				},
			},
			fixedVulns: vulnerabilityMapping{
				"test-lib/1.0.0/cve-2024-1234": "1.0.0+sp1",
			},
			excludeHighCritical: true,
			expectedExclude:     false,
			expectedComment:     "Seal bot remediation notes:\nCVE-2024-1234 was fixed in package test-lib by using version 1.0.0+sp1\n",
		},
		{
			name: "no vulnerabilities fixed",
			issue: Issue{
				ScaVulnerabilities: []ScaVulnerability{
					{
						Cve:        "CVE-2024-1234",
						OxSeverity: severityHigh,
						LibName:    "test-lib",
						LibVersion: "1.0.0",
					},
					{
						Cve:        "CVE-2024-5678",
						OxSeverity: severityCritical,
						LibName:    "test-lib",
						LibVersion: "1.0.0",
					},
				},
			},
			fixedVulns:          vulnerabilityMapping{},
			excludeHighCritical: true,
			expectedExclude:     false,
			expectedComment:     "Seal bot remediation notes:\n",
		},
		{
			name: "fixedVulns contains more vulnerabilities than issue",
			issue: Issue{
				ScaVulnerabilities: []ScaVulnerability{
					{
						Cve:        "CVE-2024-1234",
						OxSeverity: severityHigh,
						LibName:    "test-lib",
						LibVersion: "1.0.0",
					},
				},
			},
			fixedVulns: vulnerabilityMapping{
				"test-lib/1.0.0/cve-2024-1234": "1.0.0+sp1",
				"test-lib/1.0.0/cve-2024-5678": "1.0.0+sp1",
				"test-lib/1.0.0/cve-2024-5679": "1.0.0+sp1",
				"test-lib/1.0.0/cve-2024-5680": "1.0.0+sp1",
			},
			excludeHighCritical: true,
			expectedExclude:     true,
			expectedComment:     "Seal bot remediation notes:\nCVE-2024-1234 was fixed in package test-lib by using version 1.0.0+sp1\n\nThe Seal bot excluded the issue since all existing vulnerabilities were remediated",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isExcluded, comment := processIssueVulnerabilities(tt.issue, tt.fixedVulns, tt.excludeHighCritical)
			if isExcluded != tt.expectedExclude {
				t.Errorf("expected exclude %v, got %v", tt.expectedExclude, isExcluded)
			}
			if comment != tt.expectedComment {
				t.Errorf("expected comment %q, got %q", tt.expectedComment, comment)
			}
		})
	}
}
