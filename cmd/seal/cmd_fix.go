package main

import (
	"cli/cmd/seal/output"
	"cli/internal/common"
	"cli/internal/phase"
	"fmt"
	"log/slog"

	"github.com/spf13/cobra"
)

const summaryFlag = "summarize"

func getSummaryPath(cmd *cobra.Command) string {
	summaryPath, err := cmd.Flags().GetString(summaryFlag)
	if err != nil {
		// means misconfiguration in code
		panic("failed getting flag summary value")
	}
	return summaryPath
}

func dumpSummary(summary *output.Summary, summaryPath string) error {

	if summaryPath != "" {
		slog.Info("creating fix summary", "path", summaryPath)

		summaryFile, err := common.CreateFile(summaryPath)
		if err != nil {
			return common.NewPrintableError("failed creating summary file")

		}

		if err = summary.Save(summaryFile); err != nil {
			return common.NewPrintableError("failed writing to summary file")
		}
	}

	return nil
}

func printSummary(summary *output.Summary) {
	if len(summary.Fixes) > 0 {
		slog.Info("fixed packaged", "count", len(summary.Fixes))
		summary.Print()
	}

	var msg string
	fixed := len(summary.Fixes)
	switch fixed {
	case 0:
		msg = "Nothing to fix"
	case 1:
		msg = "Fixed 1 package"
	default:
		msg = fmt.Sprintf("Fixed %d packages", fixed)
	}

	fmt.Println(msg)
}

func fixCommand() *cobra.Command {
	var cmd = &cobra.Command{
		Use:   "fix [directory]",
		Short: "Apply fixes to dependencies",
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

			targetDir := extractTargetDir(args)
			verbosity := getFlag(cmd, verboseFlagKey)
			summaryPath := getSummaryPath(cmd)

			// IMPORTANT - after this point printing directly to console would mess up the progress bar, msg should be used instead
			fixPhase, err := phase.NewFixPhase(targetDir, verbosity == 0)
			if err != nil {
				slog.Error("failed initializing fix", "err", err)
				return common.FallbackPrintableMsg(err, "failed initializing fix phase")
			}

			defer fixPhase.HideProgress() // should be gone when this is over, hide just in case

			//  auth check could be run in parallel with scan sub command to improve experience
			slog.Info("authenticating")
			if err := fixPhase.Authenticate(); err != nil {
				slog.Error("auth failed", "err", err)
				return common.FallbackPrintableMsg(err, "authentication issue")
			}

			result, err := fixPhase.Scan()
			if err != nil {
				return common.FallbackPrintableMsg(err, "failed performing initial scan")
			}

			if len(result.Vulnerable) == 0 {
				fixPhase.HideProgress() // make sure before printing
				slog.Info("no vulnerable package found", "target", fixPhase.ProjectDir)
				err = dumpSummary(output.NewSummary(fixPhase.ProjectDir, nil), summaryPath) // output summary even if no fixes are relvant, so could be processed by 3rd-party
				if err != nil {
					return err
				}

				fmt.Println("No vulnerabilities found")
				return nil
			}

			fixes, err := fixPhase.Fix(result)
			if err != nil {
				return common.FallbackPrintableMsg(err, "failed performing fix")
			}

			fixPhase.HideProgress() // should be gone here, but before handling summary make sure it gone
			summary := output.NewSummary(fixPhase.ProjectDir, fixes)
			if summary == nil {
				return common.NewPrintableError("failed generating summary")
			}

			err = dumpSummary(summary, summaryPath)
			if err != nil {
				return err
			}

			printSummary(summary)
			return nil
		},
	}

	cmd.Flags().String(summaryFlag, "", "output fix summary to path")
	return cmd
}
