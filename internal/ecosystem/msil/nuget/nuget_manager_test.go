package nuget

import (
	"cli/internal/api"
	"cli/internal/common"
	"cli/internal/ecosystem/msil/utils"
	"cli/internal/ecosystem/shared"
	"os"
	"path/filepath"
	"testing"
)

func TestParseVersion4(t *testing.T) {
	data := `NuGet Version: 4.1.0.2450
usage: NuGet <command> [args] [options]
`
	version := parseVersionOutput(data)
	if version != "4.1.0.2450" {
		t.Fatalf("wrong version, got `%s`", version)
	}
}

func TestParseVersion2(t *testing.T) {
	data := `NuGet Version: 2.8.60717.93
usage: NuGet <command> [args] [options]
Type 'NuGet help <command>' for hel`
	version := parseVersionOutput(data)
	if version != "2.8.60717.93" {
		t.Fatalf("wrong version, got `%s`", version)
	}
}

func TestParseVersion6(t *testing.T) {
	data := `NuGet Version: 6.11.1.2
usage: NuGet <command> [args] [options]
Type 'NuGet help <command>' for help on a specific command.

Available commands:`
	version := parseVersionOutput(data)
	if version != "6.11.1.2" {
		t.Fatalf("wrong version, got `%s`", version)
	}
}

func TestNugetVersionSupport(t *testing.T) {
	var m *NugetManager // dont worry about it ;p
	if m.IsVersionSupported("3.4.0") {
		t.Fatalf("badly supported version")
	}
	if m.IsVersionSupported("") {
		t.Fatalf("badly supported version")
	}

	if !m.IsVersionSupported("3.5.0") {
		t.Fatalf("badly supported version")
	}

	if !m.IsVersionSupported("6.5.0") {
		t.Fatalf("badly supported version")
	}
	if !m.IsVersionSupported("6.12.1.1") {
		t.Fatalf("badly supported version")
	}
}

func TestGetDefaultAuxFileForFormat(t *testing.T) {
	f := getDefaultAuxFileForFormat(utils.FormatLegacyPackagesConfig)
	if f != utils.DefaultPackagesConfigFile {
		t.Fatalf("wrong format %s", f)
	}

	f = getDefaultAuxFileForFormat(utils.FormatLegacy)
	if f != "" {
		t.Fatalf("wrong format %s", f)
	}
}

func TestGetPackagesDirForFormat(t *testing.T) {
	f := getPackagesDirForFormat(utils.FormatLegacyPackagesConfig)
	if f != utils.DefaultPackagesDirName {
		t.Fatalf("wrong format %s", f)
	}

	f = getPackagesDirForFormat(utils.FormatLegacy)
	if f != "" {
		t.Fatalf("wrong format %s", f)
	}
}

func TestGetProjectName(t *testing.T) {
	projname := "myproj.csproj"
	m := &NugetManager{
		targetFile: filepath.Join("abc", projname),
	}

	if name := m.GetProjectName(); name != projname {
		t.Fatalf("bad project name: `%s` expected: `%s`", name, projname)
	}

}

func TestDownloadNugetPackage(t *testing.T) {

	var manager *NugetManager
	ver := "1.3.5-sp1"
	lib := "MylibARR"

	fakeServer := &api.FakeArtifactServer{
		Data: []byte("asd"),
		GetValidator: func(uri string, params, extraHdrs []api.StringPair) (data []byte, code int, err error) {
			if uri != "v3-flatcontainer/mylibarr/1.3.5-sp1/mylibarr.1.3.5-sp1.nupkg" {
				t.Fatalf("bad download uri: `%s`", uri)
			}

			return []byte("dummy"), 200, nil
		},
	}

	_, _, _ = manager.DownloadPackage(fakeServer,
		shared.DependencyDescriptor{
			AvailableFix: &api.PackageVersion{
				Version: ver,
				Library: api.Package{
					Name:           lib,
					NormalizedName: utils.NormalizeName(lib),
				},
			},
		})
}

func TestFindSolutionDir(t *testing.T) {
	base, err := os.MkdirTemp("", "test_seal_cli_*")
	if err != nil {
		panic(err)
	}

	_, _ = common.CreateFile(filepath.Join(base, "test.sln"))

	projFile := filepath.Join(base, "d1", "d2", "d3", "d4", "test.csproj")
	_ = os.MkdirAll(filepath.Dir(projFile), 0o777)
	_, _ = common.CreateFile(projFile)

	slndir := findSolutionDir(projFile)
	if slndir != base {
		t.Fatalf("failed finding sln dir; got `%s` expected `%s`", slndir, base)
	}
}

func TestFindSolutionDirNotFound(t *testing.T) {
	base, err := os.MkdirTemp("", "test_seal_cli_*")
	if err != nil {
		panic(err)
	}

	projFile := filepath.Join(base, "d1", "d2", "d3", "d4", "test.csproj")
	_ = os.MkdirAll(filepath.Dir(projFile), 0o777)
	_, _ = common.CreateFile(projFile)

	slndir := findSolutionDir(projFile)
	if slndir != "" {
		t.Fatalf("got `%s` instead of empty", slndir)
	}
}

func TestFindSolutionDirTooDeep(t *testing.T) {
	base, err := os.MkdirTemp("", "test_seal_cli_*")
	if err != nil {
		panic(err)
	}

	projFile := filepath.Join(base, "d1", "d2", "d3", "d4", "d5", "d6", "d7", "d8", "d9", "d10", "d11", "test.csproj")
	_ = os.MkdirAll(filepath.Dir(projFile), 0o777)
	_, _ = common.CreateFile(projFile)

	slndir := findSolutionDir(projFile)
	if slndir != "" {
		t.Fatalf("got `%s` instead of empty", slndir)
	}
}
