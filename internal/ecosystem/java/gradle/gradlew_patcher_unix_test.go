//go:build !windows

package gradle

import (
	"testing"
)

func TestVerifyGradleWrapperSanity(t *testing.T) {
	data := getTestFile("gradlew.sh.txt")
	cacheOverrideLine := "GRADLE_USER_HOME=/home/user/.seal-gradle"
	gradlewString := string(data)

	if err := verifyGradleWrapper(gradlewString, cacheOverrideLine); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestVerifyGradleWrapperAlreadyPatched(t *testing.T) {
	data := getTestFile("gradlew.sh.txt")
	patchLine := "GRADLE_USER_HOME=/home/user/.seal-gradle"
	gradlewString := string(data)
	gradlewString += "\n" + patchLine

	if err := verifyGradleWrapper(gradlewString, patchLine); err != nil {
		t.Fatalf("expected nil, got error %v", err)
	}
}

func TestVerifyGradleWrapperPatchedForOtherCache(t *testing.T) {
	data := getTestFile("gradlew.sh.txt")
	patchLine := "export GRADLE_USER_HOME=/home/user/.seal-gradle"
	gradlewString := string(data)
	gradlewString += "\n" + patchLine

	if err := verifyGradleWrapper(gradlewString, "export GRADLE_USER_HOME=/home/user/.other-cache"); err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestGetPatchedGradleWrapperContent(t *testing.T) {
	data := getTestFile("gradlew.sh.txt")
	patchedData := getPatchedGradleWrapperContent(data, "added_line=test")
	expectedResult := `#!/bin/sh

#
# SPDX-License-Identifier: Apache-2.0
#

##############################################################################

# Attempt to set APP_HOME

# Resolve links: $0 may be a link
app_path=$0
added_line=test

# Need this for daisy-chained symlinks.
while
    APP_HOME=${app_path%"${app_path##*/}"}  # leaves a trailing /; empty if no leading path
    [ -h "$app_path" ]
do
    ls=$( ls -ld "$app_path" )
    link=${ls#*' -> '}
    case $link in             #(
      /*)   app_path=$link ;; #(
      *)    app_path=$APP_HOME$link ;;
    esac
done
`
	if patchedData != expectedResult {
		t.Fatalf("expected %q, got %q", expectedResult, patchedData)
	}
}

func TestGetPatchedGradleWrapperContentAllComment(t *testing.T) {
	data := `#!/bin/sh
#
# SPDX-License-Identifier: Apache-2.0
#
##############################################################################
`
	patchedData := getPatchedGradleWrapperContent(data, "added_line=test")
	if patchedData != data {
		t.Fatalf("expected %q, got %q", data, patchedData)
	}
}

func TestGetPatchedGradleWrapperContentAllBlankLines(t *testing.T) {
	data := `#!/bin/sh
   
  


`
	patchedData := getPatchedGradleWrapperContent(data, "added_line=test")
	if patchedData != data {
		t.Fatalf("expected %q, got %q", data, patchedData)
	}
}
