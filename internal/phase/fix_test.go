package phase

import (
	"cli/internal/api"
	"cli/internal/ecosystem/mappings"
	"testing"
)

func TestFixMapKey(t *testing.T) {
	p := &api.PackageVersion{
		Version:                         "1.2.3",
		Library:                         api.Package{Name: "lodash", PackageManager: mappings.NpmManager},
		RecommendedLibraryVersionId:     "123123",
		RecommendedLibraryVersionString: "1.2.3-sp1",
	}

	expectedKey := "NPM|lodash@1.2.3 -> NPM|lodash@1.2.3-sp1"
	k := FormatFixKey(p)
	if k != expectedKey {
		t.Fatalf("bad key format; got `%s` expected `%s`", k, expectedKey)
	}
}
