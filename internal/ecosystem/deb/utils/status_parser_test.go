package utils

import (
	"bytes"
	"testing"
)

func assertEqual(t *testing.T, expected string, current string) {
	if expected != current {
		t.Errorf("Invalid value. Expected '%v', got '%v'", expected, current)
	}
}

func TestParseValidData(t *testing.T) {
	data := `Package: libquadmath0
Status: install ok installed
Priority: optional
Section: libs
Installed-Size: 275
Maintainer: Debian GCC Maintainers <debian-gcc@lists.debian.org>
Architecture: amd64
Multi-Arch: same
Source: gcc-4.9
Version: 4.9.2-10
Depends: gcc-4.9-base (= 4.9.2-10), libc6 (>= 2.14)
Pre-Depends: multiarch-support
Conffiles:
 /etc/deluser.conf 773fb95e98a27947de4a95abb3d3f2a2
Description: GCC Quad-Precision Math Library
 A library, which provides quad-precision mathematical functions on targets
 supporting the __float128 datatype. The library is used to provide on such
 targets the REAL(16) type in the GNU Fortran compiler.
Homepage: http://gcc.gnu.org/

Package: libedit2
Status: install ok installed
Priority: standard
Section: libs
Installed-Size: 277
Maintainer: LLVM Packaging Team <pkg-llvm-team@lists.alioth.debian.org>
Architecture: amd64
Multi-Arch: same
Source: libedit
Version: 3.1-20140620-2
Depends: libbsd0 (>= 0.0), libc6 (>= 2.17), libtinfo5
Pre-Depends: multiarch-support
Description: BSD editline and history libraries
 Command line editor library provides generic line editing,
 history, and tokenization functions.
 .
 It slightly resembles GNU readline.
Homepage: http://www.thrysoee.dk/editline/`

	reader := bytes.NewBufferString(data)
	parser := NewParser(reader)
	packages, err := parser.Parse()
	if err != nil {
		t.Errorf("Parse returned error %s", err)
	}

	if len(packages) != 2 {
		t.Errorf("Expected 2 packages, got: %v", len(packages))
	}

	pkg := packages[0]
	assertEqual(t, "libquadmath0", pkg.Package)
	assertEqual(t, "4.9.2-10", pkg.Version)
	assertEqual(t, "libs", pkg.Section)
	assertEqual(t, "Debian GCC Maintainers <debian-gcc@lists.debian.org>", pkg.Maintainer)
	assertEqual(t, "install ok installed", pkg.Status)
	assertEqual(t, "gcc-4.9", pkg.Source)
	assertEqual(t, "amd64", pkg.Architecture)
	assertEqual(t, "same", pkg.MultiArch)
	assertEqual(t, "gcc-4.9-base (= 4.9.2-10), libc6 (>= 2.14)", pkg.Depends)
	assertEqual(t, "multiarch-support", pkg.PreDepends)
	assertEqual(t, "http://gcc.gnu.org/", pkg.Homepage)
	assertEqual(t, "optional", pkg.Priority)
	assertEqual(t, "\n/etc/deluser.conf 773fb95e98a27947de4a95abb3d3f2a2", pkg.Conffiles)
	if pkg.InstalledSize != 275 {
		t.Errorf("Incorrect size: %v", pkg.InstalledSize)
	}

	pkg = packages[1]
	assertEqual(t, "libedit2", pkg.Package)
	assertEqual(t, "3.1-20140620-2", pkg.Version)
	assertEqual(t, "libs", pkg.Section)
	assertEqual(t, "LLVM Packaging Team <pkg-llvm-team@lists.alioth.debian.org>", pkg.Maintainer)
	assertEqual(t, "install ok installed", pkg.Status)
	assertEqual(t, "libedit", pkg.Source)
	assertEqual(t, "amd64", pkg.Architecture)
	assertEqual(t, "same", pkg.MultiArch)
	assertEqual(t, "libbsd0 (>= 0.0), libc6 (>= 2.17), libtinfo5", pkg.Depends)
	assertEqual(t, "multiarch-support", pkg.PreDepends)
	assertEqual(t, "http://www.thrysoee.dk/editline/", pkg.Homepage)
	assertEqual(t, "standard", pkg.Priority)
	if pkg.InstalledSize != 277 {
		t.Errorf("Incorrect size: %v", pkg.InstalledSize)
	}
}

