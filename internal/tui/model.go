package tui

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-isatty"

	"github.com/GyeongHoKim/preflight/internal/review"
)

// ReviewModel is the Bubbletea model for displaying review findings.
type ReviewModel struct {
	review *review.Review
	width  int
	choice string // "push_anyway" | "cancel" | ""
	done   bool
}

// NewReviewModel creates a ReviewModel for the given review.
func NewReviewModel(r *review.Review) ReviewModel {
	return ReviewModel{review: r}
}

// Choice returns the user's decision: "push_anyway", "cancel", or "" (not blocking).
func (m ReviewModel) Choice() string {
	return m.choice
}

// Init implements tea.Model.
func (m ReviewModel) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (m ReviewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		return m, nil
	case tea.KeyMsg:
		if m.review != nil && m.review.Blocking {
			switch msg.String() {
			case "y", "Y":
				m.choice = "push_anyway"
				m.done = true
				return m, tea.Quit
			case "n", "N", "q", "ctrl+c":
				m.choice = "cancel"
				m.done = true
				return m, tea.Quit
			}
		} else {
			// Non-blocking: any key or auto-exit.
			m.done = true
			return m, tea.Quit
		}
	}
	return m, nil
}

// View implements tea.Model.
func (m ReviewModel) View() string {
	if m.review == nil {
		return styleFooter.Render("preflight: no review available") + "\n"
	}

	w := m.width
	if w == 0 || w > maxWidth {
		w = maxWidth
	}

	var b strings.Builder

	// Header.
	header := fmt.Sprintf("preflight review — %s", m.review.Provider)
	b.WriteString(styleHeader.Width(w).Render(header))
	b.WriteString("\n\n")

	// Findings.
	if len(m.review.Findings) == 0 {
		b.WriteString(styleInfo.Render("  No issues found."))
		b.WriteString("\n")
	} else {
		for _, f := range m.review.Findings {
			sev := strings.ToUpper(f.Severity)
			loc := f.Location
			if loc != "" {
				loc = " — " + loc
			}
			label := fmt.Sprintf("[%s] %s%s", sev, f.Category, loc)
			msg := "  " + f.Message

			switch f.Severity {
			case review.SeverityCritical:
				b.WriteString(styleCritical.Render(label))
			case review.SeverityWarning:
				b.WriteString(styleWarning.Render(label))
			default:
				b.WriteString(styleInfo.Render(label))
			}
			b.WriteString("\n")
			b.WriteString(lipgloss.NewStyle().Width(w).Render(msg))
			b.WriteString("\n\n")
		}
	}

	// Summary.
	if m.review.Summary != "" {
		b.WriteString(styleFooter.Render("Summary: " + m.review.Summary))
		b.WriteString("\n\n")
	}

	// Footer / prompt.
	if m.review.Blocking && !m.done {
		b.WriteString(stylePrompt.Render("Push blocked. Push anyway? [y/n]: "))
	} else if !m.review.Blocking {
		b.WriteString(styleFooter.Render("No blocking issues — push allowed."))
		b.WriteString("\n")
	}

	return b.String()
}

// IsTTY reports whether stdout is connected to a terminal.
func IsTTY() bool {
	return isatty.IsTerminal(os.Stdout.Fd()) || isatty.IsCygwinTerminal(os.Stdout.Fd())
}
