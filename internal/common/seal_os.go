package common

import "log/slog"

// osMagic is a magic string that means scanning the OS and not an application ecosystem
const OsMagic = "os"

func GetTargetDir(target string) string {
	if target == OsMagic {
		slog.Info("using current directory as target", "target", target)
		return GetAbsDirPath(CliCWD) // When the target is `os`, we should use the current directory as the targetDir
	}

	slog.Debug("using provided target", "target", target)
	return GetAbsDirPath(target)
}
