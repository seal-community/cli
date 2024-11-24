package utils

import "strings"

// Nuget package names are case-insensitive as stated here:
// https://learn.microsoft.com/en-us/nuget/consume-packages/finding-and-choosing-packages
func NormalizeName(name string) string {
	return strings.ToLower(name)
}
