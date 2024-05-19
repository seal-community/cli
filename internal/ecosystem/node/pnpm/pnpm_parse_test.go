package pnpm

import (
	"bufio"
	"cli/internal/ecosystem/mappings"
	"io"
	"strings"
	"testing"
)

const projectPath = "/Users/mococo/proj"

func TestPnpmOutputSkipping(t *testing.T) {
	before := ""
	r := bufio.NewReader(strings.NewReader(before))
	err := skipToPackages(r, projectPath)
	if err != io.EOF {
		t.Fatalf("err: %v", err)
	}

}

func TestPnpmOutputSkippingValid(t *testing.T) {
	before := "/Users/mococo/proj:bau@0.1.123:PRIVATE"
	r := bufio.NewReader(strings.NewReader(before))
	err := skipToPackages(r, projectPath)
	if err != io.EOF {
		t.Fatalf("err: %v", err)
	}
}

func TestPnpmOutputSkippingUnsupported(t *testing.T) {
	before := "abcdef"
	r := bufio.NewReader(strings.NewReader(before))
	err := skipToPackages(r, projectPath)
	if err != io.EOF {
		t.Fatalf("err: %v", err)
	}
}

func TestPnpmOutputSkippingInvalid(t *testing.T) {
	before := " WARN  Issue while reading \"/Users/mococo/proj/.npmrc\". Failed to replace env in config: ${VARNAME}\n/Users/mococo/proj:bau@0.1.123:PRIVATE\n/Users/mococo/proj/node_modules/.pnpm/@adobe+css-tools@4.3.3/node_modules/@adobe/css-tools:@adobe/css-tools@4.3.3"
	r := bufio.NewReader(strings.NewReader(before))
	err := skipToPackages(r, projectPath)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	afterBytes, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("err reading buffer: %v", err)
	}

	after := string(afterBytes)
	if after != "/Users/mococo/proj/node_modules/.pnpm/@adobe+css-tools@4.3.3/node_modules/@adobe/css-tools:@adobe/css-tools@4.3.3" {
		t.Fatalf("skip failed - did not skip correctly, before: `%s` after: `%s`", before, after)
	}
}

func TestPnpmOutputSkippingInvalidCR(t *testing.T) {
	// should work regardless of os since \n is after \r
	before := " WARN  Issue while reading \"/Users/mococo/proj/.npmrc\". Failed to replace env in config: ${VARNAME}\r\n/Users/mococo/proj:bau@0.1.123:PRIVATE\r\n/Users/mococo/proj/node_modules/.pnpm/@adobe+css-tools@4.3.3/node_modules/@adobe/css-tools:@adobe/css-tools@4.3.3"
	r := bufio.NewReader(strings.NewReader(before))
	err := skipToPackages(r, projectPath)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	afterBytes, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("err reading buffer: %v", err)
	}

	after := string(afterBytes)
	if after != "/Users/mococo/proj/node_modules/.pnpm/@adobe+css-tools@4.3.3/node_modules/@adobe/css-tools:@adobe/css-tools@4.3.3" {
		t.Fatalf("skip failed - did not skip correctly, before: `%s` after: `%s`", before, after)
	}
}

func TestPnpmLineParse(t *testing.T) {
	d := parseLine("/Users/mococo/proj/node_modules/.pnpm/zod@3.22.4/node_modules/zod:zod@3.22.4", projectPath)
	if d == nil {
		t.Fatalf("failed parsing line")
	}

	if d.Link {
		t.Fatalf("wrongly detected link")
	}

	if d.PackageManager != mappings.NpmManager {
		t.Fatalf("did not use correct package manager")
	}

	if d.Name != "zod" {
		t.Fatalf("did not use correct package name; got %s", d.Name)
	}

	if d.Version != "3.22.4" {
		t.Fatalf("did not use correct version; got %s", d.Version)
	}

	if d.DiskPath != "/Users/mococo/proj/node_modules/.pnpm/zod@3.22.4/node_modules/zod" {
		t.Fatalf("did not use correct path; got %s", d.DiskPath)
	}
}

func TestPnpmLineParseWindowsPath(t *testing.T) {
	d := parseLine("C:\\Users\\mococo\\proj\\node_modules\\.pnpm\\zod@3.22.4\\node_modules\\zod:zod@3.22.4", projectPath)
	if d == nil {
		t.Fatalf("failed parsing line")
	}

	if d.Link {
		t.Fatalf("wrongly detected link")
	}

	if d.PackageManager != mappings.NpmManager {
		t.Fatalf("did not use correct package manager")
	}

	if d.Name != "zod" {
		t.Fatalf("did not use correct package name; got %s", d.Name)
	}

	if d.Version != "3.22.4" {
		t.Fatalf("did not use correct version; got %s", d.Version)
	}

	if d.DiskPath != "C:\\Users\\mococo\\proj\\node_modules\\.pnpm\\zod@3.22.4\\node_modules\\zod" {
		t.Fatalf("did not use correct path; got %s", d.DiskPath)
	}
}

func TestPnpmLineParseScoped(t *testing.T) {
	d := parseLine("/Users/mococo/proj/node_modules/.pnpm/@typescript-eslint+types@6.21.0/node_modules/@typescript-eslint/types:@typescript-eslint/types@6.21.0", projectPath)
	if d == nil {
		t.Fatalf("failed parsing line")
	}

	if d.PackageManager != mappings.NpmManager {
		t.Fatalf("did not use correct package manager")
	}

	if d.Name != "@typescript-eslint/types" {
		t.Fatalf("did not use correct package name; got %s", d.Name)
	}

	if d.Version != "6.21.0" {
		t.Fatalf("did not use correct version; got %s", d.Version)
	}

	if d.DiskPath != "/Users/mococo/proj/node_modules/.pnpm/@typescript-eslint+types@6.21.0/node_modules/@typescript-eslint/types" {
		t.Fatalf("did not use correct path; got %s", d.DiskPath)
	}

	if d.Link {
		t.Fatalf("wrongly detected link")
	}
}

