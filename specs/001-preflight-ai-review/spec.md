# Feature Specification: preflight — AI-Powered Pre-Push Code Review

**Feature Branch**: `001-preflight-ai-review`
**Created**: 2026-03-18
**Status**: Draft
**Input**: User description: "Build a git pre-push hook tool called preflight. When a developer runs `git push`, preflight intercepts the push, collects the diff between the local branch and its upstream, and sends it to a locally installed AI CLI tool for code review. The review result is displayed in a terminal UI so the developer can read the feedback before the push completes. If the AI identifies a critical issue, the push is blocked by default. The developer can choose to override the block and push anyway, or cancel the push to address the issues first. The goal is to help individual developers catch serious problems — security risks, silent error discarding, logic bugs — before code leaves their machine, without requiring any API tokens. preflight is intended for local developer workstations only — developers who prefer to validate changes on their own machine before pushing, not for automated server-side pipelines."

## User Scenarios & Testing *(mandatory)*

### User Story 1 — Clean Push with No Issues (Priority: P1)

A developer finishes work on a feature branch and runs `git push`. preflight intercepts the push, collects the code changes since the last upstream sync, and submits them to the locally installed AI tool for review. The AI finds no critical problems. preflight displays the review summary in a readable terminal view, then allows the push to complete automatically.

**Why this priority**: This is the most common daily interaction. The tool must stay out of the developer's way on green paths and impose no friction when code is clean.

**Independent Test**: Install preflight as a git hook on a repository, commit non-problematic code, run `git push`, observe the terminal UI displays a review summary, and confirm the push completes without intervention.

**Acceptance Scenarios**:

1. **Given** preflight is installed as a pre-push hook and a locally supported AI tool is available, **When** the developer runs `git push` with no blocking issues in the diff, **Then** the terminal shows a review summary and the push proceeds automatically without requiring user input.
2. **Given** the diff between the local branch and its upstream contains only non-critical observations, **When** the AI review completes, **Then** the exit code is `0` and the push is not blocked.
3. **Given** the repository has no uncommitted or unpushed changes beyond the expected diff, **When** preflight runs, **Then** it collects exactly the changes between the local branch tip and the upstream tracking branch.

---

### User Story 2 — Blocked Push with Developer Override (Priority: P2)

A developer pushes code that contains a critical issue (e.g., a hardcoded secret, a silently discarded error, a logic bug with data-loss potential). preflight displays the AI's findings prominently in the terminal UI and blocks the push. The developer reads the feedback and decides to either fix the issue before pushing, or explicitly acknowledge the risk and force the push through with an override.

**Why this priority**: This is the core safety mechanism. A blocked push that the developer cannot read or act on is useless; a blocked push with no override path creates hostile friction. Both the clear display and the override option are essential.

**Independent Test**: Introduce a known security anti-pattern (e.g., a hardcoded credential string) in a commit, run `git push`, observe the terminal UI highlights the critical finding, confirm the push is blocked, then select the override option and confirm the push proceeds.

**Acceptance Scenarios**:

1. **Given** the diff contains a pattern the AI classifies as a critical issue, **When** the review completes, **Then** the terminal UI displays the specific issue clearly and the push is held pending developer action.
2. **Given** the push is blocked with a critical issue, **When** the developer selects "push anyway," **Then** the override is recorded and the push completes with exit code `0`.
3. **Given** the push is blocked with a critical issue, **When** the developer selects "cancel," **Then** the push is aborted and exit code is `1`, allowing the developer to address the problem.
4. **Given** a blocked push, **When** the developer reads the review and chooses to fix the issue, **Then** preflight exits non-zero so the original push is cancelled; the developer commits the fix and pushes again.

---

### User Story 3 — AI Tool Unavailable (Fail-Open) (Priority: P3)

The developer runs `git push` but the locally installed AI CLI tool is not found on the system, or it times out during review. preflight must not silently block the push. It emits a warning message to the terminal explaining what happened, then allows the push to proceed as if preflight had not run.

**Why this priority**: preflight runs as a git hook on the developer's machine. Infrastructure failures must never silently block a push — that would erode trust and lead developers to remove the hook entirely.

**Independent Test**: Remove or rename the AI CLI binary, run `git push`, observe a warning message on stderr explaining the tool is unavailable, and confirm the push completes normally with exit code `0`.

**Acceptance Scenarios**:

