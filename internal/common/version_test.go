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
