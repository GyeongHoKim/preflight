# Tasks: preflight — AI-Powered Pre-Push Code Review

**Input**: Design documents from `/specs/001-preflight-ai-review/`
**Prerequisites**: plan.md ✅, spec.md ✅, research.md ✅, data-model.md ✅, contracts/ ✅, quickstart.md ✅

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.
**Tests**: Included inline per constitution requirement ("new behaviour MUST be covered by at least one test").

## Format: `[ID] [P?] [Story?] Description`

- **[P]**: Can run in parallel (different files, no dependencies on incomplete tasks)
- **[Story]**: Which user story this task belongs to (US1–US6)

---

## Phase 1: Setup (Project Initialization)

**Purpose**: Bare-minimum project scaffold that all subsequent tasks build on.

- [x] T001 Initialize Go module `github.com/GyeongHoKim/preflight` with `go mod init`, create `go.mod` and initial dependency list per plan.md in the repository root
- [x] T002 Create directory skeleton: `cmd/preflight/`, `internal/cli/`, `internal/config/`, `internal/diff/`, `internal/provider/`, `internal/review/`, `internal/tui/`, `internal/hook/`
- [x] T003 [P] Create `Makefile` with `build`, `test`, `lint`, `fmt`, `tidy`, `setup` targets per CLAUDE.md commands; set `./bin/preflight` as output binary
- [x] T004 [P] Create `.golangci.yml` golangci-lint configuration; enable `gofmt`, `goimports`, `govet`, `errcheck`, `staticcheck`, `godot`, `revive`
- [x] T005 Create stub `cmd/preflight/main.go` with `Version`, `CommitSHA`, `BuildTime` vars for ldflags injection and a cobra root command call

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core types and infrastructure that every user story depends on. No user story work can begin until this phase is complete.

**⚠️ CRITICAL**: Phases 3–8 all depend on this phase.

- [x] T006 Implement `Config` struct with fields `Provider`, `BlockOn`, `Timeout`, `PromptExtra`, `MaxDiffBytes` and `Load(projectPath, globalPath string) (*Config, error)` with default values and field validation in `internal/config/config.go`; write table-driven tests in `internal/config/config_test.go` covering valid YAML, missing file (uses defaults), invalid provider value, project-overrides-global
- [x] T007 [P] Implement `Finding`, `Review`, and `ProviderResult` structs in `internal/review/review.go`; add `SeverityRank(s string) int` for threshold comparison; write tests in `internal/review/review_test.go`
- [x] T008 [P] Define canonical review JSON schema constant and `SystemPrompt(extra string) string` builder in `internal/review/prompt.go`; the schema must match the `Finding` struct fields (`severity`, `category`, `message`, `location`) plus top-level `blocking bool` and `summary string`
- [x] T009 [P] Implement `PushInfo` struct with `LocalRef`, `LocalSHA`, `RemoteRef`, `RemoteSHA` fields; add `IsNewBranch() bool` and `IsDeletePush() bool` methods; implement `ParsePushInfo(r io.Reader) ([]PushInfo, error)` in `internal/diff/collect.go`; write tests in `internal/diff/collect_test.go` covering normal push, new branch (all-zero remote SHA), delete push, multi-ref stdin
- [x] T010 Implement `Runner` interface (`Run(ctx context.Context, diff []byte) (ProviderResult, error)`), sentinel errors `ErrProviderNotFound` and `ErrProviderTimeout`, and a `shouldFailOpen(err error) bool` stub (returns true for `ErrProviderNotFound` and `context.DeadlineExceeded`) in `internal/provider/runner.go`; add `MockRunner` struct for test injection; write tests for the sentinel errors and `shouldFailOpen` stub
- [x] T044 Implement `IsGitRepo(dir string) bool` in `internal/diff/collect.go` using `git rev-parse --git-dir`; call it from `hook.Run()` and return exit code `2` with message `"preflight: not a git repository"` to stderr if not in a repo; add test in `internal/diff/collect_test.go`

**Checkpoint**: Foundation ready — user story implementation can now begin.

---

## Phase 3: User Story 1 — Clean Push with No Issues (Priority: P1) 🎯 MVP

**Goal**: Developer runs `git push`, preflight collects the diff, invokes claude, displays a review summary (no critical issues), and the push completes automatically.

**Independent Test**: `make build && cd /tmp/clean-repo && git init && git remote add origin https://example.com/repo.git && <add non-problematic commit> && /path/to/bin/preflight run --provider claude` exits 0 and displays a review summary in the TUI.

