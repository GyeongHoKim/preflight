# preflight ✈️

> AI-powered code review in your terminal, before you push.

**preflight** runs an AI code review on your staged diff using locally installed AI CLI tools (Claude, Codex) or, for air‑gapped teams, an **organization-controlled [Ollama](https://ollama.com)** HTTP server. Subprocess providers need no API tokens; the Ollama path uses only the `base_url` you configure.

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

| Provider  | CLI / transport                                                                | Authentication / trust boundary  |
| --------- | ------------------------------------------------------------------------------ | -------------------------------- |
| Anthropic | [`claude`](https://code.claude.com)                                            | Claude subscription (local CLI)  |
| OpenAI    | [`codex`](https://developers.openai.com/codex/cli)                             | ChatGPT subscription (local CLI) |
| Ollama    | HTTP to your Ollama instance ([API](https://docs.ollama.com/api/introduction)) | Your server; no vendor cloud API |

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
provider: claude # claude | codex | ollama | auto

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

### Private Ollama (organization-controlled inference)

Use `provider: ollama` when reviews must stay inside your network. preflight sends the
staged **diff** and optional **repository tool** traffic (list/read/search under the
git root) **only** to `ollama.base_url` — not to Claude, Codex, or other subprocess
CLIs.

```yaml
provider: ollama
timeout: 120s
ollama:
  base_url: "http://ollama.internal:11434"
  model: "llama3"
  max_tool_turns: 25
  max_read_bytes: 65536
  max_list_entries: 500
  max_search_matches: 100
  allow_prefixes: [] # optional; empty = whole repo subject to deny_paths
  deny_paths:
    - ".env*"
    - "secrets/**"
```

If Ollama is unreachable or misconfigured, preflight **fails open** (exit `0`) with a
stderr warning, same as when a subprocess CLI is missing. See also
`specs/003-private-ollama-review/quickstart.md`.

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
  --provider string   AI provider to use (claude, codex, ollama)
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
3. The diff is passed to your configured provider: a **subprocess CLI** (claude/codex) or **Ollama HTTP** (`/api/chat`), with the same structured review schema
4. The provider returns JSON with issues and severity ratings
5. preflight renders the results in a TUI
6. If no `CRITICAL` issues exist (or you choose "Push anyway"), git push proceeds; otherwise it exits non-zero and the push is cancelled

With **claude** or **codex**, the diff is sent to your local CLI session. With **ollama**,
preflight makes HTTP requests only to the configured `base_url` (plus repository tool
calls that read the working tree locally before responding to the model).

---

## Supported Providers

| Provider  | Transport        | Notes                                                                    |
| --------- | ---------------- | ------------------------------------------------------------------------ |
| Anthropic | `claude` CLI     | `--output-format json`, `--json-schema`                                  |
| OpenAI    | `codex` CLI      | `--output-schema` file                                                   |
| Ollama    | HTTP `/api/chat` | JSON schema via `format`; tools `list_files`, `read_file`, `search_repo` |

Subprocess providers support structured JSON in non-interactive mode. Ollama uses the
same canonical review JSON on stdout. Schema enforcement is best-effort — if the
response doesn't parse, preflight may retry once then fail open.

`auto` detects subprocess CLIs in order: `claude` → `codex`. Ollama is **never**
auto-selected; set `provider: ollama` explicitly. Override with `--provider` or the
config file.

---

## FAQ

**Does this require an API key?**
No for **claude** and **codex**: preflight uses locally installed CLI tools that authenticate with your existing subscription. For **ollama**, you point at your own server URL; preflight does not send data to a vendor cloud API for that mode.

**Can I use this in CI?**
No. preflight is designed exclusively for local developer use — it relies on your personally authenticated AI CLI session, which is not available in CI environments. For AI-assisted code review in CI, refer to the official API documentation for your provider: [Claude](https://docs.anthropic.com/en/docs/about-claude/models/overview), [OpenAI](https://platform.openai.com/docs/guides/code-review).

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
