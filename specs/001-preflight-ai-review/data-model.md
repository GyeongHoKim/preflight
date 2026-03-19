# Data Model: preflight — AI-Powered Pre-Push Code Review

**Phase**: 1 — Design & Contracts
**Date**: 2026-03-18
**Branch**: `001-preflight-ai-review`

---

## Entities

### Finding

A single issue identified by the AI tool in the diff.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `Severity` | `string` enum | yes | `"critical"`, `"warning"`, `"info"` |
| `Category` | `string` enum | no | `"security"`, `"logic"`, `"quality"`, `"style"` |
| `Message` | `string` | yes | Human-readable description of the issue |
| `Location` | `string` | no | File and line reference, e.g. `"auth/token.go:42"` |

**Validation rules:**
- `Severity` must be one of the three defined values; unknown values are normalised to `"info"`.
- `Message` must be non-empty.
- `Location` is optional; omitted when the AI cannot pinpoint a specific line.

**State transitions**: none (immutable once parsed from AI output).

---

### Review

The complete output of a single AI review session.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `Findings` | `[]Finding` | yes | Ordered list of findings; may be empty |
| `Blocking` | `bool` | yes | `true` if any finding meets or exceeds the configured `block_on` threshold |
| `Summary` | `string` | yes | One-sentence overall assessment from the AI |
| `Provider` | `string` | yes | The provider that produced this review (`"claude"`, `"codex"`, etc.) |
| `DurationMS` | `int64` | yes | Wall-clock time from invocation to parsed result, in milliseconds |

**Derived property**: `CriticalCount`, `WarningCount` (computed from `Findings` — not stored).

---

### PushInfo

Information extracted from the git pre-push hook's stdin, describing what is being pushed.

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `LocalRef` | `string` | yes | Full ref name, e.g. `refs/heads/feature-x` |
| `LocalSHA` | `string` | yes | SHA of the commit being pushed |
| `RemoteRef` | `string` | yes | Ref name on the remote |
| `RemoteSHA` | `string` | yes | Current SHA on remote; all-zeros if the remote ref does not exist yet |

**Derived state**:
- `IsNewBranch() bool` — `RemoteSHA` is all-zeros
- `IsDeletePush() bool` — `LocalSHA` is all-zeros

---

### Config

User-controlled settings loaded from `.preflight.yml` (project) or `~/.config/preflight/.preflight.yml` (global). Project-level takes precedence.

| Field | YAML key | Type | Default | Description |
|-------|----------|------|---------|-------------|
| `Provider` | `provider` | `string` | `"auto"` | AI provider: `auto`, `claude`, `codex` |
| `BlockOn` | `block_on` | `string` | `"critical"` | Minimum severity that blocks a push: `"critical"` or `"warning"` |
| `Timeout` | `timeout` | `time.Duration` | `60s` | Maximum time to wait for AI CLI response |
| `PromptExtra` | `prompt_extra` | `string` | `""` | Additional instructions appended to the review system prompt |
| `MaxDiffBytes` | `max_diff_bytes` | `int` | `524288` (512 KB) | Truncate diff above this size with a warning |

**Validation rules:**
- `Provider` must be one of: `auto`, `claude`, `codex`.
- `BlockOn` must be one of: `critical`, `warning`.
- `Timeout` must be > 0.
- `MaxDiffBytes` must be > 0.

---

### ProviderResult

Raw output from a single AI CLI subprocess invocation, before semantic parsing.

| Field | Type | Description |
|-------|------|-------------|
| `Stdout` | `[]byte` | Captured stdout from the AI CLI process |
| `Stderr` | `[]byte` | Captured stderr (used for diagnostic warnings only) |
| `ExitCode` | `int` | Process exit code |
| `Duration` | `time.Duration` | Wall-clock time of the subprocess |

This is an internal transport type. It is always converted to a `Review` or an error before crossing package boundaries.

---

## Relationships

```
Config
  └── Provider (string key → ProviderRunner interface)

PushInfo
  └── used to derive git diff range → passed to ProviderRunner

ProviderRunner
  ├── input: []byte (diff content)
  └── output: ProviderResult

ProviderResult
  └── parsed by provider-specific parser → Review

Review
  └── contains []Finding
  └── displayed by TUI or plain-text renderer
  └── exit code = f(Review.Blocking, userChoice)
```

---

## Severity Ordering

For `block_on` threshold comparison:

```
critical  (3)  >  warning  (2)  >  info  (1)
```

A finding blocks if `SeverityRank(finding.Severity) >= SeverityRank(config.BlockOn)`.
