package npm

import (
	"cli/internal/common"
	"cli/internal/config"
	"cli/internal/ecosystem/node/utils"
	"cli/internal/ecosystem/shared"
	"os"

	"errors"
	"testing"

	"golang.org/x/exp/maps"
)

func TestDependencySignatureId(t *testing.T) {
	signature := "bbb|lodash@1.2.3"
	generated := common.DependencyId("bbb", "lodash", "1.2.3")
	if generated != signature {
		t.Fatalf("wrong dep version signature; generated: '%s' expected: '%s'", generated, signature)
	}
}

func TestParseNoDeps(t *testing.T) {
	conf, _ := config.New(nil)
	parser := dependencyParser{conf}

	dependencies, err := parser.Parse(getTestFile("no_deps.json"),
		defaultTestProjectDir)
	if err != nil {
		t.Fatalf("parse failed ")
	}

	if len(dependencies) != 0 {
		t.Fatalf("got more than 0 dependency: %v %v", len(dependencies), dependencies)
	}
}

func TestParseSingleDependency(t *testing.T) {
	conf, _ := config.New(nil)
	parser := dependencyParser{conf}

	dependencies, err := parser.Parse(getTestFile("single_dep.json"),
		defaultTestProjectDir)
	if err != nil {
		t.Fatalf("parse failed ")
	}

	if len(dependencies) != 1 {
		t.Fatalf("got more than 1 dependency: %v", len(dependencies))
	}

	deps := maps.Values(dependencies)
	dep := deps[0][0]

	if dep.PackageManager != shared.NpmManager {
		t.Fatalf("wrong package manager %v", dep.PackageManager)
	}

	if dep.Version != "0.0.1" {
		t.Fatalf("wrong version %v", dep.Version)
	}

	if dep.Name != "standalone" {
		t.Fatalf("wrong version %v", dep.Version)
	}

	if !dep.IsDirect() {
		t.Fatalf("did not detect as direct dep %v", dep)
	}
}

func TestParseSameDependencyMultipleVersions(t *testing.T) {
	conf, _ := config.New(nil)
	parser := dependencyParser{conf}

	dependencies, err := parser.Parse(getTestFile("multiple_versions.json"),
		defaultTestProjectDir)
	if err != nil {
		t.Fatalf("parse failed ")
	}

	if len(dependencies) != 3 {
		t.Fatalf("got wrong number of dependencies: %v %v", len(dependencies), dependencies)
	}

	lodashSignatureA := common.DependencyId(shared.NpmManager, "lodash", "4.17.11")
	lodashDepsA, ok := dependencies[lodashSignatureA]
	if !ok {
		t.Fatalf("could not find dependency: %s", lodashSignatureA)
	}

	if len(lodashDepsA) != 1 {
		t.Fatalf("got wrong number of lodash 4.17.11 deps %v", lodashDepsA)
	}

	if lodashDepsA[0].Name != "lodash" {
		t.Fatalf("expected first dep to be lodash, got %s", lodashDepsA[0].Name)
	}
	if lodashDepsA[0].Version != "4.17.11" {
		t.Fatalf("wrong version for first lodash; got '%s'", lodashDepsA[0].Version)
	}

	lodashSignatureB := common.DependencyId(shared.NpmManager, "lodash", "4.17.11")
	lodashDepsB, ok := dependencies[lodashSignatureB]
	if !ok {
		t.Fatalf("could not find dependency: %s", lodashSignatureB)
	}

	if len(lodashDepsB) != 1 {
		t.Fatalf("got wrong number of lodash 4.17.11 deps %v", lodashDepsB)
	}

	if lodashDepsB[0].Name != "lodash" {
		t.Fatalf("expected first dep to be lodash, got %s", lodashDepsB[0].Name)
	}
	if lodashDepsB[0].Version != "4.17.11" {
		t.Fatalf("wrong version for first lodash; got '%s'", lodashDepsB[0].Version)
	}

	// ignoreing dependency for cypress
}

func TestParseSingleDependencyNamespace(t *testing.T) {
	conf, _ := config.New(nil)
	parser := dependencyParser{conf}

	dependencies, err := parser.Parse(getTestFile("namespace_dep.json"),
		defaultTestProjectDir)
	if err != nil {
		t.Fatalf("parse failed ")
	}
	if len(dependencies) != 1 {
		t.Fatalf("got more than 1 dependency: %v", len(dependencies))
	}

	dep := maps.Values(dependencies)[0][0] // taking first instance

	if dep.PackageManager != shared.NpmManager {
		t.Fatalf("wrong package manager %v", dep.PackageManager)
	}
	if dep.Name != "@fastify/multipart" {
		t.Fatalf("wrong package %v", dep.Name)
	}
	if dep.Version != "8.0.0" {
		t.Fatalf("wrong version %v", dep.Version)
	}
}

func TestWrongCWD(t *testing.T) {
	conf, _ := config.New(nil)
	parser := dependencyParser{conf}

	_, err := parser.Parse(getTestFile("wrong_proj_dir.json"),
		"/Users/mococo/somefolder")
	if !errors.Is(err, utils.CwdWrongProjectDir) {
		t.Fatalf("Did not detect wrong cwd ")
	}
}

func TestSymlinkSkipped(t *testing.T) {
	// testing without npm output file since we're performing os.Lstat and needs to be created on disk
	conf, _ := config.New(nil)
	parser := dependencyParser{conf}

	// create link: .../test_syml_{random} -> .../test_seal_cli_{random}
	target, err := os.MkdirTemp("", "test_seal_cli_*")
	if err != nil {
		panic(err)
	}
	defer os.Remove(target)

	// create a temp dir then delete it to generate a safe name for symlink
	linkPath, err := os.MkdirTemp("", "test_syml_*")
	if err != nil {
		panic(err)
	}

	if err = os.Remove(linkPath); err != nil {
		panic(err)
	}

	if err := os.Symlink(target, linkPath); err != nil {
		panic(err)
	}
	defer os.Remove(linkPath) // removes the link

	// test
	if !parser.shouldSkip(&NpmPackage{
		Extraenous: false, // we don't want it to be skipped because it's extraneous
		Version:    "1.4.2",
		Name:       "name",
		Path:       linkPath,
	}) {
		t.Fatalf("did not detect symlink for %s", linkPath)
	}
}
