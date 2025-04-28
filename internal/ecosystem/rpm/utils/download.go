package utils

import (
	"cli/internal/api"
	"cli/internal/common"
	"cli/internal/ecosystem/shared"
	"fmt"
	"log/slog"
	"regexp"
)

// Return numeric value of centos version, e.g. 1.2.3-4.el7 -> 7
func getOsVersion(libraryVersion string) string {
	reg := regexp.MustCompile(`\.el(\d+)`)
	matches := reg.FindStringSubmatch(libraryVersion)
	if len(matches) < 2 {
		return ""
	}
	return matches[1]
}

// Build the URI for the RPM package
// The URI format is:
// centos/<os>/<arch>/Packages/<prefixMode>/<name>-<version>.<arch>.rpm
// e.g. centos/7/x86_64/Packages/noprefix/openssl-1.0.2k-19.el7.x86_64.rpm
// and for prefixed packages - centos/7/x86_64/Packages/openssl-1.0.2k-19.el7.x86_64.rpm
func buildUri(name string, version string, arch string, useSealedName bool) string {
	os := getOsVersion(version)
	prefixMode := ""
	if !useSealedName {
		prefixMode = "noprefix/"
	}
	_, noEpochVersion := common.GetNoEpochVersion(version)

	return fmt.Sprintf("centos/%s/%s/Packages/%s%s-%s.%s.rpm", os, arch, prefixMode, name, noEpochVersion, arch)
}

func DownloadRpmPackage(s api.ArtifactServer, name string, version string, arch string, useSealedName bool) ([]byte, string, error) {
	defer common.ExecutionTimer().Log()

	uri := buildUri(name, version, arch, useSealedName)
	filename := common.FileNameFromUri(uri)
	libraryData, err := shared.DownloadFile(s, uri)

	if err != nil {
		slog.Error("failed sending request for rpm package data", "err", err, "name", name, "version", version, "arch", arch)
		return nil, "", err
	}

	return libraryData, filename, nil
}
