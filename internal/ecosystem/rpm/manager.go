package rpm

import (
	"cli/internal/common"
	"cli/internal/config"
	"cli/internal/ecosystem/rpm/yum"
	"cli/internal/ecosystem/shared"
	"fmt"
	"log/slog"
	"runtime"
)

func isDistroSupported(os string) bool {
	return os == "centos" || os == "rhel" || os == "ol"
}

func GetPackageManager(config *config.Config, targetDir string) (shared.PackageManager, error) {
	slog.Debug("checking OS supports rpm")
	if runtime.GOOS != "linux" {
		slog.Error("OS does not support RPM", "os", runtime.GOOS)
		return nil, fmt.Errorf("OS does not support RPM")
	}

	os, err := shared.GetOSDistro()
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
