package rpm

import (
	"cli/internal/common"
	"cli/internal/config"
	"cli/internal/ecosystem/rpm/yum"
	"cli/internal/ecosystem/shared"
	"fmt"
	"log/slog"
	"runtime"

	"gopkg.in/ini.v1"
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

func isDistroSupported(os string) bool {
	return os == "centos" || os == "rhel"
}

func GetPackageManager(config *config.Config, targetDir string) (shared.PackageManager, error) {
	slog.Debug("checking OS supports rpm")
	if runtime.GOOS != "linux" {
		slog.Error("OS does not support RPM", "os", runtime.GOOS)
		return nil, fmt.Errorf("OS does not support RPM")
	}

	osRelease, err := loadOsRelease()
	if err != nil {
		slog.Error("Failed to load os-release", "err", err)
		return nil, err
	}

	os, err := getDistro(osRelease)
	if err != nil {
		slog.Error("Failed to get OS", "err", err)
		return nil, err
	}

	if !isDistroSupported(os) {
		slog.Error("OS does not support RPM", "os", os)
		return nil, common.NewPrintableError("OS does not support RPM")
	}

	slog.Debug("OS supports RPM", "os", os)
	return yum.NewYumManager(config, targetDir), nil
}
