package golang

import (
	"testing"
)

func TestGetReplaceString(t *testing.T) {
	replaceString := getReplaceString("github.com/Masterminds/semver", "1.5.0")
	expected := "-replace=github.com/Masterminds/semver@v1.5.0=sealsecurity.io/github.com/Masterminds/semver@v1.5.0"
	if replaceString != expected {
		t.Fatalf("unexpected replace string: got %s expected %s", replaceString, expected)
	}
}

func TestModulesContentAddReplaceSanity(t *testing.T) {
	content := `# github.com/Masterminds/semver v1.5.0
## explicit
github.com/Masterminds/semver
`
	err, modifiedContent := modulesContentAddReplace(content, "github.com/Masterminds/semver", "1.5.0")
	if err != nil {
		t.Fatalf("failed modifying content: %v", err)
	}

	expectedContent := `# github.com/Masterminds/semver v1.5.0 => sealsecurity.io/github.com/Masterminds/semver v1.5.0
## explicit
github.com/Masterminds/semver
`

	if modifiedContent != expectedContent {
		t.Fatalf("unexpected content: got %s expected %s", modifiedContent, expectedContent)
	}
}

func TestModulesContentAddReplaceNoModule(t *testing.T) {
	content := `# github.com/Masterminds/semver v1.2.3
## explicit
github.com/Masterminds/semver
`

	err, modifiedContent := modulesContentAddReplace(content, "github.com/Masterminds/semver", "1.5.0")
	if err == nil {
		t.Fatalf("expected error")
	}

	if modifiedContent != "" {
		t.Fatalf("unexpected content: got %s", modifiedContent)
	}
}

func TestModulesContentAddReplaceMultipleModules(t *testing.T) {
	content := `# github.com/Masterminds/semver v1.5.0
## explicit
github.com/Masterminds/semver # github.com/Masterminds/semver v1.5.0
`
	err, modifiedContent := modulesContentAddReplace(content, "github.com/Masterminds/semver", "1.5.0")
	if err == nil {
		t.Fatalf("expected error")
	}

	if modifiedContent != "" {
		t.Fatalf("unexpected content: got %s", modifiedContent)
	}
}
