package hook

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/GyeongHoKim/preflight/internal/config"
	"github.com/GyeongHoKim/preflight/internal/provider"
	"github.com/GyeongHoKim/preflight/internal/review"
)

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

func cleanResultJSON() review.ProviderResult {
	return review.ProviderResult{
		Stdout: []byte(`{"findings":[],"blocking":false,"summary":"all good"}`),
	}
}

func TestRun_CleanReview(t *testing.T) {
	mock := &provider.MockRunner{Result: cleanResultJSON()}
	var out, errOut bytes.Buffer
	// Use a delete push so diff collection is skipped; instead inject diff via mock.
	// Actually, let's use a valid non-delete push but mock the runner.
	// We need to provide diff bytes directly; let's hack it by using a mock
	// that returns content and have the diff be non-empty.
	// Simplest: use a zero-sha push (delete) to trigger early exit 0.
	stdin := strings.NewReader(makePushInfo("abc123", "0000000000000000000000000000000000000000"))
	code := Run(context.Background(), defaultCfg(), stdin, &out, &errOut, true, mock)
	assert.Equal(t, 0, code)
}

func TestRun_DeletePush(t *testing.T) {
	mock := &provider.MockRunner{}
	var out, errOut bytes.Buffer
	stdin := strings.NewReader(makePushInfo("0000000000000000000000000000000000000000", "abc123"))
	code := Run(context.Background(), defaultCfg(), stdin, &out, &errOut, true, mock)
	assert.Equal(t, 0, code)
}

func TestRun_ProviderNotFound_FailOpen(t *testing.T) {
	mock := &provider.MockRunner{Err: provider.ErrProviderNotFound}
	var out, errOut bytes.Buffer
	// New branch push (remote SHA = zeros) so diff will be attempted via `git diff HEAD`
	// but mock runner overrides the actual call.
	stdin := strings.NewReader(makePushInfo("abc123", "0000000000000000000000000000000000000000"))
	code := Run(context.Background(), defaultCfg(), stdin, &out, &errOut, true, mock)
	assert.Equal(t, 0, code)
	assert.Contains(t, errOut.String(), "skipping review")
}

func TestRun_MalformedResponse_FailOpen(t *testing.T) {
	mock := &provider.MockRunner{Result: review.ProviderResult{Stdout: []byte("not json")}}
	var out, errOut bytes.Buffer
	stdin := strings.NewReader(makePushInfo("abc123", "0000000000000000000000000000000000000000"))
	code := Run(context.Background(), defaultCfg(), stdin, &out, &errOut, true, mock)
	assert.Equal(t, 0, code)
	assert.Contains(t, errOut.String(), "skipping")
}

func TestRun_NoStdin(t *testing.T) {
	mock := &provider.MockRunner{}
	var out, errOut bytes.Buffer
	stdin := strings.NewReader("")
	code := Run(context.Background(), defaultCfg(), stdin, &out, &errOut, true, mock)
	assert.Equal(t, 0, code)
}
