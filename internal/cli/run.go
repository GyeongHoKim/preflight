package cli

import (
	"context"
	"os"

	"github.com/spf13/cobra"

	"github.com/GyeongHoKim/preflight/internal/hook"
)

// buildRunCmd creates the `preflight run` cobra subcommand.
func buildRunCmd() *cobra.Command {
	var (
		providerFlag string
		noTUIFlag    bool
		blockOnFlag  string
	)

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run a code review on the current branch diff",
		Long:  "Runs an AI code review against the current branch's diff. Called automatically by the pre-push hook.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if providerFlag != "" {
				cfg.Provider = providerFlag
			}
			if blockOnFlag != "" {
				cfg.BlockOn = blockOnFlag
			}

			noTUI := noTUIFlag || globalFlags.noTUI
			code := hook.Run(context.Background(), cfg, os.Stdin, os.Stdout, os.Stderr, noTUI, nil, nil)
			os.Exit(code)
			return nil
		},
	}

	cmd.Flags().StringVar(&providerFlag, "provider", "", "AI provider override for this run")
	cmd.Flags().BoolVar(&noTUIFlag, "no-tui", false, "plain-text output; do not launch TUI")
	cmd.Flags().StringVar(&blockOnFlag, "block-on", "", "minimum severity to block: critical or warning")

	return cmd
}
