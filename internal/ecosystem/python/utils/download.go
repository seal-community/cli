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

func getVersionUrl(libraryInfo []byte, version string, compatibleTags []string) (url.URL, error) {
	infoHtml := html.NewTokenizer(bytes.NewReader(libraryInfo))
	wheelUrls := make([]url.URL, 0)
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
			if !strings.HasSuffix(u.Path, ".whl") {
				continue
			}

			wheelUrls = append(wheelUrls, *u)
		}
	}

	for _, tag := range compatibleTags {
		for _, u := range wheelUrls {
			if strings.Contains(u.Path, tag) {
				return u, nil
			}
		}
	}

	return url.URL{}, fmt.Errorf("failed finding version url")
}

func DownloadPythonPackage(s api.Server, name string, version string, compatibleTags []string) ([]byte, error) {
	defer common.ExecutionTimer().Log()

	authHeader := api.BuildBasicAuthHeader(s.AuthToken)
	libraryInfo, statusCode, err := api.SendSealRequest[any](
		s.Client,
		"GET",
		fmt.Sprintf("%s/simple/%s/", api.PypiServer, name),
		nil,
		[]api.StringPair{authHeader},
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

	if libraryInfo == nil {
		slog.Error("no content for package description", "status", statusCode)
		return nil, fmt.Errorf("no data from server: %d", statusCode)
	}

	versionDownloadUrl, err := getVersionUrl(libraryInfo, version, compatibleTags)
	if err != nil {
		slog.Error("failed finding version url", "err", err, "name", name, "version", version, "tags", compatibleTags)
		return nil, err
	}
	slog.Info("found version url", "url", versionDownloadUrl.String())

	libraryData, statusCode, err := api.SendSealRequest[any](
		s.Client,
		"GET",
		versionDownloadUrl.String(),
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

	shaBytes := sha256.Sum256(libraryData)
	calcSha256 := hex.EncodeToString(shaBytes[:])

	expectedSha256 := strings.TrimPrefix(versionDownloadUrl.Fragment, "sha256=")

	if calcSha256 != expectedSha256 {
		return nil, fmt.Errorf("wrong checksum for package; expected: %s ; got %s", calcSha256, expectedSha256)
	}

	return libraryData, nil
}