1. **Given** no supported AI CLI tool is found on the system, **When** the developer runs `git push`, **Then** preflight emits a warning to stderr, exits `0`, and the push proceeds normally.
2. **Given** the AI CLI tool is found but does not respond within the configured timeout, **When** the timeout elapses, **Then** preflight emits a timeout warning to stderr, exits `0`, and the push is not blocked.
3. **Given** the AI CLI returns a malformed or unparseable response, **When** preflight processes the output, **Then** it emits a diagnostic warning to stderr, exits `0`, and does not block the push.

---

### User Story 4 — First-Time Installation (Priority: P4)

A developer discovers preflight and wants to start using it. They install the binary and register it as a git hook in one or more repositories with a single command. From that point forward, every `git push` in those repositories automatically runs preflight without any further configuration.

**Why this priority**: The tool is worthless if setup is painful. A one-command install path is critical for adoption.

**Independent Test**: On a clean machine with a supported AI CLI installed, run the preflight install command pointing at a git repository, then run `git push` in that repository and observe that preflight intercepts the push.

**Acceptance Scenarios**:

1. **Given** preflight is installed on the system, **When** the developer runs `preflight install` in a repository, **Then** preflight registers itself as the pre-push hook for that repository.
2. **Given** a repository already has an existing pre-push hook, **When** the developer runs `preflight install`, **Then** preflight detects the existing hook, emits a warning naming the existing hook file, and exits with an error; the developer must pass `--force` to replace it.
3. **Given** the developer wants preflight active in all future repositories, **When** they configure preflight in a global git hooks directory, **Then** every subsequent `git push` in any repository on the machine runs preflight.

---

### User Story 5 — Plain-Text Output (`--no-tui`) (Priority: P5)

A developer prefers not to use the full-screen terminal UI, or their terminal is old or limited (no reliable color or rich display). They may also run `git push` from a local script or pipe output to a file. In these cases they use `--no-tui` (or have no TTY attached), and preflight skips the Bubbletea UI and writes a plain-text review summary to standard output. The exit code contract remains the same.

**Why this priority**: Not every local environment suits a TUI; plain text must remain readable, script-friendly, and free of animation or styling artifacts.

**Independent Test**: Run preflight with `--no-tui` and pipe stdout to a file; confirm the file contains a readable plain-text summary and the process exits with the correct exit code.

**Acceptance Scenarios**:

1. **Given** preflight is invoked with `--no-tui`, **When** the review completes with no blocking issues, **Then** a plain-text summary is written to stdout and the process exits `0`.
2. **Given** preflight is invoked with `--no-tui` and a critical issue is found, **When** the review completes, **Then** the plain-text summary includes the critical finding, and the process exits `1`.
3. **Given** no terminal is attached (no TTY), **When** preflight runs, **Then** it automatically falls back to plain-text output without requiring `--no-tui`.

---

### User Story 6 — Provider Selection and Configuration (Priority: P6)

The developer has multiple AI CLI tools installed (e.g., both claude and gemini), or they prefer a specific one. They can configure preflight to use a particular provider, either via a project-level config file or a global user config file. Without any configuration, preflight auto-detects and uses the first available supported tool.

**Why this priority**: Different developers use different AI tools. The auto-detect default removes friction for first-time users; explicit configuration satisfies power users.

**Independent Test**: Create a project-level `.preflight.yml` specifying a provider, run `git push`, and confirm preflight uses the specified provider (verifiable via process list or log output).

**Acceptance Scenarios**:

1. **Given** no configuration file exists, **When** preflight searches for AI tools, **Then** it tries supported providers in the defined auto-detection order and uses the first one found.
2. **Given** a project-level config file specifies a provider, **When** preflight runs in that repository, **Then** it uses the specified provider regardless of auto-detection order.
3. **Given** both a global and a project-level config exist, **When** preflight runs, **Then** the project-level config takes precedence over the global config.
4. **Given** the configured provider is not installed, **When** preflight runs, **Then** it emits a clear warning naming the missing provider and falls back to auto-detection or fail-open behavior.

---

### Edge Cases

