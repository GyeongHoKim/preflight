# Research: Animated Waiting Spinner + Bubbletea/Lipgloss v2

**Branch**: `002-animated-waiting-spinner`  
**Date**: 2026-03-19

---

## 1. Bubbletea v2 module path and migration

**Decision**: Target **Bubbletea v2** using the published v2 module  
`charm.land/bubbletea/v2` (stable tags include `v2.0.0`–`v2.0.2` as of research date).  
Follow the upstream **`UPGRADE_GUIDE_V2.md`** (key changes: `View() tea.View` instead of `string`, `tea.NewView(...)`, `KeyPressMsg` / split mouse messages, alt-screen and mouse mode moved onto `tea.View` fields, `tea.NewProgram` simplified).

**Rationale**: v1 (`github.com/charmbracelet/bubbletea` v1.3.x) and v2 are different major APIs; the v2 line is the supported path for new features and documentation.

**Alternatives considered**:

- Stay on v1 — rejected: user requirement is explicit v2+ migration.
- Use only `github.com/charmbracelet/bubbletea` for v2 — **rejected**: public `go list` shows v1 line only on that path; v2 is distributed as `charm.land/bubbletea/v2` (verify during `go get`).

**Implementation note**: Replace imports across `internal/tui`, `internal/hook`, and tests. Re-audit `tea.WithOutput(stdout)` and program options against v2 API.

---

## 2. Lipgloss v2

**Decision**: Target **Lipgloss v2** exclusively via **`github.com/charmbracelet/lipgloss/v2`** (`v2.0.x`). Use this import path in `go.mod`, `internal/tui`, tests, and agent docs (`CLAUDE.md`).

**Rationale**: Pairs with Bubbletea v2 ecosystem; styling APIs remain `NewStyle()`, `Render()`, with layout helpers suitable for composing spinner + review layout. Locking one path removes “pick one” ambiguity in tasks and CI.

**Alternatives considered**:

- `charm.land/lipgloss/v2` — acceptable mirror but **not chosen** for this repo to keep a single canonical require line and match historical `github.com/charmbracelet/*` v1 layout.
- Keep Lipgloss v1 with Bubbletea v2 — **rejected**: mixed major versions increase friction and duplicate color/terminal stacks.

---

## 3. `teatest` vs extracted golden tests

**Decision**: **Primary regression strategy**: snapshot/golden tests on **pure frame renderer** output (strings with ANSI disabled). **Secondary**: keep or upgrade `github.com/charmbracelet/x/exp/teatest` only if it supports Bubbletea v2 after migration; if not, rely on thin integration tests (start program, feed messages) plus renderer goldens.

**Rationale**: User requirement is frame-accurate, deterministic regression tests; coupling snapshots to the full Bubbletea event loop is brittle. Renderer-level goldens are stable and fast.

**Alternatives considered**:

- teatest-only — rejected: harder to get deterministic multi-frame animation snapshots.
- E2E only — rejected: does not meet “frame snapshot” requirement.

---

## 4. Mathematical model: “liquid blob ring” (name ≠ geometry)

**Naming**: **“Liquid blob ring”** evokes looping, organic motion; it does **not** prescribe a **circular band**, **torus**, or **polar annulus** in the terminal. The visible silhouette must **not** be locked to a geometric circle/ring mask.

**Decision**: Use a **2D implicit scalar field** on a **rectangular spinner viewport** (terminal cells), in **Cartesian normalized coordinates** \((s_x, s_y) \in [0,1]^2\), discretized per cell.

### 4.1 Geometry

- Spinner occupies a **width × height** cell rectangle (from layout). Each cell maps to a sample point \((s_x, s_y)\) at cell center, normalized to the viewport.
- **No annulus / no polar ring mask**: do **not** restrict the field to \(r_{\mathrm{in}} \le r \le r_{\mathrm{out}}\) in polar coordinates. Optional **soft edge falloff** near the rectangle border (vignette) is allowed to keep the blob visually contained — this is **not** a circular ring.

### 4.2 Field (pure Go)

Define a **metaball-style** sum of 2D Gaussians plus a **planar traveling wave** and a **global pulse**:

\[
\Phi(s_x, s_y, t) = P(t) \cdot \left( B(s_x, s_y, t) + W \cdot \sin(\vec{k}\cdot(s_x, s_y) - \omega t) \right)
\]

