# Provider Contract: AI CLI Integration

**Type**: Internal interface contract between preflight and AI CLI subprocesses
**Date**: 2026-03-18
**Branch**: `001-preflight-ai-review`

---

## Provider Interface

Every AI provider adapter in preflight must satisfy this contract:

**Input**: a byte slice containing the git diff to review, a `context.Context` for timeout/cancellation, and the review system prompt.
**Output**: a `ProviderResult` (raw stdout/stderr/exit code) or an error.
**Error semantics**: errors returned from the provider adapter are categorised as either "fail-open" (tool unavailable, timeout) or "hard errors" (usage errors). Only hard errors propagate to exit code `1`.

---

## Per-Provider Invocation Details

### claude

| Attribute | Value |
|-----------|-------|
| Binary name | `claude` |
| Non-interactive flag | `-p` |
| JSON envelope flag | `--output-format json` |
| Schema enforcement | `--json-schema '<schema-json>'` |
| Diff delivery | stdin pipe |
| System prompt | `--append-system-prompt '<prompt>'` |
| Session persistence | `--no-session-persistence` |

**Full invocation:**
```
claude -p "<review prompt>" \
  --output-format json \
  --no-session-persistence \
  --json-schema '<canonical-review-schema>'
```
Diff is written to the command's stdin.

**Response parsing:**
```json
{
  "type": "result",
  "subtype": "success",
  "is_error": false,
  "result": "{\"findings\":[...],\"blocking\":true,\"summary\":\"...\"}",
  "total_cost_usd": 0.003,
  "duration_ms": 1200
}
```
Extract `result` field. Parse `result` as JSON to get the canonical review object.
If `is_error` is `true`, treat as a fail-open warning.

---

### codex

| Attribute | Value |
|-----------|-------|
| Binary name | `codex` |
| Non-interactive flag | `-q` |
| JSON envelope flag | `--json` |
| Schema enforcement | prompt injection only |
| Diff delivery | embedded in prompt string |

**Full invocation:**
```
codex -q --json "<review prompt + schema instruction + diff>"
```
No stdin piping documented. For large diffs (>100 KB), write diff to a temp file and reference its path in the prompt.

**Response parsing:** Schema not officially documented. Attempt to parse full stdout as JSON. If top-level keys include a string field (e.g., `output`, `response`, `content`, or `result`), attempt to parse that field as the canonical review JSON. If unparseable, emit a warning and fail-open.

---

## Fail-Open Conditions

The following conditions MUST result in exit code `0` with a warning on stderr — they never block a push:

| Condition | Detection |
|-----------|-----------|
| Binary not in PATH | `exec.LookPath` returns error |
| Process timeout | `context.DeadlineExceeded` |
| Non-zero exit code from AI CLI | `*exec.ExitError` with any exit code |
| stdout is empty | `len(stdout) == 0` |
| stdout is not valid JSON | `json.Unmarshal` returns error |
| Review JSON missing required fields | validation after unmarshal |

---

## Large Diff Handling

If the diff exceeds `config.MaxDiffBytes` (default 512 KB):
1. Truncate to `MaxDiffBytes`.
2. Append a warning comment to the truncated diff: `# [preflight: diff truncated at 512 KB]`.
3. Emit a warning to stderr: `preflight: diff truncated; review may be incomplete`.
4. Continue with the truncated diff.

---

## Timeout Enforcement

All AI CLI subprocesses are launched with `exec.CommandContext(ctx, ...)` where `ctx` has a deadline equal to `config.Timeout`. When the context deadline is exceeded:
- The subprocess is killed (SIGKILL via context cancellation).
- preflight emits: `preflight: <provider> timed out after <timeout>; skipping review` to stderr.
- Exit code: `0` (fail-open).
