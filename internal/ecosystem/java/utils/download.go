package utils

import (
	"cli/internal/api"
	"cli/internal/common"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"log/slog"
)

func DownloadMavenPackage(s api.ArtifactServer, name string, version string) ([]byte, string, error) {
	defer common.ExecutionTimer().Log()

	orgName, artifactName, err := SplitJavaPackageName(name)
	if err != nil {
		slog.Error("failed to split package name", "err", err)
		return nil, "", err
	}

	orgName = OrgNameToUrlPath(orgName)
	filename := GetPackageFileName(artifactName, version)
	sha1Filename := fmt.Sprintf("%s.sha1", filename)

	libraryData, statusCode, err := s.Get(
		fmt.Sprintf("%s/%s/%s/%s", orgName, artifactName, version, filename),
		nil, nil,
	)

	if err != nil {
		slog.Error("failed sending request for maven package data", "err", err, "name", name, "version", version)
		return nil, "", err
	}

	if statusCode != 200 {
		slog.Error("bad response code for maven package payload", "err", err, "status", statusCode)
		return nil, "", fmt.Errorf("bad status code for maven package data; status: %d", statusCode)
	}

	if len(libraryData) == 0 {
		slog.Error("no payload content from server")
		return nil, "", fmt.Errorf("no package content")
	}

	// Check sha1sum

	librarySha1, statusCode, err := s.Get(
		fmt.Sprintf("%s/%s/%s/%s", orgName, artifactName, version, sha1Filename),
		nil, nil,
	)

	if err != nil {
		slog.Error("failed sending request for maven package sha1", "err", err, "name", name, "version", version)
		return nil, "", err
	}

	if statusCode != 200 {
		slog.Error("bad response code for maven package sha1", "err", err, "status", statusCode)
		return nil, "", fmt.Errorf("bad status code for maven package sh1; status: %d", statusCode)
	}

	if len(librarySha1) == 0 {
		slog.Error("no sha1 file content from server")
		return nil, "", fmt.Errorf("no sha1 content")
	}

	shaBytes := sha1.Sum(libraryData)
	calcSha1 := hex.EncodeToString(shaBytes[:])
	if calcSha1 != string(librarySha1) {
		return nil, "", fmt.Errorf("wrong checksum for package; expected: %s ; got %s", librarySha1, calcSha1)
	}

	return libraryData, filename, nil
}
