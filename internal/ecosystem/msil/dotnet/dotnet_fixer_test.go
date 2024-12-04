package dotnet

import (
	"os"
	"testing"
)

func TestFormatCachePackagePath(t *testing.T) {
	root := "myroot"
	library := "my-lib"
	version := "my-ver"
	sep := string(os.PathSeparator)
	expected := root + sep + library + sep + version

	if res := formatCachePackagePath(root, library, version); res != expected {
		t.Fatalf("bad path; expected `%s` got `%s`", expected, res)
	}
}
