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
)

var MissingNodeModulesFolder = common.NewPrintableError("missing node_modules directory, please install dependencies before running")

func GetPackageManager(config *config.Config, targetDir string) (shared.PackageManager, error) {
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
		return pnpm.NewPnpmManager(config), nil
	}

	if yarn.IsYarnProjectDir(targetDir) {
		return yarn.NewYarnManager(config), nil
	}

	return npm.NewNpmManager(config), nil
}
