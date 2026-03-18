package review

import "fmt"

// reviewSchema is the canonical JSON schema for AI review responses.
const reviewSchema = `{
  "type": "object",
  "required": ["findings", "blocking", "summary"],
  "properties": {
    "findings": {
      "type": "array",
      "items": {
        "type": "object",
        "required": ["severity", "message"],
        "properties": {
          "severity": {"type": "string", "enum": ["critical", "warning", "info"]},
          "category": {"type": "string", "enum": ["security", "logic", "quality", "style"]},
          "message": {"type": "string"},
          "location": {"type": "string"}
        }
      }
    },
    "blocking": {"type": "boolean"},
    "summary": {"type": "string"}
  }
}`

// Schema returns the canonical JSON schema string for AI review responses.
func Schema() string {
	return reviewSchema
}

// SystemPrompt returns the system prompt string for AI review requests.
// extra is appended after the base instructions when non-empty.
func SystemPrompt(extra string) string {
	base := `You are a code reviewer. Review the following git diff for:
- Security vulnerabilities (hardcoded secrets, injection risks, auth bypasses)
- Silent error discarding (ignored return values, empty catch blocks)
- Logic bugs with data loss or corruption potential

Respond ONLY with a JSON object matching this schema:
` + reviewSchema
	if extra != "" {
		return fmt.Sprintf("%s\n\nAdditional instructions: %s", base, extra)
	}
	return base
}