func TestParseNoNewlineAtEOF(t *testing.T) {
	data := `Package: libquadmath0
Protected: yes
Essential: no
Status: install ok installed
Priority: optional
Section: libs
Installed-Size: 275
Maintainer: Debian GCC Maintainers <debian-gcc@lists.debian.org>
Architecture: amd64
Multi-Arch: same
Source: gcc-4.9
Version: 4.9.2-10
Provides: libquadmath0
Replaces: libquadmath0
Depends: gcc-4.9-base (= 4.9.2-10), libc6 (>= 2.14)
Pre-Depends: multiarch-support
Recommends: gcc-4.9
Suggests: gcc-4.9
Breaks: gcc-4.9
Enhances: gcc-4.9
Conflicts: gcc-4.9
Conffiles:
 /etc/deluser.conf 773fb95e98a27947de4a95abb3d3f2a2
Description: GCC Quad-Precision Math Library
 A library, which provides quad-precision mathematical functions on targets
 supporting the __float128 datatype. The library is used to provide on such
 targets the REAL(16) type in the GNU Fortran compiler.
Homepage: http://gcc.gnu.org/
Important: yes`
	reader := bytes.NewBufferString(data)
	parser := NewParser(reader)
	packages, err := parser.Parse()
	if err != nil {
		t.Errorf("Parse returned error %s", err)
	}

	if len(packages) != 1 {
		t.Errorf("Expected 1 packages, got: %v", len(packages))
	}

	pkg := packages[0]
	assertEqual(t, "libquadmath0", pkg.Package)
	assertEqual(t, "4.9.2-10", pkg.Version)
	assertEqual(t, "libs", pkg.Section)
	assertEqual(t, "Debian GCC Maintainers <debian-gcc@lists.debian.org>", pkg.Maintainer)
	assertEqual(t, "install ok installed", pkg.Status)
	assertEqual(t, "gcc-4.9", pkg.Source)
	assertEqual(t, "amd64", pkg.Architecture)
	assertEqual(t, "same", pkg.MultiArch)
	assertEqual(t, "gcc-4.9-base (= 4.9.2-10), libc6 (>= 2.14)", pkg.Depends)
	assertEqual(t, "multiarch-support", pkg.PreDepends)
	assertEqual(t, "http://gcc.gnu.org/", pkg.Homepage)
	assertEqual(t, "optional", pkg.Priority)
	assertEqual(t, "\n/etc/deluser.conf 773fb95e98a27947de4a95abb3d3f2a2", pkg.Conffiles)
	assertEqual(t, "yes", pkg.Protected)
	assertEqual(t, "no", pkg.Essential)
	assertEqual(t, "libquadmath0", pkg.Provides)
	assertEqual(t, "libquadmath0", pkg.Replaces)
	assertEqual(t, "gcc-4.9", pkg.Recommends)
	assertEqual(t, "gcc-4.9", pkg.Suggests)
	assertEqual(t, "gcc-4.9", pkg.Breaks)
	assertEqual(t, "gcc-4.9", pkg.Enhances)
	assertEqual(t, "gcc-4.9", pkg.Conflicts)
	assertEqual(t, "yes", pkg.Important)
}

func TestDump(t *testing.T) {
	data := `Package: tar
Essential: yes
Status: install ok installed
Priority: required
Section: utils
Installed-Size: 3152
Maintainer: Janos Lenart <ocsi@debian.org>
Architecture: amd64
Multi-Arch: foreign
Version: 1.34+dfsg-1+deb11u1
Replaces: cpio (<< 2.4.2-39)
Pre-Depends: libacl1 (>= 2.2.23), libc6 (>= 2.28), libselinux1 (>= 3.1~)
Suggests: bzip2, ncompress, xz-utils, tar-scripts, tar-doc
Breaks: dpkg-dev (<< 1.14.26)
Conflicts: cpio (<= 2.4.2-38)
Description: GNU version of the tar archiving utility
 Tar is a program for packaging a set of files as a single archive in tar
 format.  The function it performs is conceptually similar to cpio, and to
 things like PKZIP in the DOS world.  It is heavily used by the Debian package
 management system, and is useful for performing system backups and exchanging
 sets of files with others.
Homepage: https://www.gnu.org/software/tar/

Package: tzdata
Status: install ok installed
Priority: required
Section: localization
Installed-Size: 3454
Maintainer: GNU Libc Maintainers <debian-glibc@lists.debian.org>
Architecture: all
Multi-Arch: foreign
Version: 2024b-0+deb11u1
Provides: tzdata-bullseye
Depends: debconf (>= 0.5) | debconf-2.0
Conffiles:
 /etc/deluser.conf 773fb95e98a27947de4a95abb3d3f2a2
Description: time zone and daylight-saving time data
 This package contains data required for the implementation of
 standard local time for many representative locations around the
 globe. It is updated periodically to reflect changes made by
 political bodies to time zone boundaries, UTC offsets, and
 daylight-saving rules.
Homepage: https://www.iana.org/time-zones


`

	reader := bytes.NewBufferString(data)
	parser := NewParser(reader)
	packages, err := parser.Parse()
	if err != nil {
		t.Errorf("Parse returned error %s", err)
	}
	packages_dump := DumpPackages(packages)
	assertEqual(t, data, packages_dump)
}
