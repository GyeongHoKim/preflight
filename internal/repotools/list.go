package repotools

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// ListArgs is the JSON shape for list_files.
type ListArgs struct {
	Prefix string `json:"prefix"`
	Limit  int    `json:"limit"`
}

type listResult struct {
	Paths      []string `json:"paths"`
	Truncated  bool     `json:"truncated"`
	Truncation string   `json:"truncation_reason,omitempty"`
}

// ListFiles walks the repository (optionally under prefix) and returns relative paths.
func (e *Executor) ListFiles(ctx context.Context, argsJSON []byte) (string, error) {
	var args ListArgs
	if err := json.Unmarshal(argsJSON, &args); err != nil {
		return "", fmt.Errorf("list_files: bad arguments: %w", err)
	}
	limit := args.Limit
	if limit <= 0 || limit > e.maxList {
		limit = e.maxList
	}

	prefix := NormalizeRel(args.Prefix)
	if prefix == "." {
		prefix = ""
	}
	startDir := e.root
	if prefix != "" {
		abs, _, err := e.rules.ResolveReadable(prefix)
		if err != nil {
			// Surface policy errors to the model as tool JSON, not as a Go error.
			return listJSON(listResult{Paths: nil, Truncated: false, Truncation: err.Error()}), nil //nolint:nilerr // tool payload
		}
		st, err := os.Stat(abs)
		if err != nil {
			return listJSON(listResult{Truncation: fmt.Sprintf("stat: %v", err)}), nil
		}
		if !st.IsDir() {
			return listJSON(listResult{Paths: []string{prefix}}), nil
		}
		startDir = abs
	}

	var paths []string
	var truncated bool
	var truncReason string

	walkRoot := e.root
	_ = filepath.WalkDir(startDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil //nolint:nilerr // skip entries the walker cannot access
		}
		select {
		case <-ctx.Done():
			return fs.SkipAll
		default:
		}
		if d.IsDir() && d.Name() == ".git" {
			return filepath.SkipDir
		}
		rel, err := filepath.Rel(walkRoot, path)
		if err != nil {
			return nil //nolint:nilerr // skip unexpected paths
		}
		rel = filepath.ToSlash(rel)
		if rel == "." {
			return nil
		}
		if prefix != "" && rel != prefix && !strings.HasPrefix(rel, prefix+"/") {
			return nil
		}
		if deniedByGit(rel) || matchDeny(rel, e.rules.DenyPatterns) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if !allowed(rel, e.rules.AllowPrefixes) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if len(paths) >= limit {
			truncated = true
			truncReason = fmt.Sprintf("list capped at %d entries (max_list_entries)", limit)
			return filepath.SkipAll
		}
		paths = append(paths, rel)
		return nil
	})

	if ctx.Err() != nil {
		truncated = true
		if truncReason == "" {
			truncReason = "cancelled before listing completed"
		}
	}

	return listJSON(listResult{
		Paths:      paths,
		Truncated:  truncated,
		Truncation: truncReason,
	}), nil
}

func listJSON(r listResult) string {
	b, err := json.Marshal(r)
	if err != nil {
		return fmt.Sprintf(`{"error":%q}`, err.Error())
	}
	return string(b)
}
