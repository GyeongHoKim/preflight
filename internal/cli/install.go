package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

const managedHeader = "# Managed by preflight. Run `preflight uninstall` to remove."

const hookScript = `#!/bin/sh
# Managed by preflight. Run ` + "`preflight uninstall`" + ` to remove.
exec preflight run "$@"
`

// WriteHookScript writes the managed pre-push hook to hooksDir/pre-push.
// If an unmanaged hook already exists and force is false, it returns an error.
func WriteHookScript(hooksDir string, force bool) error {
	hookPath := filepath.Join(hooksDir, "pre-push")

	if _, err := os.Stat(hookPath); err == nil {
		// File exists — check if managed.
		if !IsManagedHook(hookPath) && !force {
			return fmt.Errorf("existing hook found at %s; use --force to replace", hookPath)
		}
	}

	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		return fmt.Errorf("install: create hooks dir: %w", err)
	}

	// 0o755 is required because the hook must be executable by the git subprocess.
	if err := os.WriteFile(hookPath, []byte(hookScript), 0o755); err != nil { //nolint:gosec // hook scripts must be executable (0755)
		return fmt.Errorf("install: write hook: %w", err)
	}
	return nil
}

// IsManagedHook reports whether the file at path was written by preflight.
func IsManagedHook(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	return strings.Contains(string(data), managedHeader)
}

// buildInstallCmd creates the `preflight install` cobra subcommand.
func buildInstallCmd() *cobra.Command {
	var (
		globalFlag bool
		forceFlag  bool
	)

	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install preflight as the git pre-push hook",
		RunE: func(cmd *cobra.Command, args []string) error {
			hooksDir, err := resolveHooksDir(globalFlag)
			if err != nil {
				return fmt.Errorf("preflight: %w", err)
			}

			if err := WriteHookScript(hooksDir, forceFlag); err != nil {
				return fmt.Errorf("preflight: %w", err)
			}

			hookPath := filepath.Join(hooksDir, "pre-push")
			cmd.Printf("preflight: hook installed at %s\n", hookPath)
			return nil
		},
	}

	cmd.Flags().BoolVar(&globalFlag, "global", false, "install into global git hooks path")
	cmd.Flags().BoolVar(&forceFlag, "force", false, "overwrite existing pre-push hook")

	return cmd
}

// resolveHooksDir returns the path to the git hooks directory.
func resolveHooksDir(global bool) (string, error) {
	if global {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("could not determine home directory: %w", err)
		}
		return filepath.Join(home, ".config", "git", "hooks"), nil
	}
	// Local: find .git directory.
	gitDir, err := findGitDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(gitDir, "hooks"), nil
}

// findGitDir returns the path to the .git directory for the current repo.
func findGitDir() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("could not determine working directory: %w", err)
	}
	// Walk up looking for .git.
	dir := wd
	for {
		candidate := filepath.Join(dir, ".git")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("not in a git repository")
}
