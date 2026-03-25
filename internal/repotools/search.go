package repotools

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// SearchArgs is the JSON shape for search_repo.
type SearchArgs struct {
	Pattern    string `json:"pattern"`
	PathPrefix string `json:"path_prefix"`
	Limit      int    `json:"limit"`
	MaxFileKB  int    `json:"max_file_kb"`
}

type searchMatch struct {
	Path    string `json:"path"`
	Line    int    `json:"line"`
	Snippet string `json:"snippet"`
}

type searchResult struct {
	Matches    []searchMatch `json:"matches"`
	Truncated  bool          `json:"truncated"`
	Truncation string        `json:"truncation_reason,omitempty"`
}

// SearchRepo searches for a literal substring in text files under the repository.
func (e *Executor) SearchRepo(ctx context.Context, argsJSON []byte) (string, error) {
	var args SearchArgs
	if err := json.Unmarshal(argsJSON, &args); err != nil {
		return "", fmt.Errorf("search_repo: bad arguments: %w", err)
	}
	if args.Pattern == "" {
		return searchJSON(searchResult{Truncation: "pattern is required"}), nil
	}
	limit := args.Limit
	if limit <= 0 || limit > e.maxSearch {
		limit = e.maxSearch
	}
	maxKB := args.MaxFileKB
	if maxKB <= 0 {
		maxKB = 512
	}
	maxBytes := maxKB * 1024

	prefix := NormalizeRel(args.PathPrefix)
	var matches []searchMatch
	var truncated bool
	var truncReason string

	_ = filepath.WalkDir(e.root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil //nolint:nilerr // skip entries the walker cannot access
		}
		select {
		case <-ctx.Done():
			return fs.SkipAll
		default:
		}
		if d.IsDir() {
			if d.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}

		rel, err := filepath.Rel(e.root, path)
		if err != nil {
			return nil //nolint:nilerr // skip unexpected paths
		}
		rel = filepath.ToSlash(rel)
		if prefix != "" && rel != prefix && !strings.HasPrefix(rel, prefix+"/") {
			return nil
		}
		if deniedByGit(rel) || matchDeny(rel, e.rules.DenyPatterns) || !allowed(rel, e.rules.AllowPrefixes) {
			return nil
		}

		data, err := os.ReadFile(path) //nolint:gosec // G122: path from WalkDir under fixed repo root gated by PathRules
		if err != nil {
			return nil //nolint:nilerr // skip unreadable files
		}
		if len(data) > maxBytes {
			return nil
		}
		if bytes.IndexByte(data, 0) >= 0 {
			return nil
		}

		sc := bufio.NewScanner(bytes.NewReader(data))
		lineNo := 0
		for sc.Scan() {
			lineNo++
			line := sc.Text()
			if strings.Contains(line, args.Pattern) {
				snippet := line
				if len(snippet) > 200 {
					snippet = snippet[:200] + "…"
				}
				matches = append(matches, searchMatch{Path: rel, Line: lineNo, Snippet: snippet})
				if len(matches) >= limit {
					truncated = true
					truncReason = fmt.Sprintf("match list capped at %d (max_search_matches)", limit)
					return filepath.SkipAll
				}
			}
		}
		return nil
	})

	if ctx.Err() != nil {
		truncated = true
		if truncReason == "" {
			truncReason = "cancelled before search completed"
		}
	}

	return searchJSON(searchResult{
		Matches:    matches,
		Truncated:  truncated,
		Truncation: truncReason,
	}), nil
}

func searchJSON(r searchResult) string {
	b, err := json.Marshal(r)
	if err != nil {
		return fmt.Sprintf(`{"error":%q}`, err.Error())
	}
	return string(b)
}
