# Contract: Canonical review JSON (reference)

**Source of truth in code**: `internal/review/prompt.go` (`Schema()` / `reviewSchema`)

This file duplicates the **logical** shape for planning; implementation MAY drift—always diff against `Schema()` before release.

## Root object

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| `findings` | array | yes | May be empty. |
| `blocking` | boolean | yes | Drives push gating with `block_on`. |
| `summary` | string | yes | Human-readable overview. |
| `verdict` | string | yes | Exactly `"patch is correct"` or `"patch is incorrect"`. |
| `confidence` | number | yes | `0.0`–`1.0` inclusive. |

## Finding object

| Field | Type | Required |
|-------|------|----------|
| `severity` | string | yes — `critical` \| `warning` \| `info` |
| `message` | string | yes |
| `category` | string | no — `security` \| `logic` \| `quality` \| `style` |
| `location` | string | no — file:line or similar |

The Ollama provider MUST emit JSON satisfying this contract so `review.ParseReview` remains unchanged.
