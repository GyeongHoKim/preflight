<!--
SYNC IMPACT REPORT
==================
Version change: (none) → 1.0.0 (initial adoption)

Modified principles:
  - All principles are new (no prior version)

Added sections:
  - Core Principles (5 principles)
  - Technology Stack
  - Quality Gates
  - Governance

Removed sections:
  - N/A

Templates requiring updates:
  - .specify/templates/plan-template.md  ✅ — Constitution Check section already present;
    gate language aligns with principles defined here. No structural changes needed.
  - .specify/templates/spec-template.md  ✅ — Requirements + Success Criteria sections
    are compatible with Go CLI project structure. No changes needed.
  - .specify/templates/tasks-template.md ✅ — Phase 1 includes linting/formatting task
    slot (T003); path conventions support single-project Go layout. No changes needed.

Follow-up TODOs:
  - None. All placeholders resolved.
-->

# preflight Constitution

## Core Principles

### I. Go Standards Compliance (NON-NEGOTIABLE)

All code MUST conform to the Go Code Review Comments guide at
https://go.dev/wiki/CodeReviewComments. This includes, but is not limited to:

- **Naming**: Use `MixedCaps`, not underscores; acronyms follow Go conventions
  (e.g., `ServeHTTP`, not `ServeHttp`).
- **Comments**: All exported symbols MUST have doc comments beginning with the
  symbol name.
- **Error strings**: Error messages MUST NOT be capitalized or end with punctuation
  (they are composed with `fmt.Errorf`).
- **Receiver names**: MUST be short, consistent, and never `self` or `this`.
- **Package names**: Single lowercase words; avoid `util`, `common`, `misc`.
- **Imports**: Grouped in order: stdlib → external → internal, separated by blank
  lines.
- **Context**: `context.Context` MUST be the first parameter when accepted.

**Rationale**: Consistent adherence to the canonical Go style guide ensures the
codebase remains idiomatic, readable, and maintainable by any Go developer without
additional context.

After every code change, the implementer MUST verify compliance with this guide
before considering the task done.

### II. Zero-Lint Policy (NON-NEGOTIABLE)

After every code change, `make lint` MUST be run and MUST produce zero golangci-lint
errors before proceeding to the next task or marking work complete.

- No lint warnings may be suppressed with `//nolint` directives without an explicit,
  inline justification comment explaining why suppression is necessary.
- New `//nolint` directives require reviewer approval.

**Rationale**: Automated linting enforces style, detects bugs early, and prevents
technical debt from accumulating. The zero-error bar is intentional — any tolerance
creates drift.

### III. Explicit Error Handling

All errors MUST be handled explicitly. The following are prohibited:

- Assigning errors to `_` unless the function is documented as infallible.
- Silently discarding errors with empty `if err != nil {}` blocks.
- Panicking in response to recoverable runtime errors.

Error messages MUST provide sufficient context for diagnosis (use `fmt.Errorf("...:
%w", err)` wrapping). Errors that originate at system boundaries (user input, AI
CLI subprocess, file I/O) MUST surface a user-friendly message via stderr and exit
non-zero.

**Rationale**: preflight runs as a git hook. Silent failures or opaque errors
confuse users and erode trust in the tool. Every failure must be diagnosable.

### IV. CLI Interface Design

The CLI MUST follow these I/O conventions:

- **stdout**: Structured output (TUI or plain text results).
- **stderr**: Warnings, errors, and progress messages.
- **Exit codes**: `0` = success / no blocking issues; `1` = blocking issues found or
  internal error; `2` = usage/argument error.
- `--no-tui` flag MUST produce machine-parseable plain text output suitable for
  pipes and CI environments.
- The tool MUST exit `0` (never block silently) when the AI CLI is unavailable or
  times out — a warning MUST be emitted to stderr.

**Rationale**: preflight runs non-interactively as a git hook. Predictable I/O
contracts let users integrate it reliably into scripts, CI, and hook managers.

### V. Simplicity & Minimal Dependencies

- Prefer the Go standard library over third-party packages for straightforward tasks.
- Every external dependency MUST be justified. Introduce a dependency only when the
  standard library cannot reasonably satisfy the requirement.
- Avoid premature abstraction (YAGNI). Three similar concrete implementations are
  better than a premature interface.
- Internal packages MUST have a clear, single responsibility. Organizational-only
  packages (e.g., `util`, `helpers`) are prohibited.

**Rationale**: preflight is a single-binary CLI tool distributed via Homebrew and
`go install`. A lean dependency graph reduces supply chain risk, speeds up builds,
and lowers the maintenance burden.

## Technology Stack

- **Language**: Go (latest stable release)
- **Build & task runner**: `make` (Makefile at repository root)
- **Linter**: golangci-lint, invoked via `make lint`
- **Testing**: `go test ./...` with table-driven tests preferred
- **Distribution**: Single statically-linked binary; released via GitHub Releases,
  Homebrew tap (`gyeongho/tap/preflight`), and `go install`
- **Supported AI providers**: `claude`, `codex`, `gemini`, `qwen` (invoked as
  subprocess; no direct API usage by preflight itself)
- **Configuration**: `preflight.yml` (project-level) or
  `~/.config/preflight/config.yml` (global)

## Quality Gates

Every pull request and every completed implementation task MUST satisfy all of the
following before merge or sign-off:

1. **Lint gate**: `make lint` exits `0` with zero golangci-lint errors.
2. **Standards gate**: Implementer has verified compliance with
   https://go.dev/wiki/CodeReviewComments for all changed files.
3. **Test gate**: `go test ./...` passes; new behavior MUST be covered by at least
   one test.
4. **Build gate**: `go build ./...` succeeds with no warnings.
5. **Exit-code contract**: Manual or automated verification that the CLI exits with
   the correct code for success, blocking-issue, and error scenarios.

No task is considered done until all five gates pass.

## Governance

This constitution supersedes all other project practices. Conflicts between this
document and any other guidance file are resolved in favor of this constitution.

**Amendment procedure**:
1. Open a pull request with the proposed change to this file.
2. The PR description MUST explain: what is changing, why, and the migration plan
   for any existing code that no longer complies.
3. At least one reviewer MUST explicitly approve the constitutional change.
4. After merge, update `LAST_AMENDED_DATE` and bump `CONSTITUTION_VERSION`
   following semantic versioning:
   - **MAJOR**: Removal or redefinition of an existing principle.
   - **MINOR**: New principle or section added.
   - **PATCH**: Clarifications, wording fixes, non-semantic refinements.

**Compliance review**: All code reviews MUST verify constitution compliance.
Reviewers are empowered to block merges that violate any principle, regardless of
test or build status.

**Runtime guidance**: For session-level development guidance, refer to
`.specify/memory/` and any `CLAUDE.md` or agent context files present in the
repository root.

**Version**: 1.0.0 | **Ratified**: 2026-03-18 | **Last Amended**: 2026-03-18
