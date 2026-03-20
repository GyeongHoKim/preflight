# Quickstart: Ollama provider (design preview)

**Feature**: 003-private-ollama-review  
**Date**: 2026-03-20

This document describes how operators will exercise the feature once implemented. Exact YAML keys MUST match the implementation; names below are **planned**.

## Prerequisites

- Ollama (or API-compatible server) reachable on the internal network, serving the [Ollama HTTP API](https://docs.ollama.com/api/introduction) (default port **11434**).
- A model pulled on that server (e.g. `ollama pull llama3` on the host running Ollama).
- `preflight` built from a branch that includes the Ollama provider.

## Configuration (planned)

Project file `.preflight.yml` (or global `~/.config/preflight/.preflight.yml`) might include:

```yaml
provider: ollama
timeout: 120s
# Planned nested or flat keys — confirm at implementation time:
ollama:
  base_url: "http://ollama.internal:11434"
  model: "llama3"
  max_tool_turns: 25
```

Trust and path restrictions (from spec FR-007):

```yaml
ollama:
  deny_paths:
    - ".env*"
    - "secrets/**"
```

## Run

1. Stage changes and push (or run the hook entrypoint as documented in the main README).
2. Expect: review runs against **only** the configured `base_url`; no subprocess to `claude`/`codex` when `provider: ollama`.
3. If the server is down: expect **fail-open** (exit `0`) with a **stderr warning**, per constitution.

## Verification

- **Privacy**: Confirm via network policy or capture that traffic only goes to `base_url`.
- **Quality**: Findings should cite files/lines when tools succeeded; stderr should mention truncation when limits apply.
