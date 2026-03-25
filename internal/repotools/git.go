package repotools

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// GitArgs is the JSON shape for git_context.
type GitArgs struct {
	Mode  string `json:"mode"`
	Path  string `json:"path"`
	Limit int    `json:"limit"`
}

type gitResult struct {
	Mode       string `json:"mode"`
	Path       string `json:"path,omitempty"`
	Output     string `json:"output,omitempty"`
	Truncated  bool   `json:"truncated"`
	Truncation string `json:"truncation_reason,omitempty"`
	Skipped    bool   `json:"skipped,omitempty"`
	SkipReason string `json:"skip_reason,omitempty"`
}

// GitContext runs a read-only allowlisted git command.
func (e *Executor) GitContext(ctx context.Context, argsJSON []byte) (string, error) {
	var args GitArgs
	if err := json.Unmarshal(argsJSON, &args); err != nil {
		return "", fmt.Errorf("git_context: bad arguments: %w", err)
	}

	mode := strings.TrimSpace(args.Mode)
	if mode == "" {
		mode = "status"
	}
	limit := args.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 200 {
		limit = 200
	}

	var cmdArgs []string
	res := gitResult{Mode: mode}
	skipReason := ""
	switch mode {
	case "status":
		cmdArgs = []string{"-C", e.root, "status", "--short", "--untracked-files=no"}
	case "log":
		cmdArgs = []string{"-C", e.root, "log", "--oneline", "-n", strconv.Itoa(limit)}
	case "log_file":
		if args.Path == "" {
			skipReason = "path is required for mode=log_file"
			break
		}
		_, rel, err := e.rules.ResolveReadable(args.Path)
		if err != nil {
			res.Path = args.Path
			skipReason = err.Error()
			break
		}
		res.Path = rel
		cmdArgs = []string{"-C", e.root, "log", "--oneline", "-n", strconv.Itoa(limit), "--", rel}
	default:
		skipReason = "mode must be one of: status, log, log_file"
	}

	if skipReason != "" {
		res.Skipped = true
		res.SkipReason = skipReason
		return gitJSON(res), nil
	}

	out, runErr := exec.CommandContext(ctx, "git", cmdArgs...).CombinedOutput()
	if runErr != nil {
		res.Skipped = true
		res.SkipReason = strings.TrimSpace(string(out))
	} else {
		res.Output = strings.TrimSpace(string(out))
	}
	return gitJSON(res), nil
}

func gitJSON(r gitResult) string {
	b, err := json.Marshal(r)
	if err != nil {
		return fmt.Sprintf(`{"error":%q}`, err.Error())
	}
	return string(b)
}
