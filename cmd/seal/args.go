package main

import (
	"cli/internal/common"
	"fmt"
	"log/slog"

	"github.com/spf13/cobra"
)

func getArgBool(cmd *cobra.Command, key string) bool {
	v, err := cmd.Flags().GetBool(key)
	if err != nil {
		// means misconfiguration in code
		slog.Error("failed getting flag", "err", err, "key", key)
		panic(fmt.Sprintf("failed getting bool key %s", key))
	}

	return v
}

func getArgString(cmd *cobra.Command, key string) string {
	v, err := cmd.Flags().GetString(key)
	if err != nil {
		// means misconfiguration in code
		slog.Error("failed getting flag", "err", err, "key", key)
		panic(fmt.Sprintf("failed getting string key %s", key))
	}

	return v
}

func getArgCount(cmd *cobra.Command, key string) int {
	val, err := cmd.Flags().GetCount(key)
	if err != nil {
		// means misconfiguration in code
		slog.Error("failed getting flag", "err", err, "key", key)
		panic(fmt.Sprintf("failed getting string key %s", key))
	}

	return val
}

// Given command line args, extract the target path and target type
func extractTarget(args []string, filesystem string, isOs bool) (string, common.TargetType) {
	if isOs && filesystem != "" {
		slog.Error("invalid target", "filesystem", filesystem, "os", isOs)
		return "", common.UnknownTarget
	}

	target := ""
	if len(args) > 0 {
		target = args[0]
	}

	// handle `seal scan os`, which is deprecated and should become `seal scan --os`
	if target == "os" || isOs {
		slog.Debug("detected os target")
		return "", common.OsTarget
	}

	if filesystem == "" {
		slog.Debug("detected manifest target")
		return target, common.ManifestTarget
	}

	switch filesystem {
	case string(common.JavaFilesTarget):
		slog.Debug("detected java files target")
		return target, common.JavaFilesTarget
	default:
		slog.Error("invalid target", "target", filesystem)
		return "", common.UnknownTarget
	}
}

func getArgArray(cmd *cobra.Command, key string) []string {
	v, err := cmd.Flags().GetStringArray(key)
	if err != nil {
		slog.Error("failed getting flag", "err", err, "key", key)
		panic(fmt.Sprintf("failed getting string array key %s", key))
	}

	return v
}
