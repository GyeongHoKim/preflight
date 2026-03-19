// Package cli implements the cobra command tree for preflight.
package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/GyeongHoKim/preflight/internal/config"
)

var (
	cfgVersion   string
	cfgCommit    string
	cfgBuildDate string
)

// rootCmd is the top-level cobra command.
var rootCmd *cobra.Command

// cfg holds the resolved config for the current run.
var cfg *config.Config

// globalFlags holds the global flag values.
var globalFlags struct {
	configPath string
	noTUI      bool
	verbose    bool
	provider   string
}

// Execute initialises the command tree and runs the CLI.
// It returns the process exit code.
func Execute(version, commit, buildDate string) int {
	cfgVersion = version
	cfgCommit = commit
	cfgBuildDate = buildDate

	rootCmd = buildRootCmd()
	if err := rootCmd.Execute(); err != nil {
		return 1
	}
	return 0
}

// buildRootCmd constructs and returns the root cobra.Command.
func buildRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "preflight",
		Short:         "AI-powered pre-push code review",
		Long:          "preflight intercepts git pushes, runs the diff through a local AI CLI, and blocks pushes with critical issues.",
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			return loadConfig(cmd)
		},
	}

	root.PersistentFlags().StringVar(&globalFlags.configPath, "config", "", "path to config file")
	root.PersistentFlags().BoolVar(&globalFlags.noTUI, "no-tui", false, "disable terminal UI; write plain text to stdout")
	root.PersistentFlags().BoolVar(&globalFlags.verbose, "verbose", false, "emit debug information to stderr")
	root.PersistentFlags().StringVar(&globalFlags.provider, "provider", "", "AI provider: auto, claude, codex, gemini, qwen")

	root.AddCommand(
		buildRunCmd(),
		buildInstallCmd(),
		buildUninstallCmd(),
		buildVersionCmd(),
	)

	return root
}

// loadConfig resolves and validates configuration for the run.
func loadConfig(_ *cobra.Command) error {
	projectPath := "./.preflight.yml"
	globalPath := os.ExpandEnv("$HOME/.config/preflight/.preflight.yml")

	if globalFlags.configPath != "" {
		projectPath = globalFlags.configPath
	}

	var err error
	cfg, err = config.Load(projectPath, globalPath)
	if err != nil {
		return fmt.Errorf("preflight: %w", err)
	}

	// --provider flag overrides config.
	if globalFlags.provider != "" {
		cfg.Provider = globalFlags.provider
		// Re-validate provider.
		validProviders := map[string]bool{
			"auto": true, "claude": true, "codex": true, "gemini": true, "qwen": true,
		}
		if !validProviders[cfg.Provider] {
			return fmt.Errorf("preflight: invalid provider %q; must be one of auto, claude, codex, gemini, qwen", cfg.Provider)
		}
	}

	if globalFlags.verbose {
		fmt.Fprintf(os.Stderr, "preflight: using provider %q\n", cfg.Provider)
	}

	return nil
}
