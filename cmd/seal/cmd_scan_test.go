package main

import (
	"cli/internal/actions"
	"cli/internal/api"
	"cli/internal/common"
	"cli/internal/ecosystem/mappings"
	"cli/internal/ecosystem/shared"
	"slices"

	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

type FakePackageManager struct {
}

func (m *FakePackageManager) Name() string {
	return "fakename"
}

func (m *FakePackageManager) GetEcosystem() string {
	return "fakeecosystem"
}

func (m *FakePackageManager) GetVersion(targetDir string) string {
	return "1.2.3"
}

func (m *FakePackageManager) IsVersionSupported(version string) bool {
	return true
}

func (m *FakePackageManager) GetProjectName(projectDir string) string {
	return ""
}

func (m *FakePackageManager) GetFixer(projectDir string, workdir string) shared.DependencyFixer {
	return nil
}

func (m *FakePackageManager) GetParser() shared.ResultParser {
	return nil
}

func (m *FakePackageManager) GetScanTargets() []string {
	return []string{"package.json"}
}

func (m *FakePackageManager) ListDependencies(targetDir string) (*common.ProcessResult, bool) {
	return nil, false
}

func (m *FakePackageManager) DownloadPackage(server api.Server, descriptor shared.DependnecyDescriptor) ([]byte, error) {
	return nil, nil
}

func (m *FakePackageManager) HandleFixes(projectDir string, fixes []shared.DependnecyDescriptor) error {
	return nil
}

func getTestVulns() []api.PackageVersion {
	vulns := []api.PackageVersion{
		{
			Version: "1.2.3",
			Library: api.Package{
				Name:           "lodash",
				PackageManager: mappings.NpmManager,
			},
			RecommendedLibraryVersionId:     "123123",
			RecommendedLibraryVersionString: "1.2.3-sp1",
		},
	}
	return vulns
}

func TestCreateActionObject(t *testing.T) {
	content := `
---
meta:
  schema-version: 0.0.0
  created-on: 2023-09-19T10:57:01Z # ISO 8601, utc time
  cli-version: 0.0.0

projects:
  project: # the same one in the config file / shown in ui
    targets: # what was used to scan, relative path to target directory scanned
      - package.json

    manager:
      ecosystem: fakeecosystem # to not confuse the user, and allow backend to tell internal 'manager'
      name: fakename
      version: 1.2.3

    overrides:
      lodash:
        1.2.3:
          use: 1.2.3-sp1
`
	manager := &FakePackageManager{}
	vulns := getTestVulns()
	actionsObject := createActionsObject(vulns, manager, "project", "projectDir")
	if actionsObject == nil {
		t.Fatalf("actionObject is nil")
	}

	actionsExpected, _ := actions.Load(strings.NewReader(content))

	if actionsObject.Meta.CliVersion != common.CliVersion {
		t.Fatalf("cli version is not equal")
	}

	if actionsObject.Meta.SchemaVersion != actions.SchemaVersion {
		t.Fatalf("schema version is not equal")
	}

	actionsYaml, _ := yaml.Marshal(actionsObject.Projects["project"])
	expectedActionsYaml, _ := yaml.Marshal(actionsExpected.Projects["project"])
	if string(actionsYaml) != string(expectedActionsYaml) {
		t.Fatalf("actions projects are not equal")
	}
}

func TestOverrideMergeSanitySamePackage(t *testing.T) {
	localDeps := common.DependencyMap{
		"NPM|lodash@1.2.3": nil,
	}
	remotePackages := []api.PackageVersion{
		{
			Version: "1.2.3",
			Library: api.Package{
				Name:           "lodash",
				PackageManager: mappings.NpmManager},
			RecommendedLibraryVersionString: "1.2.3-sp1",
		},
	}
	overrides := []api.PackageVersion{
		{
			Version: "1.2.3",
			Library: api.Package{
				Name:           "lodash",
				PackageManager: mappings.NpmManager},
			RecommendedLibraryVersionString: "1.2.3-sp1",
		},
	}

	combined := getMergedOverride(localDeps, remotePackages, overrides)
	if len(combined) != 1 {
		t.Fatalf("wrong number of combined overrides %d", len(combined))
	}

	o := combined[0]
	if o.Id() != "NPM|lodash@1.2.3" {
		t.Fatalf("wrong id for override %s", o.Id())
	}

	if o.RecommendedId() != "NPM|lodash@1.2.3-sp1" {
		t.Fatalf("wrong recommended id for override %s", o.RecommendedId())
	}
}

func TestOverrideMergeSanityFixedLocal(t *testing.T) {
	localDeps := common.DependencyMap{
		"NPM|lodash@1.2.3-sp1": nil,
	}
	remotePackages := []api.PackageVersion{}
	overrides := []api.PackageVersion{
		{
			Version: "1.2.3",
			Library: api.Package{
				Name:           "lodash",
				PackageManager: mappings.NpmManager},
			RecommendedLibraryVersionString: "1.2.3-sp1",
		},
	}
	combined := getMergedOverride(localDeps, remotePackages, overrides)
	if len(combined) != 1 {
		t.Fatalf("wrong number of combined overrides %d", len(combined))
	}

	o := combined[0]
	if o.Id() != "NPM|lodash@1.2.3" {
		t.Fatalf("wrong id for override %s", o.Id())
	}

	if o.RecommendedId() != "NPM|lodash@1.2.3-sp1" {
		t.Fatalf("wrong recommended id for override %s", o.RecommendedId())
	}
}

func TestOverrideMergeSanityNoRecommended(t *testing.T) {
	localDeps := common.DependencyMap{
		"NPM|lodash@1.2.3-sp1": nil,
	}
	remotePackages := []api.PackageVersion{
		{
			Version: "1.2.3-sp1",
			Library: api.Package{
				Name:           "lodash",
				PackageManager: mappings.NpmManager},
			RecommendedLibraryVersionString: "", // no fix available
		},
	}
	overrides := []api.PackageVersion{
		{
			Version: "1.2.3",
			Library: api.Package{
				Name:           "lodash",
				PackageManager: mappings.NpmManager},
			RecommendedLibraryVersionString: "1.2.3-sp1",
		},
	}
	combined := getMergedOverride(localDeps, remotePackages, overrides)
	if len(combined) != 1 {
		t.Fatalf("wrong number of combined overrides %d", len(combined))
	}

	o := combined[0]
	if o.Id() != "NPM|lodash@1.2.3" {
		t.Fatalf("wrong id for override %s", o.Id())
	}

	if o.RecommendedId() != "NPM|lodash@1.2.3-sp1" {
		t.Fatalf("wrong recommended id for override %s", o.RecommendedId())
	}
}

func TestOverrideMergeSanityNewSp2(t *testing.T) {
	localDeps := common.DependencyMap{
		"NPM|lodash@1.2.3-sp1": nil,
	}
	remotePackages := []api.PackageVersion{
		{
			Version: "1.2.3-sp1",
			Library: api.Package{
				Name:           "lodash",
				PackageManager: mappings.NpmManager},
			RecommendedLibraryVersionString: "1.2.3-sp2",
		},
	}
	overrides := []api.PackageVersion{
		{
			Version: "1.2.3",
			Library: api.Package{
				Name:           "lodash",
				PackageManager: mappings.NpmManager},
			RecommendedLibraryVersionString: "1.2.3-sp1",
		},
	}

	combined := getMergedOverride(localDeps, remotePackages, overrides)
	if len(combined) != 1 {
		t.Fatalf("wrong number of combined overrides %d", len(combined))
	}

	o := combined[0]
	if o.Id() != "NPM|lodash@1.2.3" {
		t.Fatalf("wrong id for override %s", o.Id())
	}

	if o.RecommendedId() != "NPM|lodash@1.2.3-sp2" {
		t.Fatalf("wrong recommended id for override %s", o.RecommendedId())
	}
}

func TestOverrideMergeOverrideRemovedIfNotInLocal(t *testing.T) {
	localDeps := common.DependencyMap{
		"NPM|lodash@1.2.3": nil,
	}
	remotePackages := []api.PackageVersion{
		{
			Version: "1.2.3",
			Library: api.Package{
				Name:           "lodash",
				PackageManager: mappings.NpmManager},
			RecommendedLibraryVersionString: "1.2.3-sp1",
		},
	}
	overrides := []api.PackageVersion{
		{
			Version: "1.0.0",
			Library: api.Package{
				Name:           "semver-regex",
				PackageManager: mappings.NpmManager},
			RecommendedLibraryVersionString: "1.0.0-sp1",
		},
	}

	combined := getMergedOverride(localDeps, remotePackages, overrides)
	if len(combined) != 1 {
		t.Fatalf("wrong number of combined overrides %d", len(combined))
	}

	o := combined[0]
	if o.Id() != "NPM|lodash@1.2.3" {
		t.Fatalf("wrong id for override %s", o.Id())
	}

	if o.RecommendedId() != "NPM|lodash@1.2.3-sp1" {
		t.Fatalf("wrong recommended id for override %s", o.RecommendedId())
	}
}

func TestOverrideMergeRemoteAddsNewOverride(t *testing.T) {
	localDeps := common.DependencyMap{
		"NPM|lodash@1.2.3":       nil,
		"NPM|semver-regex@1.0.0": nil,
	}
	remotePackages := []api.PackageVersion{
		{
			Version: "1.2.3",
			Library: api.Package{
				Name:           "lodash",
				PackageManager: mappings.NpmManager},
			RecommendedLibraryVersionString: "1.2.3-sp1",
		},
		{
			Version: "1.0.0",
			Library: api.Package{
				Name:           "semver-regex",
				PackageManager: mappings.NpmManager},
			RecommendedLibraryVersionString: "1.0.0-sp1",
		},
	}

	overrides := []api.PackageVersion{
		{
			Version: "1.2.3",
			Library: api.Package{
				Name:           "lodash",
				PackageManager: mappings.NpmManager},
			RecommendedLibraryVersionString: "1.2.3-sp1",
		},
	}

	combined := getMergedOverride(localDeps, remotePackages, overrides)
	slices.SortFunc(combined, func(a, b api.PackageVersion) int { return strings.Compare(a.Id(), b.Id()) })

	if len(combined) != 2 {
		t.Fatalf("wrong number of combined overrides %d", len(combined))
	}

	o := combined[0]
	if o.Id() != "NPM|lodash@1.2.3" {
		t.Fatalf("wrong id for override %s", o.Id())
	}

	if o.RecommendedId() != "NPM|lodash@1.2.3-sp1" {
		t.Fatalf("wrong recommended id for override %s", o.RecommendedId())
	}

	o2 := combined[1]
	if o2.Id() != "NPM|semver-regex@1.0.0" {
		t.Fatalf("wrong id for override %s", o2.Id())
	}

	if o2.RecommendedId() != "NPM|semver-regex@1.0.0-sp1" {
		t.Fatalf("wrong recommended id for override %s", o2.RecommendedId())
	}
}

func TestOverrideMergeRemoteAddsNewOverrideAfterFix(t *testing.T) {
	localDeps := common.DependencyMap{
		"NPM|lodash@1.2.3":           nil,
		"NPM|semver-regex@1.0.0-sp1": nil,
	}
	remotePackages := []api.PackageVersion{
		{
			Version: "1.2.3",
			Library: api.Package{
				Name:           "lodash",
				PackageManager: mappings.NpmManager},
			RecommendedLibraryVersionString: "1.2.3-sp1",
		},
	}

	overrides := []api.PackageVersion{
		{
			Version: "1.0.0",
			Library: api.Package{
				Name:           "semver-regex",
				PackageManager: mappings.NpmManager},
			RecommendedLibraryVersionString: "1.0.0-sp1",
		},
	}
	combined := getMergedOverride(localDeps, remotePackages, overrides)
	slices.SortFunc(combined, func(a, b api.PackageVersion) int { return strings.Compare(a.Id(), b.Id()) })

	if len(combined) != 2 {
		t.Fatalf("wrong number of combined overrides %d", len(combined))
	}

	o := combined[0]
	if o.Id() != "NPM|lodash@1.2.3" {
		t.Fatalf("wrong id for override %s", o.Id())
	}

	if o.RecommendedId() != "NPM|lodash@1.2.3-sp1" {
		t.Fatalf("wrong recommended id for override %s", o.RecommendedId())
	}

	o2 := combined[1]
	if o2.Id() != "NPM|semver-regex@1.0.0" {
		t.Fatalf("wrong id for override %s", o2.Id())
	}

	if o2.RecommendedId() != "NPM|semver-regex@1.0.0-sp1" {
		t.Fatalf("wrong recommended id for override %s", o2.RecommendedId())
	}
}

func TestOverrideMergeNoLocalDeps(t *testing.T) {
	localDeps := common.DependencyMap{}
	remotePackages := []api.PackageVersion{}
	overrides := []api.PackageVersion{
		{
			Version: "1.2.3",
			Library: api.Package{
				Name:           "lodash",
				PackageManager: mappings.NpmManager},
			RecommendedLibraryVersionString: "1.2.3-sp1",
		},
	}

	combined := getMergedOverride(localDeps, remotePackages, overrides)
	if len(combined) != 0 {
		t.Fatalf("wrong number of combined overrides %d", len(combined))
	}
}

func TestConvertingOverrideToPackageVersion(t *testing.T) {
	packages := convertActionsOverride(&actions.ActionsFile{
		Projects: map[string]actions.ProjectSection{
			"tests": {
				Manager: actions.ProjectManagerSection{
					Name:      "NPM",
					Ecosystem: "node",
				},
				Overrides: actions.LibraryOverrideMap{
					"ejs": actions.VersionOverrideMap{
						"2.7.4": actions.Override{
							Library: "ejs",
							Version: "2.7.4-sp1",
						},
					},
				},
			},
		},
	})

	expected := api.PackageVersion{
		Version: "2.7.4",
		Library: api.Package{
			Name:           "ejs",
			PackageManager: "NPM",
		},
		RecommendedLibraryVersionString: "2.7.4-sp1",
	}

	converted := packages[0]
	if expected.Version != converted.Version {
		t.Fatalf("wrong version %s", converted.Version)
	}

	if expected.RecommendedLibraryVersionString != converted.RecommendedLibraryVersionString {
		t.Fatalf("wrong recommended version %s", converted.RecommendedLibraryVersionString)
	}

	if expected.Library.Name != converted.Library.Name {
		t.Fatalf("wrong lib name %s", converted.Library.Name)
	}

	if expected.Library.PackageManager != converted.Library.PackageManager {
		t.Fatalf("wrong package manager name %s", converted.Library.PackageManager)
	}
}

func TestOverrideMergeMultipleOverrides(t *testing.T) {
	localDeps := common.DependencyMap{
		"NPM|lodash@1.2.3-sp1":       nil,
		"NPM|semver-regex@1.0.0-sp1": nil,
	}
	remotePackages := []api.PackageVersion{}
	overrides := []api.PackageVersion{
		{
			Version: "1.2.3",
			Library: api.Package{
				Name:           "lodash",
				PackageManager: mappings.NpmManager},
			RecommendedLibraryVersionString: "1.2.3-sp1",
		},
		{
			Version: "1.0.0",
			Library: api.Package{
				Name:           "semver-regex",
				PackageManager: mappings.NpmManager},
			RecommendedLibraryVersionString: "1.0.0-sp1",
		},
	}
	combined := getMergedOverride(localDeps, remotePackages, overrides)
	slices.SortFunc(combined, func(a, b api.PackageVersion) int { return strings.Compare(a.Id(), b.Id()) })

	if len(combined) != 2 {
		t.Fatalf("wrong number of combined overrides %d", len(combined))
	}

	o := combined[0]
	if o.Id() != "NPM|lodash@1.2.3" {
		t.Fatalf("wrong id for override %s", o.Id())
	}

	if o.RecommendedId() != "NPM|lodash@1.2.3-sp1" {
		t.Fatalf("wrong recommended id for override %s", o.RecommendedId())
	}

	o2 := combined[1]
	if o2.Id() != "NPM|semver-regex@1.0.0" {
		t.Fatalf("wrong id for override %s", o2.Id())
	}

	if o2.RecommendedId() != "NPM|semver-regex@1.0.0-sp1" {
		t.Fatalf("wrong recommended id for override %s", o2.RecommendedId())
	}
}
