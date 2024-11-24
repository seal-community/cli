package common

import (
	"log/slog"

	gover "github.com/hashicorp/go-version"
)

var CliVersion string

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
