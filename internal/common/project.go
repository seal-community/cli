package common

import "regexp"

const MaxProjectNameLen = 255

func NormalizeProjectName(name string) string {
	re1 := regexp.MustCompile(`[ /\\]`)
	name = re1.ReplaceAllString(name, "-")

	re2 := regexp.MustCompile(`[^a-zA-Z0-9_\-\.]`)
	name = re2.ReplaceAllString(name, "")

	// Trim the name to the maximum allowed length
	if len(name) > MaxProjectNameLen {
		name = name[:MaxProjectNameLen]
	}

	return name
}
