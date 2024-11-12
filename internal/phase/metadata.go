//go:build !mockserver
// +build !mockserver

package phase

import (
	"cli/internal/common"
	"cli/internal/ecosystem/shared"
	"fmt"
	"log/slog"
)

func managerMetadata(manager shared.PackageManager) (*PackageManagerMetadata, error) {

	name := manager.Name()
	version := manager.GetVersion()
	if version == "" {
		slog.Error("failed getting version of manager", "name", name)
		// IMPORTANT: in future we might want to return printable error here
		return nil, fmt.Errorf("failed getting package manager version")
	}

	slog.Info("package manager version", "version", version, "name", name)

	return &PackageManagerMetadata{Version: version, Name: name}, nil
}

func gatherMetadata(manager shared.PackageManager) (map[string]interface{}, error) {
	metadata := map[string]interface{}{
		"version": common.CliVersion,
	}

	managerMetadata, err := managerMetadata(manager)
	if err != nil {
		return nil, common.FallbackPrintableMsg(err, fmt.Sprintf("failed gathering %s metadata", manager.Name()))
	}

	if managerMetadata != nil {
		slog.Debug("done collecting manager metadata")
		metadata[manager.Name()] = managerMetadata
	}

	return metadata, nil
}
