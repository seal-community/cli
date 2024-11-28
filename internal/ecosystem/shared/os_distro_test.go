package shared

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

func TestGetDistroDebian(t *testing.T) {
	osReleaseText := `
PRETTY_NAME="Debian GNU/Linux 11 (bullseye)"
NAME="Debian GNU/Linux"
VERSION_ID="11"
VERSION="11 (bullseye)"
VERSION_CODENAME=bullseye
ID=debian
HOME_URL="https://www.debian.org/"
SUPPORT_URL="https://www.debian.org/support"
BUG_REPORT_URL="https://bugs.debian.org/"`

	osRelease, err := ini.Load([]byte(osReleaseText))
	if err != nil {
		t.Fatalf("failed to load os-release: %v", err)
	}

	os, err := getDistro(osRelease)
	if err != nil {
		t.Fatalf("failed to get distro: %v", err)
	}

	if os != "debian" {
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
