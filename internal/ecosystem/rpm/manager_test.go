package rpm

import (
	"testing"

	"gopkg.in/ini.v1"
)

func TestGetDistro(t *testing.T) {
	osReleaseText := `NAME="CentOS Linux"
VERSION="7 (Core)"
ID="centos"
ID_LIKE="rhel fedora"
VERSION_ID="7"
PRETTY_NAME="CentOS Linux 7 (Core)"
ANSI_COLOR="0;31"
CPE_NAME="cpe:/o:centos:centos:7"
HOME_URL="https://www.centos.org/"
BUG_REPORT_URL="https://bugs.centos.org/"

CENTOS_MANTISBT_PROJECT="CentOS-7"
CENTOS_MANTISBT_PROJECT_VERSION="7"
REDHAT_SUPPORT_PRODUCT="centos"
REDHAT_SUPPORT_PRODUCT_VERSION="7"`

	osRelease, err := ini.Load([]byte(osReleaseText))
	if err != nil {
		t.Fatalf("failed to load os-release: %v", err)
	}

	os, err := getDistro(osRelease)
	if err != nil {
		t.Fatalf("failed to get distro: %v", err)
	}

	if os != "centos" {
		t.Fatalf("expected centos, got %s", os)
	}
}

func TestGetDistroMissing(t *testing.T) {
	osReleaseText := `NAME="CentOS Linux"
VERSION="7 (Core)"
ID_LIKE="rhel fedora"
VERSION_ID="7"
PRETTY_NAME="CentOS Linux 7 (Core)"
ANSI_COLOR="0;31"
CPE_NAME="cpe:/o:centos:centos:7"
HOME_URL="https://www.centos.org/"
BUG_REPORT_URL="https://bugs.centos.org/"

CENTOS_MANTISBT_PROJECT="CentOS-7"
CENTOS_MANTISBT_PROJECT_VERSION="7"
REDHAT_SUPPORT_PRODUCT="centos"
REDHAT_SUPPORT_PRODUCT_VERSION="7"`

	osRelease, err := ini.Load([]byte(osReleaseText))
	if err != nil {
		t.Fatalf("failed to load os-release: %v", err)
	}

	_, err = getDistro(osRelease)
	if err == nil {
		t.Fatalf("failed to get distro: %v", err)
	}
}

func TestGetDistroEmpty(t *testing.T) {
	osReleaseText := ``

	osRelease, err := ini.Load([]byte(osReleaseText))
	if err != nil {
		t.Fatalf("failed to load os-release: %v", err)
	}

	_, err = getDistro(osRelease)
	if err == nil {
		t.Fatalf("failed to get distro: %v", err)
	}
}

func TestIsDistroSupported(t *testing.T) {
	if !isDistroSupported("centos") {
		t.Fatalf("centos should be supported")
	}

	if !isDistroSupported("rhel") {
		t.Fatalf("rhel should be supported")
	}

	if isDistroSupported("ubuntu") {
		t.Fatalf("ubuntu should not be supported")
	}
}
