package main

import (
	"cli/internal/common"
	"fmt"

	"github.com/spf13/cobra"
)

func versionCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Show cli version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("%s\n", common.CliVersion)
		},
	}

	return cmd
}
