//go:build mockserver
// +build mockserver

package phase

import (
	"cli/internal/ecosystem/shared"
)

func managerMetadata(manager shared.PackageManager) (*PackageManagerMetadata, error) {
	return nil, nil
}

func gatherMetadata(manager shared.PackageManager) (map[string]interface{}, error) {
	return nil, nil
}
