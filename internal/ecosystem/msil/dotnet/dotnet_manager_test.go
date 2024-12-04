package dotnet

import (
	"cli/internal/api"
	"cli/internal/config"
	"cli/internal/ecosystem/msil/utils"
	"cli/internal/ecosystem/shared"
	"fmt"
	"os"
	"path/filepath"
	"slices"
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

func TestDownloadNugetPackage(t *testing.T) {

	var manager *DotnetPackageManager
	ver := "1.3.5-sp1"
	lib := "MylibARR"

	data := []byte("dummy")
	code := 200
	var err error = nil

	fakeServer := &api.FakeArtifactServer{
		Data: []byte("asd"),
		GetValidator: func(uri string, params, extraHdrs []api.StringPair) ([]byte, int, error) {
			if uri != "v3-flatcontainer/mylibarr/1.3.5-sp1/mylibarr.1.3.5-sp1.nupkg" {
				t.Fatalf("bad download uri: `%s`", uri)
			}

			return data, code, err
		},
	}

	rData, _, rErr := manager.DownloadPackage(fakeServer,
		shared.DependencyDescriptor{
			AvailableFix: &api.PackageVersion{
				Version: ver,
				Library: api.Package{
					Name:           lib,
					NormalizedName: utils.NormalizeName(lib),
				},
			},
		})

	if !slices.Equal(rData, data) {
		t.Fatal("wrong data")
	}
	if rErr != err {
		t.Fatal("wrong err")
	}
}
