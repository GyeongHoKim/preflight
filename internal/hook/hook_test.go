package hook

import (
	"bytes"
	"context"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/GyeongHoKim/preflight/internal/config"
	"github.com/GyeongHoKim/preflight/internal/diff"
	"github.com/GyeongHoKim/preflight/internal/provider"
	"github.com/GyeongHoKim/preflight/internal/review"
	"github.com/GyeongHoKim/preflight/internal/review/reviewtest"
)

// fakeDiff is a test double for diff.Collector that returns configurable bytes.
type fakeDiff struct{ data []byte }

func (f fakeDiff) Collect(_ context.Context, _ diff.PushInfo, _ int) ([]byte, error) {
	return f.data, nil
}

// someDiff returns a non-empty diff payload suitable for triggering review logic.
func someDiff() fakeDiff {
	return fakeDiff{data: []byte("diff --git a/foo.go b/foo.go\n+added line\n")}
}

func defaultCfg() *config.Config {
	return &config.Config{
		Provider:     "claude",
		BlockOn:      "critical",
		Timeout:      10_000_000_000, // 10s
		MaxDiffBytes: 524288,
	}
}

// makePushInfo returns a synthetic pre-push stdin payload.
func makePushInfo(localSHA, remoteSHA string) string {
	return "refs/heads/main " + localSHA + " refs/heads/main " + remoteSHA + "\n"
}

func TestRun_CleanReview(t *testing.T) {
	mock := &provider.MockRunner{Result: reviewtest.CleanReview("claude")}
	var out, errOut bytes.Buffer
	stdin := strings.NewReader(makePushInfo("abc123", "0000000000000000000000000000000000000000"))
	code := Run(context.Background(), defaultCfg(), stdin, &out, &errOut, true, someDiff(), mock)
	assert.Equal(t, 0, code)
}

func TestRun_DeletePush(t *testing.T) {
	mock := &provider.MockRunner{}
	var out, errOut bytes.Buffer
	stdin := strings.NewReader(makePushInfo("0000000000000000000000000000000000000000", "abc123"))
	code := Run(context.Background(), defaultCfg(), stdin, &out, &errOut, true, someDiff(), mock)
	assert.Equal(t, 0, code)
}

func TestRun_ProviderNotFound_FailOpen(t *testing.T) {
	mock := &provider.MockRunner{Err: provider.ErrProviderNotFound}
	var out, errOut bytes.Buffer
	stdin := strings.NewReader(makePushInfo("abc123", "0000000000000000000000000000000000000000"))
	code := Run(context.Background(), defaultCfg(), stdin, &out, &errOut, true, someDiff(), mock)
	assert.Equal(t, 0, code)
	assert.Contains(t, errOut.String(), "skipping review")
}

func TestRun_MalformedResponse_FailOpen(t *testing.T) {
	mock := &provider.MockRunner{Result: reviewtest.Malformed()}
	var out, errOut bytes.Buffer
	stdin := strings.NewReader(makePushInfo("abc123", "0000000000000000000000000000000000000000"))
	code := Run(context.Background(), defaultCfg(), stdin, &out, &errOut, true, someDiff(), mock)
	assert.Equal(t, 0, code)
	assert.Contains(t, errOut.String(), "retrying once")
	assert.Contains(t, errOut.String(), "skipping")
}

func TestRun_NoStdin(t *testing.T) {
	mock := &provider.MockRunner{}
	var out, errOut bytes.Buffer
	stdin := strings.NewReader("")
	code := Run(context.Background(), defaultCfg(), stdin, &out, &errOut, true, someDiff(), mock)
	assert.Equal(t, 0, code)
}

func TestRun_MalformedResponse_RetrySucceeds(t *testing.T) {
	mock := &provider.MockRunner{
		Results: []review.ProviderResult{
			reviewtest.Malformed(),
			reviewtest.CleanReview("claude"),
		},
	}
	var out, errOut bytes.Buffer
	stdin := strings.NewReader(makePushInfo("abc123", "0000000000000000000000000000000000000000"))
	code := Run(context.Background(), defaultCfg(), stdin, &out, &errOut, true, someDiff(), mock)
	assert.Equal(t, 0, code)
	assert.Equal(t, 2, mock.CallCount)
	assert.Contains(t, errOut.String(), "retrying once")
}

// slowRunner delays completion so plain-mode stderr progress has time to run.
type slowRunner struct{}

func (slowRunner) Run(ctx context.Context, _ []byte) (review.ProviderResult, error) {
	select {
	case <-time.After(150 * time.Millisecond):
	case <-ctx.Done():
		return review.ProviderResult{}, ctx.Err()
	}
	return reviewtest.CleanReview("claude"), nil
}

func TestRun_PlainPath_NoEscOnStdoutOrStderr(t *testing.T) {
	var out, errOut bytes.Buffer
	stdin := strings.NewReader(makePushInfo("abc123", "0000000000000000000000000000000000000000"))
	code := Run(context.Background(), defaultCfg(), stdin, &out, &errOut, true, someDiff(), slowRunner{})
	require.Equal(t, 0, code)
	assert.NotContains(t, out.String(), "\x1b")
	assert.NotContains(t, errOut.String(), "\x1b")
}

func TestBuildRunner_Ollama(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, exec.Command("git", "init", dir).Run())
	cfg := &config.Config{
		Provider:     "ollama",
		BlockOn:      "critical",
		Timeout:      10 * time.Second,
		MaxDiffBytes: 1024,
		Ollama: config.OllamaConfig{
			BaseURL: "http://localhost:11434",
			Model:   "llama3",
		},
	}
	require.NoError(t, config.Validate(cfg))

	r, err := buildRunner(cfg, dir)
	require.NoError(t, err)
	_, ok := r.(*provider.OllamaRunner)
	assert.True(t, ok)
}

func TestRun_MalformedResponse_RetryAlsoFails_FailOpen(t *testing.T) {
	mock := &provider.MockRunner{
		Results: []review.ProviderResult{
			reviewtest.Malformed(),
			reviewtest.Malformed(),
		},
	}
	var out, errOut bytes.Buffer
	stdin := strings.NewReader(makePushInfo("abc123", "0000000000000000000000000000000000000000"))
	code := Run(context.Background(), defaultCfg(), stdin, &out, &errOut, true, someDiff(), mock)
	assert.Equal(t, 0, code)
	assert.Equal(t, 2, mock.CallCount)
	assert.Contains(t, errOut.String(), "skipping")
}
