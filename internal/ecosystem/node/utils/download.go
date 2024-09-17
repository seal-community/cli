package utils

import (
	"cli/internal/api"
	"cli/internal/common"
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

func DownloadNPMPackage(s api.ArtifactServer, name string, version string) ([]byte, error) {
	defer common.ExecutionTimer().Log()
	var libraryInfo npmLibraryInfo

	statusCode, err := s.GetJsonObject(
		fmt.Sprintf("%s/", name),
		nil,
		nil,
		&libraryInfo,
	)

	if err != nil {
		slog.Error("failed sending request for npm libary info", "err", err, "name", name, "version", version)
		return nil, err
	}

	if statusCode != 200 {
		slog.Error("bad response code for npm package", "err", err, "status", statusCode)
		return nil, fmt.Errorf("bad status code for npm info: %d", statusCode)
	}

	versionInfo, ok := libraryInfo.Versions[version]
	if !ok {
		slog.Error("failed finding fixed package")
		return nil, fmt.Errorf("could not find version %s in package info %s", version, name)
	}

	tarUrl, err := url.Parse(versionInfo.Distribution.Tarball)
	if err != nil {
		slog.Error("failed parsing tarball url", "raw", versionInfo.Distribution.Tarball)
		return nil, fmt.Errorf("could not parse package url: %s", versionInfo.Distribution.Tarball)
	}

	libraryData, statusCode, err := s.Get(
		tarUrl.RequestURI(), // has to be relative to the artifact server base
		nil,
		nil,
	)

	if err != nil {
		slog.Error("failed sending request for npm package data", "err", err, "name", name, "version", version)
		return nil, err
	}

	if statusCode != 200 {
		slog.Error("bad response code for npm package payload", "err", err, "status", statusCode)
		return nil, fmt.Errorf("bad status code for npm package data; status: %d", statusCode)
	}

	if len(libraryData) == 0 {
		slog.Error("no payload content from server")
		return nil, fmt.Errorf("no package content")
	}

	shaBytes := sha1.Sum(libraryData)
	calcSha1 := hex.EncodeToString(shaBytes[:])
	if calcSha1 != versionInfo.Distribution.Sha1sum {
		return nil, fmt.Errorf("wrong checksum for package; expected: %s ; got %s", versionInfo.Distribution.Sha1sum, calcSha1)
	}

	return libraryData, nil
}
