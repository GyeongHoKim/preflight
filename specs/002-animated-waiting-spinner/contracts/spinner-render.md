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

  `ComputeFrame(config BlobRingConfig, opts RenderOpts) (Frame, error)`

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

## Versioning

Breaking changes to `Frame` layout or `RenderFrame` output for the same inputs require:

1. Bumping golden files in the same PR.
2. Note in CHANGELOG or commit message under this feature.

---

## Interaction with CLI contract

- User-facing flags remain governed by [contracts/cli.md](../../001-preflight-ai-review/contracts/cli.md) (`--no-tui`, exit codes).
- This contract is **internal**; it does not add new public CLI flags unless product owners extend the spec.
