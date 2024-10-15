package main

import (
	"cli/cmd/seal/output"
	"cli/internal/actions"
	"cli/internal/api"
	"cli/internal/clients/blackduck"
	"cli/internal/common"
	"cli/internal/ecosystem/shared"
	"cli/internal/phase"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"

	"github.com/spf13/cobra"
)

const summaryFlag = "summarize"

func fixModeFromString(s string) phase.FixMode {
	modes := []phase.FixMode{phase.FixModeAll, phase.FixModeRemote, phase.FixModeLocal}
	fm := phase.FixMode(s)
	if slices.Contains(modes, fm) {
		return fm
	} else {
		return ""
	}
}

const modeFlag = "mode"
const silenceFlag = "silence"

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

func loadActionsFile(actionsFilePath string) (*actions.ActionsFile, error) {
	f, err := os.Open(actionsFilePath)
	if err != nil {
		if !os.IsNotExist(err) {
			slog.Info("failed opening conf file", "err", err, "path", actionsFilePath)
			return nil, common.NewPrintableError("could not open actions file in %s", actionsFilePath)
		}

		slog.Info("actions file not found", "path", actions.FailedParsingActionYamlInvalid)
		return nil, nil

	} else {
		defer f.Close()
	}

	// NOTE: project id from the actions file can differ from what we discover in fix phase - currently ok to ignore
	return actions.Load(f)
}

func filterVulnerablePackageForOverrides(vulnPackages []api.PackageVersion, overrides map[string]api.PackageVersion) []api.PackageVersion {
	overriddenPackages := make([]api.PackageVersion, 0, len(vulnPackages))

	for _, vulnPackage := range vulnPackages {
		if vulnPackage.RecommendedLibraryVersionId == "" {
			// do not allow to 'abuse' this to install package inplace of the vulnerable one if we don't have a SP for it
			slog.Debug("ignoring vulnerable package - does not have a sealed version", "packageId", vulnPackage.Id())
			continue
		}
		override, ok := overrides[vulnPackage.Id()]
		if !ok {
			slog.Debug("ignoring vulnerable package version, not in allowed overrides", "name", vulnPackage.Library.Name, "version", vulnPackage.Version)
			continue
		}

		slog.Debug("overriding package", "from", fmt.Sprintf("%s@%s", vulnPackage.Library.Name, vulnPackage.Version), "to", fmt.Sprintf("%s@%s", override.Library, override.Version))
		vulnPackage.RecommendedLibraryVersionString = override.RecommendedLibraryVersionString
		// keeping RecommendedLibraryVersionId as it is used to filter out non fixable packages; we will need to update this from BE

		overriddenPackages = append(overriddenPackages, vulnPackage) // will be copied, needs to keep all the other information like vulnerabilities
	}

	// this can return a 0 length slice if no packages are allowed, which is OK
	return overriddenPackages
}

func updateScanResultAccordingToActionsFile(result *phase.ScanResult, actionsFilePath string, normalizer shared.Normalizer) error {
	overrides, err := getExistingConfigOverrides(actionsFilePath, normalizer)
	if err != nil {
		slog.Error("failed opening actions file for fix", "err", err)
		return common.FallbackPrintableMsg(err, "failed loading actions file")
	}

	// replace existing slice
	slog.Info("limiting results according to actions file", "before", len(result.Vulnerable))

	// even if we have 0 after filtering, so we don't fix anything
	result.Vulnerable = filterVulnerablePackageForOverrides(result.Vulnerable, overrides)
	slog.Info("total available vulnerable after overriding", "count", len(result.Vulnerable))

	return nil
}

