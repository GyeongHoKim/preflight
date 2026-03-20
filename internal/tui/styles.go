// Package tui provides terminal UI components for preflight.
package tui

import lipgloss "charm.land/lipgloss/v2"

const maxWidth = 80

var (
	// styleCritical styles critical-severity findings.
	styleCritical = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("9"))
	// styleWarning styles warning-severity findings.
	styleWarning = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	// styleInfo styles info-severity findings.
	styleInfo = lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Faint(true)
	// styleHeader styles the review header line.
	styleHeader = lipgloss.NewStyle().Bold(true)
	// styleFooter styles the review footer line.
	styleFooter = lipgloss.NewStyle().Faint(true)
	// stylePrompt styles the blocking prompt.
	stylePrompt = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("9"))
)
