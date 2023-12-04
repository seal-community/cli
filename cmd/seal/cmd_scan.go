package main

import (
	"cli/cmd/seal/output"
	"cli/internal/api"
	"cli/internal/common"
	"cli/internal/phase"
	"fmt"
	"log/slog"

	"github.com/spf13/cobra"
)

type ResultHandler interface {
	Handle([]api.PackageVersion, common.DependencyMap) error
}

const csvFlag = "csv"

func initResultHandler(cmd *cobra.Command) (ResultHandler, error) {
	csvFilePath, err := cmd.Flags().GetString(csvFlag)
	if err != nil {
		// means misconfiguration in code
		panic("failed getting flag csv value")
	}

	if csvFilePath == "" {
		return output.ConsolePrinter{}, nil
	}

	slog.Info("exporting results to csv file", "path", csvFilePath)
	csv, err := common.CreateFile(csvFilePath)
	if err != nil {
		return nil, err
	}

	return output.CsvExporter{Writer: csv}, nil
}

func scanCommand() *cobra.Command {
	var cmd = &cobra.Command{
		Use:   "scan [directory]",
		Short: "Scan a directory",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			defer func() {
				// used to print error message on exit
				if err != nil {
					if printableErr := common.AsPrintable(err); printableErr != nil {
						fmt.Println(printableErr.Error())
					} else {
						slog.Warn("non printable error", "err", err)
					}
					// overwrite so we could distinguish between usage error and more internal ones
					err = SubCommandError
				}
			}()

			resultHandler, err := initResultHandler(cmd)
			if err != nil {
				return common.NewPrintableError("failed initializing output")
			}

			targetDir := extractTargetDir(args)
			verbosity := getFlag(cmd, verboseFlagKey)

			scanPhase, err := phase.NewScanPhase(targetDir, verbosity == 0)
			if err != nil {
				slog.Error("failed initializing scan", "err", err)
				return common.FallbackPrintableMsg(err, "failed initializing scan phase")
			}

			defer scanPhase.HideProgress() // should be gone when this is over, hide just in case

			result, err := scanPhase.Scan()
			if err != nil {
				return common.FallbackPrintableMsg(err, "failed performing scan")
			}

			scanPhase.HideProgress() // should be gone here, but before handling output

			if len(result.Vulnerable) == 0 {
				slog.Info("no vulnerable package found", "target", scanPhase.ProjectDir)
				fmt.Println("No vulnerabilities found") // Print to screen
				return nil
			}

			if err = resultHandler.Handle(result.Vulnerable, result.AllDependencies); err != nil {
				slog.Error("failed handling results", "err", err)
				return common.NewPrintableError("failed exporting results")
			}

			return nil
		},
	}

	cmd.Flags().String(csvFlag, "", "output results as csv to path")
	return cmd
}
