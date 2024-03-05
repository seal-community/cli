package utils

import (
	"fmt"
	"strings"
)

func EscapePackageName(name string) string {
	return strings.ReplaceAll(name, "-", "_")
}

func DistInfoPath(name string, version string) string {
	return fmt.Sprintf("%s-%s.dist-info", EscapePackageName(name), version)
}
