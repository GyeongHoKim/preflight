---
description: "Task list for 003-private-ollama-review (Private Ollama provider with repository tools)"
---

# Tasks: Private Ollama provider with repository tools

**Input**: Design documents from `/home/gyeonghokim/workspace/preflight/specs/003-private-ollama-review/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/

**Tests**: The product constitution (`.specify/memory/constitution.md`, Quality Gates §3) requires **new behavior to be covered by at least one test**. Below, dedicated tasks cover `internal/ollama`, `internal/repotools`, `internal/diff` (repo root), `internal/provider` (Ollama runner), and `internal/hook` beyond `internal/config/config_test.go` (T005). Tests may follow implementation in the same PR as long as `go test ./...` passes before merge.

**Organization**: Tasks are grouped by user story for independent verification; foundational work blocks all stories.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no blocking dependencies on incomplete tasks in the same wave)
- **[Story]**: User story from `spec.md` (US1–US3)
- Setup and foundational phases: no story label

## Path Conventions

This repository uses `internal/` at the repo root (`cmd/preflight`, `internal/config`, `internal/provider`, etc.) — not `src/`.

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Package boundaries and shared git/repo helpers required by later phases

- [x] T001 Add `internal/ollama/doc.go` and `internal/repotools/doc.go` with package comments describing responsibilities per `specs/003-private-ollama-review/plan.md`
- [x] T002 [P] Add `internal/diff/repo_root.go` with `TopLevel(workingDir string) (string, error)` using `git rev-parse --show-toplevel` (or equivalent) for repository root resolution

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Config, HTTP client surface, and path safety that MUST exist before user-story integration

**⚠️ CRITICAL**: No user story phase work should merge until this phase completes

- [x] T003 Extend `internal/config/config.go` with Ollama-related fields matching `specs/003-private-ollama-review/data-model.md` (`BaseURL`, `Model`, limits, allow/deny paths, etc.)
- [x] T004 Update `internal/config/config.go` `validate()` and `validProviders` so `provider: ollama` is accepted and `base_url` / `model` are required when Ollama is selected per `specs/003-private-ollama-review/contracts/ollama-provider.md`
- [x] T005 Extend `internal/config/config_test.go` with valid and invalid YAML cases for `provider: ollama` and boundary validation errors
- [x] T006 Update `internal/cli/root.go` provider flag validation and help text to include `ollama` alongside existing providers
- [x] T007 Implement `internal/ollama/client.go` — `net/http` client for `POST {normalizedBaseURL}/api/chat` with `stream: false`, request body JSON encoding, response decoding, and context-aware timeouts per `specs/003-private-ollama-review/research.md`
- [x] T008 Implement `internal/ollama/types.go` (or split files) for chat request/response structs (`messages`, `tools`, `tool_calls`, `role: tool` follow-ups) aligned with `specs/003-private-ollama-review/contracts/ollama-provider.md`
- [x] T009 Implement `internal/repotools/paths.go` — resolve user paths under `RepoRoot`, apply `AllowedPathPrefixes` and `DeniedPathGlobs`, reject escapes outside root per `specs/003-private-ollama-review/data-model.md`
- [x] T010 Add `internal/repotools/executor.go` (or equivalent) constructor that binds `RepoRoot`, limit fields from config, and exposes a single dispatch entry for tool name + JSON arguments used by the provider

**Checkpoint**: Config loads; Ollama client can POST chat; repotools can classify paths — user story wiring can begin

---

## Phase 3: User Story 1 — Private inference path (Priority: P1) 🎯 MVP

**Goal**: Pre-push review traffic uses only the user-configured organization-controlled Ollama `base_url` for inference-related requests; disclosure explains what may leave the machine (FR-001, FR-002).

**Independent Test**: Configure `provider: ollama` and `ollama.base_url` to a controlled endpoint; observe (e.g. proxy logs or `tcpdump`) that review traffic targets only that origin for the HTTP client path.

### Implementation for User Story 1

- [x] T011 [US1] Update `README.md` with a "Private Ollama" section documenting trust boundary, what data is sent to `base_url`, and pointer to `.preflight.yml` keys (FR-002)
- [x] T012 [US1] Implement `internal/provider/ollama.go` — `OllamaRunner` struct implementing `provider.Runner`, `NewOllamaRunner(...)` taking `*config.Config`, repo root string, and `review` prompt/schema strings; `Run` must use only `internal/ollama` HTTP (no `claude`/`codex` subprocess) per `specs/003-private-ollama-review/contracts/ollama-provider.md`
- [x] T013 [US1] Update `internal/hook/hook.go` `buildRunner` to resolve repo root via `internal/diff` `TopLevel(wd)` and construct `OllamaRunner` when `cfg.Provider == "ollama"`
- [x] T014 [US1] Ensure `attempt` in `internal/hook/hook.go` passes provider name `"ollama"` to `review.ParseReview` when Ollama is selected so `Review.Provider` and envelope handling stay consistent

**Checkpoint**: Ollama provider can run end-to-end against a server (tool loop may still be minimal until US2); privacy documentation exists

---

## Phase 4: User Story 2 — Repository tools & grounded review (Priority: P2)

**Goal**: Closed set of repository tools (`list_files`, `read_file`, `search_repo`) with documented limits so the model can ground feedback in real paths and contents (FR-003–FR-005, FR-007).

**Independent Test**: Run review on a repo with known conventions; confirm findings cite concrete file paths/lines and tool results include explicit truncation when limits hit.

### Implementation for User Story 2

- [x] T015 [P] [US2] Implement `list_files` in `internal/repotools/list.go` with `MaxListEntries` cap and truncation notice in output
- [x] T016 [P] [US2] Implement `read_file` in `internal/repotools/read.go` with `MaxReadBytes`, binary skip behavior, and truncation notice per `specs/003-private-ollama-review/research.md`
- [x] T017 [P] [US2] Implement `search_repo` in `internal/repotools/search.go` with `MaxSearchMatches`, scan bounds, and truncation notice
- [x] T018 [US2] Wire tool name → handler in `internal/repotools/executor.go` (or `tools.go`) with JSON argument parsing errors returned as tool-visible messages
- [x] T019 [US2] Extend `internal/provider/ollama.go` with assistant↔tool loop: append tool results as `role: tool` messages until final assistant content or `MaxToolTurns` / timeout per FR-004/FR-005
- [x] T020 [US2] Register Ollama `tools` function definitions in chat requests (names, descriptions, JSON Schema parameters) matching repotools dispatch per `specs/003-private-ollama-review/contracts/ollama-provider.md`
- [x] T021 [US2] Drive model output to canonical review JSON using `review.Schema()` and `format`/prompt strategy per `specs/003-private-ollama-review/contracts/review-output-json.md` and `internal/review/prompt.go`
- [x] T022 [P] [US2] Optional: implement read-only `git_context` helper in `internal/repotools/git.go` behind strict argument allowlisting per `specs/003-private-ollama-review/research.md` §3

**Checkpoint**: Full tool suite + limits; review JSON matches existing parser expectations

---

## Phase 5: User Story 3 — Fail-open on unhealthy endpoint (Priority: P3)

**Goal**: Unreachable Ollama, misconfiguration, timeouts, and malformed JSON after retry do not block push; stderr shows actionable warnings (FR-006, constitution fail-open).

**Independent Test**: Stop Ollama or point `base_url` at a closed port; confirm hook exits `0` with stderr warning; confirm timeout produces non-blocking exit.

### Implementation for User Story 3

- [x] T023 [US3] Map transport/HTTP failures from `internal/ollama` and malformed final responses in `internal/provider/ollama.go` to errors that classify as fail-open where appropriate (align with `internal/hook/hook.go` expectations)
- [x] T024 [US3] Extend `internal/provider/runner.go` `ShouldFailOpen` (and related sentinels if needed) so Ollama-specific unavailable/timeout errors match subprocess fail-open behavior
- [x] T025 [US3] Verify `internal/hook/hook.go` plain and TUI paths emit clear stderr messages for Ollama fail-open cases without changing exit-code contract

**Checkpoint**: Simulated outage always yields non-blocking push path with visible warning

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Documentation alignment, governance notes, constitution test coverage, quality gates

- [x] T026 [P] Reconcile `specs/003-private-ollama-review/quickstart.md` planned YAML keys with implemented `internal/config/config.go` field names and document any deltas in `README.md`
- [x] T027 [P] Align `specs/003-private-ollama-review/plan.md` with `.specify/memory/constitution.md` v1.1.0+ (Ollama HTTP policy); remove obsolete “constitution conflict” wording

### Automated tests (Constitution Quality Gate 3)

**Purpose**: Each new package or materially new behavior has table-driven or `httptest` coverage before merge (`go test ./...`).

- [x] T028 [P] Add `internal/ollama/client_test.go` using `httptest.Server` to verify successful `POST /api/chat` (`stream: false`), non-2xx mapping, and context cancellation (depends on T007–T008)
- [x] T029 [P] Add `internal/repotools/paths_test.go` table-driven tests for allow/deny prefixes, glob denial, and path escape rejection (depends on T009)
- [x] T030 Add `internal/diff/repo_root_test.go` (or `repo_root_test.go` next to `repo_root.go`) verifying `TopLevel` against a temporary `git init` repository (depends on T002)
- [x] T031 [P] Add `internal/repotools/list_test.go`, `internal/repotools/read_test.go`, and `internal/repotools/search_test.go` (or a single `repotools_test.go` if preferred) covering caps, truncation notices, and executor dispatch errors (depends on T015–T018)
- [x] T032 Add `internal/provider/ollama_test.go` with `httptest` fake Ollama responses covering at least one happy-path review JSON and one fail-open-classified error path (depends on T012–T021 and T023–T024)
- [x] T033 Extend `internal/hook/hook_test.go` with cases for `provider: ollama` runner selection and fail-open stderr behavior where practical without a live server (depends on T013–T014, T023–T025)

### Final quality gates

- [x] T034 Run `make lint`, `go test ./...`, and `go build ./...` from repo root; fix any issues in touched packages

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1** → no prerequisites
- **Phase 2** → depends on Phase 1 — **blocks** all user stories
- **Phase 3 (US1)** → depends on Phase 2
- **Phase 4 (US2)** → depends on Phase 3 (runner + client + hook wiring must exist); tool implementations can start after Phase 2 **in parallel files** but integration (T019–T021) depends on T012–T018
- **Phase 5 (US3)** → depends on Phase 4 (realistic error paths through runner)
- **Phase 6** → documentation and constitution test tasks (T028–T033) should complete **after** the code they cover exists; **T034** (lint/test/build) is the final gate before merge

### User Story Dependencies

| Story | Depends on | Notes |
|-------|------------|--------|
| US1 (P1) | Foundational | No dependency on US2/US3 |
| US2 (P2) | US1 + Foundational | Extends same runner and repotools |
| US3 (P3) | US2 recommended | Error mapping should cover tool-loop and HTTP paths |

### Within Each User Story

- US2: implement list/read/search before registry dispatch (T015–T017 before T018); tool loop (T019) after dispatch (T018)

### Parallel Opportunities

- **Phase 1**: T001 and T002 [P] — different files
- **Phase 2**: After T003–T004 land config shape, T007–T008 [P] (ollama) vs T009–T010 [P] (repotools) can proceed in parallel on different files
- **US2**: T015, T016, T017 [P] — separate files; T022 [P] optional git tool in parallel once executor pattern exists
- **Phase 6 tests**: T028, T029, T031 [P] — different packages once implementations exist; T030 and T032–T033 may serialize on shared fixtures

---

## Parallel Example: User Story 2 (tool implementations)

```bash
# After T018 prerequisites exist, implement filesystem tools concurrently:
Task T015 → internal/repotools/list.go
Task T016 → internal/repotools/read.go
Task T017 → internal/repotools/search.go
```

---

## Implementation Strategy

### MVP First (User Story 1)

1. Complete Phase 1–2 (foundation)
2. Complete Phase 3 (US1): private endpoint wiring + disclosure + `OllamaRunner` calling `internal/ollama` only
3. **STOP and VALIDATE**: network boundary test for US1
4. US2 adds tools required for FR-003–FR-005 before calling the feature “complete” for spec compliance

### Incremental Delivery

1. Foundation → config + client + path rules
2. US1 → trusted endpoint + docs
3. US2 → repotools + tool loop + JSON schema alignment
4. US3 → fail-open parity
5. Polish → quickstart alignment + constitution tests (T028–T033) + lint/test (T034)

### Suggested MVP Scope

- **Minimum shippable slice for demo**: Phase 1–3 (US1) — proves private HTTP path and configuration
- **Spec-complete for FR-003+**: Must include Phase 4 (US2)

---

## Notes

- **Libraries**: No new `go.mod` dependency in Phase 1 per plan — use `net/http` and `encoding/json`; optional later spike on `github.com/ollama/ollama/api` only if justified
- **C2 / Quality Gate 3**: T028–T033 satisfy the constitution’s requirement that new behavior has automated test coverage; T034 confirms the full suite passes
- **[P]** tasks assume no conflicting edits to the same file in parallel
- Commit after each task or logical group; stop at checkpoints to validate independently
