package tui

import (
	"fmt"
	"io"
	"strings"

	"github.com/GyeongHoKim/preflight/internal/review"
)

// PlainRender writes a structured plain-text review summary to w.
// branch is the current branch name; commitCount is the number of commits being pushed.
func PlainRender(w io.Writer, r *review.Review, branch string, commitCount int) {
	writef(w, "preflight: reviewing %d commit(s) on %s\n", commitCount, branch)

	if r == nil || (len(r.Findings) == 0 && !r.Blocking) {
		writef(w, "preflight: no issues found — push allowed.\n")
		return
	}

	if len(r.Findings) > 0 {
		writef(w, "\n")
	}

	for _, f := range r.Findings {
		sev := strings.ToUpper(f.Severity)
		loc := f.Location
		if loc != "" {
			loc = " — " + loc
		}
		cat := f.Category
		writef(w, "[%s] %s%s\n  %s\n\n", sev, cat, loc, f.Message)
	}

	// Footer.
	critical, warning := 0, 0
	for _, f := range r.Findings {
		switch f.Severity {
		case review.SeverityCritical:
			critical++
		case review.SeverityWarning:
			warning++
		}
	}

	if r.Blocking {
		writef(w, "preflight: %d critical, %d warning — push blocked.\n", critical, warning)
		writef(w, "To push anyway, run: git push --no-verify\n")
	} else {
		writef(w, "preflight: no blocking issues — push allowed.\n")
	}
}

// writef writes a formatted string to w.
// Write errors are intentionally ignored; stdout failures are unrecoverable
// in a CLI context and must not mask the primary exit code.
func writef(w io.Writer, format string, args ...interface{}) {
	fmt.Fprintf(w, format, args...) //nolint:errcheck // stdout write failures are unrecoverable in a CLI context
}
