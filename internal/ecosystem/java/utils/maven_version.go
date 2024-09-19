package utils

import (
	"bufio"
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
		// maven outputs the error to stdout
		slog.Error("running maven version returned non-zero", "err", result.Stderr, "out", result.Stdout, "exitcode", result.Code)
		return ""
	}

	version := parseMavenVersion(result.Stdout)
	return version
}

func parseMavenVersion(mavenVersionOutput string) string {
	r := bufio.NewReader(strings.NewReader(mavenVersionOutput))

	scanner := bufio.NewScanner(r) // handles line-endings correctly per os
	scanner.Scan()
	line := scanner.Text()
	line = strings.Split(line, "\r")[0] // in case for some reason we parse windows output on non windows machine
	splitted := strings.Split(line, " Maven ")

	if len(splitted) < 2 {
		slog.Error("failed parsing maven version", "output", mavenVersionOutput)
		return ""
	}

	versionWithSuffix := splitted[1]

	// strip color escape code if exists https://en.wikipedia.org/wiki/ANSI_escape_code#Colors
	// example: // "3.6.3\x1b[m\nMaven"
	if colorIdx := strings.Index(versionWithSuffix, "\x1b"); colorIdx != -1 {
		// color code escape character
		versionWithSuffix = versionWithSuffix[:colorIdx]
	}

	return strings.Split(versionWithSuffix, " ")[0]
}
