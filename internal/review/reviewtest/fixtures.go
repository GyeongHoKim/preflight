// Package reviewtest provides test fixtures and helpers for review parsing tests.
// Provider envelopes match the formats documented in specs/001-preflight-ai-review.
package reviewtest

import (
	"encoding/json"

	"github.com/GyeongHoKim/preflight/internal/review"
)

// FindingSpec describes a single finding for building canonical JSON.
type FindingSpec struct {
	Severity string
	Category string
	Message  string
	Location string
}

// canonicalPayload is the JSON shape expected by review.ParseReview.
type canonicalPayload struct {
	Findings []struct {
		Severity string `json:"severity"`
		Category string `json:"category"`
		Message  string `json:"message"`
		Location string `json:"location"`
	} `json:"findings"`
	Blocking   bool    `json:"blocking"`
	Summary    string  `json:"summary"`
	Verdict    string  `json:"verdict"`
	Confidence float64 `json:"confidence"`
}

// CanonicalJSON builds canonical review JSON including verdict and confidence.
// Use with provider envelopes (ClaudeEnvelope, CodexEnvelope, etc.).
func CanonicalJSON(summary string, blocking bool, findings []FindingSpec, verdict string, confidence float64) []byte {
	payload := canonicalPayload{Summary: summary, Blocking: blocking, Verdict: verdict, Confidence: confidence}
	for _, f := range findings {
		payload.Findings = append(payload.Findings, struct {
			Severity string `json:"severity"`
			Category string `json:"category"`
			Message  string `json:"message"`
			Location string `json:"location"`
		}{f.Severity, f.Category, f.Message, f.Location})
	}
	out, _ := json.Marshal(payload)
	return out
}

// ClaudeEnvelope wraps inner JSON in the claude CLI envelope (result field).
func ClaudeEnvelope(inner []byte) review.ProviderResult {
	b, _ := json.Marshal(map[string]string{"type": "result", "result": string(inner)})
	return review.ProviderResult{Stdout: b}
}

// CodexEnvelope wraps inner JSON in a codex-style envelope with the given top-level field.
// Codex schema is undocumented; common field names are "output", "result", "response".
func CodexEnvelope(inner []byte, field string) review.ProviderResult {
	b, _ := json.Marshal(map[string]string{field: string(inner)})
	return review.ProviderResult{Stdout: b}
}

// Malformed returns a ProviderResult with non-JSON stdout (for fail-open tests).
func Malformed() review.ProviderResult {
	return review.ProviderResult{Stdout: []byte("not json")}
}

// Empty returns an empty ProviderResult.
func Empty() review.ProviderResult {
	return review.ProviderResult{}
}

// CleanReview returns a ProviderResult that parses to a clean review (no findings, non-blocking)
// for the given provider. Use in hook tests when a successful parse is needed.
func CleanReview(provider string) review.ProviderResult {
	inner := CanonicalJSON("all good", false, nil, review.VerdictCorrect, 0.9)
	switch provider {
	case "codex", "unknown":
		return CodexEnvelope(inner, "result")
	case "ollama":
		return review.ProviderResult{Stdout: inner}
	default:
		return ClaudeEnvelope(inner)
	}
}
