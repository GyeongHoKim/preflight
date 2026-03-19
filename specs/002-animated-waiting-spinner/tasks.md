# Tasks: Animated Waiting Spinner (Bubbletea/Lipgloss v2 + 2D liquid blob field)

**Input**: Design documents from `/home/gyeonghokim/workspace/preflight/specs/002-animated-waiting-spinner/`  
**Prerequisites**: [plan.md](./plan.md), [spec.md](./spec.md), [research.md](./research.md), [data-model.md](./data-model.md), [contracts/spinner-render.md](./contracts/spinner-render.md), [quickstart.md](./quickstart.md)

**Tests**: Golden/snapshot and plain-path assertions are included because [contracts/spinner-render.md](./contracts/spinner-render.md) and [plan.md](./plan.md) require deterministic `DisableANSI` frame regression tests, **SC-002** adjacent-frame metrics (contract §Adjacent-frame smoothness), and SC-004/FR-008 verification.

**Organization**: Phases follow user story priorities P1→P3 from [spec.md](./spec.md); Bubbletea/Lipgloss v2 migration is **foundational** and blocks all stories.

**Normative references**: Plain-mode stdout/stderr rules — **FR-006** (see US2 in spec). Spinner must not affect hook exit — **FR-008** / **SC-005**.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no ordering dependency on incomplete sibling tasks)
- **[Story]**: [US1], [US2], [US3] map to spec user stories
- Paths are absolute under the repository root `/home/gyeonghokim/workspace/preflight`

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Baseline and design alignment before dependency migration

- [ ] T001 Read `/home/gyeonghokim/workspace/preflight/specs/002-animated-waiting-spinner/plan.md` and `/home/gyeonghokim/workspace/preflight/specs/002-animated-waiting-spinner/spec.md` and confirm scope (v2 migration + spinner + plain path + fail-open)
- [ ] T002 [P] Re-read `/home/gyeonghokim/workspace/preflight/specs/002-animated-waiting-spinner/contracts/spinner-render.md` and `/home/gyeonghokim/workspace/preflight/specs/002-animated-waiting-spinner/data-model.md` for entity names and golden rules
- [ ] T003 [P] Run `make lint` and `make test` from `/home/gyeonghokim/workspace/preflight` on the feature branch; record baseline (must be green before migration)

---

## Phase 2: Foundational — Bubbletea v2 + Lipgloss v2 (Blocking)

**Purpose**: Upgrade terminal stack per [research.md](./research.md); **no user story work until this phase completes**

**⚠️ CRITICAL**: All US1–US3 implementation assumes v2 APIs (`tea.View`, `KeyPressMsg`, Lipgloss v2 module path locked in `go.mod`)

- [ ] T004 Update `/home/gyeonghokim/workspace/preflight/go.mod` to require `charm.land/bubbletea/v2` and **`github.com/charmbracelet/lipgloss/v2`** (locked in [research.md](./research.md) §2 — do not split across mirror module paths); remove direct v1 `github.com/charmbracelet/bubbletea` and `github.com/charmbracelet/lipgloss` requires; run `go mod tidy` in `/home/gyeonghokim/workspace/preflight`
- [ ] T005 Migrate `/home/gyeonghokim/workspace/preflight/internal/tui/model.go` to Bubbletea v2 (`View() tea.View`, `tea.NewView`, `tea.KeyPressMsg`, window size messages per v2 upgrade guide)
- [ ] T006 [P] Migrate `/home/gyeonghokim/workspace/preflight/internal/tui/styles.go` to Lipgloss v2 import path and fix any API drift for review styles
- [ ] T007 Update `/home/gyeonghokim/workspace/preflight/internal/hook/hook.go` for Bubbletea v2 program creation and stdout output (`tea.NewProgram`, v2-compatible options)
- [ ] T008 Update `/home/gyeonghokim/workspace/preflight/internal/tui/model_test.go` for v2 messages; fix or replace `github.com/charmbracelet/x/exp/teatest` in `/home/gyeonghokim/workspace/preflight/go.mod` if incompatible with v2
- [ ] T009 Run `make lint && make test` from `/home/gyeonghokim/workspace/preflight` and fix all compile/lint failures introduced by T004–T008

**Checkpoint**: Repository builds cleanly on Bubbletea v2 + Lipgloss v2; review TUI and hook paths behave as before (minus intentional UI API changes)

