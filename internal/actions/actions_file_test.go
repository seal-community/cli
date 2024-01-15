package actions

import (
	"cli/internal/common"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestSanityYamlFile(t *testing.T) {
	content := fmt.Sprintf(`
---
meta:
  schema-version: %s
  created-on: 2023-09-19T10:57:01Z # ISO 8601, utc time
  cli-version: %s

projects:
  my-project-id: # the same one in the config file / shown in ui
    targets: # what was used to scan, relative path to target directory scanned
      - ./package.json

    manager:
      ecosystem: node # to not confuse the user, and allow backend to tell internal 'manager'
      name: yarn
      version: 1.7

    overrides:
      ejs:
        2.7.4:
          use: 2.7.4-sp1
          from: seal-ejs
`, SchemaVersion, common.CliVersion)

	actions, err := Load(strings.NewReader(content))
	if actions == nil {
		t.Fatalf("failed loading actions: %v", err)
	}

	if actions.Meta.CliVersion != common.CliVersion {
		t.Fatalf("wrong cli version in loaded actions %s", actions.Meta.CliVersion)
	}

	expected, err := time.Parse(Iso8601FormatLayout, "2023-09-19T10:57:01Z")
	if err != nil {
		t.Fatalf("failed creating expected time: %v", err)
	}

	if expected != actions.Meta.CreatedOn.Time {
		t.Fatalf("wrong cli timestamp, expected %v, got %v", expected, actions.Meta.CreatedOn.Time)
	}

	proj, ok := actions.Projects["my-project-id"]
	if !ok {
		t.Fatal("failed finding project in yaml")
	}

	if proj.Manager.Name != "yarn" {
		t.Fatalf("wrong manager name %s", proj.Manager.Name)
	}

	if proj.Manager.Ecosystem != "node" {
		t.Fatalf("wrong ecosystem %s", proj.Manager.Ecosystem)
	}

	if proj.Manager.Version != "1.7" {
		t.Fatalf("wrong manager version %s", proj.Manager.Version)
	}

	lib, ok := proj.Overrides["ejs"]
	if !ok {
		t.Fatal("failed finding lib in yaml")
	}

	sealedPackage, ok := lib["2.7.4"]
	if !ok {
		t.Fatal("failed finding version in yaml")
	}

	if sealedPackage.Library != "seal-ejs" {
		t.Fatalf("wrong library name %s", sealedPackage.Library)
	}

	if sealedPackage.Version != "2.7.4-sp1" {
		t.Fatalf("wrong library version %s", sealedPackage.Version)
	}
}

func TestEmptyConfigFile(t *testing.T) {
	content := ``
	actions, err := Load(strings.NewReader(content))
	if actions != nil {
		t.Fatalf("did not fail loading empty file: %v", err)
	}
}

func TestNoExtraFields(t *testing.T) {
	content := fmt.Sprintf(`
---
meta:
  schema-version: %s
  created-on: 2023-09-19T10:57:01Z # ISO 8601, utc time
  cli-version: %s
  other: 0.1.2

projects:
  my-project-id: # the same one in the config file / shown in ui
    targets: # what was used to scan, relative path to target directory scanned
      - ./package.json

    manager:
      ecosystem: node # to not confuse the user, and allow backend to tell internal 'manager'
      name: yarn
      version: 1.7

    overrides:
      ejs:
        2.7.4:
          use: 2.7.4-sp1
          from: seal-ejs
`, SchemaVersion, common.CliVersion)
	actions, err := Load(strings.NewReader(content))

	if actions != nil {
		t.Fatalf("allowed extraneous field in actions: %v", actions)
	}

	if err != FailedParsingActionYaml {
		t.Fatalf("should fail parsing yaml with extraneous field: %v", err)
	}
}

func TestMissingFieldsErrProjects(t *testing.T) {
	content := `
---
meta:
  schema-version: 0.1.0
  created-on: 2023-09-19T10:57:01Z # ISO 8601, utc time
  cli-version: 0.1.1

`
	actions, err := Load(strings.NewReader(content))

	if actions != nil {
		t.Fatalf("allowed missing field in actions: %v", actions)
	}

	if err != FailedParsingActionYamlInvalid {
		t.Fatalf("should fail parsing yaml with missing field: %v", err)
	}
}

func TestMissingFieldsErrInMeta(t *testing.T) {
	content := fmt.Sprintf(`
---
meta:
  schema-version: %s
  created-on: 2023-09-19T10:57:01Z # ISO 8601, utc time

projects:
  my-project-id: # the same one in the config file / shown in ui
    targets: # what was used to scan, relative path to target directory scanned
      - ./package.json

    manager:
      ecosystem: node # to not confuse the user, and allow backend to tell internal 'manager'
      name: yarn
      version: 1.7

    overrides:
      ejs:
        2.7.4:
          use: 2.7.4-sp1
          from: seal-ejs
`, SchemaVersion)
	actions, err := Load(strings.NewReader(content))

	if actions != nil {
		t.Fatalf("allowed missing field in actions: %v", actions)
	}

	if err != FailedParsingActionYamlInvalid {
		t.Fatalf("should fail parsing yaml with missing field: %v", err)
	}
}

func TestMissingFieldsErrInsideProject(t *testing.T) {
	content := fmt.Sprintf(`
---
meta:
  schema-version: %s
  created-on: 2023-09-19T10:57:01Z # ISO 8601, utc time
  cli-version: %s

projects:
  my-project-id: # the same one in the config file / shown in ui
    manager:
      ecosystem: node # to not confuse the user, and allow backend to tell internal 'manager'
      name: yarn
      version: 1.7

    overrides:
      ejs:
        2.7.4:
          use: 2.7.4-sp1
          from: seal-ejs
`, SchemaVersion, common.CliVersion)

	actions, err := Load(strings.NewReader(content))

	if actions != nil {
		t.Fatalf("allowed missing field in actions: %v", actions)
	}

	if err != FailedParsingActionYamlInvalid {
		t.Fatalf("should fail parsing yaml with missing field: %v", err)
	}
}

func TestMissingFieldsErrInsideProjectManager(t *testing.T) {
	content := fmt.Sprintf(`
---
meta:
  schema-version: %s
  created-on: 2023-09-19T10:57:01Z # ISO 8601, utc time
  cli-version: %s

projects:
  my-project-id: # the same one in the config file / shown in ui
    targets: # what was used to scan, relative path to target directory scanned
      - ./package.json

    manager:
      name: yarn
      version: 1.7

    overrides:
      ejs:
        2.7.4:
          use: 2.7.4-sp1
          from: seal-ejs
`, SchemaVersion, common.CliVersion)
	actions, err := Load(strings.NewReader(content))

	if actions != nil {
		t.Fatalf("allowed missing field in actions: %v", actions)
	}

	if err != FailedParsingActionYamlInvalid {
		t.Fatalf("should fail parsing yaml with missing field: %v", err)
	}
}

func TestMissingFieldsErrInsideProjectOverrideVersionValue(t *testing.T) {
	content := fmt.Sprintf(`
---
meta:
  schema-version: %s
  created-on: 2023-09-19T10:57:01Z # ISO 8601, utc time
  cli-version: %s

projects:
  my-project-id: # the same one in the config file / shown in ui
    targets: # what was used to scan, relative path to target directory scanned
      - ./package.json

    manager:
      ecosystem: node # to not confuse the user, and allow backend to tell internal 'manager'
      name: yarn
      version: 1.7

    overrides:
      ejs:
        2.7.4:
          from: seal-ejs
`, SchemaVersion, common.CliVersion)

	actions, err := Load(strings.NewReader(content))

	if actions != nil {
		t.Fatalf("allowed missing field in actions: %v", actions)
	}

	if err != FailedParsingActionYamlInvalid {
		t.Fatalf("should fail parsing yaml with missing field: %v", err)
	}
}

func TestMissingFieldsOfLibraryIsAllowed(t *testing.T) {
	content := fmt.Sprintf(`
---
meta:
  schema-version: %s
  created-on: 2023-09-19T10:57:01Z # ISO 8601, utc time
  cli-version: %s

projects:
  my-project-id: # the same one in the config file / shown in ui
    targets: # what was used to scan, relative path to target directory scanned
      - ./package.json

    manager:
      ecosystem: node # to not confuse the user, and allow backend to tell internal 'manager'
      name: yarn
      version: 1.7

    overrides:
      ejs:
        2.7.4:
          use: 2.7.4-sp1
`, SchemaVersion, common.CliVersion)
	actions, err := Load(strings.NewReader(content))

	if actions == nil {
		t.Fatalf("not allowed to skip the from field: %v", actions)
	}

	if err != nil {
		t.Fatalf("shouldnt fail parsing yaml with missing use field: %v", err)
	}
}

func TestEmptyFieldsTargets(t *testing.T) {
	content := fmt.Sprintf(`
---
meta:
  schema-version: %s
  created-on: 2023-09-19T10:57:01Z 
  cli-version: %s

projects:
  my-project-id: 
    targets: []

    manager:
      ecosystem: node
      name: yarn
      version: 1.7

    overrides:
      ejs:
        2.7.4:
          use: 2.7.4-sp1
          from: seal-ejs
`, SchemaVersion, common.CliVersion)

	actions, err := Load(strings.NewReader(content))

	if actions != nil {
		t.Fatalf("not allowed to skip the from field: %v", actions)
	}

	if err != FailedParsingActionYamlInvalid {
		t.Fatalf("shouldnt fail parsing yaml with missing use field: %v", err)
	}
}

func TestEmptyFieldsOverrides(t *testing.T) {
	content := fmt.Sprintf(`
---
meta:
  schema-version: %s
  created-on: 2023-09-19T10:57:01Z 
  cli-version: %s

projects:
  my-project-id: 
    targets: 
      - ./package.json

    manager:
      ecosystem: node
      name: yarn
      version: 1.7

    overrides: {}
`, SchemaVersion, common.CliVersion)

	actions, err := Load(strings.NewReader(content))
	if actions == nil {
		t.Fatalf("should be allowed to have 0 projects: %v", err)
	}

	if err != nil {
		t.Fatalf("should not fail parsing yaml without projects: %v", err)
	}
}

func TestEmptyFieldsOverridesVerison(t *testing.T) {
	content := `
---
meta:
  schema-version: 0.1.0
  created-on: 2023-09-19T10:57:01Z 
  cli-version: 0.1.1

projects:
  my-project-id: 
    targets: 
      - ./package.json

    manager:
      ecosystem: node
      name: yarn
      version: 1.7

    overrides:
      ejs: {}
`
	actions, err := Load(strings.NewReader(content))

	if actions != nil {
		t.Fatalf("not allowed to skip the from field: %v", actions)
	}

	if err != FailedParsingActionYamlInvalid {
		t.Fatalf("shouldnt fail parsing yaml with missing use field: %v", err)
	}
}

func TestProjectsAtLeastOne(t *testing.T) {
	content := fmt.Sprintf(`
---
meta:
  schema-version: %s
  created-on: 2023-09-19T10:57:01Z # ISO 8601, utc time
  cli-version: %s

projects: {}
`, SchemaVersion, common.CliVersion)

	actions, err := Load(strings.NewReader(content))
	if actions != nil {
		t.Fatalf("not allowed to have 0 projects: %v", actions)
	}

	if err != FailedParsingActionYamlInvalid {
		t.Fatalf("should fail parsing yaml without projects: %v", err)
	}
}
func TestProjectsAtMostOne(t *testing.T) {
	// for now, only 1 project is supported
	content := fmt.Sprintf(`
---
meta:
  schema-version: %s
  created-on: 2023-09-19T10:57:01Z # ISO 8601, utc time
  cli-version: %s

projects:
  my-project-id: # the same one in the config file / shown in ui
    targets: # what was used to scan, relative path to target directory scanned
      - ./package.json

    manager:
      ecosystem: node # to not confuse the user, and allow backend to tell internal 'manager'
      name: yarn
      version: 1.7

    overrides:
      ejs:
        2.7.4:
          use: 2.7.4-sp1
          from: seal-ejs

  my-project-id2: # the same one in the config file / shown in ui
    targets: # what was used to scan, relative path to target directory scanned
      - ./package.json

    manager:
      ecosystem: node # to not confuse the user, and allow backend to tell internal 'manager'
      name: yarn
      version: 1.7

    overrides:
      ejs:
        2.7.4:
          use: 2.7.4-sp1
          from: seal-ejs
`, SchemaVersion, common.CliVersion)

	actions, err := Load(strings.NewReader(content))
	if actions != nil {
		t.Fatalf("not allowed to have more than 1 project: %v", len(actions.Projects))
	}

	if err != FailedParsingActionYamlInvalid {
		t.Fatalf("should fail parsing yaml with multiple projects: %v", err)
	}
}

func TestInvalidSchemaVersionMajor(t *testing.T) {
	content := `
---
meta:
  schema-version: 100.1.0
  created-on: 2023-09-19T10:57:01Z # ISO 8601, utc time
  cli-version: 0.1.1

projects:
  my-project-id: # the same one in the config file / shown in ui
    targets: # what was used to scan, relative path to target directory scanned
      - ./package.json

    manager:
      ecosystem: node # to not confuse the user, and allow backend to tell internal 'manager'
      name: yarn
      version: 1.7

    overrides:
      ejs:
        2.7.4:
          use: 2.7.4-sp1
          from: seal-ejs
`

	actions, err := Load(strings.NewReader(content))
	if actions != nil {
		t.Fatalf("not allowed to have different major")
	}

	if err == nil {
		t.Fatalf("should with err")
	}
}

func TestValidWithDifferentMinorSchemaVersion(t *testing.T) {
	content := `
---
meta:
  schema-version: 0.21321321321.0
  created-on: 2023-09-19T10:57:01Z # ISO 8601, utc time
  cli-version: 0.1.1

projects:
  my-project-id: # the same one in the config file / shown in ui
    targets: # what was used to scan, relative path to target directory scanned
      - ./package.json

    manager:
      ecosystem: node # to not confuse the user, and allow backend to tell internal 'manager'
      name: yarn
      version: 1.7

    overrides:
      ejs:
        2.7.4:
          use: 2.7.4-sp1
          from: seal-ejs
`

	actions, err := Load(strings.NewReader(content))
	if actions == nil {
		t.Fatalf("failed loading backwards compatible minor change in schema")
	}

	if err != nil {
		t.Fatalf("should not fail for different minor %v", err)
	}
}
