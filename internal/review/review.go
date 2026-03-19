// Package review defines the types returned by AI provider adapters.
package review

// Severity levels for findings.
const (
	SeverityCritical = "critical"
	SeverityWarning  = "warning"
	SeverityInfo     = "info"
)

// Verdict constants for AI correctness assessment.
const (
	VerdictCorrect   = "patch is correct"
	VerdictIncorrect = "patch is incorrect"
)

// Finding is a single issue identified in the diff by an AI provider.
type Finding struct {
	Severity string `json:"severity"`
	Category string `json:"category,omitempty"`
	Message  string `json:"message"`
	Location string `json:"location,omitempty"`
}

// Review is the complete, normalised output of a single AI review session.
type Review struct {
	Findings   []Finding
	Blocking   bool
	Summary    string
	Verdict    string  // VerdictCorrect or VerdictIncorrect
	Confidence float64 // [0.0, 1.0]
	Provider   string
	DurationMS int64
}

// ProviderResult holds the raw output from an AI CLI subprocess invocation.
type ProviderResult struct {
	Stdout   []byte
	Stderr   []byte
	ExitCode int
	Duration int64 // milliseconds
}

// SeverityRank returns a numeric rank for severity comparison.
// Higher rank means more severe. Unknown values return 0.
func SeverityRank(s string) int {
	switch s {
	case SeverityCritical:
		return 3
	case SeverityWarning:
		return 2
	case SeverityInfo:
		return 1
	default:
		return 0
	}
}
