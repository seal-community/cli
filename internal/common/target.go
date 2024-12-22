package common

import "log/slog"

type TargetType string

const (
	UnknownTarget  TargetType = ""
	ManifestTarget TargetType = "manifest"
	// OsTarget is used to scan and fix artifacts in the OS
	// The target is the current directory
	OsTarget TargetType = "os"

	// *FilesTarget are used to scan and fix artifacts in a directory
	// The target is the directory path
	// For example, JavaFilesTarget is used to scan and fix JAR files in a directory
	JavaFilesTarget TargetType = "java"
)

func GetTargetDir(target string, targetType TargetType) string {
	if targetType == OsTarget || (targetType == JavaFilesTarget && target == "") {
		slog.Info("using current directory as target", "target", target)
		return GetAbsDirPath(CliCWD) // When the target is `os`, we should use the current directory as the targetDir
	}

	slog.Debug("using provided target", "target", target)
	return GetAbsDirPath(target)
}
