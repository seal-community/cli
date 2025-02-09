package utils

import (
	"cli/internal/api"
	"log/slog"
)

const sealPrefix = "seal-"

func getParsedPackageFromDB(dbContent string, rule api.SilenceRule) (bool, *PackageInfo) {
	packages := parseAPKDB(string(dbContent))
	packageInfo, exists := packages[rule.Library]
	return exists, &packageInfo
}

func RenamePackage(dbContent string, rule api.SilenceRule) (bool, string) {
	slog.Debug("silencing package", "name", rule.Library)
	exists, packageInfo := getParsedPackageFromDB(dbContent, rule)
	if !exists {
		return false, dbContent
	}

	return true, modifyApkDBContentForSilence(dbContent, *packageInfo, sealPrefix+rule.Library)
}
