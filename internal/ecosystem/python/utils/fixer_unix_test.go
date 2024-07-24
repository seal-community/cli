//go:build !windows

package utils

import (
	"strings"
	"testing"
)

func TestParseRecordFile(t *testing.T) {
	recordContent := `pip-23.0.1.dist-info/RECORD,,
pip-23.0.1.dist-info/entry_points.txt,sha256=w694mjHYSfmSoUVVSaHoQ9UkOBBdtKKIJbyDRLdKju8,124
pip-23.0.1.dist-info/top_level.txt,sha256=zuuue4knoyJ-UwPPXg8fezS7VCrXJQrAP7zeNuwvFQg,4
pip/__init__.py,sha256=5yroedzc2dKKbcynDrHX8vBoLxqU27KmFvvHmdqQN9w,357
pip/__main__.py,sha256=mXwWDftNLMKfwVqKFWGE_uuBZvGSIiUELhLkeysIuZc,1198`

	files, err := parseRecordFile(strings.NewReader(recordContent))
	if err != nil {
		t.Fatalf("parse failed %v", err)
	}
	if len(files) != 5 {
		t.Fatalf("got wrong number of files %v", files)
	}

	if files[0] != "pip-23.0.1.dist-info/RECORD" {
		t.Fatalf("wrong file %v", files[0])
	}
	if files[len(files)-1] != "pip/__main__.py" {
		t.Fatalf("wrong file %v", files[len(files)-1])
	}
}

func TestParseInstalledFilesFile(t *testing.T) {
	recordContent := `../networkx/utils/tests/test_unionfind.py
../networkx/utils/union_find.py
../networkx/version.py
PKG-INFO
SOURCES.txt`

	files, err := parseInstalledFilesFile(strings.NewReader(recordContent), "/usr/local/lib/site-packages/sss")
	if err != nil {
		t.Fatalf("parse failed %v", err)
	}
	if len(files) != 5 {
		t.Fatalf("got wrong number of files %v", files)
	}

	if files[0] != "networkx/utils/tests/test_unionfind.py" {
		t.Fatalf("wrong file %v", files[0])
	}
	if files[len(files)-1] != "sss/SOURCES.txt" {
		t.Fatalf("wrong file %v", files[len(files)-1])
	}
}
