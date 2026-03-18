package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// buildVersionCmd creates the `preflight version` cobra subcommand.
func buildVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("preflight %s (commit %s, built %s)\n", cfgVersion, cfgCommit, cfgBuildDate)
		},
	}
}
