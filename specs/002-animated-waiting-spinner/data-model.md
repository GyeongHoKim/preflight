# Data Model: Animated Waiting Spinner

**Branch**: `002-animated-waiting-spinner`  
**Date**: 2026-03-19  
**Spec**: [spec.md](./spec.md)

This document describes **logical entities** for the waiting animation, renderer options, and Bubbletea orchestration. It is implementation-agnostic but maps cleanly to Go types.

---

## 1. `RenderOptions`

Controls how a `Frame` is turned into terminal output.

| Field | Type (conceptual) | Description |
|-------|-------------------|-------------|
| `Width` | int | Total width in cells (must be > 0 for layout). |
| `Height` | int | Total height in cells. |
| `Seed` | uint64 | Deterministic seed for blob phases / PRNG-derived parameters. |
| `DisableColor` | bool | Luminance/glyphs only; no foreground color ANSI. |
| `DisableANSI` | bool | No escape sequences at all (plain runes + spaces for layout). |
| `Tick` | int or `time.Duration` | Monotonic step index or elapsed time quantum for animation phase. |

**Validation**: `Width`/`Height` within reasonable bounds (e.g. 10–200); invalid combos fall back to safe defaults (documented in code).

---

## 2. `BlobRingConfig`

Parameters for the **mathematical model** (see research.md). Stored separately from runtime tick so tests can reuse configs.

| Field | Description |
|-------|-------------|
| `BlobCount` | Number of Gaussian blobs \(n\) on the ring. |
| `Sigma` | Angular width of each blob (radians or normalized scale). |
| `WaveNumber` | Integer \(k\) for \(\sin(k\theta - \omega t)\). |
| `WaveAmplitude` | Scalar \(W\) weight vs blob term. |
| `AngularSpeeds` | Per-blob \(\omega_i\) (may be PRNG-derived from `Seed`). |
| `PulseOmega` | \(\Omega\) for global brightness pulse. |
| `InnerRadiusFraction` | \(r_{\mathrm{in}}\) as fraction of half min(w,h). |
| `OuterRadiusFraction` | \(r_{\mathrm{out}}\) as fraction of half min(w,h). |

**Relationships**: Derived from `Seed` in production for variety; tests pin `Seed` + explicit config for goldens.

---

## 3. `Cell` / `SemanticCell`

Single terminal cell **before** ANSI styling.

| Field | Description |
|-------|-------------|
| `Rune` | Display character (space for empty). |
| `FG` | Optional color (palette index or RGB); ignored if `DisableColor`. |
| `BG` | Optional background; often unused for spinner. |

**Invariant**: `Rune` must be a single terminal cell wide (avoid ambiguous width runes in v1; document if using Braille).

---

## 4. `Frame`

Snapshot of the spinner at one instant.

| Field | Description |
|-------|-------------|
| `Cells` | 2D grid `[row][col]SemanticCell` matching `RenderOptions` width/height (or compact region with offset metadata). |
| `Tick` | The tick index this frame was built for (traceability in tests). |

**Relationships**: Produced only by pure `anim` package; consumed by `tui` stringifier.

---

## 5. `WaitingPhase` (TUI state machine)

High-level phase while `hook.Run` waits on the provider.

| Value | Meaning |
|-------|---------|
| `loading` | Provider running; show spinner animation. |
| `review` | Provider done; show `ReviewModel` (existing). |

**Transitions**:

- `loading` → `review` when provider returns and review is parsed successfully.
- `loading` → `review` or exit plain path on fail-open / error paths per existing hook rules (spinner must not affect exit code — spec FR-008).

---

## 6. `WaitingModel` (Bubbletea v2)

Extends or composes with review display.

| Field | Description |
|-------|-------------|
| `phase` | `WaitingPhase` |
| `tick` | int — incremented on tick messages while `loading`. |
| `seed` | uint64 — from options or time. |
| `renderOpts` | `RenderOptions` subset needed for layout. |
| `review` | `*review.Review` — nil while loading. |
| `err` | Optional soft error for spinner-only failures (must not block push). |

**State transitions**: Standard Elm cycle: `Init` may return tick `Cmd`; `Update` handles window size, keys, provider completion message (custom `tea.Msg`), tick.

---

## 7. Relationships diagram (conceptual)

```text
BlobRingConfig + Seed + Tick  --->  anim.ComputeFrame()  --->  Frame
                                                              |
RenderOptions + Frame --------->  tui.FrameString() --------> string (stdout / goldens)
```

---

## 8. Validation rules (from spec)

- **FR-006 / plain path**: When `--no-tui` or non-TTY, **no** spinner ANSI animation; simple text progress only (separate code path; not `FrameString` with ANSI).
- **FR-008**: Errors in animation rendering must be logged or ignored gracefully; **must not** change push blocking semantics.
