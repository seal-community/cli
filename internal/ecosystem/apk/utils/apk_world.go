package utils

import (
	"strings"
)

const ApkWorldPath = "/etc/apk/world"

func ApkWorldRemoveHashRestriction(packageName string, world string) string {
	var newWorld string

	hashRestriction := packageName + "><"
	for _, line := range strings.Split(world, "\n") {
		if strings.Contains(line, hashRestriction) {
			line = packageName
		}
		newWorld += line + "\n"
	}

	return strings.TrimSuffix(newWorld, "\n")
}