func TestPnpmLineParseExtraAtLink(t *testing.T) {
	d := parseLine("/Users/mococo/mylib:@scope/libname@extra@link:../mylib", projectPath)
	if d == nil {
		t.Fatalf("failed parsing line")
	}

	if !d.Link {
		t.Fatalf("failed detecting link")
	}
}

func TestPnpmLineParseExtraA(t *testing.T) {
	// fake edge case, can happen locally with link, maybe also in registry
	d := parseLine("/Users/mococo/proj/node_modules/.pnpm/@scope/libname@extra@1.2.3:@scope/libname@extra@1.2.3", projectPath)
	if d == nil {
		t.Fatalf("failed parsing line")
	}

	if d.Link {
		t.Fatalf("wrongly detected link")
	}

	if d.PackageManager != mappings.NpmManager {
		t.Fatalf("did not use correct package manager")
	}

	if d.Name != "@scope/libname@extra" {
		t.Fatalf("did not use correct package name; got %s", d.Name)
	}

	if d.Version != "1.2.3" {
		t.Fatalf("did not use correct version; got %s", d.Version)
	}

	if d.DiskPath != "/Users/mococo/proj/node_modules/.pnpm/@scope/libname@extra@1.2.3" {
		t.Fatalf("did not use correct path; got %s", d.DiskPath)
	}
}

func TestPnpmLineParseLinkDirutsideProjDir(t *testing.T) {
	d := parseLine("/Users/mococo/mylib:libname@link:../mylib", projectPath)
	if d == nil {
		t.Fatalf("failed parsing line")
	}

	if !d.Link {
		t.Fatalf("failed detecting link")
	}
}

func TestPnpmLineParseLinkDirWithinProjDir(t *testing.T) {
	d := parseLine("/Users/mococo/proj/inner_proj:mylib@link:inner_proj", projectPath)
	if d == nil {
		t.Fatalf("failed parsing line")
	}

	if !d.Link {
		t.Fatalf("failed detecting link")
	}
}

func TestPnpmLineParseScopedWindowsPath(t *testing.T) {
	d := parseLine("C:\\Users\\mococo\\proj\\node_modules\\.pnpm\\@typescript-eslint+types@6.21.0\\node_modules\\@typescript-eslint\\types:@typescript-eslint\\types@6.21.0", projectPath)
	if d == nil {
		t.Fatalf("failed parsing line")
	}

	if d.PackageManager != mappings.NpmManager {
		t.Fatalf("did not use correct package manager")
	}

	if d.Name != "@typescript-eslint\\types" {
		t.Fatalf("did not use correct package name; got %s", d.Name)
	}

	if d.Version != "6.21.0" {
		t.Fatalf("did not use correct version; got %s", d.Version)
	}

	if d.DiskPath != "C:\\Users\\mococo\\proj\\node_modules\\.pnpm\\@typescript-eslint+types@6.21.0\\node_modules\\@typescript-eslint\\types" {
		t.Fatalf("did not use correct path; got %s", d.DiskPath)
	}
}

func TestPnpmLineParseGitInstall(t *testing.T) {
	// from docs: https://pnpm.io/cli/add
	// pnpm add kevva/is-positive#97edff6f525f192a3f83cea1944765f769ae2678

	d := parseLine("/Users/mococo/proj/node_modules/.pnpm/github.com+kevva+is-positive@97edff6f525f192a3f83cea1944765f769ae2678/node_modules/is-positive:is-positive@3.1.0", projectPath)
	if d == nil {
		t.Fatalf("failed parsing line")
	}

	if d.PackageManager != mappings.NpmManager {
		t.Fatalf("did not use correct package manager")
	}

	if d.Name != "is-positive" {
		t.Fatalf("did not use correct package name; got %s", d.Name)
	}

	if d.Version != "3.1.0" {
		t.Fatalf("did not use correct version; got %s", d.Version)
	}

	if d.DiskPath != "/Users/mococo/proj/node_modules/.pnpm/github.com+kevva+is-positive@97edff6f525f192a3f83cea1944765f769ae2678/node_modules/is-positive" {
		t.Fatalf("did not use correct path; got %s", d.DiskPath)
	}

	if d.Link {
		t.Fatalf("wrongly detected link")
	}
}

func TestPnpmLineParseLocalTgz(t *testing.T) {
	// from docs: https://pnpm.io/cli/add
	//  	curl -L https://registry.npmjs.org/node-forge/-/node-forge-0.10.0.tgz > z.tgz
	// 		pnpm add ./z.tgz

	d := parseLine("/Users/mococo/proj/node_modules/.pnpm/file+z.tgz/node_modules/node-forge:node-forge@0.10.0", projectPath)
	if d == nil {
		t.Fatalf("failed parsing line")
	}

	if d.PackageManager != mappings.NpmManager {
		t.Fatalf("did not use correct package manager")
	}

	if d.Name != "node-forge" {
		t.Fatalf("did not use correct package name; got %s", d.Name)
	}

	if d.Version != "0.10.0" {
		t.Fatalf("did not use correct version; got %s", d.Version)
	}

	if d.DiskPath != "/Users/mococo/proj/node_modules/.pnpm/file+z.tgz/node_modules/node-forge" {
		t.Fatalf("did not use correct path; got %s", d.DiskPath)
	}

	if d.Link {
		t.Fatalf("wrongly detected link")
	}
}
