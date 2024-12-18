package utils

import (
	"cli/internal/common"
	"fmt"
)

func BuildDebName(name, version, arch string) string {
	_, version = common.GetNoEpochVersion(version)
	return fmt.Sprintf("%s_%s_%s.deb", name, version, arch)
}
