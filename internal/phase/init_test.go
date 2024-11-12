package phase

import (
	"cli/internal/api"
	"cli/internal/common"
	"cli/internal/config"
	"cli/internal/ecosystem/mappings"
	"cli/internal/ecosystem/shared"
	"cli/internal/project"
	"fmt"
	"os"
	"testing"
)

func TestProjectNameValid(t *testing.T) {

	for _, projName := range []string{
		"test",
		"my_proj",
		"my-proj",
		"my-proj1",
		"my.proj1",
		".",
		"1",
		"a1",
		"aA1",
		"a",
		"a..",
		"1.",
		".1.",
	} {
		t.Run(fmt.Sprintf("name_%s", projName), func(t *testing.T) {
			if msg := project.ValidateProjectId(projName); msg != "" {
				t.Fatalf("incorrectly checked valid project name `%s` : `%s`", projName, msg)
			}
		})
	}
}

func TestProjectNameInvalid(t *testing.T) {

	for _, projName := range []string{
		"a ",
		" a",
		"my proj",
		"a,",
	} {
		t.Run(fmt.Sprintf("name_%s", projName), func(t *testing.T) {
			if msg := project.ValidateProjectId(projName); msg == "" {
				t.Fatalf("project name should be invalid `%s` : `%s`", projName, msg)
			}
		})
	}
}

func TestGetProjectDir(t *testing.T) {
	d := t.TempDir()
	if projDir := getProjectDirAbs(d); projDir != d {
		t.Fatalf("got %s instead of %s", projDir, d)
	}
}

func TestGetProjectDirFromFile(t *testing.T) {
	d := t.TempDir()
	f, err := os.CreateTemp(d, "test_*")
	if err != nil {
		t.Fatalf("%v", err)
	}
	defer f.Close()

	fpath := f.Name()
	if projDir := getProjectDirAbs(fpath); projDir != d {
		t.Fatalf("got %s instead of %s for file %s", projDir, d, fpath)
	}
}

func TestGetProjectDirEmpty(t *testing.T) {
	if projDir := getProjectDirAbs(""); projDir != common.CliCWD {
		t.Fatalf("got %s instead of %s for file %s", projDir, "", common.CliCWD)
	}
}

func TestGetArtifactServerUrl(t *testing.T) {
	tests := []struct {
		ecosystem   string
		expectedUrl string
	}{
		{mappings.PythonEcosystem, api.PypiServer},
		{mappings.NodeEcosystem, api.NpmServer},
		{mappings.DotnetEcosystem, api.NugetServer},
		{mappings.JavaEcosystem, api.MavenServer},
		{mappings.NodeEcosystem, api.NpmServer},
		{mappings.GolangEcosystem, api.GolangServer},
		{"bad123", ""},
	}

	for _, test := range tests {
		t.Run(test.ecosystem, func(t *testing.T) {
			if url := getArtifactServerUrl(&shared.FakePackageManager{Ecosystem: test.ecosystem}, nil); url != test.expectedUrl {
				t.Fatalf("expected %s for ecosystem %s; got %s", test.expectedUrl, test.ecosystem, url)
			}
		})
	}
}

func TestGetJfrogArtifactRepo(t *testing.T) {

	conf := config.Config{}
	conf.JFrog.MavenRepository = "mvnvnv"

	tests := []struct {
		ecosystem   string
		expectedUrl string
	}{
		{mappings.JavaEcosystem, conf.JFrog.MavenRepository},
		{mappings.PythonEcosystem, ""},
		{mappings.NodeEcosystem, ""},
		{mappings.DotnetEcosystem, ""},
		{mappings.NodeEcosystem, ""},
		{mappings.GolangEcosystem, ""},
		{"bad123", ""},
	}

	for _, test := range tests {
		t.Run(test.ecosystem, func(t *testing.T) {
			if url := getJfrogArtifactRepo(&shared.FakePackageManager{Ecosystem: test.ecosystem}, &conf); url != test.expectedUrl {
				t.Fatalf("expected %s for ecosystem %s; got %s", test.expectedUrl, test.ecosystem, url)
			}
		})
	}
}
