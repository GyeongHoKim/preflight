package repotools

import (
	"fmt"
	"path/filepath"
	"strings"
)

// PathRules enforces repository-relative path access with optional allow prefixes
// and deny patterns.
type PathRules struct {
	RepoRoot      string
	AllowPrefixes []string
	DenyPatterns  []string
}

// ErrOutsideRepo is returned when a path escapes the repository root.
var ErrOutsideRepo = fmt.Errorf("repotools: path outside repository root")

// ErrDeniedPath is returned when a path matches a deny rule or fails allow rules.
var ErrDeniedPath = fmt.Errorf("repotools: path denied by policy")

// NormalizeRel returns a cleaned slash-separated path relative to the repo root.
func NormalizeRel(rel string) string {
	rel = filepath.ToSlash(filepath.Clean(rel))
	rel = strings.TrimPrefix(rel, "./")
	return rel
}

// ResolveReadable returns the absolute filesystem path for rel if it is allowed.
func (p PathRules) ResolveReadable(rel string) (abs string, normRel string, err error) {
	if p.RepoRoot == "" {
		return "", "", fmt.Errorf("repotools: empty repo root")
	}
	rel = NormalizeRel(rel)
	if rel == ".." || strings.HasPrefix(rel, "../") {
		return "", "", fmt.Errorf("%w: %q", ErrOutsideRepo, rel)
	}

	joined := filepath.Join(p.RepoRoot, filepath.FromSlash(rel))
	absRoot, err := filepath.Abs(p.RepoRoot)
	if err != nil {
		return "", "", fmt.Errorf("repotools: abs repo root: %w", err)
	}
	absJoined, err := filepath.Abs(joined)
	if err != nil {
		return "", "", fmt.Errorf("repotools: abs target: %w", err)
	}

	relRoot, err := filepath.Rel(absRoot, absJoined)
	if err != nil || strings.HasPrefix(relRoot, "..") {
		return "", "", fmt.Errorf("%w: %q", ErrOutsideRepo, rel)
	}

	if deniedByGit(rel) {
		return "", "", fmt.Errorf("%w: %q", ErrDeniedPath, rel)
	}
	if matchDeny(rel, p.DenyPatterns) {
		return "", "", fmt.Errorf("%w: %q", ErrDeniedPath, rel)
	}
	if !allowed(rel, p.AllowPrefixes) {
		return "", "", fmt.Errorf("%w: %q", ErrDeniedPath, rel)
	}

	return absJoined, rel, nil
}

func deniedByGit(rel string) bool {
	if rel == ".git" || strings.HasPrefix(rel, ".git/") {
		return true
	}
	return false
}

func allowed(rel string, prefixes []string) bool {
	if len(prefixes) == 0 {
		return true
	}
	for _, pre := range prefixes {
		pre = NormalizeRel(pre)
		if pre == "" || pre == "." {
			return true
		}
		if rel == pre || strings.HasPrefix(rel, pre+"/") {
			return true
		}
	}
	return false
}

func matchDeny(rel string, patterns []string) bool {
	for _, pat := range patterns {
		pat = strings.TrimSpace(pat)
		if pat == "" {
			continue
		}
		if strings.Contains(pat, "**") {
			prefix := strings.TrimSuffix(pat, "**")
			prefix = strings.TrimSuffix(prefix, "/")
			prefix = NormalizeRel(prefix)
			if prefix == "" {
				return true
			}
			if rel == prefix || strings.HasPrefix(rel, prefix+"/") {
				return true
			}
			continue
		}
		ok, err := filepath.Match(pat, rel)
		if err == nil && ok {
			return true
		}
		base := filepath.Base(rel)
		ok, err = filepath.Match(pat, base)
		if err == nil && ok {
			return true
		}
	}
	return false
}
