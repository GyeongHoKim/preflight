package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/GyeongHoKim/preflight/internal/review"
)

func makeReview(blocking bool, findings ...review.Finding) *review.Review {
	return &review.Review{
		Findings: findings,
		Blocking: blocking,
		Summary:  "test summary",
		Provider: "mock",
	}
}

func keyPressRune(r rune) tea.KeyPressMsg {
	return tea.KeyPressMsg(tea.Key{Text: string(r), Code: r})
}

func TestReviewModel_WindowSizeMsg(t *testing.T) {
	m := NewReviewModel(makeReview(false))
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	rm := updated.(ReviewModel)
	assert.Equal(t, 100, rm.width)
}

func TestReviewModel_View_ContainsProviderAndFinding(t *testing.T) {
	r := makeReview(false, review.Finding{
		Severity: "warning",
		Category: "logic",
		Message:  "unchecked error",
		Location: "api/handler.go:42",
	})
	m := NewReviewModel(r)
	view := m.View()
	assert.Contains(t, view.Content, "mock")
	assert.Contains(t, view.Content, "unchecked error")
}

func TestReviewModel_View_ShowsPromptWhenBlocking(t *testing.T) {
	m := NewReviewModel(makeReview(true))
	view := m.View()
	assert.True(t, strings.Contains(view.Content, "[y/n]") || strings.Contains(view.Content, "Push blocked"))
}

func TestReviewModel_BlockingPrompt_Y(t *testing.T) {
	m := NewReviewModel(makeReview(true))
	updated, cmd := m.Update(keyPressRune('y'))
	require.NotNil(t, cmd)
	rm := updated.(ReviewModel)
	assert.Equal(t, "push_anyway", rm.Choice())
}

func TestReviewModel_BlockingPrompt_N(t *testing.T) {
	m := NewReviewModel(makeReview(true))
	updated, cmd := m.Update(keyPressRune('n'))
	require.NotNil(t, cmd)
	rm := updated.(ReviewModel)
	assert.Equal(t, "cancel", rm.Choice())
}

func TestReviewModel_BlockingPrompt_Q(t *testing.T) {
	m := NewReviewModel(makeReview(true))
	updated, cmd := m.Update(keyPressRune('q'))
	require.NotNil(t, cmd)
	rm := updated.(ReviewModel)
	assert.Equal(t, "cancel", rm.Choice())
}

func TestReviewModel_NonBlocking_AnyKeyQuits(t *testing.T) {
	m := NewReviewModel(makeReview(false))
	updated, cmd := m.Update(tea.KeyPressMsg(tea.Key{Code: tea.KeyEnter}))
	require.NotNil(t, cmd)
	rm := updated.(ReviewModel)
	assert.Equal(t, "", rm.Choice())
}
