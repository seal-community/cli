package main

import (
	"cli/cmd/seal/output"
	"cli/internal/actions"
	"cli/internal/api"
	"cli/internal/common"
	"cli/internal/phase"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"golang.org/x/exp/maps"
)

const summaryFlag = "summarize"
const actionsIgnoreFlag = "ignore-local-config"

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

func loadActionsFile(targetDir string) (*actions.ActionsFile, error) {
	actionsFilePath := filepath.Join(targetDir, actions.ActionFileName)
	f, err := os.Open(actionsFilePath)
	if err != nil {
		if !os.IsNotExist(err) {
			slog.Info("failed opening conf file", "err", err, "path", actionsFilePath)
			return nil, common.NewPrintableError("could not open local config file in %s", actionsFilePath)
		}
		slog.Info("actions file not found", "path", actions.FailedParsingActionYamlInvalid)
		return nil, nil

	} else {
		defer f.Close()
	}

	// NOTE: project id from the actions file can differ from what we discover in fix phase - currently ok to ignore
	return actions.Load(f)
}

func filterVulnerablePackageForProject(vulnPackages []api.PackageVersion, projectSection actions.ProjectSection) []api.PackageVersion {
	overriddenPackages := make([]api.PackageVersion, 0, len(vulnPackages))

	for _, vulnPackage := range vulnPackages {
		if vulnPackage.RecommendedLibraryVersionId == "" {
			// do not allow to 'abuse' this to install package inplace of the vulnerable one if we don't have a SP for it
			slog.Debug("ignoring vulnerable package - does not have a sealed version", "packageId", vulnPackage.Id())
			continue
		}

		ecosystem := vulnPackage.Ecosystem()
		if ecosystem != projectSection.Manager.Ecosystem {
			slog.Debug("ignoring vulnerable package - different ecosystem", "package-ecosystem", ecosystem, "project-ecosystem", projectSection.Manager.Ecosystem)
			continue
		}

		versionMap, ok := projectSection.Overrides[vulnPackage.Library.Name]
		if !ok {
			slog.Debug("ignoring vulnerable package - not in allowed overrides", "name", vulnPackage.Library.Name)
			continue
		}
		override, ok := versionMap[vulnPackage.Version]
		if !ok {
			slog.Debug("ignoring vulnerable package version, not in allowed overrides", "name", vulnPackage.Library.Name, "version", vulnPackage.Version)
			continue
		}

		slog.Debug("overriding package", "from", fmt.Sprintf("%s@%s", vulnPackage.Library.Name, vulnPackage.Version), "to", fmt.Sprintf("%s@%s", override.Library, override.Version))
		vulnPackage.RecommendedLibraryVersionString = override.Version
		// keeping RecommendedLibraryVersionId as it is used to filter out non fixable packages; we will need to update this from BE
		if override.Library != "" {
			vulnPackage.Library.Name = override.Library
		}

		vulnPackage.OverrideMethod = api.OverriddenFromLocal
		overriddenPackages = append(overriddenPackages, vulnPackage) // will be copied, needs to keep all the other information like vulnerabilities
	}

	// this can return a 0 length slice if no packages are allowed, which is OK
	return overriddenPackages
}

func getVulnerablePackagesAccordingToOverride(vulnPackages []api.PackageVersion, actions *actions.ActionsFile) []api.PackageVersion {
	if len(actions.Projects) > 1 {
		// should be prevented by the validator when loading
		slog.Error("found more than 1 project in actions file - using first", "count", len(actions.Projects))
	}

	projectId := maps.Keys(actions.Projects)[0]
	projectSection := actions.Projects[projectId]
	slog.Info("filtering according to overrides in project", "id", projectId)
	return filterVulnerablePackageForProject(vulnPackages, projectSection)
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
			verbosity := getArgCount(cmd, verboseFlagKey)
			ignoreActionsFile := getArgBool(cmd, actionsIgnoreFlag)
			summaryPath := getArgString(cmd, summaryFlag)

			// IMPORTANT - after this point printing directly to console would mess up the progress bar, msg should be used instead
			fixPhase, err := phase.NewFixPhase(targetDir, verbosity == 0)
			if err != nil {
				slog.Error("failed initializing fix", "err", err)
				return common.FallbackPrintableMsg(err, "failed initializing fix phase")
			}

			defer fixPhase.HideProgress() // should be gone when this is over, hide just in case

			var actions *actions.ActionsFile = nil
			if !ignoreActionsFile {
				// performing here for better experience in case of invalid file
				slog.Info("loading actions file")
				actions, err = loadActionsFile(targetDir)
				if err != nil {
					slog.Error("failed opening local config for fix", "err", err)
					return common.FallbackPrintableMsg(err, "failed loading local config")
				}
				// NOTE: ideally we change the BE to support querying any package, then we pre-fetch it here and notify user, instead of failing later
			}

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

			if actions != nil {
				// replace existing slice
				slog.Info("limiting results according to actions file", "before", len(result.Vulnerable))
				overriddenPackages := getVulnerablePackagesAccordingToOverride(result.Vulnerable, actions)
				result.Vulnerable = overriddenPackages // even if we have 0 after filtering, so we don't fix anything
				slog.Info("total available vulnerable after overriding", "count", len(overriddenPackages))
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
	cmd.Flags().Bool(actionsIgnoreFlag, false, "ignore definitions in local config")
	return cmd
}