- **Blobs**: \(B(s_x, s_y, t) = \sum_{i=1}^{n} \exp\left(-\frac{\|(s_x, s_y) - \mathbf{c}_i(t)\|^2}{2\sigma^2}\right)\)  
  with \(\mathbf{c}_i(t)\) moving inside \([0,1]^2\) (or wrapping at edges if desired for seamless flow). Initial positions and motion parameters are **seed-derived** for determinism.
- **Wave** (spec “wave”): traveling sinusoid in the **plane** (choose \(\vec{k}\) for direction; integer-related \(k_x, k_y\) give sequential lobe motion across the grid).
- **Pulse** (spec “pulse”): \(P(t) = 0.5 + 0.5\sin(\Omega t)\) (or smoothstep variant) scales overall intensity.

**Rationale**: 2D metaballs yield **liquid-like merging blobs** without imposing a circular topology; planar wave + pulse match FR-002–FR-004. Deterministic given \((t, \text{seed})\).

**Alternatives considered**:

- Annulus / polar “ring” field — **rejected**: conflicts with product intent (no geometric circular band).
- Precomputed ASCII art frames — rejected: heavy assets, no smooth evolution.
- 3D particle simulation — rejected: overkill for terminal resolution.
- Simple rotating spinner only — rejected: does not meet the liquid visual goal.

### 4.3 Discretization → glyphs and color

- Quantize \(\Phi\) into **luminance bands** mapped to a small rune ramp (e.g. ` ·░▒▓█` or Braille blocks for smoother look).
- **Gradient color**: map \((s_x, s_y, t)\) and/or \(\Phi\) to hue (HSV), then to Lipgloss colors; saturation/value modulated by \(\Phi\) and \(P(t)\).
- **No-color mode** (spec edge case): same luminance ramp, no ANSI color codes.

---

## 5. Separation Bubbletea vs rendering

**Decision**: Split into:

1. **`internal/anim`** (e.g. `liquidblob.go`): deterministic `ComputeFrame` / tick step where `Frame` is **semantic cells** (rune + optional FG/BG indices or RGB structs) — **no** Bubbletea, **no** Lipgloss.
2. **`internal/tui/spinner_view.go` (or similar)**: `Frame` → string via Lipgloss v2, controlled by `RenderOptions` (`DisableANSI`, `DisableColor`, dimensions).
3. **Bubbletea v2 model**: holds tick counter, calls into (2) in `View()`, schedules `tea.Tick` (or frame msg) while waiting; on review ready, switches to existing review UI.

**Rationale**: Golden tests import (1)+(2) with fixed tick/seed/size; Bubbletea only orchestrates time.

**Alternatives considered**: Lipgloss inside blob package — rejected: couples math tests to terminal styling.

---

## 6. Testability: seed and ANSI-off

**Decision**:

- **`Seed`**: derive blob initial positions, motion parameters, and \(\vec{k}\) components (within safe ranges) from a **PRNG** (`math/rand/v2` with explicit source) seeded for tests; production uses time-based seed or fixed aesthetic seed.
- **`DisableANSI` / plain output**: route Lipgloss through a **no-color / no-escape** renderer profile (Lipgloss v2 + `termenv`/output configured for no ANSI) so golden files are **plain text**; optional separate goldens with ANSI on for manual inspection only.

**Rationale**: Matches user ask for snapshot regression and stable automation; avoids framing as “CI product” while still supporting headless `go test`.

**Alternatives considered**: Strip ANSI with regex post-process — rejected: fragile vs Lipgloss internals.

---

## 7. Resolved clarifications (formerly NEEDS CLARIFICATION)

| Topic | Resolution |
|-------|------------|
| Exact v2 import path | Use `go get charm.land/bubbletea/v2@latest` and matching Lipgloss v2; lock versions in `go.mod`. |
| teatest compatibility | Validate during migration; goldens do not depend on it. |
| “Liquid blob ring” definition | **Not** a circular/torus band: 2D Cartesian metaball field in a rectangle + planar traveling wave + global pulse, discretized to glyphs/colors. |
| Frame snapshot location | Test `Frame` → string pipeline, not Bubbletea `View` string alone (unless ANSI off makes it stable). |

---

## References (external)

- Bubbletea v2 upgrade guide: `UPGRADE_GUIDE_V2.md` in bubbletea v2 repository.
- Charmbracelet Lipgloss v2 README for `NewStyle`, layout, rendering.
