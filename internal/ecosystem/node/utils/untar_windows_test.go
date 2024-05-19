package utils

import (
	"fmt"
	"testing"
)

func TestPackageFolderRemoved(t *testing.T) {

	outputDir := `C:\test\proj\.seal\`
	paths := [][]string{
		{`package\a\b\`, outputDir + `a\b`},
		{`package\a\b`, outputDir + `a\b`},
		{`a\b\`, outputDir + `a\b`},
		{`a\b`, outputDir + `a\b`},
	}

	for i, p := range paths {
		t.Run(fmt.Sprintf("path_%d", i), func(t *testing.T) {
			given := p[0]
			expected := p[1]
			if result := getTargetPathForNpm(outputDir, given); result != expected {
				t.Fatalf("failed to remove package prefix, expected: `%s` got: `%s`", expected, result)
			}
		})
	}

}
func TestIllegalPaths(t *testing.T) {
	badPaths := []string{
		`\\a\b\d.txt`,
		`C:\a.txt`,
		`../x.txt`,
		`..\x.txt`,
		`./abc/../x.txt`,
		`.\abc\..\x.txt`,
	}

	for i, p := range badPaths {
		t.Run(fmt.Sprintf("bad_path_%d", i), func(t *testing.T) {
			if !isIllegalPath(p) {
				t.Fatalf("failed to detect bad path `%s`", p)
			}
		})
	}
}
