package utils

import (
	"cli/internal/api"
	"cli/internal/common"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"log/slog"
)

type npmLibraryInfo struct {
	Versions map[string]struct {
		Distribution struct {
			Tarball string `json:"tarball"`
			Sha1sum string `json:"shasum"`
		} `json:"dist"`
	} `json:"versions"`
}

func DownloadNPMPackage(s api.Server, name string, version string) ([]byte, error) {
	defer common.ExecutionTimer().Log()

	authHeader := api.BuildBasicAuthHeader(s.AuthToken)
	libraryInfo, statusCode, err := api.SendSealRequestJson[any, npmLibraryInfo](
		s.Client,
		"GET",
		fmt.Sprintf("https://npm.sealsecurity.io/%s/", name),
		nil,
		[]api.StringPair{authHeader},
		[]api.StringPair{},
	)

	if err != nil {
		slog.Error("failed sending request for npm libary info", "err", err, "name", name, "version", version)
		return nil, err
	}

	if statusCode != 200 {
		slog.Error("bad response code for npm package", "err", err, "status", statusCode)
		return nil, fmt.Errorf("bad status code for npm info: %d", statusCode)
	}

	if libraryInfo == nil {
		slog.Error("no content for package description", "status", statusCode)
		return nil, fmt.Errorf("no data from server: %d", statusCode)
	}

	versionInfo, ok := libraryInfo.Versions[version]
	if !ok {
		slog.Error("failed finding fixed package")
		return nil, fmt.Errorf("could not find version %s in package info %s", version, name)
	}

	url := versionInfo.Distribution.Tarball
	libraryData, statusCode, err := api.SendSealRequest[any](
		s.Client,
		"GET",
		url,
		nil,
		[]api.StringPair{authHeader},
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
		return nil, fmt.Errorf("wrong checksum for package; expected: %s ; got %s", calcSha1, versionInfo.Distribution.Sha1sum)
	}

	return libraryData, nil
}
