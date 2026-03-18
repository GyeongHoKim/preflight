package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

// buildUninstallCmd creates the `preflight uninstall` cobra subcommand.
func buildUninstallCmd() *cobra.Command {
	var globalFlag bool

	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Remove the preflight pre-push hook",
		RunE: func(cmd *cobra.Command, args []string) error {
			hooksDir, err := resolveHooksDir(globalFlag)
			if err != nil {
				return fmt.Errorf("preflight: %w", err)
			}

			hookPath := filepath.Join(hooksDir, "pre-push")

			if _, err := os.Stat(hookPath); os.IsNotExist(err) {
				return fmt.Errorf("preflight: no hook found at %s", hookPath)
			}

			if !IsManagedHook(hookPath) {
				cmd.PrintErrf("preflight: hook at %s was not installed by preflight; not removing\n", hookPath)
				return nil
			}

			if err := os.Remove(hookPath); err != nil {
				return fmt.Errorf("preflight: remove hook: %w", err)
			}

			cmd.Printf("preflight: hook removed from %s\n", hookPath)
			return nil
		},
	}

	cmd.Flags().BoolVar(&globalFlag, "global", false, "remove from global git hooks path")
	return cmd
}
