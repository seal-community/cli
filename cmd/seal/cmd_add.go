package main

import (
	"cli/cmd/seal/output/scanners"
	"cli/internal/actions"
	"cli/internal/api"
	"cli/internal/common"
	"cli/internal/grype"
	"cli/internal/phase"
	"cli/internal/snyk"
	"fmt"
	"log/slog"
	"path/filepath"

	"github.com/spf13/cobra"
)

func parseRule(args []string) (*phase.AddRule, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("not enough arguments")
	}

	// passing more arguments is unsupported for now
	if len(args) > 2 {
		return nil, fmt.Errorf("too many arguments")
	}

	fromLibrary := args[0]
	fromVersion := args[1]

	from := actions.Override{Library: fromLibrary, Version: fromVersion}
	to := &actions.Override{} // empty values denotes remote recommendation

	return &phase.AddRule{
		From: from,
		To:   to,
	}, nil
}

func findRule(target api.PackageVersion, existingOverrides []api.PackageVersion) *api.PackageVersion {
	// finds the first rule that has the same origin package and version
	// NOTE: assumes same manager
	for i, pv := range existingOverrides {
		if pv.Library.NormalizedName != target.Library.NormalizedName {
			continue
		}

		if pv.Version != target.Version {
			continue
		}

		slog.Debug("found rule from origin", "library", target.Library.Name, "version", target.Version)
		return &(existingOverrides[i])
	}

	slog.Debug("did not find rule from origin", "library", target.Library.Name, "version", target.Version)
	return nil
}

func upsertRule(resolved phase.ResolvedRule, existingOverrides *[]api.PackageVersion) (_ api.PackageVersion, modified bool, found bool) {
	// this assumes the To value of the resolved rule is set, must be checked by caller
	// NOTE: assumes same package name until backend data changes

	newOverride := findRule(resolved.From, *existingOverrides)
	if newOverride == nil {
		slog.Debug("adding rule, not found in overrides")
		// saving the copy of resolvedFrom, to be updated later
		// needed to update these fields for the actions file logic
		resolved.From.RecommendedLibraryVersionId = resolved.To.VersionId
		resolved.From.RecommendedLibraryVersionString = resolved.To.Version
		*existingOverrides = append(*existingOverrides, resolved.From)
		return resolved.From, true, false
	}

	slog.Debug("found rule's origin in actions file")
	if newOverride.RecommendedLibraryVersionString != resolved.To.Version {
		slog.Debug("rule found with different target version")
		// needed to update these fields for the actions file logic
		newOverride.RecommendedLibraryVersionId = resolved.To.VersionId
		newOverride.RecommendedLibraryVersionString = resolved.To.Version
		return *newOverride, true, true
	}

	return *newOverride, false, true
}

func addCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add package version",
		Short: "Add a package fix to actions file",
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			defer func() {
				// used to print error message on exit
				if err != nil {
					if printableErr := common.AsPrintable(err); printableErr != nil {
						fmt.Println(printableErr.Error())
						// overwrite so we could distinguish between usage error and more internal ones;
						// only doing for printable errors since we check the commandline args
						err = SubCommandError
					} else {
						slog.Warn("non printable error", "err", err)
					}
				}
			}()

			verbosity := getArgCount(cmd, verboseFlagKey)
			outputSnykPolicy := getArgBool(cmd, snykPolicyFlag)
			outputGrypePolicy := getArgBool(cmd, grypePolicyFlag)
			target := getArgString(cmd, manifestFile)
			targetDir := common.GetAbsDirPath(target)
			configPath := getArgString(cmd, configFileKey)

			addPhase, err := phase.NewAddPhase(targetDir, configPath, verbosity == 0)
			if err != nil {
				slog.Error("failed initializing scan", "err", err)
				return common.FallbackPrintableMsg(err, "failed initializing scan phase")
			}

			defer addPhase.HideProgress() // should be gone when this is over, hide just in case

			rule, err := parseRule(args)
			if err != nil {
				slog.Info("bad rule input", "arguments", args)
				// returning the inner error so the CLI will show usage
				return err
			}

			resolved, err := addPhase.Resolve(*rule)
			if err != nil || resolved.To == nil { // checking resolved target is not nil because for now it is required
				slog.Error("failed resolving rule", "err", err, "from", rule.From.Library, "version", rule.From.Version)
				return common.FallbackPrintableMsg(err, "failed resolving version")
			}

			addPhase.HideProgress() // explicitly stop the progress bar here, allow printing

			actionsFilePath := getArgString(cmd, actionsFileKey)
			if actionsFilePath == "" {
				actionsFilePath = filepath.Join(targetDir, actions.ActionFileName)
			}

			existingOverrides, err := getExistingConfigOverrides(actionsFilePath, addPhase.Manager)
			if err != nil {
				return common.FallbackPrintableMsg(err, "failed getting existing actions file")
			}

			if existingOverrides == nil {
				existingOverrides = make(map[string]api.PackageVersion)
			}

			existingOverridesArray := make([]api.PackageVersion, 0, len(existingOverrides))
			for _, v := range existingOverrides {
				existingOverridesArray = append(existingOverridesArray, v)
			}

			newOverride, modifedOverrides, foundInActionsFile := upsertRule(*resolved, &existingOverridesArray) // returns if we modified the slice
			slog.Debug("collecting")
			depMap, err := addPhase.Collect() // calling collect instead of scan because we only want what's on disk, no need to send request to BE
			if err != nil {
				slog.Error("failed performing local scan", "err", err)
				return common.FallbackPrintableMsg(err, "failed resolving rule")
			}

			foundInDependencies := false
			if _, found := depMap[resolved.From.Id()]; found {
				slog.Info("found the rule in existing dependencies", "id", resolved.From.Id())
				foundInDependencies = true
			}

			added := false
			if modifedOverrides {
				slog.Info("updating actions file with new rule")
				if err = recreateActionsFile(actionsFilePath, existingOverridesArray, addPhase.Manager, addPhase.Config.Project, addPhase.ProjectDir); err != nil {
					return err // only a wrapper func, logged from withing
				}
				added = true
			}

			// dup in policies should be handled by our editing code
			// using the newOverride var because it holds id/strings of recommended that were changed to match the resolvedTo package data (NOTE: important for editing policy logic, for now)
			if outputSnykPolicy {
				slog.Info("generating snyk policy")
				addedSnyk, err := scanners.EditSnykPolicyFile(filepath.Join(targetDir, snyk.PolicyFileName), []api.PackageVersion{newOverride}, []api.PackageVersion{*resolved.To})
				if err != nil {
					return err // err already logged from func
				}
				added = added || addedSnyk
			}

			if outputGrypePolicy {
				slog.Info("generating grype policy")
				addedGrype, err := scanners.EditGrypePolicyFile(filepath.Join(targetDir, grype.PolicyFileName), []api.PackageVersion{newOverride}, []api.PackageVersion{*resolved.To})
				if err != nil {
					return err // err already logged from func
				}
				added = added || addedGrype
			}

			if !added {
				// notify that there was nothing to update
				slog.Info("did not add anything")
				fmt.Println(common.Colorize("Nothing to add", common.AnsiDarkGrey)) // Print to screen
			} else {
				if !foundInDependencies && !foundInActionsFile {
					slog.Warn("did find the rule's target in project")
					fmt.Println(common.Colorize("Package not found in project", common.AnsiWarnYellow))
				}
			}

			return nil
		},
	}

	cmd.Flags().Bool(snykPolicyFlag, false, fmt.Sprintf("generate or update the %s file", snyk.PolicyFileName))
	cmd.Flags().Bool(grypePolicyFlag, false, fmt.Sprintf("generate or update the %s file", grype.PolicyFileName))
	cmd.Flags().String(manifestFile, "", "path to the manifest file to use")
	return cmd
}
