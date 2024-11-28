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

func buildUri(name string, version string, arch string) string {
	os := getOsVersion(version)
	_, noEpochVersion := common.GetNoEpochVersion(version)

	return fmt.Sprintf("centos/%s/%s/Packages/%s-%s.%s.rpm", os, arch, name, noEpochVersion, arch)
}

func DownloadRpmPackage(s api.ArtifactServer, name string, version string, arch string) ([]byte, string, error) {
	defer common.ExecutionTimer().Log()

	uri := buildUri(name, version, arch)
	filename := common.FileNameFromUri(uri)
	libraryData, err := shared.DownloadFile(s, uri)

	if err != nil {
		slog.Error("failed sending request for rpm package data", "err", err, "name", name, "version", version, "arch", arch)
		return nil, "", err
	}

	return libraryData, filename, nil
}
