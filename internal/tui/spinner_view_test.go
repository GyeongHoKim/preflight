package tui

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/stretchr/testify/require"

	"github.com/GyeongHoKim/preflight/internal/anim"
)

func TestRenderFrame_DisableANSI_NoEscape(t *testing.T) {
	cfg := anim.DefaultLiquidBlobConfig()
	frame, err := anim.ComputeFrame(cfg, anim.RenderOpts{Width: 20, Height: 5, Tick: 2, Seed: 1})
	require.NoError(t, err)
	s := RenderFrame(frame, RenderOptions{DisableANSI: true})
	require.NotContains(t, s, "\x1b")
}

func TestRenderFrame_GoldenFiles(t *testing.T) {
	cfg := anim.DefaultLiquidBlobConfig()
	dir := filepath.Join("..", "anim", "testdata", "spinner_golden")
	for _, tick := range []int{0, 1, 2, 3, 4} {
		frame, err := anim.ComputeFrame(cfg, anim.RenderOpts{Width: 24, Height: 6, Tick: tick, Seed: 42})
		require.NoError(t, err)
		got := RenderFrame(frame, RenderOptions{DisableANSI: true})
		name := filepath.Join(dir, "frame_"+formatTick(tick)+".txt")
		wantBytes, err := os.ReadFile(name)
		require.NoError(t, err, "open %s", name)
		want := strings.ReplaceAll(string(wantBytes), "\r\n", "\n")
		require.Equal(t, want, got, "golden mismatch for tick %d", tick)
	}
}

func formatTick(tick int) string {
	if tick < 10 {
		return "00" + strconv.Itoa(tick)
	}
	return strconv.Itoa(tick)
}

func TestRenderFrame_AdjacentFrameSmoothness_SC002(t *testing.T) {
	const (
		w      = 24
		h      = 6
		seed   = 42
		ticks  = 30
		maxPct = 0.35
		maxDup = 0.20
	)
	cfg := anim.DefaultLiquidBlobConfig()
	cells := w * h
	maxDiff := int(maxPct*float64(cells) + 0.999999) // ceil

	var prev string
	dupPairs := 0
	totalPairs := 0
	for tck := 0; tck < ticks; tck++ {
		f1, err := anim.ComputeFrame(cfg, anim.RenderOpts{Width: w, Height: h, Tick: tck, Seed: seed})
		require.NoError(t, err)
		s1 := RenderFrame(f1, RenderOptions{DisableANSI: true})
		if tck > 0 {
			totalPairs++
			if s1 == prev {
				dupPairs++
			}
			diff := countRuneDiffGrid(s1, prev, w, h)
			require.LessOrEqual(t, diff, maxDiff, "adjacent tick %d vs %d", tck-1, tck)
		}
		prev = s1
	}
	require.LessOrEqual(t, float64(dupPairs)/float64(totalPairs), maxDup)
}

func countRuneDiffGrid(a, b string, w, h int) int {
	la := lines(a)
	lb := lines(b)
	if len(la) != len(lb) {
		return w * h
	}
	n := 0
	for i := range la {
		ra := []rune(la[i])
		rb := []rune(lb[i])
		if len(ra) != w || len(rb) != w {
			return w * h
		}
		for j := 0; j < w; j++ {
			if ra[j] != rb[j] {
				n++
			}
		}
	}
	if len(la) != h {
		return w * h
	}
	return n
}

func lines(s string) []string {
	s = strings.TrimSuffix(s, "\n")
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}

func TestRenderFrame_UTF8CellWidth(t *testing.T) {
	cfg := anim.DefaultLiquidBlobConfig()
	frame, err := anim.ComputeFrame(cfg, anim.RenderOpts{Width: 10, Height: 4, Tick: 0, Seed: 3})
	require.NoError(t, err)
	s := RenderFrame(frame, RenderOptions{DisableANSI: true})
	for _, line := range lines(s) {
		if line == "" {
			continue
		}
		require.Equal(t, 10, utf8.RuneCountInString(line))
	}
}
