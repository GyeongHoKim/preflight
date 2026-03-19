# preflight ✈️

> AI-powered code review in your terminal, before you push.

**preflight** runs an AI code review on your staged diff using locally installed AI CLI tools — no API tokens required. Works with your existing subscription to Claude, ChatGPT/Codex, Gemini, or Qwen.

```
$ git push origin main

  ┌─ preflight ──────────────────────────────────────────┐
  │                                                       │
  │  Provider: claude          Branch: feat/auth          │
  │                                                       │
  │  ● auth/jwt.go:84   [CRITICAL]                        │
  │    JWT secret falls back to hardcoded default value.  │
  │    Replace with a required env var and fail fast.     │
  │                                                       │
  │  ● api/handler.go:201  [WARNING]                      │
  │    Error from db.Query() is silently discarded.       │
  │                                                       │
  │  ● main.go:12  [INFO]                                 │
  │    Unused import: "fmt"                               │
  │                                                       │
  │  Overall: CRITICAL issue detected                     │
  │                                                       │
  │  [Push anyway]  [Cancel]                              │
  └───────────────────────────────────────────────────────┘
```

Push is blocked on `CRITICAL` by default. You stay in control.

---

## Why preflight?

Many teams have subscriptions to Claude or ChatGPT but **API token usage is not approved** — whether for security policy, cost control, or procurement reasons. This means AI-assisted review can't run in CI.

preflight fills that gap: it runs entirely on your local machine using the CLI tools you already have authenticated.

---

## Requirements

One of the following CLI tools installed and authenticated:

