package common

import "testing"

func TestVersionAtLeast(t *testing.T) {
	if valid, err := VersionAtLeast("2.8.6", "3.0.0"); err != nil || valid {
		t.Fatalf("wrongly supported version err %v valid:%v", err, valid)
	}

	if valid, err := VersionAtLeast("3.0.1", "3.0.0"); err != nil || !valid {
		t.Fatalf("wrongly unsupported version err %v valid:%v", err, valid)
	}

	if valid, err := VersionAtLeast("3.0.0", "3.0.0"); err != nil || !valid {
		t.Fatalf("wrongly unsupported version err %v valid:%v", err, valid)
	}
}

func TestGetNoEpochVersion(t *testing.T) {
	if epoch, version := GetNoEpochVersion("1:2.8.6"); epoch != "1" || version != "2.8.6" {
		t.Fatalf("wrongly parsed epoch %s version %s", epoch, version)
	}

	if epoch, version := GetNoEpochVersion("2.8.6"); epoch != "" || version != "2.8.6" {
		t.Fatalf("wrongly parsed epoch %s version %s", epoch, version)
	}
}
