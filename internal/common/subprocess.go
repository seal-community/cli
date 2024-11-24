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

func handleProcessResult(args []string, result *ProcessResult, err error) (*ProcessResult, error) {

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

func RunCmdWithArgs(targetDir string, exe string, args ...string) (*ProcessResult, error) {
	Trace("running cmd", "exe", exe, "args", args)

	cmd := exec.Command(exe, args...)
	cmd.Dir = targetDir
	var errBuffer strings.Builder
	cmd.Stderr = &errBuffer
	result := &ProcessResult{}
	output, err := cmd.Output()

	result.Stdout = string(output)
	result.Stderr = errBuffer.String()

	return handleProcessResult(args, result, err)
}

// Runs a command with arguments and returns the combined output
func RunCmdWithArgsCombinedOutput(targetDir string, exe string, args ...string) (*ProcessResult, error) {
	Trace("running cmd combined output", "exe", exe, "args", args)

	cmd := exec.Command(exe, args...)
	cmd.Dir = targetDir
	result := &ProcessResult{}
	output, err := cmd.CombinedOutput()

	result.Stdout = string(output)
	result.Stderr = result.Stdout

	return handleProcessResult(args, result, err)
}