---

## Phase 3: User Story 1 — Animated Spinner While Waiting (Priority: P1) 🎯 MVP

**Goal**: **Liquid blob** waiting animation (product name “liquid blob ring” ≠ geometric circle/torus; rectangular 2D metaball field + planar wave + pulse + gradient) during AI provider run; deterministic core + Lipgloss projection; transition to review UI when review is ready ([spec.md](./spec.md) US1, FR-001–FR-005)

**Independent Test**: On a TTY, `git push` / `preflight run` shows animated spinner until provider returns, then review view without spinner ([spec.md](./spec.md) US1)

- [ ] T010 [US1] Create `/home/gyeonghokim/workspace/preflight/internal/anim/liquidblob.go` exporting `LiquidBlobConfig`, `RenderOpts`, `Frame`, `ComputeFrame` per `/home/gyeonghokim/workspace/preflight/specs/002-animated-waiting-spinner/data-model.md` and 2D Cartesian metaball + planar wave + pulse model in `/home/gyeonghokim/workspace/preflight/specs/002-animated-waiting-spinner/research.md` — **no annulus / no polar ring mask** (stdlib only; no bubbletea/lipgloss imports)
- [ ] T011 [P] [US1] Add `/home/gyeonghokim/workspace/preflight/internal/anim/liquidblob_test.go` asserting bitwise-stable `Frame` for identical `(config, width, height, tick, seed)`
- [ ] T012 [US1] Add `/home/gyeonghokim/workspace/preflight/internal/tui/spinner_view.go` implementing `RenderFrame(frame anim.Frame, opts RenderOptions) string` with `DisableANSI` / `DisableColor` per `/home/gyeonghokim/workspace/preflight/specs/002-animated-waiting-spinner/contracts/spinner-render.md` using Lipgloss v2 when ANSI allowed
- [ ] T013 [P] [US1] Add golden text fixtures under `/home/gyeonghokim/workspace/preflight/internal/anim/testdata/spinner_golden/` and tests (e.g. in `/home/gyeonghokim/workspace/preflight/internal/tui/spinner_view_test.go`) that `RenderFrame` with `DisableANSI: true` matches files, contains no `0x1b` bytes, and enforces **[spec.md](./spec.md) SC-002** adjacent-tick rules per [contracts/spinner-render.md](./contracts/spinner-render.md) §Adjacent-frame smoothness (max differing cells per `(t,t+1)`; max fraction of identical adjacent pairs)
- [ ] T014 [US1] Add `/home/gyeonghokim/workspace/preflight/internal/tui/waiting.go` with Bubbletea v2 model: tick-based updates, calls `ComputeFrame` + `RenderFrame`, transitions to existing review model/state when provider result arrives
- [ ] T015 [US1] Refactor `/home/gyeonghokim/workspace/preflight/internal/hook/hook.go` so provider invocation runs concurrently with TUI (e.g. `tea.Cmd` wrapping provider `Run` or goroutine + custom `tea.Msg`) enabling spinner ticks during wait; preserve fail-open and exit code behavior for non-TUI branches until US2 tweaks
- [ ] T016 [US1] Handle spec edge cases in `/home/gyeonghokim/workspace/preflight/internal/tui/spinner_view.go` and/or `/home/gyeonghokim/workspace/preflight/internal/tui/waiting.go`: very narrow terminal, no-color / low-capability terminal (glyph-only path), fast provider response avoids jarring flash ([spec.md](./spec.md) Edge Cases)
- [ ] T017 [US1] Implement Ctrl+C / OS signal handling for the waiting TUI in `/home/gyeonghokim/workspace/preflight/internal/tui/waiting.go` and `/home/gyeonghokim/workspace/preflight/internal/hook/hook.go`: propagate quit cleanly, restore terminal state (alt-screen/mouse if used), satisfy FR-007 and SC-006 ([spec.md](./spec.md) Edge Cases)
- [ ] T018 [P] [US1] **Prefer** automated coverage for **SC-006** / **FR-007** in `/home/gyeonghokim/workspace/preflight/internal/tui/waiting_test.go` (synthetic `tea.Msg` / program teardown proving clean model stop and no stuck alt-screen state). Use quickstart manual checklist **only** if CI cannot approximate teardown; if manual-only, mark quickstart interrupt rows as **release-blocking** (see [spec.md](./spec.md) SC-006)

