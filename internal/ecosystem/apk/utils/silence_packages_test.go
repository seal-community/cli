package utils

import (
	"cli/internal/api"
	"testing"
)

func TestGetParsedPackageFromDB(t *testing.T) {
	db := `C:Q16skhUkFGZO7TbDnqKclzYLEZSGc=
P:libacl
V:2.2.53-r0
p:libacl=2.2.53-r0

`
	silenceRule := api.SilenceRule{
		Library: "libacl",
		Version: "2.2.53-r0",
		Manager: "APK",
	}

	exists, packageInfo := getParsedPackageFromDB(db, silenceRule)
	if !exists {
		t.Errorf("expected package to exist")
	}

	expectedPackage := PackageInfo{
		Name:     PackageInfoEntry{Value: "libacl", LineIndex: 1},
		Version:  PackageInfoEntry{Value: "2.2.53-r0", LineIndex: 2},
		Provides: PackageInfoEntry{Value: "libacl=2.2.53-r0", LineIndex: 3},
	}

	if *packageInfo != expectedPackage {
		t.Errorf("expected %v, got %v", expectedPackage, *packageInfo)
	}
}

func TestGetParsedPackageFromDBNoPackage(t *testing.T) {
	db := `C:Q16skhUkFGZO7TbDnqKclzYLEZSGc=
P:libacl
V:2.2.53-r0
p:libacl=2.2.53-r0

`
	silenceRule := api.SilenceRule{
		Library: "libacl2",
		Version: "2.2.53-r0",
		Manager: "APK",
	}

	exists, packageInfo := getParsedPackageFromDB(db, silenceRule)
	if exists {
		t.Errorf("expected package to not exist")
	}

	expectedPackage := PackageInfo{
		Name:     PackageInfoEntry{Value: "", LineIndex: 0},
		Version:  PackageInfoEntry{Value: "", LineIndex: 0},
		Provides: PackageInfoEntry{Value: "", LineIndex: 0},
	}

	if *packageInfo != expectedPackage {
		t.Errorf("expected packageInfo to be uninitialized, got %v", *packageInfo)
	}
}

func TestRenamePackage(t *testing.T) {
	db := `C:Q16skhUkFGZO7TbDnqKclzYLEZSGc=
P:libacl
V:2.2.53-r0
p:libacl-dev

`
	silenceRule := api.SilenceRule{
		Library: "libacl",
		Version: "2.2.53-r0",
		Manager: "APK",
	}
	wasRenamed, newDBContent := RenamePackage(db, silenceRule)
	if !wasRenamed {
		t.Errorf("expected package to be renamed")
	}

	expectedDB := `C:Q16skhUkFGZO7TbDnqKclzYLEZSGc=
P:seal-libacl
V:2.2.53-r0
p:libacl-dev libacl=2.2.53-r0

`
	if newDBContent != expectedDB {
		t.Errorf("expected %s, got %s", expectedDB, newDBContent)
	}
}

func TestRenamePackageNoPackage(t *testing.T) {
	db := `C:Q16skhUkFGZO7TbDnqKclzYLEZSGc=
P:libacl
V:2.2.53-r0
p:libacl=2.2.53-r0

`
	silenceRule := api.SilenceRule{
		Library: "libacl2",
		Version: "2.2.53-r0",
		Manager: "APK",
	}
	wasRenamed, newDBContent := RenamePackage(db, silenceRule)
	if wasRenamed {
		t.Errorf("expected package to not be renamed")
	}

	if newDBContent != db {
		t.Errorf("expected %s, got %s", db, newDBContent)
	}
}
