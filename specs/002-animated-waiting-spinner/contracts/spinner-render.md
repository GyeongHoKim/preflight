# Contract: Spinner Frame Rendering (test surface)

**Type**: Internal package contract (pure logic + styled string projection)  
**Date**: 2026-03-19  
**Branch**: `002-animated-waiting-spinner`

---

## Purpose

Define the **stable boundary** between:

1. **Deterministic animation math** (no Bubbletea), and  
2. **Terminal string output** (Lipgloss v2, optional ANSI),

so that **golden / snapshot tests** do not depend on the Bubbletea event loop.

---

## Package responsibilities

### `anim` (name may be `internal/anim/...` — final name in implementation)

**MUST**:

- Export a function with semantics equivalent to:

  `ComputeFrame(config LiquidBlobConfig, opts RenderOpts) (Frame, error)`

  where `RenderOpts` includes at minimum: `Width`, `Height`, `Tick`, `Seed`.

- Use only stdlib + Go types for `Frame` (no Lipgloss imports).

**MUST NOT**:

- Import `bubbletea` or `lipgloss`.

**Determinism**: For identical `(config, opts.Width, opts.Height, opts.Tick, opts.Seed)`, output `Frame` MUST be bitwise identical across OS/arch.

---

### `tui` (spinner projection)

**MUST**:

- Export:

  `RenderFrame(frame Frame, opts RenderOptions) string`

  where `RenderOptions` includes:

  - `DisableANSI bool` — when true, output MUST contain **no** ESC (`0x1b`) sequences.
  - `DisableColor bool` — when true, no color SGR sequences (may still emit cursor/layout codes unless `DisableANSI`).

**SHOULD**:

- Use Lipgloss v2 for colored mode when `DisableANSI` is false.

---

## Golden test contract

**Test data layout** (recommended):

```text
internal/anim/testdata/spinner_golden/
  frame_000.txt
  frame_001.txt
  ...
```

**Format**:

- Each file is **UTF-8** text exactly as returned by `RenderFrame` with `DisableANSI: true` (and fixed `Width`×`Height`).
- Line endings: `\n` only (normalize in test if needed).

**Stability**: Updating goldens requires intentional `go test -update` (or project-specific flag); PRs must show visual diff in review.

---

## Adjacent-frame smoothness (SC-002)

Golden file equality alone does not prove **temporal** smoothness. Tests that consume `RenderFrame(..., RenderOptions{DisableANSI: true})` **MUST** also assert **spec SC-002** for a fixed `(width, height, seed)` and a bounded tick range:

1. For each adjacent pair `(t, t+1)`, overlay the two rendered grids (same dimensions). Count cells where runes differ. That count **MUST NOT** exceed `⌈0.35 × width × height⌉`.
2. Over the same tick range, the fraction of adjacent pairs whose rendered strings are **byte-identical** **MUST NOT** exceed **20%**.

Implementations may compute metrics on `RenderFrame` output or on an intermediate grid if the test compares rune-per-cell consistently with the spec. Thresholds are defined in [spec.md](../spec.md) SC-002.

---

## Versioning

Breaking changes to `Frame` layout or `RenderFrame` output for the same inputs require:

1. Bumping golden files in the same PR.
2. Note in CHANGELOG or commit message under this feature.

---

## Interaction with CLI contract

- User-facing flags remain governed by [contracts/cli.md](../../001-preflight-ai-review/contracts/cli.md) (`--no-tui`, exit codes).
- **Plain-mode I/O**: spinner frame rendering applies only to the TUI path. For `--no-tui` / non-TTY, wait **progress** text is written to **stderr** and the final plain review to **stdout** (project constitution IV — see [spec.md](../spec.md) US2).
- This contract is **internal**; it does not add new public CLI flags unless product owners extend the spec.
