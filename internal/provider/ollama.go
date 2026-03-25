package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/GyeongHoKim/preflight/internal/config"
	"github.com/GyeongHoKim/preflight/internal/ollama"
	"github.com/GyeongHoKim/preflight/internal/repotools"
	"github.com/GyeongHoKim/preflight/internal/review"
)

// OllamaRunner performs code review via an organization-controlled Ollama HTTP API.
type OllamaRunner struct {
	cfg      *config.Config
	repoRoot string
	prompt   string
	schema   string
}

// NewOllamaRunner constructs a runner that uses cfg.Ollama and the given repository root.
func NewOllamaRunner(cfg *config.Config, repoRoot, prompt, schema string) *OllamaRunner {
	return &OllamaRunner{cfg: cfg, repoRoot: repoRoot, prompt: prompt, schema: schema}
}

// Run implements Runner for Ollama /api/chat with repository tools.
func (r *OllamaRunner) Run(ctx context.Context, diff []byte) (review.ProviderResult, error) {
	o := r.cfg.Ollama
	// HTTP client timeout is disabled; ctx from the hook enforces the review deadline.
	client, err := ollama.NewClient(o.BaseURL, 0)
	if err != nil {
		return review.ProviderResult{}, fmt.Errorf("%w: %w", ollama.ErrUnavailable, err)
	}

	ex := repotools.NewExecutor(
		r.repoRoot,
		o.AllowPrefixes,
		o.DenyPaths,
		o.MaxListEntries,
		o.MaxReadBytes,
		o.MaxSearchMatches,
	)

	format := json.RawMessage(r.schema)
	tools := ollamaToolDefs()

	userText := fmt.Sprintf(
		"Staged diff to review (UTF-8). Respond ONLY with a single JSON object matching the configured schema (no markdown fences).\n\n```diff\n%s\n```",
		string(diff),
	)

	messages := []ollama.ChatMessage{
		{Role: "system", Content: r.prompt},
		{Role: "user", Content: userText},
	}

	for turn := 0; turn < o.MaxToolTurns; turn++ {
		req := &ollama.ChatRequest{
			Model:    o.Model,
			Messages: messages,
			Stream:   false,
			Tools:    tools,
			Format:   format,
		}

		resp, err := client.Chat(ctx, req)
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return review.ProviderResult{}, err
			}
			return review.ProviderResult{}, err // includes ollama.ErrUnavailable
		}

		msg := resp.Message
		messages = append(messages, ollama.ChatMessage{
			Role:      msg.Role,
			Content:   msg.Content,
			ToolCalls: msg.ToolCalls,
		})

		if len(msg.ToolCalls) == 0 {
			raw := extractReviewJSON(msg.Content)
			if len(raw) == 0 {
				return review.ProviderResult{}, fmt.Errorf("%w: empty or non-json assistant content", review.ErrMalformedResponse)
			}
			return review.ProviderResult{Stdout: raw}, nil
		}

		for _, tc := range msg.ToolCalls {
			name := strings.TrimSpace(tc.Function.Name)
			args := normalizeToolArgs(tc.Function.Arguments)
			payload, derr := ex.Dispatch(ctx, name, args)
			if derr != nil {
				payload = fmt.Sprintf(`{"error":%q}`, derr.Error())
			}
			messages = append(messages, ollama.ChatMessage{
				Role:    "tool",
				Name:    name,
				Content: payload,
			})
		}
	}

	return review.ProviderResult{}, fmt.Errorf("%w: exceeded max_tool_turns without final JSON", review.ErrMalformedResponse)
}

func normalizeToolArgs(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return json.RawMessage(`{}`)
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil && strings.TrimSpace(s) != "" {
		return json.RawMessage(s)
	}
	return raw
}

func extractReviewJSON(s string) []byte {
	s = strings.TrimSpace(s)
	s = stripMarkdownFences(s)
	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")
	if start < 0 || end <= start {
		return nil
	}
	return []byte(s[start : end+1])
}

func stripMarkdownFences(s string) string {
	if !strings.HasPrefix(s, "```") {
		return s
	}
	rest := strings.TrimPrefix(s, "```")
	if idx := strings.Index(rest, "\n"); idx >= 0 {
		rest = rest[idx+1:]
	}
	if end := strings.LastIndex(rest, "```"); end >= 0 {
		rest = rest[:end]
	}
	return strings.TrimSpace(rest)
}

func ollamaToolDefs() []ollama.ToolDef {
	listParams := json.RawMessage(`{
  "type": "object",
  "properties": {
    "prefix": {"type": "string", "description": "Repository-relative directory prefix to list under"},
    "limit": {"type": "integer", "description": "Max files to return"}
  }
}`)
	readParams := json.RawMessage(`{
  "type": "object",
  "required": ["path"],
  "properties": {
    "path": {"type": "string", "description": "Repository-relative file path"},
    "offset": {"type": "integer", "description": "Byte offset to start reading"},
    "max_bytes": {"type": "integer", "description": "Max bytes to read in this call"}
  }
}`)
	searchParams := json.RawMessage(`{
  "type": "object",
  "required": ["pattern"],
  "properties": {
    "pattern": {"type": "string", "description": "Literal substring to search for"},
    "path_prefix": {"type": "string", "description": "Only search under this repository-relative prefix"},
    "limit": {"type": "integer", "description": "Max matches to return"},
    "max_file_kb": {"type": "integer", "description": "Skip files larger than this many KiB"}
  }
}`)
	gitParams := json.RawMessage(`{
  "type": "object",
  "properties": {
    "mode": {"type": "string", "description": "One of: status, log, log_file"},
    "path": {"type": "string", "description": "Repository-relative path (required for mode=log_file)"},
    "limit": {"type": "integer", "description": "Max git log entries for log/log_file"}
  }
}`)

	return []ollama.ToolDef{
		{
			Type: "function",
			Function: ollama.ToolFunction{
				Name:        "list_files",
				Description: "List file paths under the repository root, optionally under a prefix, with server-enforced limits",
				Parameters:  listParams,
			},
		},
		{
			Type: "function",
			Function: ollama.ToolFunction{
				Name:        "read_file",
				Description: "Read a slice of a text file under the repository root; binary files are skipped",
				Parameters:  readParams,
			},
		},
		{
			Type: "function",
			Function: ollama.ToolFunction{
				Name:        "search_repo",
				Description: "Search for a literal substring in text files under the repository",
				Parameters:  searchParams,
			},
		},
		{
			Type: "function",
			Function: ollama.ToolFunction{
				Name:        "git_context",
				Description: "Read-only git context helper (status and recent history) with strict argument allowlisting",
				Parameters:  gitParams,
			},
		},
	}
}
