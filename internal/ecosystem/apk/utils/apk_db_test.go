package utils

import (
	"testing"
)

func TestParseAPKDBSanity(t *testing.T) {
	db := `C:Q16skhUkFGZO7TbDnqKclzYLEZSGc=
P:libacl
V:2.2.53-r0
p:libacl=2.2.53-r0

`
	packages := parseAPKDB(db)
	if len(packages) != 1 {
		t.Errorf("expected 1 package, got %d", len(packages))
	}

	libacl := packages["libacl"]
	expectedPackage := PackageInfo{
		Name:     PackageInfoEntry{Value: "libacl", LineIndex: 1},
		Version:  PackageInfoEntry{Value: "2.2.53-r0", LineIndex: 2},
		Provides: PackageInfoEntry{Value: "libacl=2.2.53-r0", LineIndex: 3},
	}
	if libacl != expectedPackage {
		t.Errorf("expected %v, got %v", expectedPackage, libacl)
	}
}

func TestParseAPKDBNoProvides(t *testing.T) {
	db := `C:Q16skhUkFGZO7TbDnqKclzYLEZSGc=
P:libacl
V:2.2.53-r0

`
	packages := parseAPKDB(db)
	if len(packages) != 1 {
		t.Errorf("expected 1 package, got %d", len(packages))
	}

	libacl := packages["libacl"]
	expectedPackage := PackageInfo{
		Name:     PackageInfoEntry{Value: "libacl", LineIndex: 1},
		Version:  PackageInfoEntry{Value: "2.2.53-r0", LineIndex: 2},
		Provides: PackageInfoEntry{Value: "", LineIndex: 0},
	}

	if libacl != expectedPackage {
		t.Errorf("expected %v, got %v", expectedPackage, libacl)
	}
}

func TestParseAPKDBLongPackages(t *testing.T) {
	db := `C:Q1Ie8prbGyPSO/m8x0/ysVgvsnedU=
P:.python-rundeps
V:20250124.193449
A:noarch
S:0
I:0
T:virtual meta package
U:
L:
D:so:libbz2.so.1 so:libc.musl-x86_64.so.1 so:libcrypto.so.3 so:libffi.so.8 so:libgdbm.so.6 so:libgdbm_compat.so.4 so:liblzma.so.5 so:libncursesw.so.6 so:libnsl.so.3 so:libpanelw.so.6 so:libreadline.so.8 so:libsqlite3.so.0 so:libssl.so.3 so:libtirpc.so.3 so:libuuid.so.1 so:libz.so.1

C:Q1qKcZ+j23xssAXmgQhkOO8dHnbWw=
P:alpine-baselayout
V:3.6.5-r0
A:x86_64
S:8515
I:315392
T:Alpine base dir structure and init scripts
U:https://git.alpinelinux.org/cgit/aports/tree/main/alpine-baselayout
L:GPL-2.0-only
o:alpine-baselayout
m:Natanael Copa <ncopa@alpinelinux.org>
t:1714981135
c:66187892e05b03a41d08e9acabd19b7576a1c875
D:alpine-baselayout-data=3.6.5-r0 /bin/sh
q:1000
F:dev
F:dev/pts
F:dev/shm
F:etc
R:motd
Z:Q1SLkS9hBidUbPwwrw+XR0Whv3ww8=
F:etc/crontabs
R:root
a:0:0:600
Z:Q1vfk1apUWI4yLJGhhNRd0kJixfvY=
F:etc/modprobe.d
R:aliases.conf
Z:Q1WUbh6TBYNVK7e4Y+uUvLs/7viqk=
R:blacklist.conf
Z:Q14TdgFHkTdt3uQC+NBtrntOnm9n4=
R:i386.conf
Z:Q1pnay/njn6ol9cCssL7KiZZ8etlc=
R:kms.conf
Z:Q1ynbLn3GYDpvajba/ldp1niayeog=
F:etc/modules-load.d
F:etc/network
F:etc/network/if-down.d
F:etc/network/if-post-down.d
F:etc/network/if-pre-up.d
F:etc/network/if-up.d
F:etc/opt
F:etc/periodic
F:etc/periodic/15min
F:etc/periodic/daily
F:etc/periodic/hourly
F:etc/periodic/monthly
F:etc/periodic/weekly
F:etc/profile.d
R:20locale.sh
Z:Q1lq29lQzPmSCFKVmQ+bvmZ/DPTE4=
R:README
Z:Q135OWsCzzvnB2fmFx62kbqm1Ax1k=
R:color_prompt.sh.disabled
Z:Q11XM9mde1Z29tWMGaOkeovD/m4uU=
F:etc/sysctl.d
F:home
F:lib
F:lib/firmware
F:lib/modules-load.d
F:lib/sysctl.d
R:00-alpine.conf
Z:Q1HpElzW1xEgmKfERtTy7oommnq6c=
F:media
F:media/cdrom
F:media/floppy
F:media/usb
F:mnt
F:opt
F:proc
F:root
M:0:0:700
F:run
F:sbin
F:srv
F:sys
F:tmp
M:0:0:1777
F:usr
F:usr/bin
F:usr/lib
F:usr/lib/modules-load.d
F:usr/local
F:usr/local/bin
F:usr/local/lib
F:usr/local/share
F:usr/sbin
F:usr/share
F:usr/share/man
F:usr/share/misc
F:var
R:run
a:0:0:777
Z:Q11/SNZz/8cK2dSKK+cJpVrZIuF4Q=
F:var/cache
F:var/cache/misc
F:var/empty
M:0:0:555
F:var/lib
F:var/lib/misc
F:var/local
F:var/lock
F:var/lock/subsys
F:var/log
F:var/mail
F:var/opt
F:var/spool
R:mail
a:0:0:777
Z:Q1dzbdazYZA2nTzSIG3YyNw7d4Juc=
F:var/spool/cron
R:crontabs
a:0:0:777
Z:Q1OFZt+ZMp7j0Gny0rqSKuWJyqYmA=
F:var/tmp
M:0:0:1777

C:Q17mim+wL35iMEtCiwQEovweL8NT0=
P:alpine-baselayout-data
V:3.6.5-r0
A:x86_64
S:11235
I:77824
T:Alpine base dir structure and init scripts
U:https://git.alpinelinux.org/cgit/aports/tree/main/alpine-baselayout
L:GPL-2.0-only
o:alpine-baselayout
m:Natanael Copa <ncopa@alpinelinux.org>
t:1714981135
c:66187892e05b03a41d08e9acabd19b7576a1c875
r:alpine-baselayout
q:1000
F:etc
R:fstab
Z:Q11Q7hNe8QpDS531guqCdrXBzoA/o=
R:group
Z:Q12Otk4M39fP2Zjkobu0nC9FvlRI0=
R:hostname
Z:Q16nVwYVXP/tChvUPdukVD2ifXOmc=
R:hosts

`
	packages := parseAPKDB(db)

	if len(packages) != 3 {
		t.Errorf("expected 3 packages, got %d", len(packages))
	}

	if packages[".python-rundeps"].Name.LineIndex != 1 {
		t.Errorf("expected 0, got %d", packages[".python-rundeps"].Name.LineIndex)
	}

	if packages["alpine-baselayout"].Name.LineIndex != 12 {
		t.Errorf("expected 11, got %d", packages["alpine-baselayout"].Name.LineIndex)
	}

	if packages["alpine-baselayout-data"].Name.LineIndex != 128 {
		t.Errorf("expected 47, got %d", packages["alpine-baselayout-data"].Name.LineIndex)
	}
}