**Checkpoint**: US1 satisfied: animated waiting on TTY, goldens stable, review replaces spinner on completion; interrupt path defined and verified

---

## Phase 4: User Story 2 — Plain Text Progress (Priority: P2)

**Goal**: `--no-tui` or non-TTY: no spinner ANSI on **stdout**; readable progress on **stderr** (헌장 IV); plain review only on stdout after progress cleared ([spec.md](./spec.md) US2, FR-006, SC-004)

**Independent Test**: `preflight run --no-tui`: **stdout** has no ESC and contains review; **stderr** has plain progress text only ([spec.md](./spec.md) US2)

- [ ] T019 [US2] Implement plain progress messaging during provider wait in `/home/gyeonghokim/workspace/preflight/internal/hook/hook.go` when `noTUI || !tui.IsTTY()`: write human-readable lines to **stderr** only (e.g. `Analyzing changes...`); clear or finalize stderr line before `/home/gyeonghokim/workspace/preflight/internal/tui/plain.go` `PlainRender` writes the review to **stdout** only
- [ ] T020 [P] [US2] Extend `/home/gyeonghokim/workspace/preflight/internal/hook/hook_test.go` to capture **stdout** and **stderr** separately on plain path: assert **stdout** contains no `0x1b` bytes through wait + review (SC-004); assert **stderr** (진행 메시지 구간)에도 **`0x1b` 바이트가 없음** — US2·FR-006과 동일하게 색·커서·애니메이션용 ESC/SGR 없이 평문만 허용

**Checkpoint**: US2 independently verifiable; stdout/stderr contract matches constitution

---

## Phase 5: User Story 3 — Fail-Open: Spinner Must Not Block Push (Priority: P3)

**Goal**: Spinner/render failures never change review-based exit semantics ([spec.md](./spec.md) US3, FR-008, SC-005)

**Independent Test**: Inject render failure or panic guard in waiting path; completed review still renders and exit code matches review-only rules ([spec.md](./spec.md) US3)

- [ ] T021 [US3] Add defensive error handling in `/home/gyeonghokim/workspace/preflight/internal/tui/waiting.go` and integration points in `/home/gyeonghokim/workspace/preflight/internal/hook/hook.go` so spinner errors log to stderr (optional) and fall back to blank/minimal wait UI without returning spurious hook errors
- [ ] T022 [P] [US3] Add regression test in `/home/gyeonghokim/workspace/preflight/internal/hook/hook_test.go` or `/home/gyeonghokim/workspace/preflight/internal/tui/waiting_test.go` using injected failure/dummy renderer to assert exit code matches non-spinner baseline for same `review.Review` outcome (FR-008)

**Checkpoint**: US3 proven: animation cannot block or falsify push result

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Docs, gates, manual validation (including SC-001 / SC-003 per [spec.md](./spec.md))

- [ ] T023 [P] Update `/home/gyeonghokim/workspace/preflight/CLAUDE.md` Active Technologies to Bubbletea v2 (`charm.land/bubbletea/v2`) + Lipgloss v2 **`github.com/charmbracelet/lipgloss/v2`** (same as [research.md](./research.md) §2)
- [ ] T024 [P] Refresh `/home/gyeonghokim/workspace/preflight/specs/002-animated-waiting-spinner/quickstart.md` after implementation: final `go test` paths, **stdout vs stderr** (**FR-006**), SC-001 optional TTY helpers, SC-003 **100ms** objective note, interrupt sign-off, **Constitution IV** AI missing/timeout repro (align with T025)
- [ ] T025 Run `make lint`, `make test`, and `go build ./...` from `/home/gyeonghokim/workspace/preflight`; fix any regressions; complete manual checks from `/home/gyeonghokim/workspace/preflight/specs/002-animated-waiting-spinner/quickstart.md`. **Constitution IV (AI tool fail-open)**: confirm behavior unchanged after hook/async work — when the AI CLI is **missing** or **times out**, preflight **exits 0** and emits a **warning on stderr** (extend `/home/gyeonghokim/workspace/preflight/internal/hook/hook_test.go` or document reproducible manual steps in quickstart if no existing test covers this path)

---

## Dependencies & Execution Order

### Phase Dependencies

