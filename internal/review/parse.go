package review

import (
	"encoding/json"
	"errors"
	"fmt"
)

// ErrMalformedResponse is returned when the AI output is syntactically or
// structurally invalid. Callers may retry once before failing open.
var ErrMalformedResponse = errors.New("review: malformed response")

// rawReview is the JSON structure returned by AI providers.
type rawReview struct {
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

// extractEnvelopePayload returns the inner review payload from a provider's JSON envelope.
// Per provider contract: claude uses "result".
// Codex and "unknown" (auto) have no documented envelope; we try common field names.
func extractEnvelopePayload(providerName string, data []byte) []byte {
	var outer map[string]json.RawMessage
	if err := json.Unmarshal(data, &outer); err != nil {
		return nil
	}

	var fields []string
	switch providerName {
	case "claude":
		// https://code.claude.com/docs/en/cli-reference
		fields = []string{"result"}
	case "codex", "unknown":
		// codex exec --json schema not officially documented (contract: try common fields).
		fields = []string{"result", "response", "output", "content"}
	default:
		fields = []string{"result", "response", "output", "content"}
	}

	for _, field := range fields {
		v, ok := outer[field]
		if !ok {
			continue
		}
		var s string
		if json.Unmarshal(v, &s) == nil && s != "" {
			return []byte(s)
		}
		return v
	}
	return nil
}

// ParseReview parses a canonical review JSON payload into a Review.
// providerName is used to populate Review.Provider and to select the envelope field per provider docs.
// If the payload is empty or contains no usable content, ParseReview returns nil, nil (fail-open).
// If the payload is syntactically or structurally invalid, it returns ErrMalformedResponse.
func ParseReview(providerName string, raw ProviderResult) (*Review, error) {
	if len(raw.Stdout) == 0 {
		return nil, nil
	}

	data := raw.Stdout
	if payload := extractEnvelopePayload(providerName, raw.Stdout); len(payload) > 0 {
		data = payload
	}

	var r rawReview
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrMalformedResponse, err)
	}

	if r.Verdict != VerdictCorrect && r.Verdict != VerdictIncorrect {
		return nil, fmt.Errorf("%w: verdict %q not recognised", ErrMalformedResponse, r.Verdict)
	}
	if r.Confidence < 0.0 || r.Confidence > 1.0 {
		return nil, fmt.Errorf("%w: confidence %v out of range [0,1]", ErrMalformedResponse, r.Confidence)
	}

	rev := &Review{
		Blocking:   r.Blocking,
		Summary:    r.Summary,
		Verdict:    r.Verdict,
		Confidence: r.Confidence,
		Provider:   providerName,
		DurationMS: raw.Duration,
	}

	for _, f := range r.Findings {
		severity := f.Severity
		if SeverityRank(severity) == 0 {
			// Normalise unknown severities to info.
			severity = SeverityInfo
		}
		rev.Findings = append(rev.Findings, Finding{
			Severity: severity,
			Category: f.Category,
			Message:  f.Message,
			Location: f.Location,
		})
	}

	if rev.Summary == "" && len(rev.Findings) == 0 {
		return nil, nil
	}

	return rev, nil
}
