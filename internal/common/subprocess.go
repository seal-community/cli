package common

import (
	"log/slog"
	"os/exec"
	"strings"
)

type ProcessResult struct {
	Stdout string
	Stderr string
	Code   int
}

func RunCmdWithArgs(targetDir string, exe string, args ...string) (*ProcessResult, error) {
	cmd := exec.Command(exe, args...)
	cmd.Dir = targetDir
	var errBuffer strings.Builder
	cmd.Stderr = &errBuffer
	result := &ProcessResult{}
	output, err := cmd.Output()

	result.Stdout = string(output)
	result.Stderr = errBuffer.String()

	if err != nil {
		exitError, ok := err.(*exec.ExitError)
		if !ok {
			// this error is due to something else, not an exit error for the process
			slog.Error("command failed", "err", err, "args", args, "stderr", result.Stderr)
			return nil, err
		}
		result.Code = exitError.ExitCode()
	}

	if result.Code != 0 {
		slog.Warn("command non success exit code", "code", result.Code, "stderr", result.Stderr)
	}

	return result, nil
}
