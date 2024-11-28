package utils

import (
	"cli/internal/api"
	"cli/internal/common"
	"cli/internal/ecosystem/shared"
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

	libraryData, err := shared.DownloadFile(s, fmt.Sprintf("%s/%s/%s/%s", orgName, artifactName, version, filename))
	if err != nil {
		slog.Error("failed getting maven package data", "err", err, "name", name, "version", version)
		return nil, "", err
	}

	// Check sha1sum
	librarySha1, err := shared.DownloadFile(s, fmt.Sprintf("%s/%s/%s/%s", orgName, artifactName, version, sha1Filename))
	if err != nil {
		slog.Error("failed getting maven package sha1sum", "err", err, "name", name, "version", version)
		return nil, "", err
	}

	shaBytes := sha1.Sum(libraryData)
	calcSha1 := hex.EncodeToString(shaBytes[:])
	if calcSha1 != string(librarySha1) {
		return nil, "", fmt.Errorf("wrong checksum for package; expected: %s ; got %s", librarySha1, calcSha1)
	}

	return libraryData, filename, nil
}
