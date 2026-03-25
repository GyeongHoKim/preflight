// Package hook implements the preflight pre-push hook orchestration.
package hook

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/GyeongHoKim/preflight/internal/config"
	"github.com/GyeongHoKim/preflight/internal/diff"
	"github.com/GyeongHoKim/preflight/internal/provider"
	"github.com/GyeongHoKim/preflight/internal/review"
	"github.com/GyeongHoKim/preflight/internal/tui"
)

// defaultProviders is the auto-detection order.
var defaultProviders = []string{"claude", "codex"}

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
		providerRunner, err = buildRunner(cfg, wd)
		if err != nil {
			logf(stderr, "preflight: %v; skipping review\n", err)
			return 0
		}
	}

	// Determine provider name.
	provName := cfg.Provider
	if provName == "auto" {
		provName = "unknown"
	}

	runCtx, runCancel := context.WithTimeout(ctx, cfg.Timeout)
	defer runCancel()

	branch := branchName(pi.LocalRef)
	commitCount := 1 // approximate; exact count not critical here

	if noTUI || !tui.IsTTY() {
		stopPlain := startPlainProgress(stderr)
		rev, rerr := runReviewWithRetry(runCtx, stderr, providerRunner, diffBytes, provName)
		stopPlain()

		if rerr != nil {
			if provider.ShouldFailOpen(rerr) {
				logf(stderr, "preflight: provider unavailable (%v); skipping review\n", rerr)
				return 0
			}
			if errors.Is(rerr, review.ErrMalformedResponse) {
				logf(stderr, "preflight: could not parse review after retry; skipping\n")
				return 0
			}
			logf(stderr, "preflight: provider error: %v\n", rerr)
			return 1
		}
		if rev == nil {
			logf(stderr, "preflight: could not parse review; skipping\n")
			return 0
		}

		tui.PlainRender(stdout, rev, branch, commitCount)
		if rev.Blocking {
			return 1
		}
		return 0
	}

	// TUI mode: spinner while provider runs, then review.
	fetch := func() (*review.Review, error) {
		return runReviewWithRetry(runCtx, stderr, providerRunner, diffBytes, provName)
	}
	wm := tui.NewWaitingModel(stderr, fetch)
	p := tea.NewProgram(wm, tea.WithOutput(stdout))
	finalModel, teaErr := p.Run()
	if errors.Is(teaErr, tea.ErrInterrupted) {
		return 1
	}
	if teaErr != nil {
		logf(stderr, "preflight: tui error: %v\n", teaErr)
		return 1
	}

	wfinal, ok := finalModel.(*tui.WaitingModel)
	if !ok {
		return 1
	}
	if wfinal.UserInterrupted() {
		return 1
	}
	if wfinal.QuitEarly() {
		rev, rerr := wfinal.ProviderResult()
		if rerr != nil {
			if provider.ShouldFailOpen(rerr) {
				logf(stderr, "preflight: provider unavailable (%v); skipping review\n", rerr)
				return 0
			}
			if errors.Is(rerr, review.ErrMalformedResponse) {
				logf(stderr, "preflight: could not parse review after retry; skipping\n")
				return 0
			}
			logf(stderr, "preflight: provider error: %v\n", rerr)
			return 1
		}
		if rev == nil {
			logf(stderr, "preflight: could not parse review; skipping\n")
			return 0
		}
		// Unreachable if fetch succeeded with a review — QuitEarly should be false.
		return 0
	}

	rm := wfinal.FinalReviewModel()
	rev := rm.Review()
	if rev == nil {
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

func startPlainProgress(stderr io.Writer) func() {
	done := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				// Carriage return only — no ESC/SGR (US2 / SC-004 stderr hygiene).
				_, _ = fmt.Fprintf(stderr, "\rpreflight: analyzing changes...")
			}
		}
	}()
	return func() {
		close(done)
		wg.Wait()
		_, _ = fmt.Fprintln(stderr)
	}
}

// runReviewWithRetry mirrors the previous synchronous hook behavior (malformed retry).
func runReviewWithRetry(ctx context.Context, stderr io.Writer, runner provider.Runner, diffBytes []byte, provName string) (*review.Review, error) {
	rev, err := attempt(ctx, runner, diffBytes, provName)
	if errors.Is(err, review.ErrMalformedResponse) {
		logf(stderr, "preflight: malformed response; retrying once\n")
		rev, err = attempt(ctx, runner, diffBytes, provName)
	}
	return rev, err
}

// logf writes a formatted message to w.
// Write errors are intentionally ignored; stderr/stdout failures are
// unrecoverable in a CLI context and must not mask the primary exit code.
func logf(w io.Writer, format string, args ...interface{}) {
	fmt.Fprintf(w, format, args...) //nolint:errcheck // stderr/stdout write failures are unrecoverable in a CLI context
}

// buildRunner constructs the appropriate provider runner based on config.
func buildRunner(cfg *config.Config, workDir string) (provider.Runner, error) {
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
	case "codex":
		return provider.NewCodexRunner(prompt, schema), nil
	case "ollama":
		root, err := diff.TopLevel(workDir)
		if err != nil {
			return nil, fmt.Errorf("resolve git top-level: %w", err)
		}
		return provider.NewOllamaRunner(cfg, root, prompt, schema), nil
	default:
		return nil, fmt.Errorf("unknown provider %q", provName)
	}
}

// attempt runs the provider once and parses the result.
func attempt(ctx context.Context, runner provider.Runner, diffBytes []byte, provName string) (*review.Review, error) {
	start := time.Now()
	result, err := runner.Run(ctx, diffBytes)
	result.Duration = time.Since(start).Milliseconds()
	if err != nil {
		return nil, err
	}
	return review.ParseReview(provName, result)
}

// branchName extracts the short branch name from a full ref like refs/heads/main.
func branchName(ref string) string {
	const prefix = "refs/heads/"
	if strings.HasPrefix(ref, prefix) {
		return ref[len(prefix):]
	}
	return ref
}
