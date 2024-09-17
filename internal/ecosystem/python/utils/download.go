package utils

import (
	"bytes"
	"cli/internal/api"
	"cli/internal/common"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net/url"
	"strings"

	"golang.org/x/net/html"
)

type urlCandidates struct {
	whls   []url.URL
	targzs []url.URL
}

func getVersionUrlCandiates(libraryInfo []byte, version string) (urlCandidates, error) {
	infoHtml := html.NewTokenizer(bytes.NewReader(libraryInfo))
	whls := make([]url.URL, 0)
	targzs := make([]url.URL, 0)
	for {
		tokenType := infoHtml.Next()
		if tokenType == html.ErrorToken {
			break
		}
		if tokenType != html.StartTagToken {
			continue
		}
		token := infoHtml.Token()
		if token.Data != "a" {
			continue
		}

		for _, attr := range token.Attr {
			if attr.Key != "href" {
				continue
			}
			u, err := url.Parse(attr.Val)
			if err != nil {
				slog.Error("failed parsing url", "err", err, "url", attr.Val)
				continue
			}
			if !strings.Contains(u.Path, version) {
				continue
			}

			switch {
			case strings.HasSuffix(u.Path, ".whl"):
				whls = append(whls, *u)
			case strings.HasSuffix(u.Path, ".tar.gz"):
				targzs = append(targzs, *u)
			}
		}
	}
	return urlCandidates{
		whls:   whls,
		targzs: targzs,
	}, nil
}

func getVersionUrl(libraryInfo []byte, version string, compatibleTags []string, OnlyBinary bool) (*url.URL, error) {
	candidates, err := getVersionUrlCandiates(libraryInfo, version)
	if err != nil {
		return nil, err
	}

	// most fitting wheel comes first
	for _, tag := range compatibleTags {
		for _, u := range candidates.whls {
			if strings.Contains(u.Path, tag) {
				return &u, nil
			}
		}
	}

	slog.Debug("no compatible wheel found", "version", version, "tags", compatibleTags)

	if OnlyBinary {
		return nil, common.NewPrintableError("no compatible wheel found and configuration is set to only install wheels")
	}

	if len(candidates.targzs) == 0 {
		return nil, fmt.Errorf("no tar.gz files found")
	}
	if len(candidates.targzs) > 1 {
		slog.Warn("multiple tar.gz files found", "version", version, "tags", compatibleTags)
		return nil, fmt.Errorf("multiple tar.gz files found")
	}

	return &candidates.targzs[0], nil
}

func DownloadPythonPackage(s api.ArtifactServer, name string, version string, compatibleTags []string, OnlyBinary bool) ([]byte, error) {
	defer common.ExecutionTimer().Log()

	libraryInfo, statusCode, err := s.Get(
		fmt.Sprintf("simple/%s/", name),
		nil,
		nil,
	)

	if err != nil {
		slog.Error("failed sending request for pypi libary info", "err", err, "name", name, "version", version)
		return nil, err
	}

	if statusCode != 200 {
		slog.Error("bad response code for pypi package", "err", err, "status", statusCode)
		return nil, fmt.Errorf("bad status code for pypi info: %d", statusCode)
	}

	if len(libraryInfo) == 0 {
		slog.Error("no content for package description", "status", statusCode)
		return nil, fmt.Errorf("no data from server: %d", statusCode)
	}

	versionDownloadUrl, err := getVersionUrl(libraryInfo, version, compatibleTags, OnlyBinary)
	if err != nil {
		slog.Error("failed finding version url", "err", err, "name", name, "version", version, "tags", compatibleTags)
		return nil, err
	}
	slog.Info("found version url", "url", versionDownloadUrl.String())

	libraryData, statusCode, err := s.Get(
		versionDownloadUrl.RequestURI(), // must be relative, in case we use different server
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

	shaBytes := sha256.Sum256(libraryData)
	calcSha256 := hex.EncodeToString(shaBytes[:])

	expectedSha256 := strings.TrimPrefix(versionDownloadUrl.Fragment, "sha256=")

	if calcSha256 != expectedSha256 {
		return nil, fmt.Errorf("wrong checksum for package; expected: %s ; got %s", expectedSha256, calcSha256)
	}

	return libraryData, nil
}
