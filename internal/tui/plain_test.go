package tui

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/GyeongHoKim/preflight/internal/review"
)

func TestPlainRender_Clean(t *testing.T) {
	var buf bytes.Buffer
	r := &review.Review{
		Findings: nil,
		Blocking: false,
		Summary:  "all good",
		Provider: "claude",
	}
	PlainRender(&buf, r, "main", 3)
	out := buf.String()
	assert.Contains(t, out, "3 commit(s) on main")
	assert.Contains(t, out, "no issues found")
}

func TestPlainRender_Blocked(t *testing.T) {
	var buf bytes.Buffer
	r := &review.Review{
		Findings: []review.Finding{
			{Severity: "critical", Category: "security", Message: "hardcoded secret", Location: "auth.go:42"},
			{Severity: "warning", Category: "logic", Message: "unchecked error"},
		},
		Blocking: true,
		Summary:  "issues found",
		Provider: "claude",
	}
	PlainRender(&buf, r, "feature", 1)
	out := buf.String()
	assert.Contains(t, out, "[CRITICAL]")
	assert.Contains(t, out, "hardcoded secret")
	assert.Contains(t, out, "push blocked")
	assert.Contains(t, out, "--no-verify")
}

func TestPlainRender_Empty(t *testing.T) {
	var buf bytes.Buffer
	PlainRender(&buf, nil, "main", 0)
	out := buf.String()
	assert.True(t, strings.Contains(out, "no issues found") || strings.Contains(out, "push allowed"))
}
