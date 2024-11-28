package shared

import (
	"fmt"
	"gopkg.in/ini.v1"
	"log/slog"
)

func loadOsRelease() (*ini.File, error) {
	cfg, err := ini.Load("/etc/os-release")
	if err != nil {
		slog.Error("Fail to load os-release", "err", err)
		return nil, err
	}

	return cfg, nil
}

// Parse /etc/os-release to get the OS name
func getDistro(osRelease *ini.File) (string, error) {
	section := osRelease.Section("")
	if section == nil {
		slog.Error("No section in os-release")
		return "", fmt.Errorf("No section in os-release")
	}

	id := section.Key("ID")
	if id == nil {
		slog.Error("No ID in os-release")
		return "", fmt.Errorf("No ID in os-release")
	}

	distro := id.String()
	if distro == "" {
		slog.Error("Empty ID in os-release")
		return "", fmt.Errorf("Empty ID in os-release")
	}

	return distro, nil
}

func GetOSDistro() (string, error) {
	osRelease, err := loadOsRelease()
	if err != nil {
		slog.Error("Failed to load os-release", "err", err)
		return "", err
	}

	return getDistro(osRelease)
}
