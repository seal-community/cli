package utils

import (
	"cli/internal/common"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
)

type JavaPackageInfo struct {
	OrgName      string
	ArtifactName string
	Version      string // optional
	Scope        string // optional
}

// almost the reverse of CreateJavaPackageInfo, but no artifact type ('jar')
func (i *JavaPackageInfo) Id() string {

	if i.OrgName == "" || i.ArtifactName == "" {
		slog.Error("bad package info", "package-info", i)
		return ""
	}

	if i.Scope != "" && i.Version != "" {
		return fmt.Sprintf("%s:%s:%s:%s", i.OrgName, i.ArtifactName, i.Version, i.Scope)
	}

	if i.Version != "" {
		return fmt.Sprintf("%s:%s:%s", i.OrgName, i.ArtifactName, i.Version)
	}

	return fmt.Sprintf("%s:%s", i.OrgName, i.ArtifactName)
}

// identifier format: orgName:artifactName:packageType:version:scope, for example:
// org.apache.commons:commons-lang3:jar:3.11:compile will return {org.apache.commons, commons-lang3, 3.11, compile}, nil
func CreateJavaPackageInfo(identifier string) (*JavaPackageInfo, error) {
	parts := strings.Split(identifier, ":")

	if len(parts) < 2 {
		slog.Error("failed parsing package info from identifier", "identifier", identifier)
		return nil, common.NewPrintableError("invalid package identifier: %s", identifier)
	}

	version := ""
	if len(parts) >= 4 {
		version = parts[3]
	}

	buildScope := ""
	if len(parts) >= 5 {
		buildScope = parts[4]
	}
	return &JavaPackageInfo{OrgName: parts[0], ArtifactName: parts[1], Scope: buildScope, Version: version}, nil
}

func GetCacheDir(projectDir string) string {
	args := []string{"help:evaluate", "-Dexpression=settings.localRepository", "-q", "-DforceStdout"}
	result, err := common.RunCmdWithArgs(projectDir, MavenExeName, args...)
	if err != nil {
		return ""
	}

	if result.Code != 0 {
		// maven outputs errors to stdout
		slog.Error("getting cache dir using maven command failed", "err", result.Stderr, "out", result.Stdout, "exitcode", result.Code)
		return ""
	}

	slog.Info("maven cache dir: ", "dir", result.Stdout)
	return result.Stdout
}

// package name format: orgName:artifactName, for example:
// org.apache.commons:commons-lang3 will return org.apache.commons, commons-lang3, nil
func SplitJavaPackageName(name string) (orgName string, artifactName string, err error) {
	parts := strings.Split(name, ":")
	if len(parts) != 2 {
		return "", "", common.NewPrintableError("invalid package name: %s", name)
	}
	return parts[0], parts[1], nil
}

// returns cacheDir/orgName(split by /)/artifactName/version, for example:
// for cacheDir: /tmp/cache, packageName: org.apache.commons:commons-lang3, version: 3.11
// will return: /tmp/cache/org/apache/commons/commons-lang3/3.11
func GetJavaPackagePath(cacheDir string, packageName string, version string) string {
	orgName, artifactName, err := SplitJavaPackageName(packageName)
	if err != nil {
		slog.Error("Failed to split package name")
		return ""
	}

	orgName = OrgNameToPath(orgName)

	return filepath.Join(cacheDir, orgName, artifactName, version)
}

// example: artifactName: commons-lang3, version: 3.11
// return: commons-lang3-3.11.jar
func GetPackageFileName(artifactName string, version string) string {
	return fmt.Sprintf("%s-%s.jar", artifactName, version)
}

// GetOrgNameToPath returns the path to a Java package in the cache according to a given orgName
// will return the orgName(split by / - or \ for Windows), for example:
// for orgName: org.apache.commons will return: org/apache/commons or org\apache\commons for windows.
func OrgNameToPath(orgName string) string {
	return filepath.FromSlash(strings.Replace(orgName, ".", "/", -1))
}

// will return the orgName(split by /), for example:
// for orgName: org.apache.commons will return: org/apache/commons
func OrgNameToUrlPath(orgName string) string {
	return strings.Replace(orgName, ".", "/", -1)
}

func FormatJavaPackageName(org, artifact string) string {
	return fmt.Sprintf("%s:%s", org, artifact)
}
