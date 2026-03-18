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
	base := `You are acting as a reviewer for a proposed code change made by another engineer.
Focus on issues that impact correctness, performance, security, maintainability, or developer experience.
Flag only actionable issues introduced by the pull request.
When you flag an issue, provide a short, direct explanation and cite the affected file and line range.
Prioritize severe issues and avoid nit-level comments unless they block understanding of the diff.
After listing findings, produce an overall correctness verdict ("patch is correct" or "patch is incorrect") with a concise justification and a confidence score between 0 and 1.
Ensure that file citations and line numbers are exactly correct using the tools available; if they are incorrect your comments will be rejected.

Respond ONLY with a JSON object matching this schema:
` + reviewSchema
	if extra != "" {
		return fmt.Sprintf("%s\n\nAdditional instructions: %s", base, extra)
	}
	return base
}
