# Data Model: Animated Waiting Spinner

**Branch**: `002-animated-waiting-spinner`  
**Date**: 2026-03-19  
**Spec**: [spec.md](./spec.md)

This document describes **logical entities** for the waiting animation, renderer options, and Bubbletea orchestration. It is implementation-agnostic but maps cleanly to Go types.

---

## 1. `RenderOpts` (package `anim`)

Inputs to **`ComputeFrame(config LiquidBlobConfig, opts RenderOpts)`** — see [contracts/spinner-render.md](./contracts/spinner-render.md). **No** Lipgloss/Bubbletea; **no** `DisableANSI` / `DisableColor` here (those apply only when projecting a `Frame` to a string).

| Field | Type (conceptual) | Description |
|-------|-------------------|-------------|
| `Width` | int | Spinner viewport width in cells (> 0). |
| `Height` | int | Spinner viewport height in cells (> 0). |
| `Tick` | int | Monotonic animation step index. |
| `Seed` | uint64 | Deterministic seed for blob motion / parameters. |

**Validation**: `Width`/`Height` within reasonable bounds (e.g. 10–200); invalid combos fall back to safe defaults (documented in code).

---

## 2. `RenderOptions` (package `tui`)

Controls how a **`Frame`** becomes terminal output in **`RenderFrame(frame Frame, opts RenderOptions) string`**.

| Field | Type (conceptual) | Description |
|-------|-------------------|-------------|
| `DisableColor` | bool | Luminance/glyphs only; no foreground color ANSI. |
| `DisableANSI` | bool | No escape sequences at all (plain runes + spaces for layout); used for goldens ([spec.md](./spec.md) SC-002). |

Layout size is taken from the `Frame` (or overridden if the implementation stores dimensions on `Frame`); do not conflate this struct with `RenderOpts`.

**Hook / plain mode (not part of `Frame` / `RenderFrame`)**: While the provider runs in `--no-tui` or non-TTY mode, **wait progress strings are emitted on stderr**; **`PlainRender` review output uses stdout only** (constitution IV, [spec.md](./spec.md) FR-006).

---

## 3. `LiquidBlobConfig`

Parameters for the **2D Cartesian liquid-blob field** (see research.md). **Not** an annulus or polar “ring” geometry — blobs live in a normalized \([0,1]^2\) viewport mapped to the spinner rectangle. Stored separately from runtime tick so tests can reuse configs.

| Field | Description |
|-------|-------------|
| `BlobCount` | Number of 2D Gaussian blobs \(n\). |
| `Sigma` | Spatial spread of each blob in normalized coordinates (same units as \(s_x, s_y\)). |
| `WaveVectorX`, `WaveVectorY` | Components of \(\vec{k}\) for \(\sin(\vec{k}\cdot(s_x,s_y) - \omega t)\) (or equivalent packed fields). |
| `WaveAngularFreq` | \(\omega\) for the traveling wave term. |
| `WaveAmplitude` | Scalar \(W\) weight vs metaball term. |
| `PulseOmega` | \(\Omega\) for global brightness pulse \(P(t)\). |
| `EdgeSoftness` | Optional vignette / soft falloff at the **rectangular** viewport edge (not a circular mask). |

Blob center paths \(\mathbf{c}_i(t)\) and any motion speeds are **seed-derived** (or fixed constants) for determinism; exact representation is implementation-defined but must stay inside (or wrap) the unit square per research.md.

**Relationships**: Derived from `Seed` in production for variety; tests pin `Seed` + explicit config for goldens.

---

## 4. `Cell` / `SemanticCell`

Single terminal cell **before** ANSI styling.

| Field | Description |
|-------|-------------|
| `Rune` | Display character (space for empty). |
| `FG` | Optional color (palette index or RGB); ignored if `DisableColor`. |
| `BG` | Optional background; often unused for spinner. |

**Invariant**: `Rune` must be a single terminal cell wide (avoid ambiguous width runes in v1; document if using Braille).

---

## 5. `Frame`

Snapshot of the spinner at one instant.

| Field | Description |
|-------|-------------|
| `Cells` | 2D grid `[row][col]SemanticCell` matching the `RenderOpts.Width` × `RenderOpts.Height` used in `ComputeFrame` (or compact region with offset metadata). |
| `Tick` | The tick index this frame was built for (traceability in tests). |

**Relationships**: Produced only by pure `anim` package; consumed by `tui` stringifier.

---

## 6. `WaitingPhase` (TUI state machine)

High-level phase while `hook.Run` waits on the provider.

| Value | Meaning |
|-------|---------|
| `loading` | Provider running; show spinner animation. |
| `review` | Provider done; show `ReviewModel` (existing). |

**Transitions**:

- `loading` → `review` when provider returns and review is parsed successfully.
- `loading` → `review` or exit plain path on fail-open / error paths per existing hook rules (spinner must not affect exit code — spec FR-008).

---

## 7. `WaitingModel` (Bubbletea v2)

Extends or composes with review display.

| Field | Description |
|-------|-------------|
| `phase` | `WaitingPhase` |
| `tick` | int — incremented on tick messages while `loading`. |
| `seed` | uint64 — from options or time. |
| `animOpts` | `RenderOpts` passed into `ComputeFrame` (width, height, tick, seed). |
| `projectionOpts` | `RenderOptions` passed into `RenderFrame` (`DisableANSI`, `DisableColor`, …). |
| `review` | `*review.Review` — nil while loading. |
| `err` | Optional soft error for spinner-only failures (must not block push). |

**State transitions**: Standard Elm cycle: `Init` may return tick `Cmd`; `Update` handles window size, keys, provider completion message (custom `tea.Msg`), tick.

---

## 8. Relationships diagram (conceptual)

```text
LiquidBlobConfig + RenderOpts  --->  anim.ComputeFrame()  --->  Frame
                                                              |
RenderOptions + Frame --------->  tui.RenderFrame() --------> string (stdout / goldens)
```

---

## 9. Validation rules (from spec)

- **FR-006 / plain path**: When `--no-tui` or non-TTY, **no** spinner ANSI animation; simple text progress only (separate code path; not `RenderFrame` with ANSI on stdout).
- **SC-002**: Golden tests enforce 인접 틱 셀 변경 상한 및 정적 프레임 쌍 비율 상한 — see [spec.md](./spec.md) SC-002.
- **FR-008**: Errors in animation rendering must be logged or ignored gracefully; **must not** change push blocking semantics.
