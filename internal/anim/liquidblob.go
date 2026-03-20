// Package anim implements deterministic liquid-blob field animation math (stdlib only).
package anim

import (
	"fmt"
	"math"
)

// LiquidBlobConfig configures the 2D metaball + planar wave + pulse field.
// Zero values are replaced by safe defaults in ComputeFrame.
type LiquidBlobConfig struct {
	BlobCount       int
	Sigma           float64
	WaveVectorX     float64
	WaveVectorY     float64
	WaveAngularFreq float64
	WaveAmplitude   float64
	PulseOmega      float64
	EdgeSoftness    float64
}

// RenderOpts selects viewport size and animation step for ComputeFrame.
type RenderOpts struct {
	Width  int
	Height int
	Tick   int
	Seed   uint64
}

// Cell is one terminal cell before ANSI styling.
type Cell struct {
	Rune rune
	R    uint8
	G    uint8
	B    uint8
}

// Frame is a snapshot of the spinner grid at one tick.
type Frame struct {
	Cells [][]Cell
	Tick  int
}

// DefaultLiquidBlobConfig returns parameters used by production and golden tests.
func DefaultLiquidBlobConfig() LiquidBlobConfig {
	return LiquidBlobConfig{
		BlobCount:       5,
		Sigma:           0.11,
		WaveVectorX:     14.0,
		WaveVectorY:     9.0,
		WaveAngularFreq: 0.14,
		WaveAmplitude:   0.32,
		PulseOmega:      0.11,
		EdgeSoftness:    0.1,
	}
}

func (c LiquidBlobConfig) normalized() LiquidBlobConfig {
	if c.BlobCount <= 0 {
		c.BlobCount = 5
	}
	if c.Sigma <= 0 {
		c.Sigma = 0.11
	}
	if c.WaveVectorX == 0 && c.WaveVectorY == 0 {
		c.WaveVectorX = 14
		c.WaveVectorY = 9
	}
	if c.WaveAngularFreq <= 0 {
		c.WaveAngularFreq = 0.14
	}
	if c.WaveAmplitude < 0 {
		c.WaveAmplitude = 0
	}
	if c.PulseOmega <= 0 {
		c.PulseOmega = 0.11
	}
	if c.EdgeSoftness < 0 {
		c.EdgeSoftness = 0
	}
	return c
}

func clampSize(w, h int) (int, int) {
	if w < 3 {
		w = 3
	}
	if w > 200 {
		w = 200
	}
	if h < 3 {
		h = 3
	}
	if h > 200 {
		h = 200
	}
	return w, h
}

// ramp maps normalized intensity [0,1] to a luminance rune.
var ramp = []rune(" ··░▒▓██")

func intensityToRune(v float64) rune {
	if v <= 0 {
		return ramp[0]
	}
	if v >= 1 {
		return ramp[len(ramp)-1]
	}
	idx := int(v * float64(len(ramp)-1))
	if idx < 0 {
		idx = 0
	}
	if idx >= len(ramp) {
		idx = len(ramp) - 1
	}
	return ramp[idx]
}

func edgeFactor(sx, sy, softness float64) float64 {
	if softness <= 0 {
		return 1
	}
	// Rectangular soft vignette: 1 at center, falls toward edges (not a ring mask).
	d := math.Max(math.Abs(sx-0.5), math.Abs(sy-0.5)) * 2 // in [0,1] at edges
	edge := 1.0 - smoothstep(1.0-softness*2, 1.0, d)
	if edge < 0 {
		return 0
	}
	return edge
}

func smoothstep(edge0, edge1, x float64) float64 {
	t := (x - edge0) / (edge1 - edge0)
	if t < 0 {
		return 0
	}
	if t > 1 {
		return 1
	}
	return t * t * (3 - 2*t)
}

func hsvToRGB(h, s, v float64) (uint8, uint8, uint8) {
	h = math.Mod(h, 1)
	if h < 0 {
		h++
	}
	i := int(h * 6)
	f := h*6 - float64(i)
	p := v * (1 - s)
	q := v * (1 - f*s)
	t := v * (1 - (1-f)*s)
	var r, g, b float64
	switch i % 6 {
	case 0:
		r, g, b = v, t, p
	case 1:
		r, g, b = q, v, p
	case 2:
		r, g, b = p, v, t
	case 3:
		r, g, b = p, q, v
	case 4:
		r, g, b = t, p, v
	default:
		r, g, b = v, p, q
	}
	return uint8(r * 255), uint8(g * 255), uint8(b * 255)
}

func blobCenter(i, tick int, seed uint64) (float64, float64) {
	// Deterministic paths in the unit square (no polar ring mask).
	phase := float64(seed%1000) * 0.001
	t := float64(tick)
	fi := float64(i)
	cx := 0.5 + 0.38*math.Sin(t*0.052+fi*1.17+phase*6.28)
	cy := 0.5 + 0.38*math.Cos(t*0.041+fi*1.91+phase*4.2)
	// Wrap into [0,1] for seamless motion.
	cx = math.Mod(cx, 1)
	if cx < 0 {
		cx++
	}
	cy = math.Mod(cy, 1)
	if cy < 0 {
		cy++
	}
	return cx, cy
}

// ComputeFrame evaluates the scalar field and fills a Frame.
func ComputeFrame(config LiquidBlobConfig, opts RenderOpts) (Frame, error) {
	cfg := config.normalized()
	w, h := clampSize(opts.Width, opts.Height)
	if opts.Tick < 0 {
		return Frame{}, fmt.Errorf("anim: tick must be non-negative")
	}

	t := float64(opts.Tick)
	sigma2 := 2 * cfg.Sigma * cfg.Sigma
	pulse := 0.5 + 0.5*math.Sin(cfg.PulseOmega*t)

	cells := make([][]Cell, h)
	for row := 0; row < h; row++ {
		cells[row] = make([]Cell, w)
		sy := (float64(row) + 0.5) / float64(h)
		for col := 0; col < w; col++ {
			sx := (float64(col) + 0.5) / float64(w)
			var bsum float64
			for i := 0; i < cfg.BlobCount; i++ {
				cx, cy := blobCenter(i, opts.Tick, opts.Seed)
				dx := sx - cx
				dy := sy - cy
				bsum += math.Exp(-(dx*dx + dy*dy) / sigma2)
			}
			wave := math.Sin(cfg.WaveVectorX*sx + cfg.WaveVectorY*sy - cfg.WaveAngularFreq*t)
			phi := pulse * (bsum + cfg.WaveAmplitude*wave)
			phi *= edgeFactor(sx, sy, cfg.EdgeSoftness)
			// Normalize roughly to [0,1] for stable glyphs across sizes.
			v := math.Tanh(phi*1.4)*0.5 + 0.5
			hue := math.Mod(sx*0.55+sy*0.35+t*0.02+float64(opts.Seed%360)/360.0, 1)
			r, g, b := hsvToRGB(hue, 0.72, 0.15+0.82*v)
			cells[row][col] = Cell{
				Rune: intensityToRune(v),
				R:    r, G: g, B: b,
			}
		}
	}
	return Frame{Cells: cells, Tick: opts.Tick}, nil
}
