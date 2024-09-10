package yarn

import (
	"cli/internal/config"
	"fmt"
	"testing"
)

func TestIndicatorMatches(t *testing.T) {
	ps := []string{
		`/b/yarn.lock`,
		`C:\yarn.lock`,
		`../yarn.lock`,
		`..\yarn.lock`,
		`./abc/../yarn.lock`,
		`.\abc\..\yarn.lock`,
	}

	for i, p := range ps {
		t.Run(fmt.Sprintf("pth_%d", i), func(t *testing.T) {
			if !IsYarnIndicatorFile(p) {
				t.Fatalf("failed to detect indicator path `%s`", p)
			}
		})
	}
}

func TestNormalizePackageNames(t *testing.T) {
	c, _ := config.New(nil)
	manager := NewYarnManager(c, "")
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
