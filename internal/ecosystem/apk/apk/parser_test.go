package apk

import (
	"cli/internal/ecosystem/mappings"
	"testing"
)

func TestParseAPKVersion(t *testing.T) {
	apkVersionOutput := `apk-tools 2.14.4, compiled for x86_64.
`
	version := parseAPKVersion(apkVersionOutput)
	if version != "2.14.4" {
		t.Errorf("expected apk version 2.14.4, got %s", version)
	}
}

func TestParseNameVersion(t *testing.T) {
	tests := []struct {
		nameVersion     string
		expectedName    string
		expectedVersion string
	}{
		{
			nameVersion:     "libc++-static-17.0.6-r1",
			expectedName:    "libc++-static",
			expectedVersion: "17.0.6-r1",
		},
		{
			nameVersion:     "acl-2.3.2-r0",
			expectedName:    "acl",
			expectedVersion: "2.3.2-r0",
		},
		{
			nameVersion:     "abseil-cpp-bad-variant-access-20230802.1-r0",
			expectedName:    "abseil-cpp-bad-variant-access",
			expectedVersion: "20230802.1-r0",
		},
		{
			nameVersion:     "7zip-doc-23.01-r0",
			expectedName:    "7zip-doc",
			expectedVersion: "23.01-r0",
		},
		{
			nameVersion:     "apache-mod-auth-kerb-5.4-r9",
			expectedName:    "apache-mod-auth-kerb",
			expectedVersion: "5.4-r9",
		},
	}

	for _, tt := range tests {
		name, version := parseNameVersion(tt.nameVersion)

		if name != tt.expectedName {
			t.Errorf("expected %s, got %s", tt.expectedName, name)
		}
		if version != tt.expectedVersion {
			t.Errorf("expected %s, got %s", tt.expectedVersion, version)
		}
	}
}

func TestParseAPKListInstalledSanity(t *testing.T) {
	apkOutput := `openssl-3.3.2-r1 x86_64 {openssl} (Apache-2.0) [installed]
`
	deps, err := parseAPKListInstalled(apkOutput)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(deps) != 1 {
		t.Errorf("expected 1 dep, got %d", len(deps))
	}

	expectedDepName := "APK|openssl@3.3.2-r1"
	if len(deps[expectedDepName]) != 1 {
		t.Errorf("expected 1 openssl dep, got %d", len(deps[expectedDepName]))
	}

	openssl := deps[expectedDepName][0]
	if openssl.Name != "openssl" {
		t.Errorf("expected openssl, got %s", openssl.Name)
	}
	if openssl.Version != "3.3.2-r1" {
		t.Errorf("expected 3.3.2-r1, got %s", openssl.Version)
	}
	if openssl.Arch != "x86_64" {
		t.Errorf("expected x86_64, got %s", openssl.Arch)
	}
	if openssl.PackageManager != mappings.ApkManager {
		t.Errorf("expected apk, got %s", openssl.PackageManager)
	}
}

func TestParseAPKListInstalledEmpty(t *testing.T) {
	apkOutput := `
`
	deps, err := parseAPKListInstalled(apkOutput)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(deps) != 0 {
		t.Errorf("expected 0 deps, got %d", len(deps))
	}
}

func TestParseAPKListInstalled(t *testing.T) {
	apkOutput := `openssl-3.3.2-r1 x86_64 {openssl} (Apache-2.0) [installed]
patch-2.7.6-r10 x86_64 {patch} (GPL-3.0-or-later) [installed]
pcre2-10.43-r0 x86_64 {pcre2} (BSD-3-Clause) [installed]
pkgconf-2.2.0-r0 x86_64 {pkgconf} (ISC) [installed]
scanelf-1.3.7-r2 x86_64 {pax-utils} (GPL-2.0-only) [installed]
ssl_client-1.36.1-r29 x86_64 {busybox} (GPL-2.0-only) [installed]
sudo-1.9.15_p5-r0 x86_64 {sudo} (custom ISC) [installed]
tar-1.35-r2 x86_64 {tar} (GPL-3.0-or-later) [installed]
zlib-1.3.1-r1 x86_64 {zlib} (Zlib) [installed]
zstd-libs-1.5.6-r0 x86_64 {zstd} (BSD-3-Clause OR GPL-2.0-or-later) [installed]
`
	deps, err := parseAPKListInstalled(apkOutput)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(deps) != 10 {
		t.Errorf("expected 10 dep, got %d", len(deps))
	}

}
