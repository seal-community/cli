package main

import (
	"cli/cmd/seal/output"
	"cli/internal/actions"
	"cli/internal/api"
	"cli/internal/clients/blackduck"
	"cli/internal/clients/dependabot"
	"cli/internal/clients/ox"
	"cli/internal/common"
	"cli/internal/ecosystem/shared"
	"cli/internal/phase"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/spf13/cobra"
)

const summaryFlag = "summarize"
const SealFixModeHeader = "X-Seal-Fix-Mode"

func getSilenceRules(v []string) ([]api.SilenceRule, error) {
	var rules []api.SilenceRule
	for _, entry := range v {
		silenceParts := strings.Split(entry, "@")
		if len(silenceParts) != 2 {
			slog.Error("silence entry is not in correct format", "rule", entry)
			return nil, common.NewPrintableError("silence entry %s is not in correct format", entry)
		}
		rules = append(rules, api.SilenceRule{Library: silenceParts[0], Version: silenceParts[1]})
	}

	return rules, nil
}

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
const skipSignCheckFlag = "skip-sign-checks"

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

func loadActionsFile(actionsFilePath string) (*actions.ActionsFile, error) {
	f, err := os.Open(actionsFilePath)
	if err != nil {
		if !os.IsNotExist(err) {
			slog.Info("failed opening conf file", "err", err, "path", actionsFilePath)
			return nil, common.NewPrintableError("could not open actions file in %s", actionsFilePath)
		}

		slog.Info("actions file not found; ignoring", "path", actionsFilePath)
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
		vulnId := vulnPackage.Id()
		if vulnPackage.RecommendedLibraryVersionId == "" {
			// do not allow to 'abuse' this to install package inplace of the vulnerable one if we don't have a SP for it
			slog.Debug("ignoring vulnerable package - does not have a sealed version", "packageId", vulnId)
			continue
		}
		override, ok := overrides[vulnId]
		if !ok {
			slog.Debug("ignoring vulnerable package version, not in allowed overrides", "name", vulnPackage.Library.Name, "version", vulnPackage.Version, "id", vulnId)
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

func updateScanResultAccordingToActionsFile(result *phase.ScanResult, actionsFilePath string, normalizer shared.Normalizer) (map[string]api.PackageVersion, error) {
	overrides, err := getExistingConfigOverrides(actionsFilePath, normalizer)
	if err != nil {
		slog.Error("failed opening actions file for fix", "err", err)
		return nil, common.FallbackPrintableMsg(err, "failed loading actions file")
	}

	// replace existing slice
	slog.Info("limiting results according to actions file", "before", len(result.Vulnerable))

	// even if we have 0 after filtering, so we don't fix anything
	result.Vulnerable = filterVulnerablePackageForOverrides(result.Vulnerable, overrides)
	slog.Info("total available vulnerable after overriding", "count", len(result.Vulnerable))

	return overrides, nil
}

// dump and print a summary of the results
func outputSummary(summaryPath string, fixes []shared.DependencyDescriptor, silenced map[string][]string, projectDir string, finalMsg string) error {
	summary := output.NewSummary(projectDir, fixes, silenced)
	if summary == nil {
		return common.NewPrintableError("failed generating summary")
	}

	err := dumpSummary(summary, summaryPath)
	if err != nil {
		return err
	}

	summary.Print(finalMsg)
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
						fmt.Println("")
						fmt.Println(printableErr.Error())
					} else {
						slog.Warn("non printable error", "err", err)
					}

					// overwrite so we could distinguish between usage error and more internal ones
					err = SubCommandError
				}
			}()

			filesystemValue := getArgString(cmd, filesystemFlag)
			osValue := getArgBool(cmd, osFlag)

			target, targetType := extractTarget(args, filesystemValue, osValue)
			if targetType == common.UnknownTarget {
				slog.Error("invalid target", "target", target)
				return common.NewPrintableError("invalid target `%s`", target)
			}

			targetDir := common.GetTargetDir(target, targetType)
			if targetDir == "" {
				slog.Error("bad target input", "target", target)
				return common.NewPrintableError("target not found `%s`", target)
			}

			verbosity := getArgCount(cmd, verboseFlagKey)
			summaryPath := getArgString(cmd, summaryFlag)
			configPath := getArgString(cmd, configFileKey)
			fm := getArgString(cmd, modeFlag)
			silenceArgArray := getArgArray(cmd, silenceFlag)
			silenceArray, err := getSilenceRules(silenceArgArray)
			localModeManuallyProvided := cmd.Flags().Changed(modeFlag)
			skipSignCheck := getArgBool(cmd, skipSignCheckFlag)

			if err != nil {
				return common.FallbackPrintableMsg(err, "failed parsing silence rules")
			}

			fixModeUsed := fixModeFromString(fm)
			if fixModeUsed == "" {
				slog.Error("fix mode is unsupported", "mode", fm)
				return common.NewPrintableError("fix mode is unsupported")
			}

			slog.Info("Fix mode", "mode", fixModeUsed)
			uploadScanActivity := getArgBool(cmd, uploadResultsKey)

			// IMPORTANT - after this point printing directly to console would mess up the progress bar, msg should be used instead
			fixPhase, err := phase.NewFixPhase(target, targetType, configPath, verbosity == 0)
			if err != nil {
				slog.Error("failed initializing fix", "err", err)
				return common.FallbackPrintableMsg(err, "failed initializing fix phase")
			}

			fixPhase.ArtifactServer.SetExtraHeaders([]api.StringPair{{Name: SealFixModeHeader, Value: fm}})

			defer fixPhase.HideProgress() // should be gone when this is over, hide just in case

			if !fixPhase.Config.UseSealedNames && len(silenceArray) > 0 {
				slog.Error("silencing packages is not supported when not using sealed names")
				return common.NewPrintableError("silencing packages is not supported when not using sealed names")
			}

			if fixModeUsed == phase.FixModeRemote && len(silenceArray) > 0 {
				slog.Error("silencing specific packages in command line is not supported in remote mode")
				return common.NewPrintableError("silencing specific packages in command line is not supported in remote mode")
			}

			actionsFilePath := getArgString(cmd, actionsFileKey)
			if actionsFilePath == "" {
				actionsFilePath = filepath.Join(targetDir, actions.ActionFileName)
			}

			// auth check could be run in parallel with scan sub command to improve experience
			if err := fixPhase.InitRemoteProject(); err != nil {
				return common.FallbackPrintableMsg(err, "failed initializing project from server")
			}

			result, err := fixPhase.Scan(uploadScanActivity)
			if err != nil {
				return common.FallbackPrintableMsg(err, "failed performing initial scan")
			}

			var overrides map[string]api.PackageVersion
			if fixModeUsed == phase.FixModeLocal {
				slog.Info("trying to load actions file", "path", actionsFilePath)
				overrides, err = updateScanResultAccordingToActionsFile(result, actionsFilePath, fixPhase.Manager)
				if err != nil {
					return common.FallbackPrintableMsg(err, "failed using actions file")
				}
			}

			if fixModeUsed == phase.FixModeRemote {
				silenceArray, err = fixPhase.QuerySilenceRules()
				if err != nil {
					return common.FallbackPrintableMsg(err, "failed querying for silence rules")
				}
				slog.Debug("silence rules", "count", len(silenceArray))
			}

			if len(result.Vulnerable) == 0 {
				fixPhase.HideProgress() // make sure before printing

				if overrides == nil {
					fmt.Println("")

					var message string
					if !localModeManuallyProvided {
						message = "Using fix mode local by default, but local configuration file not found. No fixes will be applied!"
					} else {
						message = "Local configuration file not found. No fixes will be applied!"
					}
					fmt.Println(common.Colorize("Warning:", common.AnsiWarnYellow), message)
				}

				slog.Info("no vulnerable package found", "target", target)

				silenced := make(map[string][]string, 0)
				if len(silenceArray) > 0 {
					slog.Info("silencing packages", "count", len(silenceArray))
					if silenced, err = fixPhase.Manager.SilencePackages(silenceArray, result.AllDependencies); err != nil {
						return common.FallbackPrintableMsg(err, "failed silencing packages")
					}
				}

				return outputSummary(summaryPath, nil, silenced, fixPhase.BaseDir, "No applicable fix available")
			}

			slog.Info("trying to get available fixes", "mode", fixModeUsed)
			availableFixes, err := fixPhase.GetAvailableFixes(result, fixModeUsed)
			warnRemoteDisabled := false
			if errors.Is(err, api.RemoteOverrideDisabledError) {
				// progress bar is currently running and can only print later, store in variable to print later
				// Setting availableFixes to empty so we can continue the regular flow
				slog.Warn("remote configuration is disabled, no fixes available")
				warnRemoteDisabled = true
				availableFixes = []shared.DependencyDescriptor{}
			} else if err != nil {
				slog.Error("failed getting available fixes", "err", err, "mode", fixModeUsed)
				return common.FallbackPrintableMsg(err, "failed querying for fixes")
			}

			var fixes []shared.DependencyDescriptor
			slog.Debug("fixes available", "mode", fixModeUsed, "available", len(availableFixes))
			if len(availableFixes) > 0 {
				slog.Info("attempting to apply fixes", "mode", fixModeUsed, "available", len(availableFixes))
				fixes, err = fixPhase.Fix(availableFixes, skipSignCheck) // fixes can be null
				if err != nil {
					return common.FallbackPrintableMsg(err, "failed performing fix")
				}

				slog.Info("checking callbacks")
				fixPhase.HandleCallbacks(fixes, nil, &blackduck.BlackDuckCallback{Config: fixPhase.Config})
				fixPhase.HandleCallbacks(fixes, result.Vulnerable, &dependabot.DependabotCallback{Config: fixPhase.Config})
				fixPhase.HandleCallbacks(fixes, nil, &ox.OxCallback{Config: fixPhase.Config})
			}

			silenced := make(map[string][]string, 0)
			if len(silenceArray) > 0 {
				slog.Info("silencing packages", "count", len(silenceArray))
				if silenced, err = fixPhase.Manager.SilencePackages(silenceArray, result.AllDependencies); err != nil {
					return common.FallbackPrintableMsg(err, "failed silencing packages")
				}
			}

			fixPhase.HideProgress() // should be gone here, but before handling summary make sure it gone
			if warnRemoteDisabled {
				fmt.Println(common.Colorize("Warning:", common.AnsiWarnYellow), "The remote configuration is currently disabled. All rules are inactive")
			}

			return outputSummary(summaryPath, fixes, silenced, fixPhase.BaseDir, "")
		},
	}

	cmd.Flags().String(summaryFlag, "", "output fix summary to path")
	cmd.Flags().String(modeFlag, "local", fmt.Sprintf("Fix mode, options: [%s|%s|%s]", phase.FixModeLocal, phase.FixModeRemote, phase.FixModeAll))
	cmd.Flags().Bool(skipSignCheckFlag, false, "skip signatures check")
	cmd.Flags().StringArray(silenceFlag, []string{}, "silence false-positive packages in the format of 'packageName@version'")
	return cmd
}
