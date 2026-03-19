# CLI Contract: preflight

**Type**: CLI tool contract (commands, flags, exit codes, I/O streams)
**Date**: 2026-03-18
**Branch**: `001-preflight-ai-review`

---

## Commands

### `preflight install [--global]`

Registers preflight as the pre-push hook for the current git repository (or globally via git's `core.hooksPath` when `--global` is passed).

**Flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--global` | bool | false | Install into global git hooks path (`~/.config/git/hooks/`) |
| `--force` | bool | false | Overwrite an existing pre-push hook without prompting |

**stdout**: Confirmation message: path where hook was written.
**stderr**: Warnings if an existing hook was detected.
**Exit codes**: `0`=success, `1`=hook already exists and `--force` not given, `2`=not in a git repository or usage error.

---

### `preflight uninstall [--global]`

Removes the preflight pre-push hook.

**Flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--global` | bool | false | Remove from global git hooks path |

**stdout**: Confirmation message.
**stderr**: Warning if hook was not managed by preflight.
**Exit codes**: `0`=success, `1`=hook not found, `2`=usage error.

---

### `preflight run [--provider P] [--no-tui] [--timeout D]`

Runs a code review manually against the current branch's diff, without waiting for a `git push`. Useful for testing and CI.

**Flags:**

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--provider` | string | `"auto"` | AI provider: `auto`, `claude`, `codex` |
| `--no-tui` | bool | false | Plain-text output; do not launch Bubbletea UI |
| `--timeout` | duration | `60s` | Max time to wait for AI CLI |
| `--config` | string | `""` | Path to config file (overrides default resolution) |
| `--block-on` | string | `"critical"` | Minimum severity to block: `critical` or `warning` |

**stdin**: If a diff is piped via stdin, it is used directly. Otherwise, preflight collects the diff from the current branch vs. its upstream.
**stdout**: TUI (default) or plain-text review results.
**stderr**: Warnings and errors.
**Exit codes**: see Exit Code Contract below.

---

### `preflight version`

Prints the version string.

**stdout**: `preflight vX.Y.Z (commit abc1234, built 2026-03-18)`
**Exit codes**: `0` always.

---

## Global Flags (persistent across all commands)

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--config` | string | `""` | Path to config file. Resolved if empty: `./.preflight.yml` → `~/.config/preflight/.preflight.yml` |
| `--no-tui` | bool | false | Disable terminal UI; write plain text to stdout |
| `--verbose` | bool | false | Emit debug information to stderr |

---

## Exit Code Contract

| Code | Meaning |
|------|---------|
| `0` | Clean: no blocking issues, or user explicitly chose to push anyway |
| `1` | Blocked: critical (or threshold-meeting) issues found and push was not overridden; or internal error |
| `2` | Usage error: invalid arguments, not in a git repository, unsupported provider specified |

The exit code of preflight's pre-push hook controls whether `git push` proceeds:
- Exit `0` → git continues the push
- Non-zero → git aborts the push

**Fail-open rule**: Any scenario where preflight cannot perform a review (AI tool not found, timeout, parse failure) MUST produce exit `0` with a warning on stderr. The push is never silently blocked due to a tool-side failure.

---

## I/O Streams

| Stream | Content |
|--------|---------|
| **stdout** | TUI output (Bubbletea renders directly) or plain-text review summary when `--no-tui` |
| **stderr** | All warnings, errors, and diagnostic messages |
| **stdin** | Pre-push hook input (local-ref/sha remote-ref/sha lines from git), or piped diff for `preflight run` |

---

## Plain-Text Output Format (`--no-tui`)

When `--no-tui` is active, stdout emits the following structure (one section per finding):

```
preflight: reviewing <N> commits on <branch>

[CRITICAL] security — auth/token.go:42
  Hardcoded secret detected in token initialization.

[WARNING] logic — api/handler.go:88
  Error return from Unmarshal is discarded.

preflight: 1 critical, 1 warning — push blocked.
To push anyway, run: git push --no-verify
```

When no findings:
```
preflight: reviewing <N> commits on <branch>
preflight: no issues found — push allowed.
```

---

## AI Review System Prompt Contract

preflight sends the following prompt structure to every AI provider. The exact text is implementation-defined, but the structure is:

```
SYSTEM: You are a code reviewer. Review the following git diff for:
- Security vulnerabilities (hardcoded secrets, injection risks, auth bypasses)
- Silent error discarding (ignored return values, empty catch blocks)
- Logic bugs with data loss or corruption potential

Respond ONLY with a JSON object matching this schema:
{
  "findings": [
    {
      "severity": "critical|warning|info",
      "category": "security|logic|quality|style",
      "message": "<description>",
      "location": "<file:line or empty>"
    }
  ],
  "blocking": <true if any critical finding>,
  "summary": "<one sentence>"
}

USER: <diff content>
```

For `claude`, use `--json-schema` to enforce this schema at the CLI level. For other providers, include the schema in the prompt body.

---

## Config File Schema

**Location resolution order** (first match wins):
1. `--config` flag value
2. `./.preflight.yml` in the current working directory
3. `~/.config/preflight/.preflight.yml`

**YAML schema:**
```yaml
# .preflight.yml
provider: auto          # auto | claude | codex
block_on: critical      # critical | warning
timeout: 60s            # Go duration string
prompt_extra: ""        # appended to system prompt
max_diff_bytes: 524288  # 512 KB default
```

All fields are optional; unset fields use their defaults.

---

## Hook File Contract

The installed pre-push hook is a shell script at `.git/hooks/pre-push`:

```sh
#!/bin/sh
# Managed by preflight. Run `preflight uninstall` to remove.
exec preflight run "$@"
```

`git` passes two arguments to pre-push hooks: `<remote-name>` and `<remote-url>`. Git also pipes the ref list to stdin. preflight passes `"$@"` through so the runner command receives the remote name/URL as positional arguments.
