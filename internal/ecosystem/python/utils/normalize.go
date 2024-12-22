package utils

import "strings"

func NormalizePackageName(name string) string {
	return strings.ReplaceAll(strings.ToLower(name), "_", "-")
}
