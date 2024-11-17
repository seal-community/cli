package dotnet

import (
	"cli/internal/api"
	"cli/internal/common"
	"cli/internal/config"
	"cli/internal/ecosystem/shared"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDotnetManagerDetectionNoDotnetFile(t *testing.T) {
	target, err := os.MkdirTemp("", "test_seal_cli_*")
	if err != nil {
		panic(err)
	}

	defer os.Remove(target)

	indctr, err := FindDotnetIndicatorFile(target)
	if err != nil {
		t.Fatalf("had error %v", err)
	}

	if indctr != "" {
		t.Fatal("detected Dotnet")
	}
}

func TestDotnetManagerDetectionDotnetFile(t *testing.T) {
	target, err := os.MkdirTemp("", "test_seal_cli_*")
	if err != nil {
		panic(err)
	}

	defer os.Remove(target)

	for _, suffixIndicator := range dotnetSuffixIndicators {
		p := filepath.Join(target, suffixIndicator)
		f, err := os.Create(p)
		if err != nil {
			panic(err)
		}

		f.Close()

		indctr, err := FindDotnetIndicatorFile(target)
		if err != nil {
			t.Fatalf("had error %v", err)
		}

		if indctr == "" {
			t.Fatal("failed to detect Dotnet")
		}
	}
}

func TestHandleFixes(t *testing.T) {
	target, err := os.MkdirTemp("", "test_seal_cli_*")
	if err != nil {
		panic(err)
	}

	defer os.Remove(target)

	objDir := filepath.Join(target, "obj")
	err = os.Mkdir(objDir, 0755)
	if err != nil {
		panic(err)
	}

	data := getTestFile("project.assets.json")
	p := filepath.Join(objDir, "project.assets.json")
	f, err := os.Create(p)
	if err != nil {
		panic(err)
	}
	if n, err := f.Write([]byte(data)); err != nil {
		panic(err)
	} else if n != len(data) {
		panic("failed to write all data")
	}

	f.Close()

	packageVersion := api.PackageVersion{
		VersionId: "Snappier",
		Version:   "1.1.0",
		Library: api.Package{
			Name:           "Snappier",
			NormalizedName: "snappier",
			PackageManager: "dotnet",
		},
		RecommendedLibraryVersionId:     "1.1.0-sp1",
		RecommendedLibraryVersionString: "1.1.0-sp1",
	}
	fixedVersion := api.PackageVersion{
		VersionId:           "1.1.0-sp1",
		Version:             "1.1.0-sp1",
		OriginVersionString: "Snappier",
		OriginVersionId:     "Snappier",
		Library: api.Package{
			Name:           "Snappier",
			NormalizedName: "snappier",
			PackageManager: "dotnet",
		},
		RecommendedLibraryVersionId:     "",
		RecommendedLibraryVersionString: "",
	}

	fixes := []shared.DependencyDescriptor{
		{
			Locations: map[string]common.Dependency{
				"Snappier.1.1.0-sp1.nupkg": {},
			},
			FixedLocations:    []string{"Snappier.1.1.0-sp1.nupkg"},
			VulnerablePackage: &packageVersion,
			AvailableFix:      &fixedVersion,
		},
	}

	if err := handleFixes(target, fixes); err != nil {
		t.Fatalf("failed to update project.assets.json")
	}

	assetsData, err := os.ReadFile(p)
	if err != nil {
		panic(err)
	}

	count := strings.Count(string(assetsData), "1.1.0-sp1")
	if count != 6 {
		t.Fatalf("did not find 1.1.0-sp1 6 times in the file, found only %v", count)
	}
}

func TestIndicatorMatches(t *testing.T) {
	ps := []string{
		`/b/t.csproj`,
		`C:\t.csproj`,
		`../t.csproj`,
		`..\t.csproj`,
		`./abc/../t.csproj`,
		`.\abc\..\t.csproj`,
	}

	for i, p := range ps {
		t.Run(fmt.Sprintf("pth_%d", i), func(t *testing.T) {
			if !IsDotnetIndicatorFile(p) {
				t.Fatalf("failed to detect indicator path `%s`", p)
			}
		})
	}
}

func TestIndicatorDoesNotMatchPackageJson(t *testing.T) {
	// as it is intended to be handled by dir
	ps := []string{
		`/b/t.sln`,
		`C:\t.sln`,
		`../t.sln`,
		`..\t.sln`,
		`./abc/../t.sln`,
		`.\abc\..\t.sln`,
	}

	for i, p := range ps {
		t.Run(fmt.Sprintf("pth_%d", i), func(t *testing.T) {
			if IsDotnetIndicatorFile(p) {
				t.Fatalf("failed to detect indicator path `%s`", p)
			}
		})
	}
}

func TestNormalizePackageNames(t *testing.T) {
	c, _ := config.New(nil)
	manager := NewDotnetManager(c, "", "")
	names := map[string]string{
		"aaaaa": "aaaaa",
		"aaAAa": "aaaaa",
		"AAAAA": "aaaaa",
		"AAa_a": "aaa_a",
		"AaA-a": "aaa-a",
	}
	for n, expected := range names {
		t.Run(fmt.Sprintf("name_%s", n), func(t *testing.T) {
			if manager.NormalizePackageName(n) != expected {
				t.Fatalf("failed to normalize `%s`", n)
			}
		})
	}
}
