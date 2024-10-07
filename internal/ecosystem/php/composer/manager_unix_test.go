//go:build !windows

package composer

import (
	"fmt"
	"testing"
)

func TestIndicatorMatchesUnix(t *testing.T) {
	ps := []string{
		`/b/composer.lock`,
		`../composer.lock`,
		`./abc/../composer.lock`,
	}

	for i, p := range ps {
		t.Run(fmt.Sprintf("pth_%d", i), func(t *testing.T) {
			if !IsComposerIndicatorFile(p) {
				t.Fatalf("failed to detect indicator path `%s`", p)
			}
		})
	}
}

func TestIndicatorDoesNotMatchPackageJsonUnix(t *testing.T) {
	// as it is intended to be handled by dir
	ps := []string{
		`/b/package.json`,
		`../package.json`,
		`./abc/../package.json`,
	}

	for i, p := range ps {
		t.Run(fmt.Sprintf("pth_%d", i), func(t *testing.T) {
			if IsComposerIndicatorFile(p) {
				t.Fatalf("failed to detect indicator path `%s`", p)
			}
		})
	}
}
