package utils

import (
	"cli/internal/common"
	"log/slog"
	"strings"
)

const MavenExeName = "mvn"
const VersionFlag = "--version"

func GetVersion(targetDir string) string {
	result, err := common.RunCmdWithArgs(targetDir, MavenExeName, VersionFlag)
	if err != nil {
		slog.Error("failed running maven version", "err", err)
		return ""
	}
	if result.Code != 0 {
		slog.Error("running maven version returned non-zero", "result", result)
		return ""
	}

	version := parseMavenVersion(result.Stdout)
	return version
}

func parseMavenVersion(mavenVersionOutput string) string {
	versionWithSuffix := strings.TrimPrefix(mavenVersionOutput, "Apache Maven ")
	spaceIndex := strings.Index(versionWithSuffix, " ")
	version := versionWithSuffix[:spaceIndex]
	return version
}
