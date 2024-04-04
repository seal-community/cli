package yarn

import (
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