| Provider  | CLI                                                     | Authentication       |
| --------- | ------------------------------------------------------- | -------------------- |
| Anthropic | [`claude`](https://code.claude.com)                     | Claude subscription  |
| OpenAI    | [`codex`](https://developers.openai.com/codex/cli)      | ChatGPT subscription |
| Google    | [`gemini`](https://github.com/google-gemini/gemini-cli) | Gemini account       |
| Alibaba   | [`qwen`](https://github.com/QwenLM/qwen-code)           | Qwen account         |

---

## Installation

**Homebrew (macOS/Linux)**

```bash
brew install GyeongHoKim/tap/preflight
```

**curl installer**

```bash
curl -fsSL https://github.com/GyeongHoKim/preflight/releases/latest/download/install.sh | sh
```

**Go**

```bash
go install github.com/GyeongHoKim/preflight@latest
```

---

## Quick Start

```bash
# Install the pre-push hook into the current repository
preflight install

# Optionally specify a provider (auto-detected by default)
preflight install --provider codex
```

That's it. On your next `git push`, preflight will run automatically.

To uninstall:

```bash
preflight uninstall
```

---

## Hook Tool Integration

preflight works as a standalone hook or alongside your existing hook manager.

### Lefthook

```yaml
# lefthook.yml
pre-push:
  commands:
    ai-review:
      run: preflight run
```

### Husky

```bash
# .husky/pre-push
preflight run
```

### pre-commit (Python ecosystem)

```yaml
# .pre-commit-config.yaml
repos:
  - repo: local
    hooks:
      - id: preflight
        name: AI code review
        entry: preflight run
        language: system
        stages: [pre-push]
        pass_filenames: false
        always_run: true
```

### cargo-husky (Rust ecosystem)

```toml
# Cargo.toml
[dev-dependencies]
cargo-husky = { version = "1", features = ["user-hooks"] }
```

```bash
# .cargo-husky/hooks/pre-push
#!/bin/sh
preflight run
```

### Overcommit (Ruby ecosystem)

```yaml
# .overcommit.yml
PrePush:
  Preflight:
    enabled: true
    description: "AI code review"
    command: ["preflight", "run"]
    required_executable: "preflight"
```

### Plain git hook (no manager)

```bash
# .git/hooks/pre-push  (chmod +x required)
#!/bin/sh
preflight run
```

---

## Configuration

Create `.preflight.yml` in your repository root (or `~/.config/preflight/.preflight.yml` for global config):

```yaml
# .preflight.yml
provider: claude # claude | codex | gemini | qwen | auto

# Severity threshold for blocking a push.
# INFO and WARNING show in the TUI but never block.
# CRITICAL blocks by default unless overridden here or with --force at push time.
block_on: critical # critical | warning | never

# Additional context for the AI reviewer
prompt_extra: |
  This is a Go service. Focus on data races, error handling, and context propagation.
  We follow the Uber Go style guide.

# Files and paths to exclude from review
exclude:
  - "**/*_test.go"
  - "vendor/**"
  - "*.pb.go"
```

### Environment variables

| Variable             | Description                                              |
| -------------------- | -------------------------------------------------------- |
| `PREFLIGHT_PROVIDER` | Override the configured provider                         |
| `PREFLIGHT_BLOCK_ON` | Override block severity (`critical`, `warning`, `never`) |
| `PREFLIGHT_SKIP`     | Set to `1` to skip review for one push                   |

---

## Usage

```
preflight <command> [flags]

Commands:
  install     Install preflight as a pre-push hook in the current repository
  uninstall   Remove the preflight hook
  run         Run a review manually (reads current diff against upstream)
  check       Verify that a supported AI CLI is installed and authenticated

Flags:
  --provider string   AI provider to use (claude, codex, gemini, qwen)
  --force             Push even if CRITICAL issues are found
  --no-tui            Print results as plain text (useful for CI or pipes)
  --config string     Path to config file (default: ./.preflight.yml)
```

**Run a review without pushing:**

```bash
preflight run
```

**Review against a specific base branch:**

```bash
preflight run --base main
```

**Skip the review once:**

```bash
PREFLIGHT_SKIP=1 git push
```

**Plain text output (for scripting or minimal environments):**

```bash
preflight run --no-tui
```

---

## How It Works

1. `git push` triggers the `pre-push` hook
2. preflight collects the diff between your branch and its upstream
3. The diff is passed to your configured AI CLI with a structured review prompt
4. The CLI returns a JSON response with issues and severity ratings
5. preflight renders the results in a TUI
6. If no `CRITICAL` issues exist (or you choose "Push anyway"), git push proceeds; otherwise it exits non-zero and the push is cancelled

The diff never leaves your machine except to the AI CLI, which uses your existing local session — no separate network calls are made by preflight itself.

---

## Supported Providers

| Provider  | CLI                                                     | Non-interactive    | JSON output flag              |
| --------- | ------------------------------------------------------- | ------------------ | ----------------------------- |
| Anthropic | [`claude`](https://code.claude.com)                     | `claude -p "..."`  | `--output-format json`        |
| OpenAI    | [`codex`](https://developers.openai.com/codex/cli)      | `codex exec "..."` | `--output-schema schema.json` |
| Google    | [`gemini`](https://github.com/google-gemini/gemini-cli) | `gemini -p "..."`  | `--output-format json`        |
| Alibaba   | [`qwen`](https://github.com/QwenLM/qwen-code)           | `qwen -p "..."`    | `--output-format json`        |

All four providers support structured JSON output in non-interactive mode. Schema enforcement varies by provider and is treated as best-effort — if the response doesn't parse against the expected schema, preflight shows the raw response in the TUI and lets you decide whether to block or continue.

Provider is auto-detected in order: `claude` → `codex` → `gemini` → `qwen`. Override with `--provider` or the config file.

---

## FAQ

**Does this require an API key?**
No. preflight uses locally installed CLI tools that authenticate with your existing subscription (Claude, ChatGPT, etc.). No API key or token is needed.

**Can I use this in CI?**
No. preflight is designed exclusively for local developer use — it relies on your personally authenticated AI CLI session, which is not available in CI environments. For AI-assisted code review in CI, refer to the official API documentation for your provider: [Claude](https://docs.anthropic.com/en/docs/about-claude/models/overview), [OpenAI](https://platform.openai.com/docs/guides/code-review), [Gemini](https://ai.google.dev/gemini-api/docs).

**What if the AI CLI is slow or unavailable?**
preflight has a configurable timeout (default: 60s). If the review times out or the CLI is not found, preflight warns and exits 0 — it will never silently block a push due to a tool failure.

**Can I review only specific files?**
Use `exclude` in the config to skip generated files, vendor directories, test files, etc.

**Will this work with monorepos?**
Yes. preflight reviews only the diff for the current push, so only changed files are sent to the AI.

---

## Development

**Requirements**

| Tool    | Minimum version | Purpose                          |
| ------- | --------------- | -------------------------------- |
| Go      | 1.26.1          | Build and test                   |
| Node.js | 18.0.0          | commitlint (commit message lint) |

```bash
# One-time setup after cloning — installs Node deps and wires git hooks
make setup
```

---

## License

MIT
