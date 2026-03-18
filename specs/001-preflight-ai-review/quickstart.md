# Quickstart: preflight Development Setup

**Date**: 2026-03-18
**Branch**: `001-preflight-ai-review`

---

## Prerequisites

- Go (latest stable) — `go version`
- `make` — `make --version`
- golangci-lint — `golangci-lint --version` (or `make setup` installs it)
- Node.js + npm (for lefthook) — `node --version`
- At least one of: `claude`, `codex`, `gemini`, `qwen` installed and authenticated

---

## One-Time Setup

```bash
git clone <repo-url> preflight
cd preflight
make setup          # npm install + lefthook install
```

---

## Daily Development Commands

```bash
make build          # compile binary → ./bin/preflight
make test           # go test -race -count=1 ./...
make lint           # golangci-lint run (must exit 0 before marking any task done)
make fmt            # gofmt + goimports via golangci-lint
make tidy           # go mod tidy
```

Run a single test:
```bash
go test -run TestProviderDetect ./internal/provider/
```

---

## Testing the Hook Manually

```bash
# Build first
make build

# Install into a test repository
cd /tmp/test-repo
git init && git remote add origin https://example.com/repo.git
/path/to/preflight/bin/preflight install

# Make a commit and push (hook fires automatically)
echo "hello" > file.txt
git add . && git commit -m "test"
git push origin main       # preflight intercepts here
```

Testing the plain-text path:
```bash
git diff HEAD~1 | ./bin/preflight run --no-tui --provider claude
```

---

## Project Layout

```
preflight/
├── cmd/
│   └── preflight/
│       └── main.go               # Entry point; wires cobra root command
├── internal/
│   ├── cli/                      # cobra command definitions
│   │   ├── root.go               # PersistentPreRunE: config load + flag validation
│   │   ├── install.go            # `preflight install` command
│   │   ├── uninstall.go          # `preflight uninstall` command
│   │   └── run.go                # `preflight run` command (also called by hook)
│   ├── config/                   # YAML config loading and validation
│   │   └── config.go
│   ├── diff/                     # Git diff collection
│   │   ├── collect.go            # Reads pre-push stdin, runs git diff
│   │   └── collect_test.go
│   ├── provider/                 # AI CLI subprocess adapters
│   │   ├── runner.go             # Runner interface + auto-detection
│   │   ├── claude.go             # claude-specific invocation + response parsing
│   │   ├── codex.go              # codex-specific invocation + response parsing
│   │   ├── gemini.go             # gemini-specific invocation + response parsing
│   │   ├── qwen.go               # qwen-specific invocation + response parsing
│   │   └── runner_test.go        # Unit tests with MockRunner
│   ├── review/                   # Review result types and parsing
│   │   ├── review.go             # Finding, Review, severity ranking
│   │   └── review_test.go
│   ├── tui/                      # Bubbletea TUI
│   │   ├── model.go              # ReviewModel, Update, View
│   │   ├── styles.go             # lipgloss style definitions
│   │   ├── plain.go              # Plain-text renderer (--no-tui)
│   │   └── model_test.go         # Update/View unit tests
│   └── hook/                     # Pre-push hook logic
│       ├── hook.go               # Orchestrates: diff → provider → tui → exit code
│       └── hook_test.go
├── .goreleaser.yaml              # goreleaser release config
├── Makefile
├── go.mod
└── go.sum
```

---

## Key Dependencies

| Package | Purpose | Justification |
|---------|---------|---------------|
| `github.com/spf13/cobra` | CLI argument parsing | Industry standard; clean subcommand support |
| `github.com/charmbracelet/bubbletea` | Terminal UI | Elm Architecture; pure model/update/view |
| `github.com/charmbracelet/lipgloss` | TUI styling | Standard companion to Bubbletea |
| `github.com/mattn/go-isatty` | TTY detection | Lightweight; single responsibility |
| `github.com/stretchr/testify` | Test assertions | Reduces boilerplate vs. stdlib `testing` |
| `gopkg.in/yaml.v3` | Config file parsing | Minimal; avoids viper's unused features |

All other functionality uses the Go standard library (`os/exec`, `encoding/json`, `context`, `io`, etc.).

---

## Quality Gate Checklist (run before every PR)

```bash
make lint           # must exit 0
make test           # must exit 0; new behaviour must have test coverage
make build          # must succeed
```

Then manually verify:
- [ ] All exported symbols have doc comments
- [ ] Error strings are lowercase without trailing punctuation
- [ ] No `_` error discards
- [ ] Exit-code contract tested for affected code paths
- [ ] `context.Context` is first parameter where applicable
