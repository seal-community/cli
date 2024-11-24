package utils

import (
	"cli/internal/api"
	"cli/internal/common"
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

// Return version with no epoch, e.g. 1:1.2.3-4.el7 -> 1.2.3-4.el7
func getNoEpochVersion(libraryVersion string) string {
	reg := regexp.MustCompile(`^(\d+:)?(.*)`)
	matches := reg.FindStringSubmatch(libraryVersion)
	if len(matches) < 3 {
		return ""
	}
	return matches[2]
}

func buildUri(name string, version string, arch string) string {
	os := getOsVersion(version)
	noEpochVersion := getNoEpochVersion(version)

	return fmt.Sprintf("centos/%s/%s/Packages/%s-%s.%s.rpm", os, arch, name, noEpochVersion, arch)
}

func DownloadRpmPackage(s api.ArtifactServer, name string, version string, arch string) ([]byte, string, error) {
	defer common.ExecutionTimer().Log()

	uri := buildUri(name, version, arch)
	filename := common.FileNameFromUri(uri)

	libraryData, statusCode, err := s.Get(
		uri,
		nil,
		nil,
	)

	if err != nil {
		slog.Error("failed sending request for golang package data", "err", err, "name", name, "version", version)
		return nil, "", err
	}

	if statusCode != 200 {
		slog.Error("bad response code for golang package payload", "err", err, "status", statusCode)
		return nil, "", fmt.Errorf("bad status code golang package data; status: %d", statusCode)
	}

	if len(libraryData) == 0 {
		slog.Error("no payload content from server")
		return nil, "", fmt.Errorf("no package content")
	}

	return libraryData, filename, nil
}