- [x] T011 [US1] Implement `CollectDiff(ctx context.Context, info PushInfo) ([]byte, error)` in `internal/diff/collect.go` using `git diff <remote>...<local>` via `exec.CommandContext`; handle empty diff (return nil, emit info); truncate at `maxBytes` with a warning comment prepended; add tests in `internal/diff/collect_test.go`
- [x] T012 [US1] Implement `ClaudeRunner` in `internal/provider/claude.go` — invokes `claude -p "<prompt>" --output-format json --no-session-persistence --json-schema '<schema>'` with diff as stdin; parses response envelope (`result` field); wraps errors with context; returns `ErrProviderNotFound` when binary not in PATH
- [x] T013 [P] [US1] Implement `ParseReview(providerName string, raw ProviderResult) (*review.Review, error)` in `internal/review/review.go`; handles claude envelope (`result` field → JSON parse); falls back gracefully if JSON is malformed (returns nil, nil for fail-open cases); write tests in `internal/review/review_test.go` covering valid review, missing fields normalized, malformed JSON → nil
- [x] T014 [P] [US1] Add table-driven unit tests for `ClaudeRunner` in `internal/provider/runner_test.go` using `MockRunner`; cover: success with findings, empty diff skip, binary not found (`ErrProviderNotFound`), non-zero exit code (fail-open)
- [x] T015 [US1] Implement `ReviewModel` in `internal/tui/model.go` — Bubbletea model displaying findings list; `Init()`, `Update(msg)` (handles `tea.WindowSizeMsg`), `View()` renders findings panel with lipgloss; no blocking prompt yet (auto-exits after display for clean reviews); define lipgloss styles in `internal/tui/styles.go` (critical=bold red, warning=yellow, info=dim cyan, 80-column max width)
- [x] T016 [P] [US1] Write `model_test.go` in `internal/tui/` — unit-test `ReviewModel.Update()` with synthetic `tea.WindowSizeMsg`; assert `View()` output contains provider name and finding message text
- [x] T017 [US1] Implement `hook.Run(ctx context.Context, cfg *config.Config, stdin io.Reader, stdout, stderr io.Writer, noTUI bool) int` in `internal/hook/hook.go` — orchestrates: check git repo (calls `IsGitRepo`) → parse push info → collect diff → detect/run provider → parse review → render TUI (or plain text when `noTUI` is true) → return exit code; for clean-review path returns 0
- [x] T018 [US1] Implement `preflight run` cobra subcommand in `internal/cli/run.go`; wire `--provider`, `--timeout`, `--block-on` flags; call `hook.Run()`; add root command in `internal/cli/root.go` with `PersistentPreRunE` that loads config and validates `--provider` enum

**Checkpoint**: `preflight run --provider claude` on a branch with no critical issues must exit 0 and display a review.

---

## Phase 4: User Story 2 — Blocked Push with Developer Override (Priority: P2)

**Goal**: When the AI identifies a critical issue, the TUI blocks the push and presents Override / Cancel options. Developer can choose either path.

**Independent Test**: Introduce a hardcoded secret in a test diff, run `preflight run --provider claude`, observe the TUI prompts for action; press `y` (override) → exits 0; press `n` (cancel) → exits 1.

- [x] T019 [US2] Extend `ReviewModel` in `internal/tui/model.go` to add a blocking prompt mode: when `review.Blocking == true`, show "Push anyway? [y/n]" inline after the findings list; `Update()` handles `y` key → set `choice = "push_anyway"` → `tea.Quit()`; `n` / `ctrl+c` / `q` → set `choice = "cancel"` → `tea.Quit()`; `p.Run()` result carries the choice
- [x] T020 [P] [US2] Update `internal/tui/model_test.go` — add tests for blocking prompt: synthetic `tea.KeyMsg{Type: tea.KeyRune, Rune: 'y'}` sets choice to `"push_anyway"`; `'n'` sets `"cancel"`; verify `View()` renders the prompt when `blocking == true`
- [x] T021 [US2] Update `hook.Run()` in `internal/hook/hook.go` — after TUI exits: if `review.Blocking && choice == "cancel"` return 1; if `review.Blocking && choice == "push_anyway"` return 0; emit `"push override recorded"` to stderr when override chosen
- [x] T022 [P] [US2] Add `hook_test.go` in `internal/hook/` — integration tests using `MockRunner`; cover: clean review → exit 0; blocking + override → exit 0; blocking + cancel → exit 1