func TestModifyApkDBContentForSilenceSanity(t *testing.T) {
	db := `C:Q16skhUkFGZO7TbDnqKclzYLEZSGc=
P:libacl
V:2.2.53-r0
p:so:libacl

`
	rulePackageInfo := PackageInfo{
		Name:     PackageInfoEntry{Value: "libacl", LineIndex: 1},
		Version:  PackageInfoEntry{Value: "2.2.53-r0", LineIndex: 2},
		Provides: PackageInfoEntry{Value: "so:libacl", LineIndex: 3},
	}

	newDBContent := modifyApkDBContentForSilence(db, rulePackageInfo, "seal-libacl")
	expected := `C:Q16skhUkFGZO7TbDnqKclzYLEZSGc=
P:seal-libacl
V:2.2.53-r0
p:so:libacl libacl=2.2.53-r0

`
	if newDBContent != expected {
		t.Errorf("expected\n%sgot\n\n%s", expected, newDBContent)
	}
}

func TestModifyApkDBContentForSilenceNoProvides(t *testing.T) {
	db := `C:Q16skhUkFGZO7TbDnqKclzYLEZSGc=
P:libacl
V:2.2.53-r0

`
	rulePackageInfo := PackageInfo{
		Name:     PackageInfoEntry{Value: "libacl", LineIndex: 1},
		Version:  PackageInfoEntry{Value: "2.2.53-r0", LineIndex: 2},
		Provides: PackageInfoEntry{Value: "", LineIndex: 0},
	}

	newDBContent := modifyApkDBContentForSilence(db, rulePackageInfo, "seal-libacl")
	expected := `C:Q16skhUkFGZO7TbDnqKclzYLEZSGc=
P:seal-libacl
p:libacl=2.2.53-r0
V:2.2.53-r0

`
	if newDBContent != expected {
		t.Errorf("expected\n%sgot\n\n%s", expected, newDBContent)
	}
}
