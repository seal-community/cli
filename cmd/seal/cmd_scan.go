package main

import (
	"cli/cmd/seal/output"
	"cli/cmd/seal/output/scanners"
	"cli/internal/actions"
	"cli/internal/api"
	"cli/internal/common"

	"cli/internal/ecosystem/mappings"
	"cli/internal/ecosystem/shared"
	"cli/internal/grype"
	"cli/internal/phase"
	"cli/internal/project"
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
const actionFlagNew = "generate-actions-file"
const snykPolicyFlag = "generate-snyk-policy"
const grypePolicyFlag = "generate-grype-policy"
const manifestFile = "manifest"

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

func createActionsObject(packages []api.PackageVersion, manager shared.PackageManager, projectTag string, projectDir string) *actions.ActionsFile {

	// we need to normalize the names to not constantly change windows <-> linux paths
	targets := manager.GetScanTargets()
	normTargets := make([]string, 0, len(targets))
	for _, target := range targets {
		normTargets = append(normTargets, project.NormalizeTarget(target))
	}

	ps := actions.ProjectSection{
		Manager: actions.ProjectManagerSection{
			Ecosystem: manager.GetEcosystem(),
			Name:      manager.Name(),
			Version:   manager.GetVersion(),
		},
		Targets:   normTargets,
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
	actionFile.Projects = map[string]actions.ProjectSection{projectTag: ps}

	return actionFile
}

func recreateActionsFile(actionsFilePath string, overrides []api.PackageVersion, manager shared.PackageManager, project string, projectDir string) error {
	slog.Info("recreating actions file", "path", actionsFilePath)

	ao := createActionsObject(overrides, manager, project, projectDir) // should not fail
	w, err := common.CreateFile(actionsFilePath)
	if err != nil {
		return common.NewPrintableError("failed creating actions file")
	}

	err = actions.SaveActionFile(ao, w)
	if err != nil {
		slog.Error("failed saving action file", "err", err)
		return common.FallbackPrintableMsg(err, "failed saving to actions file")
	}

	return nil
}

// creates fake PackageVersion for each override, assumes only 1 project
func convertActionsOverride(af *actions.ActionsFile, normalizer shared.Normalizer) map[string]api.PackageVersion {
	packages := make(map[string]api.PackageVersion)
	if len(af.Projects) > 1 {
		slog.Warn("more than 1 project, not supported")
	}

	for projectName, projectSection := range af.Projects {
		slog.Debug("converting override in project", "project", projectName)
		for libraryName, versionMap := range projectSection.Overrides {
			slog.Debug("converting override in library", "lib", libraryName)
			for version, override := range versionMap {
				p := api.PackageVersion{
					Version:                         version,
					RecommendedLibraryVersionString: override.Version,
					// should add recommended library when supported from BE
					Library: api.Package{
						Name:           libraryName,
						NormalizedName: normalizer.NormalizePackageName(libraryName),
						PackageManager: mappings.EcosystemToBackendManager(projectSection.Manager.Ecosystem),
					}, // ideally the ecosystem would be validated to be from currently supported list
				}
				if _, ok := packages[p.Id()]; ok {
					slog.Warn("duplicate override found", "id", p.Id())
				}
				packages[p.Id()] = p
			}
		}
		break // supports only 1 project for now
	}

	return packages
}

func getExistingConfigOverrides(actionsFilePath string, normalizer shared.Normalizer) (map[string]api.PackageVersion, error) {
	slog.Info("loading existing actions file", "path", actionsFilePath)
	actions, err := loadActionsFile(actionsFilePath)
	if err != nil {
		slog.Error("failed opening actions file", "err", err)
		return nil, common.FallbackPrintableMsg(err, "failed opening actions file")
	}

	if actions == nil {
		slog.Info("no actions config found", "path", actionsFilePath)
		return nil, nil
	}

	return convertActionsOverride(actions, normalizer), nil
}

func getMergedOverride(allDeps common.DependencyMap, remotePackages []api.PackageVersion, oldOverrides map[string]api.PackageVersion) []api.PackageVersion {

	// the following maps are used to find remote package recommendations, even if the old overrides have been installed, therefore sending the 'installed' version; in case of old-override of `origin->sp1`, the server could return `sp1->sp2`, so we need to updated the override so it becomes `origin->sp2`
	overrideIds := make(map[string]api.PackageVersion)
	overrideRecommendedIds := make(map[string]api.PackageVersion)

	// filter out stale overrides, that are not present on disk at all (neither fixed, nor vulnerable versions)
	for identifier, override := range oldOverrides {
		if _, found := allDeps[identifier]; !found {
			if _, found = allDeps[override.RecommendedId()]; !found {
				slog.Debug("ignoring old override - not found in local deps", "id", identifier, "recommended id", override.RecommendedId())
				continue
			}
		}

		slog.Debug("keeping old override", "id", override.Id(), "recommended id", override.RecommendedId())
		overrideIds[identifier] = override
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

			target := extractTarget(args)

			targetDir := common.GetTargetDir(target)
			if targetDir == "" {
				slog.Error("bad target input", "target", target)
				return common.NewPrintableError("target not found `%s`", target)
			}

			configPath := getArgString(cmd, configFileKey)
			verbosity := getArgCount(cmd, verboseFlagKey)
			genActionsFile := getArgBool(cmd, actionFlag) || getArgBool(cmd, actionFlagNew)
			uploadScanActivity := getArgBool(cmd, uploadResultsKey)

			scanPhase, err := phase.NewScanPhase(target, configPath, verbosity == 0)
			if err != nil {
				slog.Error("failed initializing scan", "err", err)
				return common.FallbackPrintableMsg(err, "failed initializing scan phase")
			}

			defer scanPhase.HideProgress() // should be gone when this is over, hide just in case

			// auth is optional for scaning purposes
			if scanPhase.CanAuthenticate {
				slog.Info("authenticating")
				if err := scanPhase.InitRemoteProject(); err != nil {
					return common.FallbackPrintableMsg(err, "failed initializing project from server")
				}
			}

			result, err := scanPhase.Scan(uploadScanActivity)

			if err != nil {
				return common.FallbackPrintableMsg(err, "failed performing scan")
			}

			scanPhase.HideProgress() // should be gone here, but before handling output

			// printing allowed from here

			if genActionsFile {

				actionsFilePath := getArgString(cmd, actionsFileKey)
				if actionsFilePath == "" {
					actionsFilePath = filepath.Join(targetDir, actions.ActionFileName)
				}

				slog.Info("loading actions file", "path", actionsFilePath)
				configOverrides := result.Vulnerable
				oldOverrides, err := getExistingConfigOverrides(actionsFilePath, scanPhase.Manager)
				if err != nil {
					return common.FallbackPrintableMsg(err, "failed getting existing actions file")
				}

				if len(oldOverrides) > 0 {
					slog.Info("merging existing overrides file", "count", len(oldOverrides))
					configOverrides = getMergedOverride(result.AllDependencies, configOverrides, oldOverrides)
					slog.Info("new overrides", "count", len(configOverrides))
				}

				// Project name isn't validated when creating the actions file!
				if err = recreateActionsFile(actionsFilePath, configOverrides, scanPhase.Manager, scanPhase.Project.Tag, scanPhase.BaseDir); err != nil {
					// only a wrapper func, logged from withing
					return err
				}

				// only available if we are generating actions file
				genSnykPolicy, genGrypePolicy := getArgBool(cmd, snykPolicyFlag), getArgBool(cmd, grypePolicyFlag)
				snykUpdated, grypeUpdated := false, false

				if (genSnykPolicy || genGrypePolicy) && len(configOverrides) > 0 {
					availableFixes, err := scanPhase.QueryRecommendedPackages(configOverrides)
					if err != nil {
						slog.Error("failed querying fixes", "err", err)
						return common.WrapWithPrintable(err, "failed to get package metadata") // using wrap to show prettier error than internal server one
					}

					if len(availableFixes) > 0 {
						// using overridden packages with versions from actions file too
						// errs are logged from scanner funcs
						if genSnykPolicy {
							slog.Info("generating snyk policy")
							policyFilePath := filepath.Join(targetDir, snyk.PolicyFileName)
							if snykUpdated, err = scanners.EditSnykPolicyFile(policyFilePath, configOverrides, availableFixes); err != nil {
								return err
							}
						}

						if genGrypePolicy {
							slog.Info("generating grype policy") // may be printed after snyk
							policyFilePath := filepath.Join(targetDir, grype.PolicyFileName)
							if grypeUpdated, err = scanners.EditGrypePolicyFile(policyFilePath, configOverrides, availableFixes); err != nil {
								return err
							}
						}
					}
				}

				if genSnykPolicy && !snykUpdated {
					slog.Info("no available fixes, skipping snyk")
					fmt.Println(common.Colorize(fmt.Sprintf("Nothing to add to %s file", snyk.PolicyFileName), common.AnsiDarkGrey)) // Print to screen
				}

				if genGrypePolicy && !grypeUpdated {
					slog.Info("no available fixes, skipping grype")
					fmt.Println(common.Colorize(fmt.Sprintf("Nothing to add to %s file", grype.PolicyFileName), common.AnsiDarkGrey)) // Print to screen
				}
			}

			if len(result.Vulnerable) == 0 {
				slog.Info("no vulnerable package found", "target", scanPhase.BaseDir)
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

	cmd.Flags().Bool(actionFlag, false, "generate a new actions file")
	_ = cmd.Flags().MarkHidden(actionFlag) // will still work, but will not be shown anywhere

	cmd.Flags().String(csvFlag, "", "output results as csv to path")
	cmd.Flags().Bool(actionFlagNew, false, "generate a new seal actions file")
	cmd.Flags().Bool(snykPolicyFlag, false, fmt.Sprintf("generate or update the .snyk file (can only be used with --%s)", actionFlagNew))
	cmd.Flags().Bool(grypePolicyFlag, false, fmt.Sprintf("generate or update the .grype.yaml file (can only be used with --%s)", actionFlagNew))

	return cmd
}
