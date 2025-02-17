package golang

import (
	"cli/internal/common"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

const SealDir = "sealsecurity.io"

func getReplaceString(packageName string, packageVersion string) string {
	return fmt.Sprintf("-replace=%s@v%s=%s/%s@v%s", packageName, packageVersion, SealDir, packageName, packageVersion)
}

func replaceInModFile(vendorDir string, packageName string, packageVersion string) error {
	projectDir := filepath.Dir(vendorDir)
	editOutput, err := common.RunCmdWithArgs(projectDir, goExe, "mod", "edit", getReplaceString(packageName, packageVersion)) // exists since go mod existed in go 1.11
	if err != nil {
		slog.Error("failed running go mod edit", "err", err)
		return err
	}

	if editOutput.Code != 0 {
		slog.Error("running go mod edit returned non-zero", "result", editOutput, "exitcode", editOutput.Code)
		return fmt.Errorf("running go mod edit returned non-zero")
	}

	return nil
}

func modulesContentAddReplace(modulesFile string, packageName string, packageVersion string) (error, string) {
	oldString := fmt.Sprintf("# %s v%s\n", packageName, packageVersion)
	if count := strings.Count(modulesFile, oldString); count != 1 {
		slog.Error("unexpected number of occurrences of package in modules.txt", "package", packageName, "version", packageVersion, "count", count)
		return fmt.Errorf("unexpected number of occurrences of package in modules.txt"), ""
	}

	newString := fmt.Sprintf("# %s v%s => %s/%s v%s\n", packageName, packageVersion, SealDir, packageName, packageVersion)
	return nil, strings.Replace(modulesFile, oldString, newString, 1)
}

func modifyModulesFile(vendorDir, packageName string, packageVersion string) error {
	modulesFile, err := os.ReadFile(filepath.Join(vendorDir, "modules.txt"))
	if err != nil {
		slog.Error("failed reading modules.txt", "err", err)
		return err
	}

	err, newModulesContent := modulesContentAddReplace(string(modulesFile), packageName, packageVersion)
	if err != nil {
		return err
	}

	err = os.WriteFile(filepath.Join(vendorDir, "modules.txt"), []byte(newModulesContent), os.ModePerm)
	if err != nil {
		slog.Error("failed writing modules.txt", "err", err)
	}

	return err
}

// Symlinks the package directory to a seal directory in the vendor directory
// Both directories need to appear in the vendor directory
// Otherwise go build won't work
// before:
//
//	vendor/google.com/protobuf
//
// after
//
//	vendor/google.com/protobuf
//	vendor/sealsecurity.io/google.com/protobuf -> vendor/google.com/protobuf
func moveToVendorSealDir(vendorDir, packageName string) error {
	newPackageDir := filepath.Join(vendorDir, SealDir, packageName)
	err := os.MkdirAll(filepath.Dir(newPackageDir), os.ModePerm)
	if err != nil {
		slog.Error("failed creating seal directory", "err", err)
		return err
	}

	oldPackageDir := filepath.Join(vendorDir, packageName)
	err = os.Symlink(oldPackageDir, newPackageDir)
	if err != nil {
		slog.Error("failed moving package to seal directory", "err", err)
		return err
	}

	return nil
}

func renamePackage(vendorDir string, packageName string, packageVersion string) error {
	err := replaceInModFile(vendorDir, packageName, packageVersion)
	if err != nil {
		return err
	}

	err = modifyModulesFile(vendorDir, packageName, packageVersion)
	if err != nil {
		return err
	}

	return moveToVendorSealDir(vendorDir, packageName)
}
