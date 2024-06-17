package maven

import (
	"cli/internal/config"
	"fmt"
	"testing"
)

func TestIndicatorMatches(t *testing.T) {
	ps := []string{
		`/b/pom.xml`,
		`C:\pom.xml`,
		`../pom.xml`,
		`..\pom.xml`,
		`./abc/../pom.xml`,
		`.\abc\..\pom.xml`,
	}

	for i, p := range ps {
		t.Run(fmt.Sprintf("pth_%d", i), func(t *testing.T) {
			if !IsMavenIndicatorFile(p) {
				t.Fatalf("failed to detect indicator path `%s`", p)
			}
		})
	}
}

func TestIndicatorDoesNotMatchOtherXml(t *testing.T) {
	// as it is intended to be handled by dir
	ps := []string{
		`/b/package.xml`,
		`C:\package.xml`,
		`../package.xml`,
		`..\package.xml`,
		`./abc/../package.xml`,
		`.\abc\..\package.xml`,
	}

	for i, p := range ps {
		t.Run(fmt.Sprintf("pth_%d", i), func(t *testing.T) {
			if IsMavenIndicatorFile(p) {
				t.Fatalf("failed to detect indicator path `%s`", p)
			}
		})
	}
}

func TestNormalizePackageNames(t *testing.T) {
	c, _ := config.New(nil)
	manager := NewMavenManager(c, "", "")
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
