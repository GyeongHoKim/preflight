package review_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/GyeongHoKim/preflight/internal/review"
	"github.com/GyeongHoKim/preflight/internal/review/reviewtest"
)

func TestParseReview_DirectCanonical(t *testing.T) {
	raw := review.ProviderResult{Stdout: reviewtest.CanonicalJSON("direct summary", false, nil)}
	rev, err := review.ParseReview("claude", raw)
	require.NoError(t, err)
	require.NotNil(t, rev)
	assert.Equal(t, "direct summary", rev.Summary)
}

func TestParseReview_ValidReview(t *testing.T) {
	inner := reviewtest.CanonicalJSON("critical issue found", true, []reviewtest.FindingSpec{
		{Severity: "critical", Category: "security", Message: "hardcoded secret", Location: "main.go:10"},
	})
	raw := reviewtest.ClaudeEnvelope(inner)
	rev, err := review.ParseReview("claude", raw)
	require.NoError(t, err)
	require.NotNil(t, rev)
	assert.True(t, rev.Blocking)
	assert.Equal(t, "critical issue found", rev.Summary)
	require.Len(t, rev.Findings, 1)
	assert.Equal(t, "critical", rev.Findings[0].Severity)
}

func TestParseReview_EmptyStdout(t *testing.T) {
	rev, err := review.ParseReview("claude", reviewtest.Empty())
	require.NoError(t, err)
	assert.Nil(t, rev)
}

func TestParseReview_MalformedJSON(t *testing.T) {
	rev, err := review.ParseReview("claude", reviewtest.Malformed())
	require.NoError(t, err)
	assert.Nil(t, rev)
}

func TestParseReview_MissingFieldsNormalized(t *testing.T) {
	inner := reviewtest.CanonicalJSON("ok", false, []reviewtest.FindingSpec{
		{Severity: "unknown_val", Message: "some issue"},
	})
	raw := reviewtest.ClaudeEnvelope(inner)
	rev, err := review.ParseReview("claude", raw)
	require.NoError(t, err)
	require.NotNil(t, rev)
	assert.Equal(t, review.SeverityInfo, rev.Findings[0].Severity)
}

func TestParseReview_EnvelopeResult(t *testing.T) {
	inner := reviewtest.CanonicalJSON("no issues", false, nil)
	raw := reviewtest.ClaudeEnvelope(inner)
	rev, err := review.ParseReview("claude", raw)
	require.NoError(t, err)
	require.NotNil(t, rev)
	assert.Equal(t, "no issues", rev.Summary)
}

func TestParseReview_GeminiEnvelopeUsesResponse(t *testing.T) {
	inner := reviewtest.CanonicalJSON("gemini review", false, nil)
	raw := reviewtest.GeminiEnvelope(inner)
	rev, err := review.ParseReview("gemini", raw)
	require.NoError(t, err)
	require.NotNil(t, rev)
	assert.Equal(t, "gemini review", rev.Summary)
}

func TestParseReview_QwenEnvelopeUsesResult(t *testing.T) {
	inner := reviewtest.CanonicalJSON("qwen review", false, nil)
	raw := reviewtest.QwenEnvelope(inner)
	rev, err := review.ParseReview("qwen", raw)
	require.NoError(t, err)
	require.NotNil(t, rev)
	assert.Equal(t, "qwen review", rev.Summary)
}

func TestParseReview_CodexEnvelopeTriesOutputThenResult(t *testing.T) {
	inner := reviewtest.CanonicalJSON("codex review", false, nil)
	raw := reviewtest.CodexEnvelope(inner, "output")
	rev, err := review.ParseReview("codex", raw)
	require.NoError(t, err)
	require.NotNil(t, rev)
	assert.Equal(t, "codex review", rev.Summary)
}
