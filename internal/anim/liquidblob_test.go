package anim

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestWriteSpinnerGoldens regenerates ../testdata/spinner_golden/frame_*.txt — run:
//
//	WRITE_SPINNER_GOLDEN=1 go test ./internal/anim -run TestWriteSpinnerGoldens -count=1
func TestWriteSpinnerGoldens(t *testing.T) {
	if os.Getenv("WRITE_SPINNER_GOLDEN") == "" {
		t.Skip("set WRITE_SPINNER_GOLDEN=1 to regenerate goldens")
	}
	cfg := DefaultLiquidBlobConfig()
	dir := filepath.Join("testdata", "spinner_golden")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	for tick := 0; tick < 5; tick++ {
		f, err := ComputeFrame(cfg, RenderOpts{Width: 24, Height: 6, Tick: tick, Seed: 42})
		require.NoError(t, err)
		s := renderFramePlainForGolden(f)
		name := filepath.Join(dir, fmt.Sprintf("frame_%03d.txt", tick))
		require.NoError(t, os.WriteFile(name, []byte(s), 0o644))
	}
}

// renderFramePlainForGolden duplicates tui.RenderFrame plain path without importing tui.
func renderFramePlainForGolden(f Frame) string {
	var b []byte
	for _, row := range f.Cells {
		for _, c := range row {
			r := c.Rune
			if r == 0 {
				r = ' '
			}
			b = append(b, string(r)...)
		}
		b = append(b, '\n')
	}
	return string(b)
}

func TestComputeFrame_Deterministic(t *testing.T) {
	cfg := DefaultLiquidBlobConfig()
	opts := RenderOpts{Width: 32, Height: 8, Tick: 3, Seed: 4242}
	a, err := ComputeFrame(cfg, opts)
	require.NoError(t, err)
	b, err := ComputeFrame(cfg, opts)
	require.NoError(t, err)
	require.Equal(t, a, b)
	require.Equal(t, opts.Tick, a.Tick)
	require.Len(t, a.Cells, 8)
	require.Len(t, a.Cells[0], 32)
}

func TestComputeFrame_DifferentTick(t *testing.T) {
	cfg := DefaultLiquidBlobConfig()
	o1 := RenderOpts{Width: 20, Height: 6, Tick: 1, Seed: 99}
	o2 := RenderOpts{Width: 20, Height: 6, Tick: 2, Seed: 99}
	a, err := ComputeFrame(cfg, o1)
	require.NoError(t, err)
	b, err := ComputeFrame(cfg, o2)
	require.NoError(t, err)
	require.NotEqual(t, a, b)
}
