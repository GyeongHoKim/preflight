package diff

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// TopLevel returns the absolute path to the git repository root for the working
// directory workDir using `git rev-parse --show-toplevel`.
func TopLevel(workDir string) (string, error) {
	cmd := exec.Command("git", "-C", workDir, "rev-parse", "--show-toplevel")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("diff: git rev-parse --show-toplevel: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return strings.TrimSpace(stdout.String()), nil
}