**Checkpoint**: Blocking push with Override/Cancel fully functional and tested.

---

## Phase 5: User Story 3 — AI Tool Unavailable (Fail-Open) (Priority: P3)

**Goal**: When the AI CLI is missing, times out, or returns unparseable output, the push is never blocked — a warning is emitted to stderr and exit code is 0.

**Independent Test**: Remove or rename the claude binary from PATH, run `preflight run`, observe warning message on stderr and exit code 0.

- [x] T023 [US3] Expand `shouldFailOpen(err error) bool` in `internal/provider/runner.go` (stub created in T010) — add `*exec.ExitError` classification: non-zero exit codes from the AI CLI subprocess are treated as fail-open; add inline comments explaining each case; note that auth/rate-limit failures are indistinguishable from other non-zero exits at this level and are intentionally fail-open to honour the never-block-a-push guarantee
- [x] T024 [P] [US3] Implement `Detect(providers []string) (string, error)` in `internal/provider/runner.go` using `exec.LookPath`; return `ErrProviderNotFound` when none are found; write tests covering: one found, none found, explicit provider not found
- [x] T025 [US3] Update `hook.Run()` in `internal/hook/hook.go` — when provider returns a fail-open condition, write `"warning: <provider> unavailable; skipping review"` to stderr and return 0; when `ParseReview` returns nil (malformed response), write `"warning: could not parse review; skipping"` to stderr and return 0
- [x] T026 [P] [US3] Add fail-open test cases to `internal/provider/runner_test.go` and `internal/hook/hook_test.go` — cover: binary not found → exit 0 + warning; timeout → exit 0 + warning; empty stdout → exit 0 + warning; malformed JSON → exit 0 + warning

**Checkpoint**: All fail-open paths verified; push never silently blocked by tool-side failures.

---

## Phase 6: User Story 4 — First-Time Installation (Priority: P4)

**Goal**: Developer installs preflight as a git pre-push hook with a single command.

**Independent Test**: `preflight install` in a git repo writes `.git/hooks/pre-push`; running `git push` fires the hook.

- [x] T027 [US4] Implement `WriteHookScript(hooksDir string, force bool) error` and `IsManagedHook(path string) bool` in `internal/cli/install.go` — `WriteHookScript` writes the managed hook script (`#!/bin/sh\n# Managed by preflight. Run \`preflight uninstall\` to remove.\nexec preflight run "$@"\n`) to `<hooksDir>/pre-push`; checks for existing unmanaged hook and returns error unless `force == true`; marks file executable (0755); `IsManagedHook` checks for the managed comment header
- [x] T028 [P] [US4] Implement `preflight install [--global] [--force]` cobra subcommand in `internal/cli/install.go`; resolves hooks dir from `.git/hooks` (local) or `~/.config/git/hooks` (global); calls `WriteHookScript()`; prints confirmation to stdout; on unmanaged-hook-exists error prints `"preflight: existing hook found at <path>; use --force to replace"` to stderr and exits 1
- [x] T029 [P] [US4] Implement `preflight uninstall [--global]` cobra subcommand in `internal/cli/uninstall.go`; calls `IsManagedHook()` before removing; warns to stderr if hook is not managed by preflight (does not remove it in that case)
- [x] T030 [P] [US4] Write tests in `internal/cli/install_test.go` — cover: install into empty hooks dir; install with existing managed hook (idempotent); install with unmanaged hook without --force → error + message; install with --force → overwrites; uninstall managed hook; uninstall non-managed hook → warning, file preserved

**Checkpoint**: `preflight install` and `preflight uninstall` work correctly; hook fires on `git push`.

---

## Phase 7: User Story 5 — Headless / CI Mode (Priority: P5)

**Goal**: `--no-tui` and auto-TTY-detection produce machine-parseable plain-text output; no TUI is launched in non-interactive environments.

**Independent Test**: `git diff HEAD~1 | preflight run --no-tui --provider claude > output.txt`; `output.txt` contains human-readable findings; `grep -i "critical\|warning\|no issues" output.txt` succeeds.

