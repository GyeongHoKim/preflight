# Implementation Plan: Private Ollama provider with repository tools

**Branch**: `003-private-ollama-review` | **Date**: 2026-03-20 | **Spec**: [spec.md](./spec.md)  
**Input**: Feature specification + implementation notes (Ollama API, tools package, interface alignment)

## Summary

Deliver an **`ollama` provider** for preflight that talks to an **organization-controlled Ollama HTTP API** (per [Ollama docs](https://docs.ollama.com/api/introduction)), keeps **wire details inside a dedicated internal client package**, and supplies **repository exploration tools** in a **separate package** so local models can match—within limits—the grounded review quality of subprocess-based CLIs. Review output stays the **existing canonical JSON** consumed by `review.ParseReview`.

## Technical Context

**Language/Version**: Go 1.26.x (repo `go.mod`)  
**Primary Dependencies**: Existing (bubbletea, cobra, yaml, testify); **no new dependency in Phase 1 decision** — use `net/http` + `encoding/json` for Ollama (see [research.md](./research.md)); optional spike on `github.com/ollama/ollama/api` during implementation only if justified.  
**Storage**: N/A (no persistent datastore; ephemeral chat state in memory).  
**Testing**: `go test ./...`, table-driven tests; HTTP via `httptest.Server`; tools tested without live Ollama.  
**Target Platform**: Linux/macOS/WSL developer machines; hook runs in user repo context.  
**Project Type**: Single-binary CLI (`cmd/preflight`) + `internal/*` libraries.  
**Performance Goals**: Complete one review within configured `timeout`; tool loop MUST cap turns and bytes (spec FR-004/FR-005).  
**Constraints**: Fail-open on provider failure (constitution); no `util` package names; explicit error handling; `make lint` clean.  
**Scale/Scope**: Single-repo working tree; large-repo behavior via truncation + warnings.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Go Standards | OK | New packages need doc comments on exported symbols. |
| II. Zero-Lint | OK | Planned code must pass `make lint`. |
| III. Explicit errors | OK | HTTP and tool errors wrapped with context. |
| IV. CLI I/O | OK | Runner returns `ProviderResult`; hook stdout/stderr contract unchanged. |
| V. Minimal dependencies | OK | Prefer stdlib for Ollama HTTP; see Complexity if adding `ollama/api`. |

### Constitution alignment (Ollama HTTP)

The repository constitution (v1.1.0+) explicitly allows **user-configured local or
organization-controlled HTTP inference** (e.g. Ollama) alongside subprocess CLI
providers, and forbids *requiring* third-party cloud APIs for core review. This
plan matches that policy.

### Post-design re-check

| Item | Status |
|------|--------|
| New packages have single responsibility (`ollama` transport vs `repotools`) | OK |
| Fail-open semantics preserved for timeouts / connection errors | OK |
| Review JSON contract unchanged | OK |

## Project Structure

### Documentation (this feature)

```text
specs/003-private-ollama-review/
├── plan.md
├── research.md
├── data-model.md
├── quickstart.md
├── contracts/
│   ├── ollama-provider.md
│   └── review-output-json.md
└── spec.md
```

### Source Code (repository root)

```text
cmd/preflight/
└── main.go

internal/
├── cli/
├── hook/
├── provider/
│   ├── runner.go
│   ├── claude.go
│   ├── codex.go
│   └── ollama.go          # NEW: OllamaRunner implements Runner; orchestrates chat+tools
├── ollama/                # NEW: HTTP client + types; hides /api/chat details
├── repotools/             # NEW: list/read/search (+ optional git helpers)
├── review/
├── diff/
├── tui/
├── anim/
└── config/

```

**Structure Decision**: Add **`internal/ollama`** for API-facing code (private types + client) and **`internal/repotools`** for filesystem/git-safe tools. **`internal/provider/ollama.go`** (or `ollama_runner.go`) implements `Runner`, wires prompt/schema from `review`, runs the tool loop, and writes **final canonical JSON** to `ProviderResult.Stdout`. This matches the user request to hide API details behind an interface while keeping tools maintainable.

## Complexity Tracking

> Fill ONLY if Constitution Check has violations that must be justified

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| HTTP instead of subprocess for inference | Ollama is a **server**; multi-turn **tool calling** maps to `/api/chat`, not a single `ollama run` one-shot | One-shot CLI does not implement the required tool loop + schema reliably |
| (Optional) `github.com/ollama/ollama/api` | Official types/streaming helpers | Rejected for v1 in research — stdlib first; revisit if duplication grows |

## Generated Artifacts (this command)

| Artifact | Path |
|----------|------|
| Research | [research.md](./research.md) |
| Data model | [data-model.md](./data-model.md) |
| Contracts | [contracts/ollama-provider.md](./contracts/ollama-provider.md), [contracts/review-output-json.md](./contracts/review-output-json.md) |
| Quickstart | [quickstart.md](./quickstart.md) |

## Phase 2 (out of scope for this command)

`tasks.md` will be produced by `/speckit.tasks`, not here.
