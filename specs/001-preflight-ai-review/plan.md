# Implementation Plan: preflight — AI-Powered Pre-Push Code Review

**Branch**: `001-preflight-ai-review` | **Date**: 2026-03-18 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/001-preflight-ai-review/spec.md`

---

## Summary

preflight is a Go CLI tool that installs as a git pre-push hook. On every `git push`, it collects the diff between the local branch and its upstream, invokes a locally installed AI CLI (claude, codex, gemini, or qwen) as a subprocess to perform a code review, and renders the findings in a Bubbletea TUI. If the AI identifies a critical issue, the push is blocked; the developer can override the block or cancel. When the AI CLI is unavailable or times out, preflight exits 0 (fail-open) so it never silently blocks a push. Distributed as a single statically-linked binary via goreleaser and Homebrew.

---

## Technical Context

**Language/Version**: Go (latest stable, ≥1.22)
**Primary Dependencies**: cobra, bubbletea, lipgloss, go-isatty, testify, yaml.v3
**Storage**: N/A (no persistent state; config read from YAML files, hook written to `.git/hooks/pre-push`)
**Testing**: `go test ./...` with testify; table-driven tests; mock Runner interface for os/exec
**Target Platform**: Linux + macOS (amd64 + arm64); Windows out of scope for v1
**Project Type**: CLI tool / git hook
**Performance Goals**: Review visible in terminal within 30 seconds for ≤500-line diffs (bottleneck is AI CLI, not preflight itself)
**Constraints**: CGO_ENABLED=0 (static binary); no direct API calls; subprocess-only AI integration; <5s overhead when diff is empty
**Scale/Scope**: Single-developer workstation tool; single git push at a time; no concurrency required

---

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

| Principle | Status | Notes |
|-----------|--------|-------|
| I. Go Standards Compliance | ✅ PASS | All exported symbols will have doc comments; naming follows Go conventions; imports grouped stdlib→external→internal |
| II. Zero-Lint Policy | ✅ PASS | golangci-lint runs via `make lint` after every change |
| III. Explicit Error Handling | ✅ PASS | Runner interface returns explicit errors; all subprocess errors wrapped with `fmt.Errorf("...: %w", err)`; fail-open logic explicit |
| IV. CLI Interface Design | ✅ PASS | stdout=TUI/plain-text, stderr=errors; exit 0/1/2 contract defined; `--no-tui` for CI; fail-open on AI unavailability |
| V. Simplicity & Minimal Dependencies | ✅ PASS | 6 external deps, each justified; yaml.v3 instead of viper; stdlib for exec/json/io; no util/helpers packages |

**No violations.** Complexity Tracking section not required.

---

## Project Structure

### Documentation (this feature)

```text
specs/001-preflight-ai-review/
├── plan.md              # This file (/speckit.plan command output)
├── research.md          # Phase 0 output — AI CLI schemas, git diff patterns, tech choices
├── data-model.md        # Phase 1 output — Finding, Review, PushInfo, Config entities
├── quickstart.md        # Phase 1 output — dev setup and project layout
├── contracts/
│   ├── cli.md           # CLI command/flag/exit-code contract
│   └── provider.md      # Per-provider invocation and response parsing contract
└── tasks.md             # Phase 2 output (/speckit.tasks command — NOT created here)
```

### Source Code (repository root)

```text
cmd/
└── preflight/
    └── main.go                   # Entry point; wires cobra root command; version vars

internal/
├── cli/                          # cobra command definitions
│   ├── root.go                   # Root command; PersistentPreRunE loads config + validates flags
│   ├── install.go                # `preflight install [--global] [--force]`; WriteHookScript(), IsManagedHook()
│   ├── install_test.go           # Tests for hook file creation, --force, idempotency
│   ├── uninstall.go              # `preflight uninstall [--global]`
│   └── run.go                    # `preflight run` — also called by the installed hook
│
├── config/                       # Config loading and validation
│   ├── config.go                 # Config struct, Load(), defaults, validation
│   └── config_test.go
│
├── diff/                         # Git diff collection
│   ├── collect.go                # ParsePushInfo() from stdin; CollectDiff() via git CLI
│   └── collect_test.go
│
├── provider/                     # AI CLI subprocess adapters
│   ├── runner.go                 # Runner interface; Detect() auto-detection; RealRunner
│   ├── claude.go                 # claude: -p --output-format json --json-schema
│   ├── codex.go                  # codex: -q --json
│   ├── gemini.go                 # gemini: --prompt --output-format json
│   ├── qwen.go                   # qwen: -p --output-format json (same pattern as claude)
│   └── runner_test.go            # Table-driven tests with MockRunner
│
├── review/                       # Review result types and JSON parsing
│   ├── review.go                 # Finding, Review, ProviderResult; ParseReview()
│   └── review_test.go
│
├── tui/                          # Terminal UI
│   ├── model.go                  # ReviewModel (Bubbletea); Update(); View()
│   ├── styles.go                 # lipgloss style definitions (colors, borders)
│   ├── plain.go                  # PlainRender() — --no-tui plain-text output
│   └── model_test.go             # Direct Update()/View() unit tests
│
└── hook/                         # Pre-push hook orchestration
    ├── hook.go                   # Run(): diff → provider → tui/plain → exit code
    └── hook_test.go              # Integration tests for exit code paths

