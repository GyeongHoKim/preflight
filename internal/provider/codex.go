package provider

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/GyeongHoKim/preflight/internal/review"
)

const codexLargeDiffThreshold = 100 * 1024 // 100 KB

// CodexRunner invokes the codex CLI to perform a code review.
type CodexRunner struct {
	prompt string
}

// NewCodexRunner creates a CodexRunner with the given prompt.
func NewCodexRunner(prompt string) *CodexRunner {
	return &CodexRunner{prompt: prompt}
}

// Run implements Runner for the codex CLI.
func (r *CodexRunner) Run(ctx context.Context, diff []byte) (review.ProviderResult, error) {
	path, err := exec.LookPath("codex")
	if err != nil {
		return review.ProviderResult{}, ErrProviderNotFound
	}

	var fullPrompt string
	if len(diff) > codexLargeDiffThreshold {
		// Write diff to a temp file.
		f, err := os.CreateTemp("", "preflight-diff-*")
		if err != nil {
			return review.ProviderResult{}, fmt.Errorf("codex: create temp file: %w", err)
		}
		defer os.Remove(f.Name()) //nolint:errcheck // best-effort cleanup of temp file
		if _, err := f.Write(diff); err != nil {
			_ = f.Close() // best-effort close before returning error
			return review.ProviderResult{}, fmt.Errorf("codex: write temp file: %w", err)
		}
		if err := f.Close(); err != nil {
			return review.ProviderResult{}, fmt.Errorf("codex: close temp file: %w", err)
		}
		fullPrompt = fmt.Sprintf("%s\n\nSee diff content in: %s", r.prompt, f.Name())
	} else {
		fullPrompt = r.prompt + "\n\n" + string(diff)
	}

	cmd := exec.CommandContext(ctx, path, "-q", "--json", fullPrompt)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	result := review.ProviderResult{
		Stdout: stdout.Bytes(),
		Stderr: stderr.Bytes(),
	}
	if err != nil {
		if asErr, ok := err.(*exec.ExitError); ok { //nolint:errorlint // direct assertion needed to extract exit code
			result.ExitCode = asErr.ExitCode()
		}
		return result, fmt.Errorf("codex: run: %w", err)
	}
	return result, nil
}
