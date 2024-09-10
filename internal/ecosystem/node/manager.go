package node

import (
	"cli/internal/common"
	"cli/internal/config"
	"cli/internal/ecosystem/node/npm"
	"cli/internal/ecosystem/node/pnpm"
	"cli/internal/ecosystem/node/utils"
	"cli/internal/ecosystem/node/yarn"
	"cli/internal/ecosystem/shared"
	"fmt"
	"log/slog"
	"strings"
)

var MissingNodeModulesFolder = common.NewPrintableError("missing node_modules directory, please install dependencies before running")

func getPackageManagerForTargetFile(config *config.Config, targetFile string, targetDir string) (shared.PackageManager, error) {
	if pnpm.IsPnpmIndicatorFile(targetFile) {
		slog.Debug("pnpm manager supports target", "target-file", targetFile, "target-dir", targetDir)
		return pnpm.NewPnpmManager(config, targetDir), nil
	}

	if yarn.IsYarnIndicatorFile(targetFile) {
		slog.Debug("yarn manager supports target", "target-file", targetFile, "target-dir", targetDir)
		return yarn.NewYarnManager(config, targetDir), nil
	}

	if npm.IsNpmIndicatorFile(targetFile) {
		slog.Debug("npm manager supports target", "target-file", targetFile, "target-dir", targetDir)
		return npm.NewNpmManager(config, targetDir), nil
	}

	return nil, fmt.Errorf("failed detecting npm indicator for file %s", targetFile)
}

func getPackageManagerForTargetDir(config *config.Config, targetDir string) (shared.PackageManager, error) {
	if !utils.ContainsNodeModules(targetDir) {
		return nil, MissingNodeModulesFolder
	}

	isNpmDir, err := npm.IsNpmProjectDir(targetDir)
	if err != nil {
		return nil, fmt.Errorf("failed detecting npm directory %w", err)
	}

	if !isNpmDir {
		// propagate error message
		return nil, utils.CwdWrongProjectDir
	}

	if pnpm.IsPnpmProjectDir(targetDir) {
		return pnpm.NewPnpmManager(config, targetDir), nil
	}

	if yarn.IsYarnProjectDir(targetDir) {
		return yarn.NewYarnManager(config, targetDir), nil
	}

	return npm.NewNpmManager(config, targetDir), nil
}

func GetPackageManager(config *config.Config, targetDir string, targetFile string) (shared.PackageManager, error) {
	slog.Debug("checking provided target for node indicator", "file", targetFile, "dir", targetDir)

	if targetFile == "" || strings.HasSuffix(targetFile, utils.PackageJsonFile) {
		// treat target file of `package.json` the same as dir, look for other manager indicators due to possible user error
		slog.Debug("checking package manager for target dir")
		return getPackageManagerForTargetDir(config, targetDir)
	}

	return getPackageManagerForTargetFile(config, targetFile, targetDir)
}
