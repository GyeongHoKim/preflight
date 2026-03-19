package review

import (
	"encoding/json"
)

// rawReview is the JSON structure returned by AI providers.
type rawReview struct {
	Findings []struct {
		Severity string `json:"severity"`
		Category string `json:"category"`
		Message  string `json:"message"`
		Location string `json:"location"`
	} `json:"findings"`
	Blocking bool   `json:"blocking"`
	Summary  string `json:"summary"`
}

// extractEnvelopePayload returns the inner review payload from a provider's JSON envelope.
// Per provider contract: claude/qwen use "result", gemini uses "response".
// Codex and "unknown" (auto) have no documented envelope; we try common field names.
func extractEnvelopePayload(providerName string, data []byte) []byte {
	var outer map[string]json.RawMessage
	if err := json.Unmarshal(data, &outer); err != nil {
		return nil
	}

	var fields []string
	switch providerName {
	case "claude", "qwen":
		// https://code.claude.com/docs/en/cli-reference; qwen-code shares same schema.
		fields = []string{"result"}
	case "gemini":
		// https://github.com/google-gemini/gemini-cli docs: --output-format json → "response".
		fields = []string{"response"}
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
// If the payload is nil or malformed, ParseReview returns nil, nil (fail-open).
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
		// Malformed JSON — fail-open.
		return nil, nil //nolint:nilerr // intentional fail-open: malformed AI response must not block a push
	}

	rev := &Review{
		Blocking:   r.Blocking,
		Summary:    r.Summary,
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