// dump and print a summary of the results
func outputSummary(summaryPath string, fixes []shared.DependnecyDescriptor, silenced map[string][]string, projectDir string) error {
	summary := output.NewSummary(projectDir, fixes, silenced)
	if summary == nil {
		return common.NewPrintableError("failed generating summary")
	}

	err := dumpSummary(summary, summaryPath)
	if err != nil {
		return err
	}

	printSummary(summary)
	return nil
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

			target := extractTarget(args)

			targetDir := common.GetTargetDir(target)
			if targetDir == "" {
				slog.Error("bad target input", "target", target)
				return common.NewPrintableError("target not found `%s`", target)
			}

			verbosity := getArgCount(cmd, verboseFlagKey)
			summaryPath := getArgString(cmd, summaryFlag)
			configPath := getArgString(cmd, configFileKey)
			fm := getArgString(cmd, modeFlag)
			silenceArray := getArgArray(cmd, silenceFlag)

			fixModeUsed := fixModeFromString(fm)
			if fixModeUsed == "" {
				slog.Error("fix mode is unsupported", "mode", fm)
				return common.NewPrintableError("fix mode is unsupported")
			}

			slog.Info("Fix mode", "mode", fixModeUsed)
			uploadScanActivity := getArgBool(cmd, uploadResultsKey)

			// IMPORTANT - after this point printing directly to console would mess up the progress bar, msg should be used instead
			fixPhase, err := phase.NewFixPhase(target, configPath, verbosity == 0)
			if err != nil {
				slog.Error("failed initializing fix", "err", err)
				return common.FallbackPrintableMsg(err, "failed initializing fix phase")
			}

			defer fixPhase.HideProgress() // should be gone when this is over, hide just in case

			if !fixPhase.Config.UseSealedNames && len(silenceArray) > 0 {
				slog.Error("silencing packages is not supported when not using sealed names")
				return common.NewPrintableError("silencing packages is not supported when not using sealed names")
			}

			actionsFilePath := getArgString(cmd, actionsFileKey)
			if actionsFilePath == "" {
				actionsFilePath = filepath.Join(targetDir, actions.ActionFileName)
			}

			// auth check could be run in parallel with scan sub command to improve experience
			slog.Info("authenticating")
			if err := fixPhase.InitRemoteProject(); err != nil {
				return common.FallbackPrintableMsg(err, "failed initializing project from server")
			}

			result, err := fixPhase.Scan(uploadScanActivity)
			if err != nil {
				return common.FallbackPrintableMsg(err, "failed performing initial scan")
			}

			if fixModeUsed == phase.FixModeLocal {
				slog.Info("trying to load actions file", "path", actionsFilePath)
				if err := updateScanResultAccordingToActionsFile(result, actionsFilePath, fixPhase.Manager); err != nil {
					return common.FallbackPrintableMsg(err, "failed using actions file")
				}
			}

			if len(result.Vulnerable) == 0 {
				fixPhase.HideProgress() // make sure before printing
				slog.Info("no vulnerable package found", "target", target)

				silenced := make(map[string][]string, 0)
				if len(silenceArray) > 0 {
					slog.Info("silencing packages", "count", len(silenceArray))
					if silenced, err = fixPhase.Manager.SilencePackages(silenceArray, result.AllDependencies); err != nil {
						return common.FallbackPrintableMsg(err, "failed silencing packages")
					}
				}

				err = dumpSummary(output.NewSummary(fixPhase.BaseDir, nil, silenced), summaryPath) // output summary even if no fixes are relvant, so could be processed by 3rd-party
				if err != nil {
					return err
				}

				fmt.Println("No applicable fix available") // ok to print here
				return nil
			}

			slog.Info("trying to get available fixes", "mode", fixModeUsed)
			availableFixes, err := fixPhase.GetAvailableFixes(result, fixModeUsed)
			if err != nil {
				slog.Error("failed getting available fixes", "err", err, "mode", fixModeUsed)
				return common.FallbackPrintableMsg(err, "failed querying for fixes")
			}

			var fixes []shared.DependnecyDescriptor
			slog.Debug("fixes available", "mode", fixModeUsed, "available", len(availableFixes))
			if len(availableFixes) > 0 {
				slog.Info("attempting to apply fixes", "mode", fixModeUsed, "available", len(availableFixes))
				fixes, err = fixPhase.Fix(availableFixes) // fixes can be null
				if err != nil {
					return common.FallbackPrintableMsg(err, "failed performing fix")
				}

				slog.Info("checking callbacks")
				fixPhase.HandleCallbacks(fixes, &blackduck.BlackDuckCallback{Config: fixPhase.Config})
			}

			silenced := make(map[string][]string, 0)
			if len(silenceArray) > 0 {
				slog.Info("silencing packages", "count", len(silenceArray))
				if silenced, err = fixPhase.Manager.SilencePackages(silenceArray, result.AllDependencies); err != nil {
					return common.FallbackPrintableMsg(err, "failed silencing packages")
				}
			}

			fixPhase.HideProgress() // should be gone here, but before handling summary make sure it gone
			return outputSummary(summaryPath, fixes, silenced, fixPhase.BaseDir)
		},
	}

	cmd.Flags().String(summaryFlag, "", "output fix summary to path")
	cmd.Flags().String(modeFlag, "local", fmt.Sprintf("Fix mode, options: [%s|%s|%s]", phase.FixModeLocal, phase.FixModeRemote, phase.FixModeAll))
	cmd.Flags().StringArray(silenceFlag, []string{}, "silence false-positive packages in the format of 'packageName:version'")
	return cmd
}
