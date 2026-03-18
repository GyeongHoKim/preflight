package provider

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"

	"github.com/GyeongHoKim/preflight/internal/review"
)

// GeminiRunner invokes the gemini CLI to perform a code review.
type GeminiRunner struct {
	prompt string
}

// NewGeminiRunner creates a GeminiRunner with the given prompt.
func NewGeminiRunner(prompt string) *GeminiRunner {
	return &GeminiRunner{prompt: prompt}
}

// Run implements Runner for the gemini CLI.
func (r *GeminiRunner) Run(ctx context.Context, diff []byte) (review.ProviderResult, error) {
	path, err := exec.LookPath("gemini")
	if err != nil {
		return review.ProviderResult{}, ErrProviderNotFound
	}

	fullPrompt := r.prompt + "\n\n" + string(diff)
	cmd := exec.CommandContext(ctx, path, "--prompt", fullPrompt, "--output-format", "json")

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
		return result, fmt.Errorf("gemini: run: %w", err)
	}
	return result, nil
}
