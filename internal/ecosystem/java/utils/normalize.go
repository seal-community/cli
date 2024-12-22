package utils

// all maven packages are supposed to be lower case according to
// https://docs.oracle.com/javase/tutorial/java/package/namingpkgs.html
// However, there are some packages that doesn't follow this rule and the current behavior is case sensitive
func NormalizePackageName(name string) string {
	return name
}
