package utils

import (
	"path/filepath"
	"testing"
)

func TestSplitJavaPackageName(t *testing.T) {
	tests := []struct {
		name          string
		expectedOrg   string
		expectedName  string
		expectedError bool
	}{
		{"com.example:package", "com.example", "package", false},
		{"com.example:pack:age", "", "", true},
		{"com.example_package", "", "", true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			org, name, err := SplitJavaPackageName(test.name)
			if org != test.expectedOrg {
				t.Fatalf("wrong org, expected: `%s` got: `%s`", test.expectedOrg, org)
			}
			if name != test.expectedName {
				t.Fatalf("wrong name, expected: `%s` got: `%s`", test.expectedName, name)
			}
			if (err == nil) == test.expectedError {
				t.Fatalf("wrong error, expected: `%v` got: `%v`", test.expectedError, err)
			}
		})
	}
}

func TestGetJavaPackagePath(t *testing.T) {
	tests := []struct {
		cacheDir    string
		packageName string
		version     string
		expected    string
	}{
		{"cache", "com.example:package", "1.2.3", filepath.FromSlash("cache/com/example/package/1.2.3")},
		{"cache", "com.example:pack:age", "1.2.3", ""},
	}
	for _, test := range tests {
		t.Run(test.packageName, func(t *testing.T) {
			result := GetJavaPackagePath(test.cacheDir, test.packageName, test.version)
			if result != test.expected {
				t.Fatalf("wrong result, expected: `%s` got: `%s`", test.expected, result)
			}
		})
	}
}

func TestCreateJavaPackageInfo(t *testing.T) {
	tests := []struct {
		identifier           string
		expectedOrgName      string
		expectedArtifactName string
		expectedVersion      string
		expectedScope        string
		expectedError        bool
	}{
		{"com.example:package:jar:1.2.3:compile", "com.example", "package", "1.2.3", "compile", false},
		{"com.example:package:jar:1.2.3", "com.example", "package", "1.2.3", "", false},
		{"com.example:package:jar", "com.example", "package", "", "", false},
		{"com.example:package", "com.example", "package", "", "", false},
		{"com.example:package:jar:1.2.3:compile:stillworks", "com.example", "package", "1.2.3", "compile", false},
		{"com.example", "", "", "", "", true},
	}
	for _, test := range tests {
		t.Run(test.identifier, func(t *testing.T) {
			result, err := CreateJavaPackageInfo(test.identifier)
			if err != nil {
				if !test.expectedError {
					t.Fatalf("wrong result, expected error: %v, got: %v", test.expectedError, err)
				}
			} else {
				if result.OrgName != test.expectedOrgName {
					t.Fatalf("wrong result, expected: `%s` got: `%s`", test.expectedOrgName, result.OrgName)
				}
				if result.ArtifactName != test.expectedArtifactName {
					t.Fatalf("wrong result, expected: `%s` got: `%s`", test.expectedArtifactName, result.ArtifactName)
				}
				if result.Version != test.expectedVersion {
					t.Fatalf("wrong result, expected: `%s` got: `%s`", test.expectedVersion, result.Version)
				}
				if result.Scope != test.expectedScope {
					t.Fatalf("wrong result, expected: `%s` got: `%s`", test.expectedScope, result.Scope)
				}
			}
		})
	}
}

func TestGetPackageFileName(t *testing.T) {
	tests := []struct {
		artifactName  string
		version 			string
		expected    	string
	}{
		{"example-app", "1.2.3", "example-app-1.2.3.jar"},
		{"example.app", "1.2.3+sp1", "example.app-1.2.3+sp1.jar"},
	}
	for _, test := range tests {
		t.Run(test.artifactName, func(t *testing.T) {
			result := GetPackageFileName(test.artifactName, test.version)
			if result != test.expected {
				t.Fatalf("wrong result, expected: `%s` got: `%s`", test.expected, result)
			}
		})
	}
}

func TestOrgNameToPath(t *testing.T) {
	tests := []struct {
		orgName  string
		expected string
	}{
		{"com.example.app", filepath.FromSlash("com/example/app")},
		{"com.exam-ple.app", filepath.FromSlash("com/exam-ple/app")},
	}
	for _, test := range tests {
		t.Run(test.orgName, func(t *testing.T) {
			result := OrgNameToPath(test.orgName)
			if result != test.expected {
				t.Fatalf("wrong result, expected: `%s` got: `%s`", test.expected, result)
			}
		})
	}
}

func TestOrgNameToUrlPath(t *testing.T) {
	tests := []struct {
		orgName  string
		expected string
	}{
		{"com.example.app", "com/example/app"},
		{"com.exam-ple.app", "com/exam-ple/app"},
	}
	for _, test := range tests {
		t.Run(test.orgName, func(t *testing.T) {
			result := OrgNameToUrlPath(test.orgName)
			if result != test.expected {
				t.Fatalf("wrong result, expected: `%s` got: `%s`", test.expected, result)
			}
		})
	}
}

