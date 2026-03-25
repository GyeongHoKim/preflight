package review_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/GyeongHoKim/preflight/internal/review"
	"github.com/GyeongHoKim/preflight/internal/review/reviewtest"
)

func TestParseReview_DirectCanonical(t *testing.T) {
	raw := review.ProviderResult{Stdout: reviewtest.CanonicalJSON("direct summary", false, nil, review.VerdictCorrect, 0.9)}
	rev, err := review.ParseReview("claude", raw)
	require.NoError(t, err)
	require.NotNil(t, rev)
	assert.Equal(t, "direct summary", rev.Summary)
}

func TestParseReview_OllamaDirect(t *testing.T) {
	raw := review.ProviderResult{Stdout: reviewtest.CanonicalJSON("ollama ok", false, nil, review.VerdictCorrect, 0.85)}
	rev, err := review.ParseReview("ollama", raw)
	require.NoError(t, err)
	require.NotNil(t, rev)
	assert.Equal(t, "ollama", rev.Provider)
	assert.Equal(t, "ollama ok", rev.Summary)
}

func TestParseReview_ValidReview(t *testing.T) {
	inner := reviewtest.CanonicalJSON("critical issue found", true, []reviewtest.FindingSpec{
		{Severity: "critical", Category: "security", Message: "hardcoded secret", Location: "main.go:10"},
	}, review.VerdictIncorrect, 0.95)
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
	assert.True(t, errors.Is(err, review.ErrMalformedResponse))
	assert.Nil(t, rev)
}

func TestParseReview_MissingFieldsNormalized(t *testing.T) {
	inner := reviewtest.CanonicalJSON("ok", false, []reviewtest.FindingSpec{
		{Severity: "unknown_val", Message: "some issue"},
	}, review.VerdictCorrect, 0.9)
	raw := reviewtest.ClaudeEnvelope(inner)
	rev, err := review.ParseReview("claude", raw)
	require.NoError(t, err)
	require.NotNil(t, rev)
	assert.Equal(t, review.SeverityInfo, rev.Findings[0].Severity)
}

func TestParseReview_EnvelopeResult(t *testing.T) {
	inner := reviewtest.CanonicalJSON("no issues", false, nil, review.VerdictCorrect, 0.9)
	raw := reviewtest.ClaudeEnvelope(inner)
	rev, err := review.ParseReview("claude", raw)
	require.NoError(t, err)
	require.NotNil(t, rev)
	assert.Equal(t, "no issues", rev.Summary)
}

func TestParseReview_CodexEnvelopeTriesOutputThenResult(t *testing.T) {
	inner := reviewtest.CanonicalJSON("codex review", false, nil, review.VerdictCorrect, 0.9)
	raw := reviewtest.CodexEnvelope(inner, "output")
	rev, err := review.ParseReview("codex", raw)
	require.NoError(t, err)
	require.NotNil(t, rev)
	assert.Equal(t, "codex review", rev.Summary)
}

func TestParseReview_VerdictAndConfidencePopulated(t *testing.T) {
	inner := reviewtest.CanonicalJSON("looks good", false, nil, review.VerdictCorrect, 0.85)
	raw := reviewtest.ClaudeEnvelope(inner)
	rev, err := review.ParseReview("claude", raw)
	require.NoError(t, err)
	require.NotNil(t, rev)
	assert.Equal(t, review.VerdictCorrect, rev.Verdict)
	assert.InDelta(t, 0.85, rev.Confidence, 1e-9)
}

func TestParseReview_InvalidVerdict_ReturnsErrMalformedResponse(t *testing.T) {
	inner := reviewtest.CanonicalJSON("summary", false, nil, "totally wrong verdict", 0.5)
	raw := reviewtest.ClaudeEnvelope(inner)
	rev, err := review.ParseReview("claude", raw)
	assert.True(t, errors.Is(err, review.ErrMalformedResponse))
	assert.Nil(t, rev)
}

func TestParseReview_ConfidenceOutOfRange_High(t *testing.T) {
	inner := reviewtest.CanonicalJSON("summary", false, nil, review.VerdictCorrect, 1.5)
	raw := reviewtest.ClaudeEnvelope(inner)
	rev, err := review.ParseReview("claude", raw)
	assert.True(t, errors.Is(err, review.ErrMalformedResponse))
	assert.Nil(t, rev)
}

func TestParseReview_ConfidenceOutOfRange_Negative(t *testing.T) {
	inner := reviewtest.CanonicalJSON("summary", false, nil, review.VerdictCorrect, -0.1)
	raw := reviewtest.ClaudeEnvelope(inner)
	rev, err := review.ParseReview("claude", raw)
	assert.True(t, errors.Is(err, review.ErrMalformedResponse))
	assert.Nil(t, rev)
}

func TestParseReview_EmptyStdout_StillNilNil(t *testing.T) {
	rev, err := review.ParseReview("claude", reviewtest.Empty())
	require.NoError(t, err)
	assert.Nil(t, rev)
}

func TestParseReview_EmptySummaryNoFindings_StillNilNil(t *testing.T) {
	// Even with valid verdict/confidence, empty summary+findings → nil, nil.
	inner := reviewtest.CanonicalJSON("", false, nil, review.VerdictCorrect, 0.9)
	raw := reviewtest.ClaudeEnvelope(inner)
	rev, err := review.ParseReview("claude", raw)
	require.NoError(t, err)
	assert.Nil(t, rev)
}
