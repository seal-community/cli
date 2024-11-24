package nuget

import (
	"path/filepath"
	"testing"
)

func TestFormatPackagesFolderEntry(t *testing.T) {
	e := formatPackagesFolderEntry("mydir", "MyLIB", "1.2.3")
	if e != filepath.Join("mydir", "MyLIB.1.2.3") {
		t.Fatalf("bad format: `%s`", e)
	}
}
