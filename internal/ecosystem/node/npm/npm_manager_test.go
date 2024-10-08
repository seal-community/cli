package npm

import (
	"cli/internal/config"
	"fmt"
	"testing"
)

func TestIndicatorMatches(t *testing.T) {
	ps := []string{
		`/b/package-lock.json`,
		`C:\package-lock.json`,
		`../package-lock.json`,
		`..\package-lock.json`,
		`./abc/../package-lock.json`,
		`.\abc\..\package-lock.json`,
	}

	for i, p := range ps {
		t.Run(fmt.Sprintf("pth_%d", i), func(t *testing.T) {
			if !IsNpmIndicatorFile(p) {
				t.Fatalf("failed to detect indicator path `%s`", p)
			}
		})
	}
}

func TestIndicatorDoesNotMatchPackageJson(t *testing.T) {
	// as it is intended to be handled by dir
	ps := []string{
		`/b/package.json`,
		`C:\package.json`,
		`../package.json`,
		`..\package.json`,
		`./abc/../package.json`,
		`.\abc\..\package.json`,
	}

	for i, p := range ps {
		t.Run(fmt.Sprintf("pth_%d", i), func(t *testing.T) {
			if IsNpmIndicatorFile(p) {
				t.Fatalf("failed to detect indicator path `%s`", p)
			}
		})
	}
}

func TestNormalizePackageNames(t *testing.T) {
	c, _ := config.New(nil)
	manager := NewNpmManager(c, "")
	names := []string{
		"aaaaa",
		"aaAAa",
		"AAAAA",
		"AAa_a",
	}
	for i, n := range names {
		t.Run(fmt.Sprintf("name_%d", i), func(t *testing.T) {
			if manager.NormalizePackageName(n) != n {
				t.Fatalf("failed to normalize `%s`", n)
			}
		})
	}
}
