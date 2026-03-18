package provider

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"

	"github.com/GyeongHoKim/preflight/internal/review"
)

// QwenRunner invokes the qwen CLI to perform a code review.
type QwenRunner struct {
	prompt string
}

// NewQwenRunner creates a QwenRunner with the given prompt.
func NewQwenRunner(prompt string) *QwenRunner {
	return &QwenRunner{prompt: prompt}
}

// Run implements Runner for the qwen CLI.
func (r *QwenRunner) Run(ctx context.Context, diff []byte) (review.ProviderResult, error) {
	path, err := exec.LookPath("qwen")
	if err != nil {
		return review.ProviderResult{}, ErrProviderNotFound
	}

	fullPrompt := r.prompt + "\n\n" + string(diff)
	args := []string{
		"-p", fullPrompt,
		"--output-format", "json",
		"--yolo",
	}
	cmd := exec.CommandContext(ctx, path, args...)

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
		return result, fmt.Errorf("qwen: run: %w", err)
	}
	return result, nil
}
