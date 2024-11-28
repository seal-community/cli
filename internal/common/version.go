package common

import (
	"log/slog"
	"regexp"
	"strings"

	gover "github.com/hashicorp/go-version"
)

var CliVersion string

const EpochRegex = `^(\d+:)?(.*)`

// this is not strict semver comparison
func VersionAtLeast(version string, minimal string) (bool, error) {

	required, err := gover.NewVersion(minimal)
	if err != nil {
		slog.Error("failed parsing minimal version requirement", "err", err, "version", minimal)
		return false, err
	}

	v, err := gover.NewVersion(version)
	if err != nil {
		slog.Error("failed parsing current version", "err", err, "version", version)
		return false, err
	}

	return v.Equal(required) || v.GreaterThan(required), nil
}

// Return split version and epoch, e.g. 1:1.2.3 -> 1, 1.2.3
func GetNoEpochVersion(version string) (string, string) {
	reg := regexp.MustCompile(EpochRegex)
	matches := reg.FindStringSubmatch(version)
	if len(matches) < 3 {
		slog.Debug("failed parsing epoch from version", "version", version)
		return "", ""
	}

	epoch, _ := strings.CutSuffix(matches[1], ":")
	return epoch, matches[2]
}
