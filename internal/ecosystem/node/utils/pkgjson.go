package utils

import (
	"cli/internal/common"
	"errors"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/iancoleman/orderedmap"
)

const PackageJsonFile = "package.json"
const SealSecurityNamespace = "@seal-security"

func GetProjectName(dir string) string {
	pgk := loadPackageJson(dir)
	if pgk == nil {
		return ""
	}
	return getProjectName(pgk)
}

func GetVersion(dir string) string {
	pgk := loadPackageJson(dir)
	if pgk == nil {
		return ""
	}
	return getStringAttrFromPackageLock(pgk, "version")
}

func loadPackageJson(dir string) *orderedmap.OrderedMap {
	p := getPackageJsonFilePath(dir)
	data := common.JsonLoad(p)
	if data == nil {
		slog.Error("failed opening package json file", "path", p)
		return nil
	}

	return data
}

func getPackageJsonFilePath(dir string) string {
	return filepath.Join(dir, PackageJsonFile)
}

func getProjectName(pgk *orderedmap.OrderedMap) string {
	return getStringAttrFromPackageLock(pgk, "name")
}

func getStringAttrFromPackageLock(pgk *orderedmap.OrderedMap, attrName string) string {
	if pgk == nil {
		return ""
	}
	val, ok := pgk.Get(attrName)
	if !ok {
		slog.Warn("attr not found in package json", "attr", attrName)
		return ""
	}

	sVal, ok := val.(string)
	if !ok {
		slog.Warn("attr value is bad type", "attr", attrName)
		return ""
	}

	return sVal
}

// gets one of the two possible formats:
// 1. name -> @seal-security/name
// 2. @scope/name -> @seal-security/scope-sealsec-name
// Note that this function doesn't support Windows paths right now
func CalculateSealedName(originalName string) string {
	// if name name is empty, return empty string
	if originalName == "" {
		return ""
	}

	// if the name is not namespaced, just add the sealsecurity namespace
	if originalName[0] != '@' {
		// add the sealsecurity namespace
		return SealSecurityNamespace + "/" + originalName
	}

	// if package has a name space - escape it with a `-sealsec-`
	// for example: @angular/core -> @seal-security@angular-sealsec-core
	parts := strings.Split(originalName, "/")
	if len(parts) != 2 {
		return ""
	}

	owner := parts[0]
	name := parts[1]
	return SealSecurityNamespace + "/" + owner[1:] + "-sealsec-" + name
}

// addSealPrefixToPackageLockFile adds a "@seal-security" namespace to the name field in the package.json file located at the given disk path.
// in the case that it's already namespaced, it'll add an escaping `-sealsec-` to the name as follows:
// 1. @owner/name -> @seal-security/owner-sealsec-name
// It loads the package.json file, modifies the name field, and saves the changes back to the file.
//
// Parameters:
//   - diskPath: The file system path where the package.json file is located.
//
// Returns:
//   - error: An error if the package.json file cannot be loaded or saved, otherwise nil.
func AddSealPrefixToPackageJsonFile(diskPath string) error {
	packageJson := loadPackageJson(diskPath)
	if packageJson == nil {
		return errors.New("failed loading package.json")
	}
	originalName := getProjectName(packageJson)
	if originalName == "" {
		return errors.New("failed getting package name")
	}
	sealedPackageName := CalculateSealedName(originalName)
	if sealedPackageName == "" {
		return errors.New("failed calculating sealed package name")
	}
	packageJson.Set("name", sealedPackageName)
	slog.Debug("changing package to sealed name", "original", originalName, "sealed", sealedPackageName)
	packageJsonFilePath := getPackageJsonFilePath(diskPath)
	if common.JsonSave(packageJson, packageJsonFilePath) != nil {
		return common.NewPrintableError("failed saving package.json")
	}

	return nil
}
