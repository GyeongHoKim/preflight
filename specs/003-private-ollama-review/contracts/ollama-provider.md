# Contract: Ollama provider integration

**Feature**: 003-private-ollama-review  
**Date**: 2026-03-20

## 1. Provider runner (`internal/provider`)

### Inputs

| Input | Contract |
|-------|----------|
| `context.Context` | Honoured for HTTP and tool execution; deadline triggers fail-open path when classified as timeout. |
| `diff []byte` | Same staged diff as today; UTF-8 text; may be truncated per `max_diff_bytes` with user warning unchanged. |

### Outputs

| Output | Contract |
|--------|----------|
| `review.ProviderResult` | `Stdout` MUST contain **only** the canonical review JSON (or be empty for fail-open). `Stderr` MAY contain diagnostic logs for humans (not parsed). `ExitCode` set when process would have failed; HTTP errors map to Go `error` per implementation doc. |

### Parsing

- `review.ParseReview("ollama", raw)` MUST succeed when `Stdout` is the canonical JSON object (no provider-specific envelope required).
- If the model returns markdown fences or prose, the runner MUST normalize or fail with `ErrMalformedResponse` (hook retries once, then fail-open).

## 2. Ollama wire protocol (internal to `internal/ollama/…`)

Not a public API of preflight; documented so tests can mock.

| Operation | Contract |
|-----------|----------|
| Chat completion | `POST {baseURL}/api/chat` with `Content-Type: application/json`, body including `model`, `messages`, `tools` (optional), `stream: false` unless streaming is explicitly implemented. |
| Tool follow-up | Append `role: "tool"` messages with content matching Ollama’s expected shape for the active API version. |
| Errors | Non-2xx or unreachable host → return wrapped error; hook classifies fail-open vs hard error per existing rules. |

Base URL MUST be normalized (no trailing slash ambiguity) inside the client package.

## 3. Repository tools (`internal/repotools/…`)

### Shared rules

- All paths are resolved relative to a single **repository root** provided by the hook (git top-level).
- Deny rules win over allow rules.
- Every result MUST state when output was truncated.

### Tool: `list_files` (name TBD at implementation)

**Input schema (conceptual)**:

- `prefix` or `glob`: string  
- `limit`: integer (capped by config)

**Output**: Ordered list of relative paths, possibly truncated with reason.

### Tool: `read_file`

**Input**: `path`, optional `offset`/`limit` or `max_bytes`  
**Output**: File text or error string; binary files skipped with message.

### Tool: `search_repo`

**Input**: `pattern`, optional `path_prefix`, `limit`  
**Output**: Match list with file:line snippets, capped.

## 4. Configuration contract

New keys MUST be validated at load time:

- `base_url` required when `provider: ollama`.
- `model` required.
- Numeric limits MUST be positive integers with sensible defaults documented in `config` package tests.

## 5. Constitution alignment

- **stdout/stderr/exit codes**: Unchanged for the hook binary; only the runner implementation differs.
- **Fail-open**: Unavailable Ollama host, timeouts, and malformed JSON after retry remain non-blocking where `ShouldFailOpen` applies.
