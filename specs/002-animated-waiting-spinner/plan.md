# Implementation Plan: Animated Waiting Spinner + Bubbletea/Lipgloss v2

**Branch**: `002-animated-waiting-spinner` | **Date**: 2026-03-19 | **Spec**: [spec.md](./spec.md)  
**Input**: Feature specification from `/home/gyeonghokim/workspace/preflight/specs/002-animated-waiting-spinner/spec.md` plus planning directive: migrate Bubbletea/Lipgloss to v2+, implement **liquid blob / “liquid blob ring”** waiting animation (별칭 “liquid blob ring” is **not** a geometric circular band; see [research.md](./research.md)) with pure Go math, extract rendering for golden frame tests (deterministic seed, ANSI-off option).

---

## Summary

preflight will **upgrade** terminal stack to **Bubbletea v2** (`charm.land/bubbletea/v2`) and **Lipgloss v2** (`github.com/charmbracelet/lipgloss/v2`), then show a **deterministic, testable** liquid-blob spinner while the AI provider runs. The **mathematical core** is a **2D Cartesian metaball-style field** in a rectangular viewport plus **planar traveling wave** and **global pulse** ([research.md](./research.md)) — **no annulus / no circular ring mask**. It lives in a **stdlib-only internal package**; **Lipgloss** applies color/glyphs; **Bubbletea** only drives ticks and lifecycle. **Regression tests** compare **plain-text** frame snapshots (`DisableANSI`) so `go test` needs no real terminal. **`--no-tui` / non-TTY** paths remain plain progress text per spec FR-006.

---

## Technical Context

**Language/Version**: Go 1.26.x (matches repo `go.mod`)  
**Primary Dependencies**: `charm.land/bubbletea/v2`, Lipgloss v2 **`github.com/charmbracelet/lipgloss/v2`** (locked in [research.md](./research.md) §2; record same path in `go.mod` and [CLAUDE.md](../../CLAUDE.md) via tasks); existing cobra, go-isatty, testify, yaml.v3  
**Storage**: N/A  
**Testing**: `go test ./...` + golden files under `internal/anim/testdata/` or adjacent package; race via `make test`  
**Target Platform**: Linux + macOS (same as project)  
**Project Type**: CLI / git hook TUI  
**Performance Goals** *(engineering guidance — **not** spec-gated unless promoted to [spec.md](./spec.md))*: Spinner tick 10–30 FPS feel; negligible CPU vs AI subprocess; frame compute for typical width×height < 1ms where practical. Use for implementation tuning only; no dedicated task unless NFR is added to the spec.  
**Constraints**: No new “util/helpers” packages; spinner failure must not affect push exit code (FR-008); deterministic output for fixed `(seed, tick, size)` when ANSI disabled  
**Scale/Scope**: Single spinner region + transition to existing review UI; hook orchestration may need async provider invocation under Bubbletea

---

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Go Standards Compliance | ✅ PASS | New exported APIs need doc comments; errors wrapped per project rules. |
| II. Zero-Lint Policy | ✅ PASS | `make lint` after each change. |
| III. Explicit Error Handling | ✅ PASS | Spinner/render errors must not panic; must not change review exit semantics (warn or degrade visually). |
| IV. CLI Interface Design | ✅ PASS | `--no-tui` / non-TTY unchanged: plain output, no animation ANSI. **Regression** (post-hook/async): AI CLI missing/timeout → exit `0` + stderr warning ([tasks.md](./tasks.md) T025). |
| V. Simplicity & Minimal Dependencies | ✅ PASS | v2 upgrades replace v1 lines (not additive); new internal package is single-purpose animation math, not `util`. |

**No violations.** Complexity Tracking not required.

### Post-design re-evaluation (Phase 1 complete)

Design artifacts ([research.md](./research.md), [data-model.md](./data-model.md), [contracts/spinner-render.md](./contracts/spinner-render.md)) keep **ANSI-free goldens** on the internal contract — compatible with Principle IV (no pollution of plain mode). **Bubbletea v2** remains justified as the TUI framework; extraction of `anim` satisfies testability without extra user-facing dependencies.

---

## Project Structure

### Documentation (this feature)

```text
/home/gyeonghokim/workspace/preflight/specs/002-animated-waiting-spinner/
├── plan.md              # This file
├── research.md          # Phase 0
├── data-model.md        # Phase 1
├── quickstart.md        # Phase 1
├── contracts/
│   └── spinner-render.md
├── spec.md
└── checklists/
```

### Source Code (repository root) — planned deltas

```text
/home/gyeonghokim/workspace/preflight/
├── go.mod                          # bubbletea v2 + lipgloss v2; teatest revision TBD
├── internal/
│   ├── anim/                       # NEW: 2D liquid-blob field, Frame grid, deterministic seed
│   │   ├── liquidblob.go
│   │   ├── liquidblob_test.go
│   │   └── testdata/               # golden frame *.txt (optional placement per contract)
│   ├── tui/
│   │   ├── model.go                # MIGRATE: Bubbletea v2, optional WaitingModel compose
│   │   ├── styles.go               # MIGRATE: Lipgloss v2
│   │   ├── plain.go                # review plain-text render only; stderr wait/progress orchestration lives in hook.go (T019)
│   │   ├── spinner_view.go         # NEW: Frame -> string (RenderOptions: DisableANSI)
│   │   └── model_test.go           # MIGRATE: teatest or slimmer tests
│   └── hook/
│       └── hook.go                 # MIGRATE: tea API; possibly wire loading phase + Cmd for provider
```

**Structure Decision**: **`internal/anim`** holds pure logic (constitution-friendly single responsibility). **`internal/tui`** owns Lipgloss projection and Bubbletea models. Hook stays orchestration-only.

---

## Implementation Phases

### Phase 0 — Research ✅

**Output**: [research.md](./research.md) — v2 module paths, upgrade guide notes, mathematical model, testing strategy, teatest caveat.

---

### Phase 1 — Design & contracts ✅

**Outputs**:

- [data-model.md](./data-model.md) — `RenderOptions`, `LiquidBlobConfig`, `Frame`, `WaitingPhase`, `WaitingModel`.
- [contracts/spinner-render.md](./contracts/spinner-render.md) — determinism + golden file rules.
- [quickstart.md](./quickstart.md) — dev workflow for this branch.

**Agent context**: run `update-agent-context.sh cursor-agent` (see quickstart).

---

### Phase 2 — Task breakdown ✅

**Output**: [tasks.md](./tasks.md) — dependency-ordered implementation tasks (T001–T025), mapped to user stories US1–US3 plus polish.

**Themes covered**: go.mod v2 migration; `internal/anim` (2D liquid-blob field, **not** annulus) + goldens; `spinner_view` + `waiting.go`; async hook + **stderr** plain progress (헌장 IV); Ctrl+C cleanup; quality gates.

---

## Risk register

| Risk | Mitigation |
|------|------------|
| teatest incompatible with Bubbletea v2 | Prefer golden tests on `RenderFrame`; slim integration tests. |
| Hook blocking model prevents spinner ticks | Refactor provider call into async pattern with Bubbletea. |
| Lipgloss ANSI in tests | `DisableANSI` / no-color renderer profile (documented in contract). |

---

## Dependency justification (delta)

| Change | Why needed |
|--------|------------|
| Bubbletea v2 | User requirement; v2 API for future maintenance. |
| Lipgloss v2 | Aligned stack with Bubbletea v2 / terminal color pipeline. |
| No new third-party deps for math | Spec requires pure Go logic; stdlib floats + PRNG sufficient. |
