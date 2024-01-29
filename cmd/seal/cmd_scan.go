package main

import (
	"cli/cmd/seal/output"
	"cli/internal/actions"
	"cli/internal/api"
	"cli/internal/common"
	"cli/internal/ecosystem/shared"
	"cli/internal/phase"
	"cli/internal/snyk"
	"fmt"
	"log/slog"
	"path/filepath"

	"github.com/spf13/cobra"
)

type ResultHandler interface {
	Handle([]api.PackageVersion, common.DependencyMap) error
}

const csvFlag = "csv"
const actionFlag = "generate-local-config"
const snykPolicyFlag = "generate-snyk-policy"

func initResultHandler(cmd *cobra.Command) (ResultHandler, error) {
	csvFilePath := getArgString(cmd, csvFlag)
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

func createActionsObject(packages []api.PackageVersion, manager shared.PackageManager, project string, projectDir string, targetDir string) *actions.ActionsFile {
	ps := actions.ProjectSection{
		Manager: actions.ProjectManagerSection{
			Ecosystem: manager.GetEcosystem(),
			Name:      manager.Name(),
			Version:   manager.GetVersion(projectDir),
		},
		Targets:   manager.GetScanTargets(),
		Overrides: make(actions.LibraryOverrideMap),
	}

	for _, p := range packages {
		if p.RecommendedLibraryVersionString == "" {
			slog.Debug("skipping package - no recommended version", "id", p.Id())
			continue
		}

		if ps.Overrides[p.Library.Name] == nil {
			ps.Overrides[p.Library.Name] = make(actions.VersionOverrideMap)
		}

		ps.Overrides[p.Library.Name][p.Version] = actions.Override{Version: p.RecommendedLibraryVersionString}
	}

	actionFile := actions.New()
	actionFile.Projects = map[string]actions.ProjectSection{project: ps}

	return actionFile
}

func convertActionsOverride(af *actions.ActionsFile) []api.PackageVersion {
	packages := make([]api.PackageVersion, 0, 10)
	if len(af.Projects) > 1 {
		slog.Warn("more than 1 project, not supported")
	}

	for projectName, projectSection := range af.Projects {
		slog.Debug("converting override in project", "project", projectName)
		for libraryName, versionMap := range projectSection.Overrides {
			slog.Debug("converting override in library", "lib", libraryName)
			for version, override := range versionMap {
				packages = append(packages, api.PackageVersion{
					Version:                         version,
					RecommendedLibraryVersionString: override.Version,
					// should add recommended library when supported from BE
					Library: api.Package{Name: libraryName, PackageManager: api.EcosystemToBackendManager(projectSection.Manager.Ecosystem)}, // ideally the ecosystem would be validated to be from currently supported list
				})
			}
		}
		break // supports only 1 project for now
	}

	return packages
}

func getExistingConfigOverrides(targetDir string) ([]api.PackageVersion, error) {
	actions, err := loadActionsFile(targetDir)
	if err != nil {
		slog.Error("failed opening local config", "err", err)
		return nil, common.FallbackPrintableMsg(err, "failed opening local config file")
	}

	if actions == nil {
		slog.Info("no local config found", "targetdir", targetDir)
		return nil, nil
	}

	return convertActionsOverride(actions), nil
}

func getMergedOverride(allDeps common.DependencyMap, remotePackages []api.PackageVersion, oldOverrides []api.PackageVersion) []api.PackageVersion {

	// the following maps are used to find remote package recommendations, even if the old overrides have been installed, therefore sending the 'installed' version; in case of old-override of `origin->sp1`, the server could return `sp1->sp2`, so we need to updated the override so it becomes `origin->sp2`
	overrideIds := make(map[string]api.PackageVersion)
	overrideRecommendedIds := make(map[string]api.PackageVersion)

	// filter out stale overrides, that are not present on disk at all (neither fixed, nor vulnerable versions)
	for _, override := range oldOverrides {
		if _, found := allDeps[override.Id()]; !found {
			if _, found = allDeps[override.RecommendedId()]; !found {
				slog.Debug("ignoring old override - not found in local deps", "id", override.Id(), "recommended id", override.RecommendedId())
				continue
			}
		}

		slog.Debug("keeping old override", "id", override.Id(), "recommended id", override.RecommendedId())
		overrideIds[override.Id()] = override
		overrideRecommendedIds[override.RecommendedId()] = override
	}

	combined := make([]api.PackageVersion, 0, len(oldOverrides)+len(remotePackages))

	// look for remote packages that update existing overrides
	// if not found, will be added as a new rule
	for _, remote := range remotePackages {
		override, found := overrideIds[remote.Id()]
		if !found {
			override, found = overrideRecommendedIds[remote.Id()]
		}

		if found {
			if remote.RecommendedLibraryVersionString != "" {
				slog.Debug("adding new override, using the remote recommendation", "id", override.Id(), "recommended-version", remote.RecommendedLibraryVersionString)
				override.RecommendedLibraryVersionString = remote.RecommendedLibraryVersionString
			} else {
				slog.Debug("remote is vulnerable with no recommended, keeping as is", "id", override.Id())
			}
			combined = append(combined, override)
			delete(overrideIds, override.Id())
		} else {
			slog.Debug("adding new override from remote", "id", remote.Id()) // remote doesn't necessarily have a recommended id
			combined = append(combined, remote)
		}
	}

	// add all remaining overrides that did not have any match from remote
	for _, override := range overrideIds {
		combined = append(combined, override)
	}

	return combined
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
			verbosity := getArgCount(cmd, verboseFlagKey)
			genActionsFile := getArgBool(cmd, actionFlag)
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

			// printing allowed from here

			if genActionsFile {
				slog.Info("generating local actions file")
				configOverrides := result.Vulnerable
				oldOverrides, err := getExistingConfigOverrides(targetDir)
				if err != nil {
					return common.FallbackPrintableMsg(err, "failed getting existing local config file")
				}

				if len(oldOverrides) > 0 {
					slog.Info("merging existing overrides file", "count", len(oldOverrides))
					configOverrides = getMergedOverride(result.AllDependencies, configOverrides, oldOverrides)
					slog.Info("new overrides", "count", len(configOverrides))
				}

				// Project name isn't validated when creating the actions file!
				ao := createActionsObject(configOverrides, scanPhase.Manager, scanPhase.Config.Project, scanPhase.ProjectDir, targetDir)

				w, err := common.CreateFile(filepath.Join(targetDir, actions.ActionFileName))
				if err != nil {
					return common.NewPrintableError("failed creating local config file")
				}

				err = actions.SaveActionFile(ao, w)
				if err != nil {
					slog.Error("failed saving action file", "err", err)
					return common.FallbackPrintableMsg(err, "failed saving to local config file")
				}

				genSnykPolicy := getArgBool(cmd, snykPolicyFlag) // only available if we are generating actions file
				snykUpdated := false
				if genSnykPolicy && len(configOverrides) > 0 {
					availableFixes, err := scanPhase.QueryFixesForPackages(configOverrides)
					if err != nil {
						slog.Error("failed querying fixes", "err", err)
						return common.WrapWithPrintable(err, "failed to get package metadata") // using wrap to show prettier error than internal server one
					}

					if len(availableFixes) > 0 {
						slog.Info("generating snyk policy")
						policyFilePath := filepath.Join(targetDir, snyk.PolicyFileName)
						// using overridden packages with versions from actions file too
						if err = output.EditSnykPolicyFile(policyFilePath, configOverrides, availableFixes); err != nil {
							return err // err already logged from func
						}
						snykUpdated = true
					}
				}

				if !snykUpdated {
					slog.Info("no available fixes, skipping snyk")
					fmt.Println(common.Colorize("Nothing to add to .snyk file", common.AnsiDarkGrey)) // Print to screen
				}
			}

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
	cmd.Flags().Bool(actionFlag, false, "generate a new local config file")
	cmd.Flags().Bool(snykPolicyFlag, false, fmt.Sprintf("generate or update the .snyk file (can only be used with --%s)", actionFlag))
	return cmd
}
