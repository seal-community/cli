package deb

import (
	"cli/internal/common"
	"cli/internal/config"
	"cli/internal/ecosystem/deb/dpkg"
	"cli/internal/ecosystem/deb/dpkgless"
	"cli/internal/ecosystem/shared"
	"fmt"
	"log/slog"
	"os/exec"
	"runtime"
)

func isDistroSupported(os string) bool {
	return os == "debian" || os == "ubuntu"
}

func isDpkgAvaliable() bool {
	_, err := exec.LookPath("dpkg")
	return err == nil
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

	if isDpkgAvaliable() {
		slog.Debug("OS supports dpkg", "os", os)
		return dpkg.NewDpkgManager(config, targetDir), nil
	}

	slog.Debug("dpkg not found, using dpkgless")
	return dpkgless.NewDpkglessManager(config, targetDir), nil
}
