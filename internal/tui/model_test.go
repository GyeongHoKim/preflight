package tui

import (
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
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

// --- Pure unit tests: no event loop needed ---

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
	assert.Contains(t, view, "mock")
	assert.Contains(t, view, "unchecked error")
}

func TestReviewModel_View_ShowsPromptWhenBlocking(t *testing.T) {
	m := NewReviewModel(makeReview(true))
	view := m.View()
	assert.True(t, strings.Contains(view, "[y/n]") || strings.Contains(view, "Push blocked"))
}

// --- teatest: key-press tests through the real event loop ---

// TestReviewModel_BlockingPrompt_Y sends the 'y' key through the actual
// tea.Program and asserts the final model records "push_anyway".
func TestReviewModel_BlockingPrompt_Y(t *testing.T) {
	tm := teatest.NewTestModel(
		t,
		NewReviewModel(makeReview(true)),
		teatest.WithInitialTermSize(80, 24),
	)

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})

	final, ok := tm.FinalModel(t, teatest.WithFinalTimeout(time.Second)).(ReviewModel)
	require.True(t, ok, "final model must be ReviewModel")
	assert.Equal(t, "push_anyway", final.Choice())
}

// TestReviewModel_BlockingPrompt_N sends the 'n' key and asserts "cancel".
func TestReviewModel_BlockingPrompt_N(t *testing.T) {
	tm := teatest.NewTestModel(
		t,
		NewReviewModel(makeReview(true)),
		teatest.WithInitialTermSize(80, 24),
	)

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})

	final, ok := tm.FinalModel(t, teatest.WithFinalTimeout(time.Second)).(ReviewModel)
	require.True(t, ok, "final model must be ReviewModel")
	assert.Equal(t, "cancel", final.Choice())
}

// TestReviewModel_BlockingPrompt_Q sends 'q' and asserts "cancel".
func TestReviewModel_BlockingPrompt_Q(t *testing.T) {
	tm := teatest.NewTestModel(
		t,
		NewReviewModel(makeReview(true)),
		teatest.WithInitialTermSize(80, 24),
	)

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})

	final, ok := tm.FinalModel(t, teatest.WithFinalTimeout(time.Second)).(ReviewModel)
	require.True(t, ok, "final model must be ReviewModel")
	assert.Equal(t, "cancel", final.Choice())
}

// TestReviewModel_NonBlocking_AnyKeyQuits verifies a non-blocking review quits
// on any keypress and leaves choice empty.
func TestReviewModel_NonBlocking_AnyKeyQuits(t *testing.T) {
	tm := teatest.NewTestModel(
		t,
		NewReviewModel(makeReview(false)),
		teatest.WithInitialTermSize(80, 24),
	)

	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	final, ok := tm.FinalModel(t, teatest.WithFinalTimeout(time.Second)).(ReviewModel)
	require.True(t, ok, "final model must be ReviewModel")
	assert.Equal(t, "", final.Choice())
}

// TestReviewModel_Output_ContainsProviderName uses WaitFor to assert the
// rendered output contains the provider name before the program exits.
func TestReviewModel_Output_ContainsProviderName(t *testing.T) {
	tm := teatest.NewTestModel(
		t,
		NewReviewModel(makeReview(false)),
		teatest.WithInitialTermSize(80, 24),
	)

	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return strings.Contains(string(out), "mock")
	}, teatest.WithDuration(time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})
	tm.WaitFinished(t, teatest.WithFinalTimeout(time.Second))
}
