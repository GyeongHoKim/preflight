package hook

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/GyeongHoKim/preflight/internal/config"
	"github.com/GyeongHoKim/preflight/internal/diff"
	"github.com/GyeongHoKim/preflight/internal/provider"
	"github.com/GyeongHoKim/preflight/internal/review"
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

func cleanResultJSON() review.ProviderResult {
	return review.ProviderResult{
		Stdout: []byte(`{"findings":[],"blocking":false,"summary":"all good"}`),
	}
}

func TestRun_CleanReview(t *testing.T) {
	mock := &provider.MockRunner{Result: cleanResultJSON()}
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
	mock := &provider.MockRunner{Result: review.ProviderResult{Stdout: []byte("not json")}}
	var out, errOut bytes.Buffer
	stdin := strings.NewReader(makePushInfo("abc123", "0000000000000000000000000000000000000000"))
	code := Run(context.Background(), defaultCfg(), stdin, &out, &errOut, true, someDiff(), mock)
	assert.Equal(t, 0, code)
	assert.Contains(t, errOut.String(), "skipping")
}

func TestRun_NoStdin(t *testing.T) {
	mock := &provider.MockRunner{}
	var out, errOut bytes.Buffer
	stdin := strings.NewReader("")
	code := Run(context.Background(), defaultCfg(), stdin, &out, &errOut, true, someDiff(), mock)
	assert.Equal(t, 0, code)
}