.goreleaser.yaml                  # goreleaser release config (Linux/macOS, arm64+amd64)
.github/
└── workflows/
    └── release.yml               # GitHub Actions: release on tag push
Makefile                          # build, test, lint, fmt, tidy, setup targets
go.mod
go.sum
```

**Structure Decision**: Standard single Go module with `internal/` packages, each with a single clear responsibility. `cmd/preflight/main.go` is the only file in `cmd/`; all logic lives in `internal/`. This aligns with Constitution Principle V (no util/helpers packages; every package has a single responsibility).

---

## Implementation Phases

### Phase 1: Foundation — Config, Diff, and CLI Skeleton

**Goal**: A working binary that installs/uninstalls the hook and collects a diff, with no AI integration yet.

**Packages**: `cmd/preflight`, `internal/cli`, `internal/config`, `internal/diff`, `internal/hook`

**Deliverables**:
- `config.Load()` reads YAML from project-level and global paths, merges correctly, validates fields
- `diff.ParsePushInfo()` parses pre-push stdin format; handles new-branch and delete-push edge cases
- `diff.CollectDiff()` runs `git diff <remote>...<local>` via os/exec; returns `[]byte`; truncates at `MaxDiffBytes`
- `preflight install` writes hook script to `.git/hooks/pre-push`; respects `--force`
- `preflight uninstall` removes managed hook
- All quality gates pass

---

### Phase 2: Provider Adapters and Review Parsing

**Goal**: Each AI CLI can be invoked and its response parsed into a `Review`.

**Packages**: `internal/provider`, `internal/review`

**Deliverables**:
- `Runner` interface with `Run(ctx context.Context, diff []byte) (ProviderResult, error)`
- `claude.go`, `gemini.go`, `codex.go`, `qwen.go` each implement `Runner`
- `provider.Detect()` tries providers in order via `exec.LookPath`
- `review.ParseReview()` extracts canonical JSON from each provider's response envelope
- Fail-open conditions all return `nil, nil` with a warning written to provided `io.Writer`
- Table-driven unit tests with `MockRunner` covering: success, timeout, not-found, malformed JSON

---

### Phase 3: TUI and Plain-Text Renderer

**Goal**: Review results are displayed correctly in both interactive and non-interactive modes.

**Packages**: `internal/tui`

**Deliverables**:
- `ReviewModel` Bubbletea model with review panel + blocking prompt
- `model.Update()` handles: arrow keys to navigate findings, Enter to confirm, `y`/`n` shortcuts
- `styles.go`: critical=bold red, warning=yellow, info=dim cyan; 80-column width constraint
- `plain.PlainRender(w io.Writer, r *review.Review, branch string, commitCount int)` — testable without stdout
- TTY detection via `go-isatty`; auto-fallback to plain-text when no TTY
- `model_test.go` tests `Update()` with synthetic key messages; `View()` contains expected strings

---

### Phase 4: Hook Orchestration and Exit Code Contract

**Goal**: `internal/hook.Run()` wires all packages together and enforces the exit code contract.

**Packages**: `internal/hook`

**Deliverables**:
- `hook.Run(ctx context.Context, cfg *config.Config, stdin io.Reader, stdout, stderr io.Writer, noTUI bool) int` — returns exit code
- Correct exit codes for all paths: clean=0, blocked=1, overridden=0, fail-open=0, usage error=2
- `preflight run` command calls `hook.Run()` (no code duplication between hook and manual run)
- Integration test: synthetic git diff + `MockRunner` → assert correct exit code and output

---

### Phase 5: Distribution and Release

**Goal**: Single binary release on GitHub with Homebrew tap formula.

**Deliverables**:
- `.goreleaser.yaml`: CGO_ENABLED=0, linux+darwin, amd64+arm64, ldflags version injection, Homebrew tap
- `.github/workflows/release.yml`: trigger on `v*` tags, run goreleaser
- `Makefile` `release-dry-run` target: `goreleaser release --snapshot --clean`
- `cmd/preflight/main.go` exposes `Version`, `CommitSHA`, `BuildTime` vars for ldflags injection

---

## Dependency Justification

| Package | Why needed | stdlib alternative rejected because |
|---------|-----------|-------------------------------------|
| `github.com/spf13/cobra` | Subcommand parsing, help generation, flag binding | `flag` pkg has no subcommand support |
| `github.com/charmbracelet/bubbletea` | Elm-architecture TUI with keyboard handling | Raw `termios` manipulation is ~300 lines of platform-specific code |
| `github.com/charmbracelet/lipgloss` | TUI styling (colors, borders, width) | Standard companion to Bubbletea; not an independent choice |
| `github.com/mattn/go-isatty` | TTY detection for `--no-tui` auto-fallback | `os.File.Stat()` + `ModeCharDevice` is fragile across terminal emulators |
| `github.com/stretchr/testify` | `assert`/`require` in tests | stdlib `t.Errorf` patterns require significantly more boilerplate |
| `gopkg.in/yaml.v3` | YAML config file parsing | stdlib has no YAML support |
