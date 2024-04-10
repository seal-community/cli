package main

import (
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
	summaryPath, err := cmd.Flags().GetString(key)
	if err != nil {
		// means misconfiguration in code
		slog.Error("failed getting flag", "err", err, "key", key)
		panic(fmt.Sprintf("failed getting string key %s", key))
	}
	
	return summaryPath
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

func extractTarget(args []string) string {
	if len(args) > 0 {
		return args[0]
	}

	return ""
}
