package deb

import (
	"cli/internal/common"
	"cli/internal/config"
	"cli/internal/ecosystem/deb/dpkg"
	"cli/internal/ecosystem/shared"
	"fmt"
	"log/slog"
	"runtime"
)

func isDistroSupported(os string) bool {
	return os == "debian" || os == "ubuntu"
}

func GetPackageManager(config *config.Config, targetDir string) (shared.PackageManager, error) {
	slog.Debug("checking OS supports dpkg")
	if runtime.GOOS != "linux" {
		slog.Error("OS does not support dpkg", "os", runtime.GOOS)
		return nil, fmt.Errorf("OS does not support dpkg")
	}

	os, err := shared.GetOSDistro()
	if err != nil {
		slog.Error("Failed to get OS", "err", err)
		return nil, err
	}

	if !isDistroSupported(os) {
		slog.Error("OS does not support dpkg", "os", os)
		return nil, common.NewPrintableError("OS does not support dpkg")
	}

	slog.Debug("OS supports dpkg", "os", os)
	return dpkg.NewDPKGManager(config, targetDir), nil
}
