// Package hook implements the preflight pre-push hook orchestration.
package hook

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/GyeongHoKim/preflight/internal/config"
	"github.com/GyeongHoKim/preflight/internal/diff"
	"github.com/GyeongHoKim/preflight/internal/provider"
	"github.com/GyeongHoKim/preflight/internal/review"
	"github.com/GyeongHoKim/preflight/internal/tui"
)

// defaultProviders is the auto-detection order.
var defaultProviders = []string{"claude", "codex", "gemini", "qwen"}

// Run orchestrates a preflight review run and returns the process exit code.
//
// Exit codes:
//   - 0: clean review, fail-open condition, or user chose to push anyway
//   - 1: blocking review and user cancelled (or internal error after review)
//   - 2: usage error (not a git repo)
func Run(ctx context.Context, cfg *config.Config, stdin io.Reader, stdout, stderr io.Writer, noTUI bool, diffCollector diff.Collector, providerRunner provider.Runner) int {
	// Check git repo.
	wd, err := os.Getwd()
	if err != nil {
		logf(stderr, "preflight: could not determine working directory: %v\n", err)
		return 2
	}
	if !diff.IsGitRepo(wd) {
		logf(stderr, "preflight: not a git repository\n")
		return 2
	}

	// Parse push info from stdin.
	pushInfos, err := diff.ParsePushInfo(stdin)
	if err != nil {
		logf(stderr, "preflight: parse push info: %v\n", err)
		return 1
	}
	if len(pushInfos) == 0 {
		// Nothing to push; exit cleanly.
		return 0
	}
	pi := pushInfos[0]

	if pi.IsDeletePush() {
		return 0
	}

	// Collect diff.
	diffCtx, cancel := context.WithTimeout(ctx, cfg.Timeout)
	defer cancel()

	if diffCollector == nil {
		diffCollector = diff.GitCollector{}
	}
	diffBytes, err := diffCollector.Collect(diffCtx, pi, cfg.MaxDiffBytes)
	if err != nil {
		logf(stderr, "preflight: collect diff: %v\n", err)
		return 1
	}
	if len(diffBytes) == 0 {
		logf(stderr, "preflight: no diff to review\n")
		return 0
	}

	// Check if diff was truncated.
	if len(diffBytes) >= cfg.MaxDiffBytes {
		logf(stderr, "preflight: diff truncated; review may be incomplete\n")
	}

	// Determine provider runner.
	if providerRunner == nil {
		providerRunner, err = buildRunner(cfg)
		if err != nil {
			logf(stderr, "preflight: %v; skipping review\n", err)
			return 0
		}
	}

	// Run provider.
	start := time.Now()
	runCtx, runCancel := context.WithTimeout(ctx, cfg.Timeout)
	defer runCancel()

	result, runErr := providerRunner.Run(runCtx, diffBytes)
	result.Duration = time.Since(start).Milliseconds()

	if runErr != nil {
		if provider.ShouldFailOpen(runErr) {
			logf(stderr, "preflight: provider unavailable (%v); skipping review\n", runErr)
			return 0
		}
		logf(stderr, "preflight: provider error: %v\n", runErr)
		return 1
	}

	// Parse review.
	provName := cfg.Provider
	if provName == "auto" {
		provName = "unknown"
	}
	rev, parseErr := review.ParseReview(provName, result)
	if parseErr != nil {
		logf(stderr, "preflight: parse review error: %v\n", parseErr)
		return 1
	}
	if rev == nil {
		logf(stderr, "preflight: could not parse review; skipping\n")
		return 0
	}

	// Render.
	branch := branchName(pi.LocalRef)
	commitCount := 1 // approximate; exact count not critical here

	if noTUI || !tui.IsTTY() {
		tui.PlainRender(stdout, rev, branch, commitCount)
		if rev.Blocking {
			return 1
		}
		return 0
	}

	// TUI mode.
	model := tui.NewReviewModel(rev)
	p := tea.NewProgram(model, tea.WithOutput(stdout))
	finalModel, teaErr := p.Run()
	if teaErr != nil {
		logf(stderr, "preflight: tui error: %v\n", teaErr)
		return 1
	}

	rm, ok := finalModel.(tui.ReviewModel)
	if !ok {
		return 1
	}

	if rev.Blocking {
		if rm.Choice() == "push_anyway" {
			logf(stderr, "preflight: push override recorded\n")
			return 0
		}
		return 1
	}
	return 0
}

// logf writes a formatted message to w.
// Write errors are intentionally ignored; stderr/stdout failures are
// unrecoverable in a CLI context and must not mask the primary exit code.
func logf(w io.Writer, format string, args ...interface{}) {
	fmt.Fprintf(w, format, args...) //nolint:errcheck // stderr/stdout write failures are unrecoverable in a CLI context
}

// buildRunner constructs the appropriate provider runner based on config.
func buildRunner(cfg *config.Config) (provider.Runner, error) {
	provName := cfg.Provider
	if provName == "auto" {
		detected, err := provider.Detect(defaultProviders)
		if err != nil {
			return nil, fmt.Errorf("no AI provider found in PATH")
		}
		provName = detected
	}

	prompt := review.SystemPrompt(cfg.PromptExtra)
	schema := review.Schema()

	switch provName {
	case "claude":
		return provider.NewClaudeRunner(prompt, schema), nil
	case "gemini":
		return provider.NewGeminiRunner(prompt), nil
	case "codex":
		return provider.NewCodexRunner(prompt), nil
	case "qwen":
		return provider.NewQwenRunner(prompt, schema), nil
	default:
		return nil, fmt.Errorf("unknown provider %q", provName)
	}
}

// branchName extracts the short branch name from a full ref like refs/heads/main.
func branchName(ref string) string {
	const prefix = "refs/heads/"
	if strings.HasPrefix(ref, prefix) {
		return ref[len(prefix):]
	}
	return ref
}
