package npm

import (
	"cli/internal/common"
	"cli/internal/config"
	"cli/internal/ecosystem/mappings"
	"cli/internal/ecosystem/node/utils"
	"os"
	"path/filepath"
	"strings"

	"errors"
	"testing"

	"golang.org/x/exp/maps"
)

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

	if dep.PackageManager != mappings.NpmManager {
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

	lodashSignatureA := common.DependencyId(mappings.NpmManager, "lodash", "4.17.11")
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

	lodashSignatureB := common.DependencyId(mappings.NpmManager, "lodash", "4.17.11")
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

	if dep.PackageManager != mappings.NpmManager {
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
	root := &NpmPackage{
		Extraenous: false, // we don't want it to be skipped because it's extraneous
		Version:    "1.4.2",
		Name:       "name",
		Path:       linkPath,
	}

	if !parser.shouldSkip(root, root) {
		t.Fatalf("did not detect symlink for %s", linkPath)
	}
}

func TestWorkspaceIsNotSkipped(t *testing.T) {
	// testing without npm output file since we're performing os.Lstat and needs to be created on disk
	conf, _ := config.New(nil)
	parser := dependencyParser{conf}

	target, err := os.MkdirTemp("", "test_seal_cli_*")
	if err != nil {
		panic(err)
	}

	defer os.Remove(target)
	
	if err := os.MkdirAll(filepath.Join(target, "workspace1"), 0755) ; err != nil {
		panic(err)
	}

	if err := os.MkdirAll(filepath.Join(target, "workspace2"), 0755) ; err != nil {
		panic(err)
	}

	if err := os.MkdirAll(filepath.Join(target, "node_modules", "@babel", "cli"), 0755) ; err != nil {
		panic(err)
	}

	if err := os.MkdirAll(filepath.Join(target, "node_modules", "yup"), 0755) ; err != nil {
		panic(err)
	}
	if err := os.Symlink(filepath.Join(target,"workspace1"), filepath.Join(target, "node_modules", "workspace1")); err != nil {
		panic(err)
	}
	if err := os.Symlink(filepath.Join(target,"workspace2"), filepath.Join(target, "node_modules", "workspace2")); err != nil {
		panic(err)
	}

	workspacesTemplate := getTestFile("workspaces.json")
	workspacesTemplate = strings.ReplaceAll(workspacesTemplate, "BASE_DIR", strings.ReplaceAll(target, "\\", "\\\\"))
	// test
	dependencies, err := parser.Parse(workspacesTemplate, target)
	if err != nil {
		t.Fatalf("failed parsing %v", err)
	}

	workspace1 := dependencies[common.DependencyId("NPM", "workspace1-package", "1.0.0")]
	if len(workspace1) != 2 {
		t.Fatalf("did not detect workspace1")
	}

	if workspace1[0].DiskPath != filepath.Join(target, "node_modules", "workspace1") {
		t.Fatalf("wrong path for workspace1 %s %s", workspace1[0].DiskPath, filepath.Join(target, "node_modules", "workspace1"))
	}

	if workspace1[1].DiskPath != filepath.Join(target, "node_modules", "workspace1") {
		t.Fatalf("wrong path for workspace1 %s %s", workspace1[1].DiskPath, filepath.Join(target, "node_modules", "workspace1"))
	}

	if ! ((workspace1[0].Branch == "" && workspace1[1].Branch == "workspace2@1.8.0") || 
		(workspace1[0].Branch == "workspace2@1.8.0" && workspace1[1].Branch == "")) {
		t.Fatalf("wrong branch for workspace1 %s %s", workspace1[0].Branch, workspace1[1].Branch)
	}

	workspace2 := dependencies[common.DependencyId("NPM", "workspace2-package", "1.8.0")]
	if len(workspace2) != 1 {
		t.Fatalf("did not detect workspace2")
	}

	if workspace2[0].DiskPath != filepath.Join(target, "node_modules", "workspace2") {
		t.Fatalf("wrong path for workspace2 %s", workspace2[0].DiskPath)
	}

	if workspace2[0].Branch != "" {
		t.Fatalf("wrong branch for workspace0 %s", workspace2[0].Branch)
	}
	babelCli := dependencies[common.DependencyId("NPM", "@babel/cli", "7.23.4")]
	if len(babelCli) != 1 {
		t.Fatalf("did not detect @babel/cli")
	}

	if babelCli[0].DiskPath != filepath.Join(target, "node_modules", "@babel", "cli") {
		t.Fatalf("wrong path for @babel/cli %s", babelCli[0].DiskPath)
	}
	
	if babelCli[0].Branch != "workspace1@1.0.0" {
		t.Fatalf("wrong branch for @babel/cli %s", babelCli[0].Branch)
	}

	yup := dependencies[common.DependencyId("NPM", "yup", "1.3.3")]
	if len(yup) != 1 {
		t.Fatalf("did not detect yup")
	}

	if yup[0].DiskPath != filepath.Join(target, "node_modules", "yup") {
		t.Fatalf("wrong path for yup %s", yup[0].DiskPath)
	}
	
	if yup[0].Branch != "workspace2@1.8.0" {
		t.Fatalf("wrong branch for yup %s", yup[0].Branch)
	}
}