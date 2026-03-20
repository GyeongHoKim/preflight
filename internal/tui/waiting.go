package tui

import (
	"fmt"
	"io"
	"os"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/GyeongHoKim/preflight/internal/anim"
	"github.com/GyeongHoKim/preflight/internal/review"
)

// WaitingPhase distinguishes loading vs review in the TUI.
type WaitingPhase int

const (
	// PhaseLoading shows the animated spinner while the provider runs.
	PhaseLoading WaitingPhase = iota
	// PhaseReview shows the review UI.
	PhaseReview
)

type tickMsg struct{}

type providerDoneMsg struct {
	review *review.Review
	err    error
}

// WaitingModel shows a liquid-blob spinner until the provider fetch completes, then the review UI.
type WaitingModel struct {
	stderr io.Writer

	fetch func() (*review.Review, error)

	phase WaitingPhase
	tick  int
	seed  uint64
	w, h  int

	animCfg anim.LiquidBlobConfig

	reviewModel ReviewModel

	providerReview *review.Review
	providerErr    error
	quitEarly      bool

	// RenderFrameHook, if non-nil, replaces RenderFrame (tests / failure injection).
	RenderFrameHook func(anim.Frame, RenderOptions) string

	// ComputeFrameHook, if non-nil, replaces anim.ComputeFrame (tests).
	ComputeFrameHook func(anim.LiquidBlobConfig, anim.RenderOpts) (anim.Frame, error)

	userInterrupted bool
}

// NewWaitingModel builds a model that runs fetch asynchronously via tea.Cmd.
// fetch must return the same semantics as hook.attempt/retry (review + error).
func NewWaitingModel(stderr io.Writer, fetch func() (*review.Review, error)) *WaitingModel {
	return &WaitingModel{
		stderr:  stderr,
		fetch:   fetch,
		phase:   PhaseLoading,
		seed:    uint64(time.Now().UnixNano()),
		animCfg: anim.DefaultLiquidBlobConfig(),
		w:       56,
		h:       10,
	}
}

// ProviderResult returns the last provider outcome after a quit-before-review exit.
func (m *WaitingModel) ProviderResult() (*review.Review, error) {
	return m.providerReview, m.providerErr
}

// QuitEarly reports whether the program exited before the review phase (provider error/nil review).
func (m *WaitingModel) QuitEarly() bool {
	return m.quitEarly
}

// UserInterrupted reports whether the user hit Ctrl+C / SIGINT.
func (m *WaitingModel) UserInterrupted() bool {
	return m.userInterrupted
}

// FinalReviewModel returns the review model after a completed interactive review session.
func (m *WaitingModel) FinalReviewModel() ReviewModel {
	return m.reviewModel
}

// Init implements tea.Model.
func (m *WaitingModel) Init() tea.Cmd {
	return tea.Batch(
		tea.Tick(time.Second/22, func(time.Time) tea.Msg { return tickMsg{} }),
		m.runProviderCmd,
	)
}

func (m *WaitingModel) runProviderCmd() tea.Msg {
	rev, err := m.fetch()
	return providerDoneMsg{review: rev, err: err}
}

func (m *WaitingModel) compute(opts anim.RenderOpts) (anim.Frame, error) {
	if m.ComputeFrameHook != nil {
		return m.ComputeFrameHook(m.animCfg, opts)
	}
	return anim.ComputeFrame(m.animCfg, opts)
}

func (m *WaitingModel) renderFrame(fr anim.Frame) (s string) {
	noColor := os.Getenv("NO_COLOR") != ""
	opts := RenderOptions{DisableANSI: false, DisableColor: noColor}
	defer func() {
		if r := recover(); r != nil {
			s = ""
		}
	}()
	if m.RenderFrameHook != nil {
		return m.RenderFrameHook(fr, opts)
	}
	return RenderFrame(fr, opts)
}

// Update implements tea.Model.
func (m *WaitingModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.InterruptMsg:
		m.userInterrupted = true
		return m, tea.Quit

	case tea.WindowSizeMsg:
		m.w = msg.Width
		if m.w < 12 {
			m.w = 12
		}
		if m.w > maxWidth {
			m.w = maxWidth
		}
		m.h = msg.Height - 4
		if m.h < 4 {
			m.h = 4
		}
		if m.h > 30 {
			m.h = 30
		}
		return m, nil

	case tickMsg:
		if m.phase != PhaseLoading {
			return m, nil
		}
		m.tick++
		return m, nil

	case providerDoneMsg:
		m.providerReview = msg.review
		m.providerErr = msg.err
		if msg.err != nil {
			m.quitEarly = true
			return m, tea.Quit
		}
		if msg.review == nil {
			m.quitEarly = true
			return m, tea.Quit
		}
		m.phase = PhaseReview
		m.reviewModel = NewReviewModel(msg.review)
		return m, nil

	case tea.KeyPressMsg:
		if m.phase == PhaseReview {
			upd, cmd := m.reviewModel.Update(msg)
			m.reviewModel = upd.(ReviewModel)
			return m, cmd
		}
		if msg.String() == "ctrl+c" {
			m.userInterrupted = true
			return m, tea.Quit
		}
	}

	return m, nil
}

// View implements tea.Model.
func (m *WaitingModel) View() tea.View {
	if m.phase == PhaseReview {
		return m.reviewModel.View()
	}
	fr, err := m.compute(anim.RenderOpts{Width: m.w, Height: m.h, Tick: m.tick, Seed: m.seed})
	if err != nil {
		if m.stderr != nil {
			_, _ = fmt.Fprintf(m.stderr, "preflight: spinner frame unavailable (%v)\n", err)
		}
		return tea.NewView("preflight: waiting for AI review...\n")
	}
	s := m.renderFrame(fr)
	if s == "" {
		return tea.NewView("preflight: waiting for AI review...\n")
	}
	header := "preflight — analyzing staged changes\n\n"
	return tea.NewView(header + s)
}
