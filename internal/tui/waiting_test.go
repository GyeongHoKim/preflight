package tui

import (
	"io"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/GyeongHoKim/preflight/internal/anim"
	"github.com/GyeongHoKim/preflight/internal/review"
	"github.com/GyeongHoKim/preflight/internal/review/reviewtest"
)

func TestWaitingModel_InterruptMsg_SetsFlag(t *testing.T) {
	m := NewWaitingModel(io.Discard, func() (*review.Review, error) {
		return nil, nil
	})
	upd, cmd := m.Update(tea.InterruptMsg{})
	require.NotNil(t, cmd)
	wm := upd.(*WaitingModel)
	assert.True(t, wm.UserInterrupted())
}

func TestWaitingModel_RenderFrameHookPanic_DegradesGracefully(t *testing.T) {
	parsed, err := review.ParseReview("claude", reviewtest.CleanReview("claude"))
	require.NoError(t, err)
	m := NewWaitingModel(io.Discard, func() (*review.Review, error) {
		return parsed, nil
	})
	m.RenderFrameHook = func(anim.Frame, RenderOptions) string {
		panic("injected")
	}
	fr, err := m.compute(anim.RenderOpts{Width: 12, Height: 5, Tick: 0, Seed: 1})
	require.NoError(t, err)
	s := m.renderFrame(fr)
	assert.Equal(t, "", s)
	v := m.View()
	assert.Contains(t, v.Content, "waiting for AI review")
}

func TestWaitingModel_ProviderDone_TransitionsToReview(t *testing.T) {
	rev, err := review.ParseReview("claude", reviewtest.CleanReview("claude"))
	require.NoError(t, err)
	m := NewWaitingModel(io.Discard, func() (*review.Review, error) {
		return rev, nil
	})
	upd, _ := m.Update(providerDoneMsg{review: rev, err: nil})
	wm := upd.(*WaitingModel)
	assert.Equal(t, PhaseReview, wm.phase)
}
