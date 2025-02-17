package golang

import (
	"cli/internal/common"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
)

func NormalizePackageName(name string) string {
	return strings.ToLower(name)
}

func isVendorDirExist(projectDir string) (bool, error) {
	vendorDir := filepath.Join(projectDir, "vendor")
	return common.DirExists(vendorDir)
}

// Run `go mod vendor` to create a vendor directory with all dependencies
// do nothing if it exists
func PrepareVendorDir(projectDir string) error {
	slog.Info("running go mod vendor", "projectDir", projectDir)
	pr, err := common.RunCmdWithArgs(projectDir, goExe, "mod", "vendor")
	if err != nil {
		slog.Error("failed running go mod vendor", "err", err)
		return err
	}

	if pr.Code != 0 {
		slog.Error("running go mod vendor returned non-zero", "result", pr)
		return fmt.Errorf("running go mod vendor returned non-zero")
	}

	return nil
}
