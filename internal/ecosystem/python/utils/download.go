package utils

import (
	"bytes"
	"cli/internal/api"
	"cli/internal/common"
	"cli/internal/ecosystem/shared"
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

func DownloadPythonPackage(s api.ArtifactServer, name string, version string, compatibleTags []string, OnlyBinary bool) ([]byte, string, error) {
	defer common.ExecutionTimer().Log()

	libraryInfo, err := shared.DownloadFile(s, fmt.Sprintf("simple/%s/", name))
	if err != nil {
		slog.Error("failed sending request for pypi libary info", "err", err, "name", name, "version", version)
		return nil, "", err
	}

	versionDownloadUrl, err := getVersionUrl(libraryInfo, version, compatibleTags, OnlyBinary)
	if err != nil {
		slog.Error("failed finding version url", "err", err, "name", name, "version", version, "tags", compatibleTags)
		return nil, "", err
	}
	slog.Info("found version url", "url", versionDownloadUrl.String())

	libraryData, err := shared.DownloadFile(s, versionDownloadUrl.RequestURI())
	if err != nil {
		slog.Error("failed getting python package data", "err", err, "name", name, "version", version)
		return nil, "", err
	}

	shaBytes := sha256.Sum256(libraryData)
	calcSha256 := hex.EncodeToString(shaBytes[:])

	expectedSha256 := strings.TrimPrefix(versionDownloadUrl.Fragment, "sha256=")

	if calcSha256 != expectedSha256 {
		return nil, "", fmt.Errorf("wrong checksum for package; expected: %s ; got %s", expectedSha256, calcSha256)
	}

	return libraryData, common.FileNameFromURL(versionDownloadUrl), nil
}
