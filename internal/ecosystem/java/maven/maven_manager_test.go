package maven

import (
	"cli/internal/common"
	"cli/internal/config"
	"fmt"
	"path/filepath"
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

func TestGetJavaIndicatorFileAbsPath(t *testing.T) {
	tmp := t.TempDir()
	dst := filepath.Join(tmp, "pom.xml")
	fi, err := common.CreateFile(dst)
	if fi == nil || err != nil {
		t.Fatalf("faile: %v %v", fi, err)
	}
	defer fi.Close()

	p, err := GetJavaIndicatorFile(tmp)
	if err != nil {
		t.Fatalf("failed getting indicator %v", err)
	}

	if p != dst {
		t.Fatalf("excepted %s; got %s", dst, p)
	}
}

func TestParseSilenceInput(t *testing.T) {
	tests := []struct {
		input                  string
		expectedPackageName    string
		expectedPackageVersion string
	}{
		{"name@version", "name", "version"},
		{"name@version@other", "", ""},
		{"name", "", ""},
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("test_%d", i), func(t *testing.T) {
			pn, pv := parseSilenceInput(test.input)
			if pn != test.expectedPackageName || pv != test.expectedPackageVersion {
				t.Fatalf("failed to parse `%s`", test.input)
			}
		})
	}
}
