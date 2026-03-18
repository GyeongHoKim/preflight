package review

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeRaw(t *testing.T, v interface{}) ProviderResult {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	return ProviderResult{Stdout: b}
}

func TestParseReview_ValidReview(t *testing.T) {
	raw := makeRaw(t, map[string]interface{}{
		"findings": []map[string]interface{}{
			{"severity": "critical", "category": "security", "message": "hardcoded secret", "location": "main.go:10"},
		},
		"blocking": true,
		"summary":  "critical issue found",
	})
	rev, err := ParseReview("claude", raw)
	require.NoError(t, err)
	require.NotNil(t, rev)
	assert.True(t, rev.Blocking)
	assert.Equal(t, "critical issue found", rev.Summary)
	require.Len(t, rev.Findings, 1)
	assert.Equal(t, "critical", rev.Findings[0].Severity)
}

func TestParseReview_EmptyStdout(t *testing.T) {
	rev, err := ParseReview("claude", ProviderResult{})
	require.NoError(t, err)
	assert.Nil(t, rev)
}

func TestParseReview_MalformedJSON(t *testing.T) {
	raw := ProviderResult{Stdout: []byte("not json")}
	rev, err := ParseReview("claude", raw)
	require.NoError(t, err)
	assert.Nil(t, rev)
}

func TestParseReview_MissingFieldsNormalized(t *testing.T) {
	raw := makeRaw(t, map[string]interface{}{
		"findings": []map[string]interface{}{
			{"severity": "unknown_val", "message": "some issue"},
		},
		"blocking": false,
		"summary":  "ok",
	})
	rev, err := ParseReview("claude", raw)
	require.NoError(t, err)
	require.NotNil(t, rev)
	assert.Equal(t, SeverityInfo, rev.Findings[0].Severity)
}

func TestParseReview_EnvelopeResult(t *testing.T) {
	inner, _ := json.Marshal(map[string]interface{}{
		"findings": []interface{}{},
		"blocking": false,
		"summary":  "no issues",
	})
	raw := makeRaw(t, map[string]interface{}{
		"type":   "result",
		"result": string(inner),
	})
	rev, err := ParseReview("claude", raw)
	require.NoError(t, err)
	require.NotNil(t, rev)
	assert.Equal(t, "no issues", rev.Summary)
}
