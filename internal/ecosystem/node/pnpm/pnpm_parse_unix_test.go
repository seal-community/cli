//go:build !windows

package pnpm

import (
	"cli/internal/common"
	"cli/internal/config"
	"cli/internal/ecosystem/shared"
	"os"
	"path/filepath"
	"testing"

	"golang.org/x/exp/maps"
)

const PlatformTestDir = "unix"
const defaultTestProjectDir = "/Users/fuwawa/proj"

func getTestFile(name string) string {
	// fetch file from current package's testdata folder
	// ref: https://pkg.go.dev/cmd/go/internal/test
	p := filepath.Join("testdata", PlatformTestDir, name)
	data, err := os.ReadFile(p)
	if err != nil {
		panic(err)
	}

	return string(data)
}
func TestParseSingleDependencyPnpm(t *testing.T) {
	conf, _ := config.New(nil)
	parser := pnpmDependencyParser{conf}

	dependencies, err := parser.Parse(getTestFile("pnpm_single_dep.json"),
		"/test/output/pnpm/single_dep_test")
	if err != nil {
		t.Fatalf("parse failed %v", err)
	}

	if len(dependencies) != 1 {
		t.Fatalf("got more than 1 dependency: %v", len(dependencies))
	}

	deps := maps.Values(dependencies)
	dep := deps[0][0]

	if dep.PackageManager != shared.NpmManager {
		t.Fatalf("wrong package manager %v", dep.PackageManager)
	}

	if dep.Version != "1.0.0" {
		t.Fatalf("wrong version %v", dep.Version)
	}

	if dep.Name != "semver-regex" {
		t.Fatalf("wrong package %v", dep.Name)
	}

	if !dep.IsDirect() {
		t.Fatalf("did not detect as direct dep %v", dep)
	}
}

func TestDependencyDevDepsDisabledPnpm(t *testing.T) {
	conf, _ := config.New(nil)
	conf.Pnpm.ProdOnlyDeps = true
	parser := pnpmDependencyParser{conf}

	dependencies, err := parser.Parse(getTestFile("pnpm_dev_dep.json"),
		defaultTestProjectDir)
	if err != nil {
		t.Fatalf("parse failed ")
	}

	if len(dependencies) != 0 {
		t.Fatalf("got more than 0 dependency: %v", len(dependencies))
	}
}

func TestDependencyDevDepsPnpm(t *testing.T) {
	conf, _ := config.New(nil)
	conf.Pnpm.ProdOnlyDeps = false
	parser := pnpmDependencyParser{conf}

	dependencies, err := parser.Parse(getTestFile("pnpm_dev_dep.json"),
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

	if dep.Version != "1.0.0" {
		t.Fatalf("wrong version %v", dep.Version)
	}

	if dep.Name != "semver-regex" {
		t.Fatalf("wrong package %v", dep.Name)
	}

	if !dep.IsDirect() {
		t.Fatalf("did not detect as direct dep %v", dep)
	}
}

func TestDependencyTransitivePnpm(t *testing.T) {
	conf, _ := config.New(nil)
	conf.Pnpm.ProdOnlyDeps = false
	parser := pnpmDependencyParser{conf}

	dependencies, err := parser.Parse(getTestFile("pnpm_transitive_deps.json"),
		defaultTestProjectDir)
	if err != nil {
		t.Fatalf("parse failed ")
	}

	if len(dependencies) != 2 {
		t.Fatalf("got more than 2 dependency: %v", len(dependencies))
	}

	cypressInstances := dependencies[common.DependencyId(shared.NpmManager, "cypress", "3.4.0")]
	if len(cypressInstances) != 1 {
		t.Fatalf("too many instances of dep %d", len(cypressInstances))
	}

	cypress := cypressInstances[0]
	if cypress.PackageManager != shared.NpmManager {
		t.Fatalf("wrong package manager %v", cypress.PackageManager)
	}

	if cypress.Version != "3.4.0" {
		t.Fatalf("wrong version %v", cypress.Version)
	}

	if cypress.Name != "cypress" {
		t.Fatalf("wrong package %v", cypress.Name)
	}

	if !cypress.IsDirect() {
		t.Fatalf("did not detect as direct cypress %v", cypress)
	}

	if len(cypressInstances) != 1 {
		t.Fatalf("too many instances of dep %d", len(cypressInstances))
	}

	archInstances := dependencies[common.DependencyId(shared.NpmManager, "arch", "2.1.1")]
	if len(archInstances) != 1 {
		t.Fatalf("too many instances of dep %d", len(archInstances))
	}
	arch := archInstances[0]
	if arch.PackageManager != shared.NpmManager {
		t.Fatalf("wrong package manager %v", arch.PackageManager)
	}

	if arch.Version != "2.1.1" {
		t.Fatalf("wrong version %v", arch.Version)
	}

	if arch.Name != "arch" {
		t.Fatalf("wrong package %v", arch.Name)
	}

	if arch.IsDirect() {
		t.Fatalf("did not detect as indirect arch %v", arch)
	}
}
