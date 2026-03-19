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

**Decision**: Target **Lipgloss v2** via `github.com/charmbracelet/lipgloss/v2` or `charm.land/lipgloss/v2` (both list `v2.0.x`; pick one module path and use it consistently with `go mod tidy`).

**Rationale**: Pairs with Bubbletea v2 ecosystem; styling APIs remain `NewStyle()`, `Render()`, with layout helpers suitable for composing spinner + review layout.

**Alternatives considered**: Keep Lipgloss v1 with Bubbletea v2 — **rejected**: mixed major versions increase friction and duplicate color/terminal stacks.

---

## 3. `teatest` vs extracted golden tests

**Decision**: **Primary regression strategy**: snapshot/golden tests on **pure frame renderer** output (strings with ANSI disabled). **Secondary**: keep or upgrade `github.com/charmbracelet/x/exp/teatest` only if it supports Bubbletea v2 after migration; if not, rely on thin integration tests (start program, feed messages) plus renderer goldens.

**Rationale**: User requirement is frame-accurate, deterministic regression tests; coupling snapshots to the full Bubbletea event loop is brittle. Renderer-level goldens are stable and fast.

**Alternatives considered**:

- teatest-only — rejected: harder to get deterministic multi-frame animation snapshots.
- E2E only — rejected: does not meet “frame snapshot” requirement.

---

## 4. Mathematical model: “liquid blob ring”

**Decision**: Use a **closed-loop (periodic) implicit field on an annulus** in polar coordinates, discretized to the terminal grid.

### 4.1 Geometry

- Center \((c_x, c_y)\) from layout box; each cell \((x,y)\) maps to polar \((r, \theta)\) with \(\theta \in [0, 2\pi)\).
- **Annulus mask**: ring only where \(r_{\mathrm{in}} \le r \le r_{\mathrm{out}}\); outside mask the cell is blank (space).

### 4.2 Field (pure Go)

Define a **metaball-style** sum of wrapped Gaussians on the circle plus a **traveling wave** and a **global pulse** (brightness breathing):

\[
\Phi(\theta, t) = P(t) \cdot \left( B(\theta, t) + W \cdot \sin(k\theta - \omega t) \right)
\]

- **Blobs**: \(B(\theta, t) = \sum_{i=1}^{n} \exp\left(-\frac{\mathrm{wrap}(\theta - \theta_i(t))^2}{2\sigma^2}\right)\)  
  with \(\theta_i(t) = (\theta_{0,i} + \omega_i t) \bmod 2\pi\) and `wrap` the shortest angular distance on the circle.
- **Wave** (spec “wave”): traveling sinusoid around the ring with integer \(k\) lobes.
- **Pulse** (spec “pulse”): \(P(t) = 0.5 + 0.5\sin(\Omega t)\) (or smoothstep variant) scales overall intensity.

**Rationale**: Metaballs on a circle produce **organic merging “liquid”** silhouettes; the wave term adds sequential motion; the pulse adds periodic brightness change. All operations are standard float64 math, deterministic given \((t, \text{seed})\).

**Alternatives considered**:

- Precomputed ASCII art frames — rejected: heavy assets, no smooth evolution.
- 3D particle simulation — rejected: overkill for terminal resolution.
- Simple rotating spinner only — rejected: does not meet “liquid blob ring” visual goal.

### 4.3 Discretization → glyphs and color

- Quantize \(\Phi\) into **luminance bands** mapped to a small rune ramp (e.g. ` ·░▒▓█` or Braille blocks for smoother look).
- **Gradient color**: map \((\theta, t)\) to hue on an HSV wheel, then to Lipgloss colors; saturation/value modulated by \(\Phi\) and \(P(t)\).
- **No-color mode** (spec edge case): same luminance ramp, no ANSI color codes.

---

## 5. Separation Bubbletea vs rendering

**Decision**: Split into:

1. **`internal/anim/blobring` (name TBD in implementation)**: deterministic `Step(tick, seed) -> Frame` where `Frame` is **semantic cells** (rune + optional FG/BG indices or RGB structs) — **no** Bubbletea, **no** Lipgloss.
2. **`internal/tui/spinner_view.go` (or similar)**: `Frame` → string via Lipgloss v2, controlled by `RenderOptions` (`DisableANSI`, `DisableColor`, dimensions).
3. **Bubbletea v2 model**: holds tick counter, calls into (2) in `View()`, schedules `tea.Tick` (or frame msg) while waiting; on review ready, switches to existing review UI.

**Rationale**: Golden tests import (1)+(2) with fixed tick/seed/size; Bubbletea only orchestrates time.

**Alternatives considered**: Lipgloss inside blob package — rejected: couples math tests to terminal styling.

---

## 6. Testability: seed and ANSI-off

**Decision**:

- **`Seed`**: derive all \(\theta_{0,i}\), \(\omega_i\) (within safe ranges) from a **PRNG** (`math/rand/v2` with explicit source) seeded for tests; production uses time-based seed or fixed aesthetic seed.
- **`DisableANSI` / plain output**: route Lipgloss through a **no-color / no-escape** renderer profile (Lipgloss v2 + `termenv`/output configured for no ANSI) so golden files are **plain text**; optional separate goldens with ANSI on for manual inspection only.

**Rationale**: Matches user ask for snapshot regression and stable automation; avoids framing as “CI product” while still supporting headless `go test`.

**Alternatives considered**: Strip ANSI with regex post-process — rejected: fragile vs Lipgloss internals.

---

## 7. Resolved clarifications (formerly NEEDS CLARIFICATION)

| Topic | Resolution |
|-------|------------|
| Exact v2 import path | Use `go get charm.land/bubbletea/v2@latest` and matching Lipgloss v2; lock versions in `go.mod`. |
| teatest compatibility | Validate during migration; goldens do not depend on it. |
| “Liquid blob” definition | Metaball sum on circle + traveling wave + global pulse, discretized to glyphs/colors. |
| Frame snapshot location | Test `Frame` → string pipeline, not Bubbletea `View` string alone (unless ANSI off makes it stable). |

---

## References (external)

- Bubbletea v2 upgrade guide: `UPGRADE_GUIDE_V2.md` in bubbletea v2 repository.
- Charmbracelet Lipgloss v2 README for `NewStyle`, layout, rendering.
