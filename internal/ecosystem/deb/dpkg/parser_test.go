package dpkg

import (
	"cli/internal/ecosystem/mappings"
	"testing"
)

func TestParseDPKGVersion(t *testing.T) {
	dpkgOutput := `Debian 'dpkg' package management program version 1.20.13 (amd64).
This is free software; see the GNU General Public License version 2 or
later for copying conditions. There is NO warranty.`
	version := parseDPKGVersion(dpkgOutput)
	if version != "1.20.13" {
		t.Errorf("expected version 1.20.13, got %s", version)
	}
}

func TestParseDPKGListInstalled(t *testing.T) {
	dpkgQueryOutput := `tar 1.34+dfsg-1.2+deb12u1 amd64  install ok installed
tzdata 2024a-0+deb12u1 all  install ok installed
`
	deps, err := parseDPKGQueryInstalled(dpkgQueryOutput)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(deps) != 2 {
		t.Errorf("expected 2 deps, got %d", len(deps))
	}

	if len(deps["DEB|tar@1.34+dfsg-1.2+deb12u1"]) != 1 {
		t.Errorf("expected 1 tar dep, got %d", len(deps["DEB|tar@1.34+dfsg-1.2+deb12u1"]))
	}

	tar := deps["DEB|tar@1.34+dfsg-1.2+deb12u1"][0]
	if tar.Name != "tar" {
		t.Errorf("expected tar, got %s", tar.Name)
	}
	if tar.Version != "1.34+dfsg-1.2+deb12u1" {
		t.Errorf("expected 2.2.53-3.el8, got %s", tar.Version)
	}
	if tar.Arch != "amd64" {
		t.Errorf("expected x86_64, got %s", tar.Arch)
	}
	if tar.PackageManager != mappings.DebGManager {
		t.Errorf("expected deb, got %s", tar.PackageManager)
	}

	if len(deps["DEB|tzdata@2024a-0+deb12u1"]) != 1 {
		t.Errorf("expected 1 tzdata dep, got %d", len(deps["DEB|tzdata@2024a-0+deb12u1"]))
	}

	tzdata := deps["DEB|tzdata@2024a-0+deb12u1"][0]
	if tzdata.Name != "tzdata" {
		t.Errorf("expected tzdata, got %s", tzdata.Name)
	}
	if tzdata.Version != "2024a-0+deb12u1" {
		t.Errorf("expected 11-5.el8, got %s", tzdata.Version)
	}
	if tzdata.Arch != "all" {
		t.Errorf("expected all, got %s", tzdata.Arch)
	}
	if tzdata.PackageManager != mappings.DebGManager {
		t.Errorf("expected deb, got %s", tzdata.PackageManager)
	}
}

func TestParseDPKGListInstalledNoPrefix(t *testing.T) {
	dpkgQueryOutput := `tar 1.34+dfsg-1.2+deb12u1 amd64 not-installed
`
	deps, err := parseDPKGQueryInstalled(dpkgQueryOutput)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(deps) != 0 {
		t.Errorf("expected 0 dep, got %d", len(deps))
	}
}

func TestParseDPKGListInstalledEmpty(t *testing.T) {
	dpkgOutput := ``
	deps, err := parseDPKGQueryInstalled(dpkgOutput)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(deps) != 0 {
		t.Errorf("expected 0 deps, got %d", len(deps))
	}
}

func TestIsStatusInstalled(t *testing.T) {
	tests := []struct {
		fields            []string
		expectedStatus    string
		expectedInstalled bool
	}{
		{[]string{"zlib1g", "1:1.2.13.dfsg-1", "amd64", "install", "ok", "installed"}, "install ok installed", true},
		{[]string{"zlib1g", "1:1.2.13.dfsg-1", "amd64", "install", "ok", "half-installed"}, "install ok half-installed", true},
		{[]string{"zlib1g", "1:1.2.13.dfsg-1", "amd64", "failed-config"}, "failed-config", true},
		{[]string{"zlib1g", "1:1.2.13.dfsg-1", "amd64", "not-installed"}, "not-installed", false},
		{[]string{"zlib1g", "1:1.2.13.dfsg-1", "amd64", "really long status that is not installed"}, "really long status that is not installed", false},
	}

	for _, test := range tests {
		t.Run(test.expectedStatus, func(t *testing.T) {
			installed, status := isStatusInstalled(test.fields)
			if installed != test.expectedInstalled {
				t.Errorf("expected installed %t, got %t", test.expectedInstalled, installed)
			}
			if status != test.expectedStatus {
				t.Errorf("expected status %s, got %s", test.expectedStatus, status)
			}
		})
	}
}
