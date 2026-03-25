package repotools

import (
	"context"
	"encoding/json"
	"fmt"
)

// Executor runs repository tools under PathRules and numeric limits.
type Executor struct {
	rules     PathRules
	root      string
	maxList   int
	maxRead   int
	maxSearch int
}

// NewExecutor builds an Executor for repoRoot with the given policy and caps.
func NewExecutor(repoRoot string, allowPrefixes, denyPatterns []string, maxList, maxRead, maxSearch int) *Executor {
	return &Executor{
		rules: PathRules{
			RepoRoot:      repoRoot,
			AllowPrefixes: allowPrefixes,
			DenyPatterns:  denyPatterns,
		},
		root:      repoRoot,
		maxList:   maxList,
		maxRead:   maxRead,
		maxSearch: maxSearch,
	}
}

// Dispatch runs the named tool and returns a string payload for the model (JSON text).
func (e *Executor) Dispatch(ctx context.Context, name string, args json.RawMessage) (string, error) {
	switch name {
	case "list_files":
		return e.ListFiles(ctx, args)
	case "read_file":
		return e.ReadFile(args)
	case "search_repo":
		return e.SearchRepo(ctx, args)
	case "git_context":
		return e.GitContext(ctx, args)
	default:
		return "", fmt.Errorf("repotools: unknown tool %q", name)
	}
}
