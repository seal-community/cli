package golang

import "strings"

func NormalizePackageName(name string) string {
	return strings.ToLower(name)
}
