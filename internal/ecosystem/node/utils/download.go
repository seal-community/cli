package utils

import (
	"cli/internal/api"
	"cli/internal/common"
	"cli/internal/ecosystem/shared"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/url"
)

type npmLibraryInfo struct {
	Versions map[string]struct {
		Distribution struct {
			Tarball string `json:"tarball"`
			Sha1sum string `json:"shasum"`
		} `json:"dist"`
	} `json:"versions"`
}

func DownloadNPMPackage(s api.ArtifactServer, name string, version string) ([]byte, string, error) {
	defer common.ExecutionTimer().Log()
	var libraryInfo npmLibraryInfo
	filename := ""
	statusCode, err := s.GetJsonObject(
		fmt.Sprintf("%s/", name),
		nil,
		nil,
		&libraryInfo,
	)

	if err != nil {
		slog.Error("failed sending request for npm libary info", "err", err, "name", name, "version", version)
		return nil, filename, err
	}

	if statusCode != 200 {
		slog.Error("bad response code for npm package", "err", err, "status", statusCode)
		return nil, filename, fmt.Errorf("bad status code for npm info: %d", statusCode)
	}

	versionInfo, ok := libraryInfo.Versions[version]
	if !ok {
		slog.Error("failed finding fixed package")
		return nil, filename, fmt.Errorf("could not find version %s in package info %s", version, name)
	}

	tarUrl, err := url.Parse(versionInfo.Distribution.Tarball)
	if err != nil {
		slog.Error("failed parsing tarball url", "raw", versionInfo.Distribution.Tarball)
		return nil, filename, fmt.Errorf("could not parse package url: %s", versionInfo.Distribution.Tarball)
	}

	libraryData, err := shared.DownloadFile(s, tarUrl.RequestURI())
	if err != nil {
		slog.Error("failed getting npm package data", "err", err, "name", name, "version", version)
		return nil, filename, err
	}

	shaBytes := sha1.Sum(libraryData)
	calcSha1 := hex.EncodeToString(shaBytes[:])
	if calcSha1 != versionInfo.Distribution.Sha1sum {
		return nil, filename, fmt.Errorf("wrong checksum for package; expected: %s ; got %s", versionInfo.Distribution.Sha1sum, calcSha1)
	}

	filename = common.FileNameFromUri(versionInfo.Distribution.Tarball)

	return libraryData, filename, nil
}
