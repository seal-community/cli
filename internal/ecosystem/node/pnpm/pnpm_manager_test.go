package pnpm

import (
	"fmt"
	"testing"
)

func TestIndicatorMatches(t *testing.T) {
	ps := []string{
		`/b/pnpm-lock.yaml`,
		`C:\pnpm-lock.yaml`,
		`../pnpm-lock.yaml`,
		`..\pnpm-lock.yaml`,
		`./abc/../pnpm-lock.yaml`,
		`.\abc\..\pnpm-lock.yaml`,
	}

	for i, p := range ps {
		t.Run(fmt.Sprintf("pth_%d", i), func(t *testing.T) {
			if !IsPnpmIndicatorFile(p) {
				t.Fatalf("failed to detect indicator path `%s`", p)
			}
		})
	}
}
