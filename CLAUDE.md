# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
make build        # compile binary to ./bin/preflight
make test         # run all tests with race detector (-race -count=1)
make lint         # run golangci-lint — must exit 0 before any task is done
make fmt          # format code via golangci-lint (gofmt + goimports)
make tidy         # go mod tidy
make setup        # one-time post-clone: npm install + lefthook install
```

Run a single test:
```bash
go test -run TestFunctionName ./path/to/package/
```

## Architecture

preflight is a Git pre-push hook that pipes the staged diff to a locally installed AI CLI (claude, codex, gemini, or qwen) as a subprocess and renders the structured JSON response in a TUI.

Key design points:
- **No API calls**: preflight invokes AI CLIs as subprocesses using the user's existing authenticated local session. It never makes direct API requests.
- **stdout / stderr / exit code contract**: stdout = TUI or plain-text results; stderr = errors and warnings; exit 0 = clean, exit 1 = blocking issues or internal error, exit 2 = usage error.
- **Fail-open on tool unavailability**: if the AI CLI is not found or times out, preflight MUST exit 0 and emit a warning to stderr — it must never silently block a push due to a tool failure.
- **Provider auto-detection**: tried in order `claude → codex → gemini → qwen`; overridable via `--provider` or config file.
- **Config resolution**: project-level `preflight.yml` overrides global `~/.config/preflight/config.yml`.

## Non-Negotiable Rules (from `.specify/memory/constitution.md`)

1. **`make lint` must exit 0** after every change. `//nolint` directives require an inline justification comment.
2. **Explicit error handling**: no `_` discard unless the function is documented infallible; wrap errors with `fmt.Errorf("context: %w", err)`.
3. **Go Code Review Comments compliance**: naming, doc comments on all exported symbols, error strings lowercase without trailing punctuation, `context.Context` first parameter. See https://go.dev/wiki/CodeReviewComments.
4. **No `util`/`helpers`/`common` packages**: every internal package must have a single clear responsibility.
5. **Minimal dependencies**: prefer stdlib; every external dependency must be justified.

## Quality Gates (all must pass before a task is done)

1. `make lint` exits 0
2. `go test ./...` passes; new behaviour has test coverage
3. `go build ./...` succeeds
4. Manual check against https://go.dev/wiki/CodeReviewComments for changed files
5. Exit-code contract verified for affected code paths

## Active Technologies
- Go (latest stable, ≥1.22) + cobra, bubbletea, lipgloss, go-isatty, testify, yaml.v3 (001-preflight-ai-review)
- N/A (no persistent state; config read from YAML files, hook written to `.git/hooks/pre-push`) (001-preflight-ai-review)

## Recent Changes
- 001-preflight-ai-review: Added Go (latest stable, ≥1.22) + cobra, bubbletea, lipgloss, go-isatty, testify, yaml.v3
