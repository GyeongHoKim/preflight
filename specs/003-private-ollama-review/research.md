# Phase 0 Research: Private Ollama provider & repository tools

**Feature**: 003-private-ollama-review  
**Date**: 2026-03-20  
**Sources**: [Ollama API introduction](https://docs.ollama.com/api/introduction), [Chat API](https://docs.ollama.com/api/chat), [Generate API](https://docs.ollama.com/api/generate), [OpenAI compatibility](https://docs.ollama.com/api/openai-compatibility), [Streaming](https://docs.ollama.com/api/streaming), `pkg.go.dev` for `github.com/ollama/ollama/api`

## 1. Ollama HTTP API surface (official)

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/api/chat` | POST | Multi-turn chat; supports `messages`, optional `tools` (function definitions), `format` (`json` or JSON schema), `stream` (default **true**), `options`, `keep_alive`, etc. |
| `/api/generate` | POST | Single-turn completion from `prompt`; optional `system`, `format`, `stream`, images, etc. |
| `/v1/chat/completions` | POST | OpenAI-compatible chat (tools, streaming, JSON mode) |
| `/v1/responses` | POST | OpenAI Responses-style API (tools, streaming; feature set evolves) |

**Base URL**: Default local install serves under `http://localhost:11434` with API paths under `/api` (e.g. full chat URL `http://localhost:11434/api/chat`). Cloud models use `https://ollama.com/api` per docs.

**Streaming**: Many endpoints stream **newline-delimited JSON** (`application/x-ndjson`) by default. For deterministic parsing in preflight, requests SHOULD set `stream: false` unless/until the hook implements NDJSON aggregation.

**Tool calling** (native API): `POST /api/chat` accepts a `tools` array of function definitions (`type: function`, `function.name`, `function.description`, `function.parameters` as JSON Schema). Assistant messages may include `tool_calls`; the client MUST send follow-up messages with `role: "tool"` and matching content until the model returns a final assistant message (or limits hit).

## 2. HTTP client: library vs stdlib

### Decision

Use **`net/http` + `encoding/json`** with **internal request/response types** living in a dedicated package (see plan) that **hide** URL paths, query/body shapes, and streaming details from `provider` and `hook`. Optionally wrap or replace internals later with `github.com/ollama/ollama/api` **only if** an audit shows materially less code and acceptable `go.mod` weight.

### Rationale

- Aligns with **Constitution V (Simplicity & Minimal Dependencies)**: Ollama’s contract is stable JSON-over-HTTP; stdlib is sufficient for `stream: false` chat + tool loops.
- Full control over **timeouts**, **context cancellation**, and **error mapping** to fail-open semantics.
- Avoids pinning to a third-party wrapper that may lag server features or pull unrelated code.

### Alternatives considered

| Option | Pros | Cons |
|--------|------|------|
| **Official `github.com/ollama/ollama/api`** | Maintained alongside server; handles streaming helpers | New module dependency; must verify Chat + tools coverage and transitive footprint |
| **Community clients** (e.g. `ollama-go`, `go-ollama`) | Convenience | Extra maintainer risk; constitution prefers stdlib unless clear win |
| **OpenAI-compatible client only** (`/v1/chat/completions`) | Familiar types | Extra compatibility layer; native `/api/chat` tool docs are the reference for this feature |

## 3. Repository “tools” (replacing CLI yolo exploration)

Ollama does not run `claude`-style embedded tools. The product MUST implement a **closed set** of tools, executed **inside preflight** on behalf of the model:

| Tool (conceptual name) | Purpose | Safety |
|------------------------|---------|--------|
| **list_files** | List paths under repo root (with glob or prefix), bounded count | Respect max entries; deny paths outside repo root |
| **read_file** | Read file slice with offset/length or max bytes | Truncate; deny binary or oversize reads |
| **search_repo** | Search for pattern (substring or regex subset) across allowed files | Limit matches, file count, and bytes scanned |
| **git_context** (optional) | Structured output of `git log` / related refs for touched files | Read-only git subprocess with argument allowlist |

**Package layout**: Implement tools under a **separate Go package** (e.g. `internal/repotools/`) with small interfaces so tests do not require a live Ollama server.

**Limits**: All tools MUST enforce configurable caps (per FR-004/FR-005 in spec) — max tool turns, max bytes per read, max list size, deny glob for sensitive paths (FR-007).

## 4. Output shape vs existing providers

Existing flow expects a **canonical review JSON** (`internal/review` schema). The Ollama runner SHOULD drive the model until it emits **one JSON object** conforming to that schema (via `format` with JSON schema where supported, or strict prompt + validation + repair loop). **Stdout** of the runner remains the canonical payload for `review.ParseReview` (provider name `ollama`).

## 5. Resolved clarifications (formerly NEEDS CLARIFICATION)

- **Primary API**: `POST /api/chat` with `stream: false`, tool loop until final message or cap.
- **Client implementation**: stdlib-first, dependency on `ollama/ollama/api` optional after spike.
- **Tools**: Dedicated `internal/repotools/` (or equivalent) package; not embedded in `provider` beyond orchestration.
