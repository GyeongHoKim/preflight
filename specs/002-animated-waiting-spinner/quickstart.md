# Quickstart: 002 Animated Waiting Spinner (planning / implementation)

**Date**: 2026-03-19  
**Branch**: `002-animated-waiting-spinner`  
**Spec**: [spec.md](./spec.md)  
**Plan**: [plan.md](./plan.md)  
**Tasks**: [tasks.md](./tasks.md)

---

## Scope of this feature

1. **Migrate** Bubbletea and Lipgloss to **v2**: `charm.land/bubbletea/v2` + **`github.com/charmbracelet/lipgloss/v2`** (locked in [research.md](./research.md) §2).
2. Implement a **2D liquid blob** waiting animation (see [research.md](./research.md)). The phrase **“liquid blob ring”** is a **visual nickname only** — it does **not** mean a geometric circular/torus band or annulus mask.
3. Keep **`--no-tui` / non-TTY** path free of animation ESC on **stdout**; **progress lines go to stderr**, final plain review to **stdout** (**FR-006**, 헌장 IV — see [spec.md](./spec.md) US2).
4. Add **deterministic golden tests** for animation frames via extracted render pipeline.

---

## Terminology (`RenderOpts` vs `RenderOptions`)

- **`internal/anim`**: `RenderOpts` (or equivalent name) — width, height, tick, seed for `ComputeFrame` only ([data-model.md](./data-model.md)).
- **`internal/tui`**: `RenderOptions` — adds `DisableANSI`, `DisableColor`, etc., for `RenderFrame` ([contracts/spinner-render.md](./contracts/spinner-render.md)).

---

## Prerequisites

Same as sibling feature [quickstart](../001-preflight-ai-review/quickstart.md): Go, `make`, golangci-lint.

---

## Commands

```bash
cd /home/gyeonghokim/workspace/preflight
make tidy            # after go.mod changes
make test            # must pass including new golden tests
make lint            # must exit 0
```

**Spinner / anim unit tests** (after implementation):

```bash
go test ./internal/anim/... -count=1
go test ./internal/tui/... -count=1
```

---

## Plain mode: stdout vs stderr

When verifying `--no-tui` or piped runs:

- **stdout**: only the final structured plain review from `PlainRender` — must not contain `0x1b` (SC-004).
- **stderr**: human-readable wait/progress messages — plain text, no TUI animation escapes.

Example capture:

```bash
./bin/preflight run --no-tui --provider claude 2>stderr.txt 1>stdout.txt
# Inspect both files per spec US2
```

---

## Manual checklist (SC-001, SC-003, interrupt)

Use a real TTY and a repo with a slow enough AI step (or mocked provider during dev):

| ID | Check | Pass criteria |
|----|--------|----------------|
| SC-001 | Start `git push` / `preflight run` on TTY; wait >0.5s before provider returns | Liquid-blob spinner visible for entire wait (not a literal circular ring shape) |
| SC-003 | When provider returns | Spinner disappears and review visible within **100ms** (objective target in [spec.md](./spec.md) SC-003; use `time` / screen recording if disputed) |
| FR-007 / SC-006 | Press Ctrl+C while spinner shows | Terminal returns to normal shell; no garbled alt-screen; next command works |

If T018 cannot automate teardown in CI, the **FR-007 / SC-006** row remains **mandatory** for release sign-off.

### Optional: reproducible SC-001 helper (non-TTY automation stays out of scope)

SC-001 needs a **TTY** and a wait **>0.5s**. When no slow provider is available:

- Run `preflight` from a **real terminal** (not CI headless) with a **stub** or **mock** subprocess that sleeps ≥600ms before emitting review JSON (dev-only harness), **or**
- Use `script(1)` to capture a session: `script -q -c "./bin/preflight run …" typescript.txt` and verify spinner glyphs appeared before the review block in the transcript.

These are **optional** developer aids; the table above remains the acceptance source of truth.

---

## Constitution IV: AI CLI missing / timeout (T025)

After hook/async refactors, re-verify **unchanged** behavior:

- **Given** configured provider binary is absent **or** review subprocess times out (per existing config), **Then** preflight **exits 0** and prints a **warning on stderr** (no silent block).

Add or extend a test in `internal/hook/hook_test.go` when feasible; otherwise document exact repro steps here after implementation.

---

## Implementation order (recommended)

1. **Dependencies**: `go get` Bubbletea v2 + Lipgloss v2; run `go mod tidy`; fix compile errors (`View` → `tea.View`, key/msg types, program options).
2. **Verify** `teatest` compatibility; if broken, switch `model_test.go` to v2-compatible patterns or rely more on golden + direct `Update` tests.
3. **Add** pure `anim` package + `RenderFrame` with `DisableANSI`.
4. **Wire** waiting phase into `hook.Run` + Bubbletea model (spinner while provider runs — async provider pattern: `tea.Cmd` wrapping provider call).
5. **Goldens**: add `testdata` and update workflow documented in [contracts/spinner-render.md](./contracts/spinner-render.md).
6. **Plain path**: stderr progress, stdout review only ([tasks.md](./tasks.md) T019–T020).

---

## Docs index (this feature)

| File | Purpose |
|------|---------|
| [spec.md](./spec.md) | User stories + FR/SC |
| [plan.md](./plan.md) | Technical plan + constitution check |
| [tasks.md](./tasks.md) | Implementation tasks |
| [research.md](./research.md) | v2 migration + math model decisions |
| [data-model.md](./data-model.md) | Entities and state machine |
| [contracts/spinner-render.md](./contracts/spinner-render.md) | Internal render/test contract |

---

## Agent context

After editing `plan.md`, refresh Cursor rules:

```bash
bash /home/gyeonghokim/workspace/preflight/.specify/scripts/bash/update-agent-context.sh cursor-agent
```
