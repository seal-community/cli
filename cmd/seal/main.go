package main

import (
	"cli/internal/common"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"runtime/debug"

	"github.com/spf13/cobra"
)

type ErrorCode int

// requires to prevent lint error on context keys
type contextKey int

const (
	logfileKey contextKey = iota
)

const verboseFlagKey = "verbose"

var SubCommandError = errors.New("") // used to differentiate between cobra usage error and our errors

const (
	Success              ErrorCode = 0
	FailedInitLog        ErrorCode = 1
	FailedNoBuildVersion ErrorCode = 2
	FailedPanic          ErrorCode = 3
	FailedCommand        ErrorCode = 10
	FailedCommandUsage   ErrorCode = 11
)

// implemented just to hide the default command added by cobra
func completionCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:    "completion",
		Short:  "Generate the autocompletion script for the specified shell",
		Hidden: true,
	}
	
	return cmd
}

func rootCmd() *cobra.Command {
	cmd := &cobra.Command{
		SilenceErrors: true, // to not print cobra formatted error, but handle it ourselves
		SilenceUsage:  true, // to not print usage every error that happens when code runs
		Use:           "seal [command]",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// this will be called before any subcommand is run, perform common setup that is beyond the commands scope
			verbosity := getArgCount(cmd, verboseFlagKey)
			logfile := cmd.Context().Value(logfileKey).(*os.File) // will panic if misconfigured in code

			setupLogging(logfile, verbosity)
			slog.Info("cli started", "version", common.CliVersion, "sessionId", common.SessionId, "logfile", logfile.Name())

			return nil
		},
	}

	cmd.AddCommand(
		scanCommand(),
		fixCommand(),
		versionCommand(),
		addCommand(),
		completionCommand(),
	)

	cmd.PersistentFlags().CountP(verboseFlagKey, "v", "counted verbosity")
	return cmd
}

func cli() (errCode ErrorCode) {
	var verbosity int

	defer func() {
		// used to recover from panics, might not have logging.
		// we do not want to show stacktrace to users, only minimal info
		if panciObj := recover(); panciObj != nil {
			slog.Error("panic caught", "err", panciObj, "trace", string(debug.Stack()))
			fmt.Println("Internal error")
			errCode = FailedPanic
		}
	}()

	logfile, err := os.CreateTemp("", "seal-cli-*.log")
	if err != nil {
		// will not hide log files from console if we failed to create a file
		fmt.Println(common.Colorize("Warning: Failed initializing log file", common.AnsiWarnYellow))
		logfile = os.Stdout
	} else {
		hideLogging() // used to disable logging before it is set up (in case of panics etc)
	}

	defer func() {
		// delete log file unless had error or verbose logging enabled
		if errCode == Success {
			if verbosity > 0 {
				fmt.Printf("\nsee log: %v\n", common.Colorize(logfile.Name(), common.AnsiDarkGrey))
			} else {
				os.Remove(logfile.Name())
			}
		} else if errCode == FailedCommand {
			// log only applicable when we had an error, not a cobra usage error
			fmt.Printf("Check the log for more details: %v\n", common.Colorize(logfile.Name(), common.AnsiDarkGrey))
		}
	}()

	defer logfile.Close()

	if common.CliVersion == "" {
		// can't log yet
		fmt.Println("Error: version string not set during compilation")
		return FailedNoBuildVersion
	}

	cmd := rootCmd()
	if cmd, err := cmd.ExecuteContextC(context.WithValue(context.Background(), logfileKey, logfile)); err != nil {
		// checking if root pre run init was done
		slog.Warn("command failed", "name", cmd.Name())
		if errors.Is(err, SubCommandError) {
			return FailedCommand
		}

		_ = cmd.Usage()
		return FailedCommandUsage
	}

	// if succeeded we might want to keep the log file, depending on verbosity value
	verbosity = getArgCount(cmd, verboseFlagKey)

	return Success
}

func main() {
	code := cli()
	os.Exit(int(code))
}
