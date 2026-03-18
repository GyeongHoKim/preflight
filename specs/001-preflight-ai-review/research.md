# Research: preflight — AI-Powered Pre-Push Code Review

**Phase**: 0 — Outline & Research
**Date**: 2026-03-18
**Branch**: `001-preflight-ai-review`

---

## 1. AI CLI Invocation Patterns

### 1.1 Claude (`claude` / claude-code)

**Decision**: Use `claude -p` (print mode) with `--json-schema` for guaranteed structured output. For providers where schema enforcement is unavailable, embed the desired JSON schema in the prompt body.

**Invocation flags** (sourced from https://code.claude.com/docs/en/cli-reference):

| Flag | Purpose |
|------|---------|
| `-p` / `--print` | Non-interactive print mode — query and exit |
| `--output-format json` | Single JSON object on stdout |
| `--output-format stream-json` | NDJSON event stream |
| `--output-format text` | Plain text (default) |
| `--system-prompt "..."` | Replace entire system prompt |
| `--append-system-prompt "..."` | Append to default system prompt |
| `--json-schema '{...}'` | Enforce output matches a JSON Schema (print mode only) |
| `--no-session-persistence` | Don't save session to disk (useful for hooks) |

**Recommended invocation for preflight:**
```bash
git diff <remote>..<local> | claude -p "$(cat system-prompt.txt)" \
  --output-format json \
  --no-session-persistence
```

Or with schema enforcement (preferred — guarantees valid JSON):
```bash
git diff <remote>..<local> | claude -p "Review the diff on stdin for critical issues. Respond with JSON only." \
  --json-schema '{"type":"object","properties":{"findings":{"type":"array"},"blocking":{"type":"boolean"}},"required":["findings","blocking"]}' \
  --no-session-persistence
```

**`--output-format json` response schema:**
```json
{
  "type": "result",
  "subtype": "success",
  "is_error": false,
  "duration_ms": 1234,
  "duration_api_ms": 1000,
  "num_turns": 1,
  "result": "<text or JSON string from model>",
  "session_id": "abc-123",
  "total_cost_usd": 0.003
}
```
The useful content is in `result`. When `--json-schema` is used, `result` is already validated JSON matching the schema; otherwise it is free-form text.

**Piping stdin**: `claude -p "PROMPT"` reads stdin automatically when input is piped (`cat file | claude -p "..."`).

**Rationale**: `--json-schema` is the cleanest approach — no text-to-JSON parsing heuristics needed, and claude validates the output internally. Use `--output-format json` as the envelope to get the full result object with error/success metadata.

---

### 1.2 Gemini (`gemini` / gemini-cli)

**Decision**: Use `gemini -p` with `--output-format json`.

**Invocation flags** (sourced from https://github.com/google-gemini/gemini-cli `docs/cli/headless.md`):

| Flag | Purpose |
|------|---------|
| `-p` / `--prompt "..."` | Non-interactive headless mode |
| `--output-format json` | Single JSON object response |
| `--output-format stream-json` | NDJSON event stream |

Headless mode also auto-activates when stdin is not a TTY.

**`--output-format json` response schema:**
```json
{
  "response": "<model's final answer as string>",
  "stats": {
    "<token usage and API latency metrics>"
  },
  "error": {
    "<optional — present only on failure>"
  }
}
```
The useful content is in `response`. Since `response` is a plain string, the model must be prompted to output JSON within that string.

**Exit codes**: `0`=success, `1`=error, `42`=input error, `53`=turn limit exceeded.

**Recommended invocation:**
```bash
git diff <remote>..<local> | gemini --prompt "$(cat system-prompt.txt)" --output-format json
```

**Rationale**: `--output-format json` gives a stable envelope. Parse `response`, then attempt JSON parse of its content. Fall back gracefully if `response` is not valid JSON.

---

### 1.3 Codex (`codex` / openai-codex)

**Decision**: Use `codex -q` (quiet mode) with `--json`.

**Invocation flags** (sourced from https://github.com/openai/codex README):

| Flag | Purpose |
|------|---------|
| `-q` | Non-interactive quiet mode — suppresses interactive UI |
| `--json` | Machine-readable JSON output |
| `CODEX_QUIET_MODE=1` | Environment variable equivalent to `-q` |

**Recommended invocation:**
```bash
codex -q --json "Review the following diff for critical issues. Output JSON only.\n\n$(git diff @{u})"
```

**JSON output schema**: Not officially documented in the README. Based on the quiet+json mode pattern, it returns a single JSON object. The exact field names (likely `output` or `response`) require empirical verification against the installed version.

**Gotcha**: `codex` does not appear to support reading from stdin as a diff payload the way `claude` and `gemini` do. The entire prompt must be passed as an argument string. For large diffs, this may hit shell arg-length limits (~2MB on Linux). Mitigation: write diff to a temp file and reference it in the prompt, or truncate large diffs with a warning.

---

### 1.4 Qwen (`qwen` / qwen-code)

**Decision**: Use `qwen -p` — same interface as `claude` (qwen-code is derived from claude-code).

**Invocation flags** (sourced from https://github.com/QwenLM/qwen-code README):

| Flag | Purpose |
|------|---------|
| `-p "..."` | Non-interactive print mode |
| `--output-format json` | JSON output (same as claude-code) |

The qwen-code CLI shares the same codebase architecture as claude-code. The `--output-format json` response schema is expected to be identical to claude's.

**Recommended invocation:**
```bash
git diff <remote>..<local> | qwen -p "PROMPT" --output-format json --no-session-persistence
```

---

### 1.5 Provider Auto-Detection Order

Try in order: `claude` → `codex` → `gemini` → `qwen`.
Detection via `exec.LookPath(provider)` — returns the binary path if found in `$PATH`.

**Alternatives considered**: Detecting by checking `--version` output (slower, more fragile than LookPath). Rejected.

---

### 1.6 Canonical Review JSON Schema (for prompt injection)

When `--json-schema` is not available (gemini, codex, qwen), embed this schema in the system prompt and ask the model to conform to it:

```json
{
  "type": "object",
  "required": ["findings", "blocking", "summary"],
  "properties": {
    "findings": {
      "type": "array",
      "items": {
        "type": "object",
        "required": ["severity", "message"],
        "properties": {
          "severity": { "type": "string", "enum": ["critical", "warning", "info"] },
          "category": { "type": "string", "enum": ["security", "logic", "quality", "style"] },
          "message":  { "type": "string" },
          "location": { "type": "string", "description": "file:line reference, optional" }
        }
      }
    },
    "blocking": { "type": "boolean", "description": "true if any critical findings exist" },
    "summary":  { "type": "string", "description": "one-sentence overall assessment" }
  }
}
```

The `block_on` config field (default: `"critical"`) determines which severity levels trigger a push block. The AI sets `blocking: true` when it finds issues at or above that threshold.

---

## 2. Git Diff Collection

### 2.1 Pre-Push Hook stdin Format

Git passes information to a pre-push hook via stdin in this format:
```
<local-ref> SP <local-sha> SP <remote-ref> SP <remote-sha> LF
```
Example: `refs/heads/main abc123... refs/heads/main 000000...`

Special cases:
- **New branch** (no upstream yet): `<remote-sha>` is all-zeros (`0000000...`)
- **Delete push**: `<local-ref>` is `(delete)`, `<local-sha>` is all-zeros

### 2.2 Getting the Correct Diff

Use the refs from stdin rather than `@{u}` to get the exact diff of what is being pushed:

```bash
# For an existing branch: diff between remote tip and what we're pushing
git diff <remote-sha>...<local-sha>

# For a new branch (remote-sha = 0000...): diff against merge-base with default branch
git diff $(git merge-base HEAD origin/HEAD)...HEAD
```

`git diff A...B` (three-dot) shows only changes that are in B but not in A's history — i.e., commits that will be new on the remote. This is the correct range for a push review.

### 2.3 Edge Cases

| Scenario | Handling |
|----------|---------|
| No upstream set | Parse from pre-push stdin. Remote SHA is all-zeros → diff vs merge-base or HEAD~N |
| Detached HEAD | `git symbolic-ref HEAD` exits non-zero; emit warning, fail-open |
| Empty diff | `git diff` returns empty stdout; skip review, exit 0 with info message |
| Large diff (>500 KB) | Truncate to a configurable max byte limit with a warning comment prepended |
| Delete push | `local-sha` is all-zeros; skip review, exit 0 |

### 2.4 Go Implementation Pattern

```go
// Stream diff to avoid loading entire diff into memory:
cmd := exec.CommandContext(ctx, "git", "diff", remote+"..."+local)
stdout, err := cmd.StdoutPipe()
cmd.Start()
// Stream stdout to AI CLI stdin via io.Pipe or collect with io.ReadAll up to maxBytes
cmd.Wait()
```

**Alternatives considered**: Using `git log` + per-commit diffs (more complex, same output for code review). Rejected in favour of a single `git diff` call.

---

## 3. Bubbletea TUI

### 3.1 Architecture

Follows The Elm Architecture — Model/Update/View:

- **Model**: holds all state (`[]Finding`, `cursor int`, `choice string`, `mode string`)
- **Update(msg)**: pure function — processes keyboard input, returns new model + optional Cmd
- **View()**: renders current state to a string using lipgloss for styling

### 3.2 No-TTY Detection and Fallback

```go
import "github.com/mattn/go-isatty"

func isTTY() bool {
    return isatty.IsTerminal(os.Stdout.Fd()) && isatty.IsTerminal(os.Stdin.Fd())
}
```

If `!isTTY() || noTUIFlag`, skip Bubbletea entirely and write plain-text results to stdout.

### 3.3 Blocking Until User Choice

`p.Run()` blocks until `tea.Quit()` is returned from `Update`. The final model value after `p.Run()` returns contains the user's choice. Extract it with a type assertion.

### 3.4 Lipgloss Styling

Use `lipgloss` alongside Bubbletea for colors, borders, and 80-column width constraints:
- Critical findings: bold red
- Warning findings: yellow
- Info findings: dim cyan
- Selection prompt: reverse-video highlight on active option

### 3.5 Alt Screen

Use `tea.WithAltScreen()` for the full review panel so it doesn't permanently pollute the terminal scroll buffer. On exit, the terminal returns to its previous state.

**Alternatives considered**: Using `termenv` directly without Bubbletea (more code, same result). Rejected.

---

## 4. CLI Framework: cobra + yaml.v3

### 4.1 cobra

Use `github.com/spf13/cobra` for command parsing.

**Command structure:**
- `preflight install` — registers the pre-push hook in a git repository
- `preflight uninstall` — removes the hook
- `preflight run` — runs a review manually (for testing outside a push)
- Root `PersistentPreRunE` — loads config, validates `--provider` enum

**Key flag decisions:**
- `--provider string` (default `"auto"`) — enum: auto/claude/codex/gemini/qwen
- `--no-tui bool` — plain-text output
- `--timeout duration` (default `60s`) — max time for AI CLI
- `--config string` — path to config file (default: `preflight.yml`, then `~/.config/preflight/config.yml`)
- `--block-on string` (default `"critical"`) — severity threshold

### 4.2 yaml.v3 (not viper)

Use `gopkg.in/yaml.v3` directly. The config schema has ≤5 fields; viper's multi-source merging, environment aliasing, and hot-reload are unnecessary overhead. Direct yaml.v3 keeps the dependency graph minimal per the constitution.

**Config schema:**
```yaml
provider: claude          # auto, claude, codex, gemini, qwen
block_on: critical        # critical, warning
timeout: 60s
prompt_extra: ""          # additional instructions appended to the system prompt
```

---

## 5. goreleaser Distribution

### 5.1 Minimal .goreleaser.yaml

- `CGO_ENABLED=0` for fully static binaries (no libc dependency)
- `goos: [linux, darwin]`, `goarch: [amd64, arm64]`
- `ldflags: [-s -w -X main.Version={{.Version}}]` for version injection
- `archives: [{format: tar.gz}]`
- `brews: [...]` targeting `gyeongho/homebrew-tap`

### 5.2 GitHub Actions Release Workflow

Trigger on `push: tags: [v*]`. Uses `goreleaser/goreleaser-action@v5`. Requires `GITHUB_TOKEN` for release creation and a separate `HOMEBREW_TAP_TOKEN` (PAT with repo write access) for tap formula commits.

**Alternatives considered**: Manual release script with `go build` + `tar` (no versioned Homebrew tap). Rejected in favour of goreleaser.

---

## 6. Testing Strategy

### 6.1 os/exec Mocking

Wrap `exec.CommandContext` behind a `Runner` interface:
```go
type Runner interface {
    Run(ctx context.Context, name string, args ...string) ([]byte, error)
}
```
Inject `RealRunner` in production code; inject `MockRunner` (with pre-configured responses) in tests. This avoids shell-out in unit tests.

### 6.2 testify Usage

- `require.NoError` / `require.Error` for setup-time preconditions (stops test on failure)
- `assert.Equal` / `assert.Contains` for outcome assertions (reports all failures)
- Table-driven tests for provider parsing, config loading, diff edge cases

### 6.3 Bubbletea Unit Testing

Test `Update()` and `View()` directly without running `tea.NewProgram()`:
```go
model := NewReviewModel(findings)
newModel, _ := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
assert.Equal(t, "cancel", newModel.(ReviewModel).choice)
```

### 6.4 Cobra Command Testing

```go
cmd.SetOut(&buf)
cmd.SetArgs([]string{"install", "--dry-run"})
err := cmd.Execute()
require.NoError(t, err)
```

---

## 7. Key Decisions Summary

| Decision | Choice | Rationale |
|----------|--------|-----------|
| AI structured output | `--json-schema` (claude), prompt-injected schema (others) | Guaranteed valid JSON for claude; best-effort for others |
| Git diff range | Three-dot `remote...local` from pre-push stdin | Exact set of new commits being pushed |
| Config library | `yaml.v3` direct | Minimal deps; viper is overkill for ≤5 fields |
| TUI library | Bubbletea + lipgloss | Standard Go TUI stack; pure Model/Update/View |
| TTY detection | `go-isatty` | Lightweight, widely adopted |
| os/exec mocking | `Runner` interface | Testable without subprocess invocation |
| Distribution | goreleaser | Homebrew tap + GitHub Releases in one workflow |
| Large diff handling | Truncate at configurable byte limit | Prevents hanging/OOM on AI CLI |
