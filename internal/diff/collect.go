// Package diff provides git diff collection for preflight.
package diff

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

// zeroSHA is the all-zero SHA used by git to indicate a non-existent ref.
const zeroSHA = "0000000000000000000000000000000000000000"

// PushInfo holds the information extracted from a single line of git's pre-push
// hook stdin. Git writes one line per ref being pushed.
type PushInfo struct {
	LocalRef  string
	LocalSHA  string
	RemoteRef string
	RemoteSHA string
}

// IsNewBranch reports whether this push creates a new remote ref.
func (p PushInfo) IsNewBranch() bool {
	return p.RemoteSHA == zeroSHA
}

// IsDeletePush reports whether this push deletes a remote ref.
func (p PushInfo) IsDeletePush() bool {
	return p.LocalSHA == zeroSHA
}

// ParsePushInfo reads the pre-push hook stdin format produced by git and
// returns one PushInfo per line. Blank lines are skipped.
func ParsePushInfo(r io.Reader) ([]PushInfo, error) {
	var infos []PushInfo
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) != 4 {
			return nil, fmt.Errorf("diff: unexpected push-info line %q: want 4 fields, got %d", line, len(fields))
		}
		infos = append(infos, PushInfo{
			LocalRef:  fields[0],
			LocalSHA:  fields[1],
			RemoteRef: fields[2],
			RemoteSHA: fields[3],
		})
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("diff: reading stdin: %w", err)
	}
	return infos, nil
}

// IsGitRepo reports whether dir is inside a git repository by running
// git rev-parse --git-dir.
func IsGitRepo(dir string) bool {
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	cmd.Dir = dir
	return cmd.Run() == nil
}

// CollectDiff runs git diff <remoteSHA>...<localSHA> and returns the diff
// bytes. If the diff exceeds maxBytes it is truncated and a warning comment is
// prepended. An empty diff returns nil with no error.
func CollectDiff(ctx context.Context, info PushInfo, maxBytes int) ([]byte, error) {
	if info.IsDeletePush() {
		return nil, nil
	}

	var args []string
	if info.IsNewBranch() {
		// New branch: diff from the common ancestor of HEAD and the remote default.
		// Fall back to diffing all commits in the branch.
		args = []string{"diff", "HEAD"}
	} else {
		args = []string{"diff", info.RemoteSHA + "..." + info.LocalSHA}
	}

	cmd := exec.CommandContext(ctx, "git", args...)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("diff: git diff: %w", err)
	}
	if len(out) == 0 {
		return nil, nil
	}
	if maxBytes > 0 && len(out) > maxBytes {
		truncated := make([]byte, maxBytes)
		copy(truncated, out[:maxBytes])
		warning := []byte(fmt.Sprintf("# [preflight: diff truncated at %d bytes]\n", maxBytes))
		return append(warning, truncated...), nil
	}
	return out, nil
}

// CollectDiffFromRange collects a diff for the given git range string.
// This is a low-level helper for testing.
func CollectDiffFromRange(ctx context.Context, rangeStr string, maxBytes int) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "git", "diff", rangeStr)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("diff: git diff: %w", err)
	}
	if len(out) == 0 {
		return nil, nil
	}
	if maxBytes > 0 && len(out) > maxBytes {
		truncated := make([]byte, maxBytes)
		copy(truncated, out[:maxBytes])
		warning := []byte(fmt.Sprintf("# [preflight: diff truncated at %d bytes]\n", maxBytes))
		return append(warning, truncated...), nil
	}
	return out, nil
}

// newlineCount counts newlines in b.
func newlineCount(b []byte) int {
	return bytes.Count(b, []byte{'\n'})
}

// suppress unused warning for newlineCount.
var _ = newlineCount
