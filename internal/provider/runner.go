// Package provider implements AI CLI subprocess adapters for preflight.
package provider

import (
	"context"
	"errors"
	"os/exec"

	"github.com/GyeongHoKim/preflight/internal/review"
)

// ErrProviderNotFound is returned when the requested AI CLI binary is not in PATH.
var ErrProviderNotFound = errors.New("provider: binary not found in PATH")

// ErrProviderTimeout is returned when the AI CLI subprocess exceeds its deadline.
var ErrProviderTimeout = errors.New("provider: timed out")

// Runner is the interface implemented by every AI provider adapter.
type Runner interface {
	// Run invokes the AI CLI with diff as input and returns the raw result.
	Run(ctx context.Context, diff []byte) (review.ProviderResult, error)
}

// shouldFailOpen reports whether err represents a condition under which preflight
// must exit 0 (fail-open) rather than blocking the push.
//
// Fail-open conditions:
//   - ErrProviderNotFound: binary not in PATH
//   - context.DeadlineExceeded: AI CLI timed out
//   - *exec.ExitError: non-zero exit from the AI CLI subprocess
func shouldFailOpen(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, ErrProviderNotFound) {
		return true
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		// Non-zero exit codes from the AI CLI are treated as fail-open.
		// Auth failures, rate limits, and other transient errors are
		// indistinguishable from other non-zero exits at this level; all are
		// intentionally fail-open to honour the never-block-a-push guarantee.
		return true
	}
	return false
}

// ShouldFailOpen is the exported version of shouldFailOpen for use in hook.
func ShouldFailOpen(err error) bool {
	return shouldFailOpen(err)
}

// Detect returns the name of the first AI CLI found in PATH from the given
// ordered list of providers. It returns ErrProviderNotFound if none are found.
func Detect(providers []string) (string, error) {
	for _, p := range providers {
		if _, err := exec.LookPath(p); err == nil {
			return p, nil
		}
	}
	return "", ErrProviderNotFound
}

// MockRunner is a test double for Runner that returns configurable results.
// If Results is non-empty, it is used as a sequence (one entry per call);
// the last entry is repeated if calls exceed len(Results). Otherwise Result is used.
type MockRunner struct {
	Result    review.ProviderResult
	Results   []review.ProviderResult // sequence mode; overrides Result when non-empty
	Err       error
	CallCount int
}

// Run implements Runner.
func (m *MockRunner) Run(_ context.Context, _ []byte) (review.ProviderResult, error) {
	m.CallCount++
	if len(m.Results) > 0 {
		idx := m.CallCount - 1
		if idx >= len(m.Results) {
			idx = len(m.Results) - 1
		}
		return m.Results[idx], m.Err
	}
	return m.Result, m.Err
}