- [x] T031 [US5] Implement `PlainRender(w io.Writer, r *review.Review, branch string, commitCount int)` in `internal/tui/plain.go` — writes the structured plain-text format defined in `contracts/cli.md`: header line, one `[SEVERITY] category — location\n  message` block per finding, footer line with counts and push-blocked/allowed verdict; write tests in `internal/tui/plain_test.go` covering: clean review output, blocked review output, empty findings
- [x] T032 [P] [US5] Add `isTTY(stdout *os.File) bool` using `github.com/mattn/go-isatty` in `internal/tui/model.go`; auto-fall-back to `PlainRender` when no TTY regardless of `--no-tui` flag
- [x] T033 [P] [US5] Wire `--no-tui` flag in `internal/cli/run.go` to pass `noTUI bool` into `hook.Run()` (signature already accepts `noTUI bool` from T017); implement the `noTUI || !isTTY()` branch in `hook.Run()` to call `PlainRender` instead of launching Bubbletea; add test verifying `--no-tui` produces plain-text and not TUI output

**Checkpoint**: `preflight run --no-tui` produces parseable plain-text; piped invocations auto-detect non-TTY.

---

## Phase 8: User Story 6 — Provider Selection and Configuration (Priority: P6)

**Goal**: Developer can select a provider via config file or `--provider` flag; auto-detection tries providers in order; project config overrides global.

**Independent Test**: Create `preflight.yml` with `provider: gemini`; run `preflight run`; observe gemini binary is invoked (visible via verbose flag or process list).

- [x] T034 [US6] Implement `GeminiRunner` in `internal/provider/gemini.go` — invokes `gemini --prompt "<prompt>" --output-format json` with diff embedded in prompt string; parses `response` field from JSON envelope; handles exit codes 42 and 53 as fail-open; write tests in `internal/provider/runner_test.go`
- [x] T035 [P] [US6] Implement `CodexRunner` in `internal/provider/codex.go` — invokes `codex -q --json "<prompt + diff>"`; for diffs exceeding 100 KB, write diff to a temp file using `os.CreateTemp("", "preflight-diff-*")` with `defer os.Remove(path)` and embed `"\n\nSee diff content in: <path>"` at the end of the prompt string; best-effort JSON parse of stdout (try candidate fields in order: `output`, `response`, `content`, `result`; fall back to raw stdout if none match); write tests
- [x] T036 [P] [US6] Implement `QwenRunner` in `internal/provider/qwen.go` — identical invocation pattern to `ClaudeRunner` (`-p --output-format json --no-session-persistence --json-schema`); write tests
- [x] T037 [P] [US6] Update `Config.Load()` in `internal/config/config.go` to fully parse and validate all fields: `provider`, `block_on`, `timeout` (Go duration), `prompt_extra`, `max_diff_bytes`; verify project-level config correctly overrides global-level defaults; add missing test cases to `internal/config/config_test.go`
- [x] T038 [US6] Wire provider selection end-to-end in `internal/cli/root.go` `PersistentPreRunE`: flag `--provider` overrides config `provider`; `"auto"` triggers `provider.Detect()`; invalid provider value returns exit code 2; add `--verbose` flag that logs detected provider to stderr; add table-driven test in `internal/cli/` covering priority order: (1) explicit flag beats config value, (2) config value beats auto-detect, (3) auto-detect uses first found provider, (4) invalid provider → exit 2

**Checkpoint**: All 4 providers selectable via flag or config; auto-detection works; project config overrides global.

---

## Phase 9: Polish & Cross-Cutting Concerns

**Purpose**: Distribution, version command, final quality gates.

