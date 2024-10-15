package yum

import (
	"cli/internal/ecosystem/mappings"
	"testing"
)

func TestParseYumVersion(t *testing.T) {
	yumOutput := `4.7.0
  Installed: dnf-0:4.7.0-20.el8.noarch at Tue Sep 17 13:50:17 2024
  Built    : Red Hat, Inc. <http://bugzilla.redhat.com/bugzilla> at Mon Oct 16 13:53:08 2023

  Installed: rpm-0:4.14.3-31.el8.x86_64 at Tue Sep 17 13:50:16 2024
  Built    : Red Hat, Inc. <http://bugzilla.redhat.com/bugzilla> at Wed Dec 13 13:45:49 2023
`
	version := parseYumVersion(yumOutput)
	if version != "4.7.0" {
		t.Errorf("expected version 4.7.0, got %s", version)
	}
}

func TestParseNameArch(t *testing.T) {
	name, arch := parseNameArch("acl.x86_64")
	if name != "acl" {
		t.Errorf("expected acl, got %s", name)
	}
	if arch != "x86_64" {
		t.Errorf("expected x86_64, got %s", arch)
	}
}

func TestParseNameArchDot(t *testing.T) {
	name, arch := parseNameArch("acl.test.x86_64")
	if name != "acl.test" {
		t.Errorf("expected acl, got %s", name)
	}
	if arch != "x86_64" {
		t.Errorf("expected x86_64, got %s", arch)
	}
}

func TestParseYumListInstalled(t *testing.T) {
	yumOutput := `Updating Subscription Management repositories.
Unable to read consumer identity

This system is not registered with an entitlement server. You can use subscription-manager to register.

Installed Packages
acl.x86_64                                                                                                                                                              2.2.53-3.el8                                                                                                                                              @System
basesystem.noarch                                                                                                                                                       11-5.el8                                                                                                                                                  @System
crypto-policies-scripts.noarch                                                                                                                                          20230731-1.git3177e06.el8                                                                                                                                 @System
`
	normalizer := NewYumManager(nil, "")
	deps, err := parseYumListInstalled(yumOutput, normalizer)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(deps) != 3 {
		t.Errorf("expected 3 deps, got %d", len(deps))
	}

	if len(deps["RPM|acl@2.2.53-3.el8"]) != 1 {
		t.Errorf("expected 1 acl dep, got %d", len(deps["RPM|acl@2.2.53-3.el8"]))
	}

	acl := deps["RPM|acl@2.2.53-3.el8"][0]
	if acl.Name != "acl" {
		t.Errorf("expected acl, got %s", acl.Name)
	}
	if acl.Version != "2.2.53-3.el8" {
		t.Errorf("expected 2.2.53-3.el8, got %s", acl.Version)
	}
	if acl.Arch != "x86_64" {
		t.Errorf("expected x86_64, got %s", acl.Arch)
	}
	if acl.PackageManager != mappings.RpmManager {
		t.Errorf("expected yum, got %s", acl.PackageManager)
	}

	if len(deps["RPM|basesystem@11-5.el8"]) != 1 {
		t.Errorf("expected 1 basesystem dep, got %d", len(deps["RPM|basesystem@11-5.el8"]))
	}

	basesystem := deps["RPM|basesystem@11-5.el8"][0]
	if basesystem.Name != "basesystem" {
		t.Errorf("expected basesystem, got %s", basesystem.Name)
	}
	if basesystem.Version != "11-5.el8" {
		t.Errorf("expected 11-5.el8, got %s", basesystem.Version)
	}
	if basesystem.Arch != "noarch" {
		t.Errorf("expected noarch, got %s", basesystem.Arch)
	}
	if basesystem.PackageManager != mappings.RpmManager {
		t.Errorf("expected yum, got %s", basesystem.PackageManager)
	}

	if len(deps["RPM|crypto-policies-scripts@20230731-1.git3177e06.el8"]) != 1 {
		t.Errorf("expected 1 crypto-policies-scripts dep, got %d", len(deps["RPM|crypto-policies-scripts@20230731-1.git3177e06.el8"]))
	}

	crypto := deps["RPM|crypto-policies-scripts@20230731-1.git3177e06.el8"][0]
	if crypto.Name != "crypto-policies-scripts" {
		t.Errorf("expected crypto-policies-scripts, got %s", crypto.Name)
	}
	if crypto.Version != "20230731-1.git3177e06.el8" {
		t.Errorf("expected 20230731-1.git3177e06.el8, got %s", crypto.Version)
	}
	if crypto.Arch != "noarch" {
		t.Errorf("expected noarch, got %s", crypto.Arch)
	}
	if crypto.PackageManager != mappings.RpmManager {
		t.Errorf("expected yum, got %s", crypto.PackageManager)
	}
}

func TestParseYumListInstalledNoPrefix(t *testing.T) {
	yumOutput := `Installed Packages
acl.x86_64                                                                                                                                                              2.2.53-3.el8                                                                                                                                              @System
`
	normalizer := NewYumManager(nil, "")
	deps, err := parseYumListInstalled(yumOutput, normalizer)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(deps) != 1 {
		t.Errorf("expected 1 dep, got %d", len(deps))
	}

	if len(deps["RPM|acl@2.2.53-3.el8"]) != 1 {
		t.Errorf("expected 1 acl dep, got %d", len(deps["RPM|acl@2.2.53-3.el8"]))
	}

	acl := deps["RPM|acl@2.2.53-3.el8"][0]
	if acl.Name != "acl" {
		t.Errorf("expected acl, got %s", acl.Name)
	}
	if acl.Version != "2.2.53-3.el8" {
		t.Errorf("expected 2.2.53-3.el8, got %s", acl.Version)
	}
	if acl.Arch != "x86_64" {
		t.Errorf("expected x86_64, got %s", acl.Arch)
	}
	if acl.PackageManager != mappings.RpmManager {
		t.Errorf("expected yum, got %s", acl.PackageManager)
	}
}

func TestParseYumListInstalledEmpty(t *testing.T) {
	yumOutput := `Installed Packages
`
	normalizer := NewYumManager(nil, "")
	deps, err := parseYumListInstalled(yumOutput, normalizer)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(deps) != 0 {
		t.Errorf("expected 0 deps, got %d", len(deps))
	}
}

func TestParseYumListInstalledWrapped(t *testing.T) {
	yumOutput := `Loaded plugins: ovl
Installed Packages
devtoolset-10-gcc-gfortran.x86_64    10.2.1-11.2.el7             @centos-sclo-rh
devtoolset-10-libquadmath-devel.x86_64
                                     10.2.1-11.2.el7             @centos-sclo-rh
devtoolset-10-libstdc++-devel.x86_64 10.2.1-11.2.el7             @centos-sclo-rh
devtoolset-10-libstdc++-devel-longlonglonglonglonglonglonglonglonglonglonglonglonglonglonglonglonglonglonglonglonglonglonglonglonglonglonglong.x86_64
		                                     10.2.1-11.2.el7             @centos-sclo-rh
seal-libxml2.x86_64                  2.9.1-6.el7_9.6+sp1         @/libxml2-2.9.1-6.el7_9.6.x86_64
`
	normalizer := NewYumManager(nil, "")
	deps, err := parseYumListInstalled(yumOutput, normalizer)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(deps) != 5 {
		t.Errorf("expected 1 dep, got %d", len(deps))
	}

}
