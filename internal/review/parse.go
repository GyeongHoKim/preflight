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

// ParseReview parses a canonical review JSON payload into a Review.
// providerName is used to populate Review.Provider.
// If the payload is nil or malformed, ParseReview returns nil, nil (fail-open).
func ParseReview(providerName string, raw ProviderResult) (*Review, error) {
	if len(raw.Stdout) == 0 {
		return nil, nil
	}

	data := raw.Stdout

	// Attempt to unwrap a provider envelope if the outer object has no "findings" key.
	// Try common envelope fields: result, response, output, content.
	var outer map[string]json.RawMessage
	if err := json.Unmarshal(data, &outer); err == nil {
		for _, field := range []string{"result", "response", "output", "content"} {
			if v, ok := outer[field]; ok {
				// The value may be a JSON string containing nested JSON, or an object.
				var s string
				if json.Unmarshal(v, &s) == nil && s != "" {
					data = []byte(s)
					break
				}
				data = v
				break
			}
		}
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