- [x] T039 Implement `preflight version` cobra subcommand in `internal/cli/root.go` — prints `preflight vX.Y.Z (commit abc1234, built 2026-03-18T00:00:00Z)` using injected ldflags vars from `cmd/preflight/main.go`
- [x] T040 [P] Create `.goreleaser.yaml` at repository root — `CGO_ENABLED=0`, `goos: [linux, darwin]`, `goarch: [amd64, arm64]`, `ldflags: [-s -w -X main.Version={{.Version}} -X main.CommitSHA={{.ShortCommit}} -X main.BuildTime={{.Date}}]`, `archives: [{format: tar.gz}]`, `brews` section targeting Homebrew tap
- [x] T041 [P] Create `.github/workflows/release.yml` — trigger on `push: tags: [v*]`; steps: checkout with `fetch-depth: 0`, setup-go stable, goreleaser-action@v5 with `GITHUB_TOKEN`
- [x] T042 [P] Add `release-dry-run` target to `Makefile`: `goreleaser release --snapshot --clean`
- [x] T043 Run full quality-gate checklist: `make lint` (must exit 0), `make test` (must exit 0 with race detector), `go build ./...` (must succeed); manually verify all exported symbols have Go-style doc comments beginning with the symbol name (per constitution Principle I and https://go.dev/wiki/CodeReviewComments#doc-comments); verify exit-code contract for all paths (clean=0, blocked=1, override=0, fail-open=0, usage-error=2, not-a-git-repo=2)

---

## Dependencies & Execution Order

### Phase Dependencies

```
Phase 1 (Setup)
  └── Phase 2 (Foundational)   ← BLOCKS everything below
        ├── Phase 3 (US1 - Clean Push)      🎯 MVP
        │     └── Phase 4 (US2 - Blocked)   builds on hook.Run() and tui model
        │           └── Phase 5 (US3 - Fail-Open)  adds error classification to existing code
        ├── Phase 6 (US4 - Install)          independent of US1-US3 TUI code
        ├── Phase 7 (US5 - Headless)         extends US1 with plain renderer
        └── Phase 8 (US6 - Providers)        extends US1 with 3 more adapters
              └── Phase 9 (Polish)
```

### User Story Dependencies

| Story | Depends On | Reason |
|-------|-----------|--------|
| US1 (P1) | Phase 2 only | Foundational types + claude adapter sufficient for MVP |
| US2 (P2) | US1 | Extends `ReviewModel` and `hook.Run()` from US1 |
| US3 (P3) | US1, US2 | Adds fail-open error paths to completed orchestration |
| US4 (P4) | Phase 2 only | `install` command is independent of TUI/provider work |
| US5 (P5) | US1 | `PlainRender` needs `Review` type; extends `hook.Run()` |
| US6 (P6) | US1, US5 | Adds 3 more providers; needs working orchestration |

### Within Each User Story

- Types/structs before services that use them
- `MockRunner` in foundational phase enables all provider tests
- `hook.Run()` grows incrementally: US1 (clean) → US2 (blocking) → US3 (fail-open) → US5 (no-tui)
- Provider adapters (T034–T036) are fully parallel once `Runner` interface exists

### Parallel Opportunities

- **Phase 1**: T003, T004 are fully parallel after T001–T002
- **Phase 2**: T007, T008, T009, T010 are fully parallel after T006
- **Phase 3**: T013, T014, T016 are fully parallel; T012 and T015 can proceed in parallel after T011 completes
- **Phase 6**: US4 (T027–T030) is fully parallel with US3 and US5 work
- **Phase 8**: T034, T035, T036, T037 are fully parallel

---

## Parallel Example: User Story 1

```
After T010 (Runner interface + MockRunner):

  Parallel track A:               Parallel track B:              Parallel track C:
  T012 (ClaudeRunner)             T015 (ReviewModel TUI)         T013 (ParseReview)
  T014 (ClaudeRunner tests)       T016 (model tests)             (tests for parsing)
        ↓ both complete                   ↓ complete
  T017 (hook.Run - clean path)          (merge into hook.Run)
  T018 (preflight run command)
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1 (Setup) — ~5 tasks
2. Complete Phase 2 (Foundational) — ~5 tasks, can partially parallelize
3. Complete Phase 3 (US1 — Clean Push) — ~8 tasks
4. **STOP and VALIDATE**: `preflight run --provider claude --no-tui` on a test repo exits 0 and shows findings
5. This is already a usable tool for the common happy path

### Incremental Delivery

1. **v0.1** (Phase 1–3): `preflight run --provider claude --no-tui` — clean push review
2. **v0.2** (+ Phase 4): Interactive TUI with block/override — fully safe gating
3. **v0.3** (+ Phase 5): Fail-open guarantees — production-safe hook
4. **v0.4** (+ Phase 6): `preflight install` — one-command setup
5. **v0.5** (+ Phase 7): `--no-tui`, CI mode
6. **v1.0** (+ Phase 8–9): All 4 providers, config file, goreleaser distribution

---

## Notes

- **[P]** tasks touch different files with no shared incomplete dependencies — safe to parallelize
- **[USN]** labels map each task to a user story for traceability back to spec.md
- `hook.Run()` is the single integration point — it grows across US1→US2→US3→US5 phases; keep it small and delegate to packages
- Commit after each task or logical group; run `make lint && make test` before each commit (lefthook enforces this)
- Stop at Phase 3 checkpoint to validate MVP before proceeding
- All 44 tasks (T001–T043 + T044) verified to have exact file paths and follow the checklist format
