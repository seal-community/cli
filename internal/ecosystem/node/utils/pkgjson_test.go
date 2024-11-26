//go:build !windows

package utils

import (
	"cli/internal/api"
	"cli/internal/common"
	"fmt"
	"os"
	"testing"

	"github.com/iancoleman/orderedmap"
)

func TestCalculateSealedName(t *testing.T) {
	namesToTest := []api.StringPair{
		{Name: "", Value: ""},
		{Name: "blah", Value: "@seal-security/blah"},
		{Name: "@test/blah", Value: "@seal-security/test-blah"},
	}
	for i, pair := range namesToTest {
		t.Run(fmt.Sprintf("name_%d", i), func(t *testing.T) {
			res := calculateSealedName(pair.Name)
			if res != pair.Value {
				t.Fatalf("failed to calculate sealed name `%s` -> %s != %s", pair, res, pair.Value)
			}
		})
	}
}

func TestAddSealPrefixToPackageLockFile(t *testing.T) {
	target, err := os.MkdirTemp("", "test_seal_cli_*")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(target)
	fakePkgJson := orderedmap.New()
	if fakePkgJson == nil {
		t.Fatalf("failed to create fake package json")
	}
	fakePkgJson.Set("name", "test")
	fakePkgJson.Set("version", "1.0.0")
	packageJsonFilePath := getPackageJsonFilePath(target)
	if packageJsonFilePath == "" {
		t.Fatalf("failed to get package json file path")
	}
	err = common.JsonSave(fakePkgJson, packageJsonFilePath)
	if err != nil {
		t.Fatalf("failed to save fake package json")
	}
	err = AddSealPrefixToPackageJsonFile(target)
	if err != nil {
		t.Fatalf("failed to add seal prefix to package lock file - %s", err)
	}
	afterPackageJson := loadPackageJson(target)
	if afterPackageJson == nil {
		t.Fatalf("failed to load altered package json")
	}
	alteredName := getProjectName(afterPackageJson)
	if alteredName != "@seal-security/test" {
		t.Fatalf("failed to add seal prefix to package json file -  %s", alteredName)
	}
}