```text
Phase 1 (Setup)
  └── Phase 2 (Foundational v2 migration)   ← BLOCKS all user stories
        └── Phase 3 (US1)                    ← MVP: spinner + hook async + goldens + Ctrl+C (T017–T018)
              └── Phase 4 (US2)             ← stderr progress + stdout plain review (depends on hook layout from US1)
                    └── Phase 5 (US3)       ← error isolation; **logically** only needs US1 `waiting.go`, but **ordered after** Phase 4 to reduce `hook.go` merge churn (plain path lands before spinner fail-open hardening)
                          └── Phase 6 (Polish)
```

### User Story Dependencies

| Story | Depends on | Notes |
|-------|------------|--------|
| US1 (P1) | Phase 2 | Needs v2 stack + new `internal/anim` + `waiting.go` + hook async |
| US2 (P2) | US1 hook structure | Plain branch after US1 hook+waiting land (through T018); T019–T020 |
| US3 (P3) | US1 `waiting.go` (+ Phase 4 complete for hook stability) | Failure paths in spinner pipeline; Phase 4→5 ordering is merge hygiene, not a logical prerequisite |

### Within Each User Story

- US1: `anim` types before `spinner_view.go`; goldens after `RenderFrame` exists; hook refactor after local waiting model compiles; T017–T018 after spinner path is stable
- US2: Tests after plain progress implemented
- US3: Tests after error handling hooks exist

### Parallel Opportunities

- **Phase 1**: T002 and T003 parallel after T001
- **Phase 2**: T006 parallel with T005 once T004 done (both touch only `styles.go` vs `model.go`); T007 after T005–T006 stabilize compile
- **Phase 3**: T011 parallel with T012 after T010; T013 parallelizable once T012 API stable (fixtures + test file); T018 [P] after T017
- **Phase 4**: T020 [P] after T019 lands
- **Phase 5**: T022 [P] after T021
- **Phase 6**: T023 and T024 parallel

---

## Parallel Example: User Story 1

```bash
# After T010 completes, in parallel:
# - T011 liquidblob_test.go determinism tests
# - T012 spinner_view.go RenderFrame implementation (imports anim)

# After T012 completes, generate goldens and tests:
# - T013 internal/anim/testdata/spinner_golden/*.txt + spinner_view_test.go
#   Include SC-002 adjacent-tick metrics (see contracts/spinner-render.md §Adjacent-frame smoothness)
```

---

## Implementation Strategy

### MVP First (User Story 1 only)

1. Complete Phase 1 (T001–T003)
2. Complete Phase 2 (T004–T009) — **critical gate**
3. Complete Phase 3 (T010–T018)
4. **STOP and VALIDATE**: TTY smoke + `go test ./internal/anim/... ./internal/tui/...` + goldens + interrupt checklist
5. Demo spinner + review transition

### Incremental Delivery

1. Setup + Foundational → v2 baseline shipped internally
2. + US1 → MVP visual feature + regression snapshots
3. + US2 → plain path polish + SC-004 test
4. + US3 → FR-008 hardening
5. + Polish → docs + full gates

### Parallel Team Strategy

- Dev A: Phase 2 `model.go` + `hook.go`
- Dev B: Phase 2 `styles.go` + `model_test.go` / teatest
- After Phase 2: Dev A `internal/anim`, Dev B `spinner_view.go`, merge then goldens + `waiting.go`

---

## Task Summary

| Metric | Value |
|--------|-------|
| **Total tasks** | 25 (T001–T025) |
| **Phase 1** | 3 |
| **Phase 2** | 6 |
| **US1 (P1)** | 9 (T010–T018) |
| **US2 (P2)** | 2 (T019–T020) |
| **US3 (P3)** | 2 (T021–T022) |
| **Polish** | 3 (T023–T025) |
| **Tasks with [P]** | 11 |

**Format validation**: Every task uses `- [ ]`, sequential `TNNN`, optional `[P]`, `[USn]` only on story phases 3–5, and includes an absolute `/home/gyeonghokim/workspace/preflight/...` file path in the description.

---

## Notes

- Do not add `internal/util` or `helpers` packages; keep `internal/anim` and `internal/tui` boundaries per [plan.md](./plan.md)
- Golden updates must be intentional and reviewed (visual/text diff)
- Plain mode: **progress → stderr**, **review → stdout** — normative detail in **FR-006** / 헌장 IV
- **`plain.go`**: final plain review rendering; **stderr** wait lines are orchestrated in **`hook.go`** (T019), not duplicated in `plain.go`
