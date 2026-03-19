# Quickstart: 002 Animated Waiting Spinner (planning / implementation)

**Date**: 2026-03-19  
**Branch**: `002-animated-waiting-spinner`  
**Spec**: [spec.md](./spec.md)  
**Plan**: [plan.md](./plan.md)

---

## Scope of this feature

1. **Migrate** Bubbletea and Lipgloss to **v2** (`charm.land/bubbletea/v2` + Lipgloss v2 module — verify exact paths in `go.mod`).
2. Implement a **liquid blob ring** waiting animation (see [research.md](./research.md)) while the AI provider runs.
3. Keep **plain-text / `--no-tui`** path free of animation ANSI (existing spec FR-006).
4. Add **deterministic golden tests** for animation frames via extracted render pipeline.

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

## Implementation order (recommended)

1. **Dependencies**: `go get` Bubbletea v2 + Lipgloss v2; run `go mod tidy`; fix compile errors (`View` → `tea.View`, key/msg types, program options).
2. **Verify** `teatest` compatibility; if broken, switch `model_test.go` to v2-compatible patterns or rely more on golden + direct `Update` tests.
3. **Add** pure `anim` package + `RenderFrame` with `DisableANSI`.
4. **Wire** waiting phase into `hook.Run` + Bubbletea model (spinner while provider runs — may require async provider pattern: `tea.Cmd` wrapping provider call).
5. **Goldens**: add `testdata` and update workflow documented in [contracts/spinner-render.md](./contracts/spinner-render.md).

---

## Docs index (this feature)

| File | Purpose |
|------|---------|
| [spec.md](./spec.md) | User stories + FR/SC |
| [plan.md](./plan.md) | Technical plan + constitution check |
| [research.md](./research.md) | v2 migration + math model decisions |
| [data-model.md](./data-model.md) | Entities and state machine |
| [contracts/spinner-render.md](./contracts/spinner-render.md) | Internal render/test contract |

---

## Agent context

After editing `plan.md`, refresh Cursor rules:

```bash
bash /home/gyeonghokim/workspace/preflight/.specify/scripts/bash/update-agent-context.sh cursor-agent
```
