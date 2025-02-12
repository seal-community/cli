package utils

import (
	"testing"
)

func TestApkWorldRemoveHashRestriction(t *testing.T) {
	world := `abuild
alpine-baselayout
alpine-keys
rsync><Q1tiuntRv4yr62xZ+pySTHmSK92mI=
`

	newWorld := ApkWorldRemoveHashRestriction("rsync", world)
	expected := `abuild
alpine-baselayout
alpine-keys
rsync
`
	if newWorld != expected {
		t.Errorf("expected\n%sgot\n\n%s", expected, newWorld)
	}
}
