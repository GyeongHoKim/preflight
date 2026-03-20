# Data Model: Ollama provider & repository tools

**Feature**: 003-private-ollama-review  
**Date**: 2026-03-20

Entities extend the [feature spec](./spec.md) key entities with implementation-oriented records. Field types are logical, not database-specific.

## Configuration (runtime)

### OllamaProviderConfig

| Field | Description |
|-------|-------------|
| `BaseURL` | HTTP(S) origin for the Ollama server (e.g. `http://ollama.internal:11434`). |
| `Model` | Model tag as understood by Ollama (`llama3`, `qwen2.5-coder`, etc.). |
| `Timeout` | Upper bound for a full review (may inherit global `timeout`). |
| `MaxToolTurns` | Maximum assistant↔tool round trips per review. |
| `MaxReadBytes` | Per `read_file` response cap. |
| `MaxListEntries` | Per `list_files` cap. |
| `MaxSearchMatches` | Per `search_repo` cap. |
| `AllowedPathPrefixes` | Optional allowlist under repo root (empty = entire repo subject to deny rules). |
| `DeniedPathGlobs` | Glob or path prefixes excluded from all tools (e.g. `.env`, `secrets/`). |

## Chat session (ephemeral)

### ChatTurn

| Field | Description |
|-------|-------------|
| `Role` | `system`, `user`, `assistant`, or `tool`. |
| `Content` | Text or structured tool result string. |
| `ToolCalls` | Present on assistant messages when the model requests tools (names + arguments). |
| `ToolName` / `ToolCallID` | For `tool` role messages, link result to request. |

### ToolInvocation (internal)

| Field | Description |
|-------|-------------|
| `Name` | Tool identifier exposed to the model. |
| `Arguments` | JSON object per tool schema. |
| `Result` | JSON or text returned to the model (truncation noted in payload). |

## Repository context

### RepoRoot

| Field | Description |
|-------|-------------|
| `AbsPath` | Absolute path to git working tree root. |
| `GitDir` | Optional path to `.git` if needed for git-based tools. |

## Review output

Unchanged canonical model from `internal/review`:

- `Review`: `Findings[]`, `Blocking`, `Summary`, `Verdict`, `Confidence`, `Provider`, `DurationMS`.
- `Finding`: `Severity`, `Category`, `Message`, `Location`.

## State transitions

1. **Config loaded** → validate provider + Ollama fields.
2. **Runner started** → build initial `messages` (system + user with diff).
3. **Loop**: POST chat → if `tool_calls` → execute `repotools` → append tool messages → repeat until assistant content with final JSON or limit/timeout.
4. **Parse** → `review.ParseReview("ollama", ProviderResult{Stdout: payload})`.

## Validation rules (from spec)

- Paths resolved MUST stay under `RepoRoot` after `Clean`/`Eval` or equivalent.
- Denied globs MUST be checked before any read/list/search.
- Any truncation MUST be explicit in tool result text so the model does not assume completeness.