- What happens when the local branch has no upstream tracking branch? preflight must emit a clear message and exit `0` (fail-open), not hang or crash.
- What happens when the diff is empty (e.g., `git push` of an already-pushed branch)? preflight should skip the review, emit a brief notice, and exit `0`.
- What happens when the diff is extremely large (thousands of files or megabytes of changes)? preflight should handle large diffs gracefully — either truncating with a warning or forwarding the diff as-is — without hanging indefinitely.
- What happens when the developer's machine has no internet access? Since preflight only invokes local CLI tools with existing sessions, this must not affect functionality.
- What happens when two developers share a repository but only one has preflight installed? preflight only affects the machine it runs on; it has no server-side component and does not modify the remote.
- What happens when preflight is run outside of a git repository? It must exit with a usage error (`2`) and a clear message.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The tool MUST register itself as a git pre-push hook in a repository with a single install command.
- **FR-002**: The tool MUST collect the diff between the local branch tip and its upstream tracking branch at push time.
- **FR-003**: The tool MUST invoke a locally installed AI CLI as a subprocess to perform the code review, passing the diff as input.
- **FR-004**: The tool MUST display the AI review result in an interactive terminal UI before the push completes.
- **FR-005**: When the AI identifies a critical issue, the tool MUST block the push by default.
- **FR-006**: When a push is blocked, the developer MUST be able to choose: (a) push anyway with override, or (b) cancel the push to fix the issue.
- **FR-007**: When no supported AI CLI is found or the AI CLI times out, the tool MUST exit `0` and emit a warning to stderr — it MUST NOT silently block the push.
- **FR-008**: The tool MUST support a `--no-tui` flag that produces plain-text output suitable for users who opt out of the TUI, for piping or logging on the local machine, and for terminals that do not reliably support the interactive UI or colors.
- **FR-009**: The tool MUST auto-detect available AI providers in a defined order and use the first one found.
- **FR-010**: The tool MUST allow the user to specify a preferred AI provider via `--provider` flag or a configuration file.
- **FR-011**: Project-level configuration MUST override global user configuration.
- **FR-012**: The tool MUST exit with code `0` for a clean review or successful override, `1` for a blocked push or internal error, and `2` for usage/argument errors.
- **FR-013**: When the local branch has no upstream tracking branch or the diff is empty, the tool MUST fail open (exit `0`) with an informative message.
- **FR-014**: The tool MUST NOT make direct network requests or require API tokens; all AI interaction is through the locally installed CLI subprocess.

### Key Entities

- **Diff**: The set of code changes between the local branch tip and its upstream, collected at push time. Represents the unit of work submitted for review.
- **Review**: The structured output from the AI tool, containing findings categorized by severity (at minimum: critical vs. non-critical).
- **Finding**: A single issue identified by the AI. Has a severity level, a description, and ideally a reference to the relevant code location.
- **Provider**: A locally installed AI CLI tool that preflight can invoke as a subprocess. Examples: claude, codex, gemini, qwen.
- **Configuration**: Settings that control tool behavior. Exists at two scopes — project-level (checked into the repository or local to the project) and global user-level.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A developer with a supported AI CLI already installed can go from zero to an intercepted push review in under 2 minutes of setup time.
- **SC-002**: For diffs up to 500 changed lines, the review completes and results are visible in the terminal within 30 seconds of `git push` being invoked.
- **SC-003**: When the AI CLI is unavailable, the push is never blocked — the fail-open path is exercised correctly 100% of the time.
- **SC-004**: A developer who has never used preflight before can read the terminal UI output and understand whether their push was blocked or approved without consulting documentation.
- **SC-005**: The tool does not add more than 5 seconds of overhead to pushes where the diff is empty or the review produces no findings.
- **SC-006**: The plain-text (`--no-tui`) output is parseable by standard shell tools (grep, awk) without post-processing; each finding appears on a predictable line prefixed with its severity in brackets (e.g., `[CRITICAL]`).

## Assumptions

- The developer already has at least one supported AI CLI tool (claude, codex, gemini, or qwen) installed and authenticated on their machine before running preflight.
- "Critical issue" is determined entirely by the AI tool's output. preflight's responsibility is to surface the AI's severity classification faithfully, not to define what constitutes a critical issue.
- The AI CLI tools accept diff content as stdin and produce structured output (e.g., JSON) that preflight can parse for severity classification.
- The tool targets individual developer workstations (local use before push); multi-user or server-side automation scenarios are out of scope for this version.
- Supported platforms are macOS and Linux (Windows support is out of scope for this version).
- The configuration file format uses YAML for both project-level and global configs.
