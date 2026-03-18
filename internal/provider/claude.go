package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"

	"github.com/GyeongHoKim/preflight/internal/review"
)

// claudeEnvelope matches the JSON envelope returned by `claude --output-format json`.
type claudeEnvelope struct {
	Type    string `json:"type"`
	Subtype string `json:"subtype"`
	IsError bool   `json:"is_error"`
	Result  string `json:"result"`
}

// ClaudeRunner invokes the claude CLI to perform a code review.
type ClaudeRunner struct {
	prompt string
	schema string
}

// NewClaudeRunner creates a ClaudeRunner with the given prompt and JSON schema.
func NewClaudeRunner(prompt, schema string) *ClaudeRunner {
	return &ClaudeRunner{prompt: prompt, schema: schema}
}

// Run implements Runner for the claude CLI.
func (r *ClaudeRunner) Run(ctx context.Context, diff []byte) (review.ProviderResult, error) {
	path, err := exec.LookPath("claude")
	if err != nil {
		return review.ProviderResult{}, ErrProviderNotFound
	}

	args := []string{
		"-p", r.prompt,
		"--output-format", "json",
		"--no-session-persistence",
		"--json-schema", r.schema,
	}
	cmd := exec.CommandContext(ctx, path, args...)
	cmd.Stdin = bytes.NewReader(diff)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err = cmd.Run()
	result := review.ProviderResult{
		Stdout: stdout.Bytes(),
		Stderr: stderr.Bytes(),
	}
	if err != nil {
		var exitErr *exec.ExitError
		if asErr, ok := err.(*exec.ExitError); ok { //nolint:errorlint // direct assertion needed to extract exit code
			exitErr = asErr
			result.ExitCode = exitErr.ExitCode()
		}
		return result, fmt.Errorf("claude: run: %w", err)
	}

	// Validate that the envelope is not an error.
	var env claudeEnvelope
	if jsonErr := json.Unmarshal(stdout.Bytes(), &env); jsonErr == nil && env.IsError {
		return result, fmt.Errorf("claude: response is_error=true")
	}

	return result, nil
}

// ParseClaudeResult extracts the canonical review JSON from a claude response envelope.
// Returns nil if the result cannot be parsed (fail-open).
func ParseClaudeResult(raw review.ProviderResult) ([]byte, error) {
	if len(raw.Stdout) == 0 {
		return nil, nil
	}
	var env claudeEnvelope
	if err := json.Unmarshal(raw.Stdout, &env); err != nil {
		// Not a JSON envelope; return raw stdout for best-effort parsing.
		return raw.Stdout, nil //nolint:nilerr // intentional: non-envelope stdout is passed through as-is
	}
	if env.IsError {
		return nil, nil
	}
	if env.Result == "" {
		return raw.Stdout, nil
	}
	return []byte(env.Result), nil
}
