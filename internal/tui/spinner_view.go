package tui

import (
	"fmt"
	"image/color"
	"strings"

	lipgloss "charm.land/lipgloss/v2"

	"github.com/GyeongHoKim/preflight/internal/anim"
)

// RenderOptions controls how RenderFrame projects a Frame to terminal output.
type RenderOptions struct {
	DisableColor bool
	DisableANSI  bool
}

// RenderFrame renders a semantic frame to a string for stdout or golden tests.
func RenderFrame(frame anim.Frame, opts RenderOptions) (s string) {
	defer func() {
		if r := recover(); r != nil {
			s = ""
		}
	}()
	if len(frame.Cells) == 0 {
		return ""
	}
	if opts.DisableANSI {
		return renderFramePlain(frame)
	}
	var b strings.Builder
	h := len(frame.Cells)
	if h == 0 {
		return ""
	}
	w := len(frame.Cells[0])
	for row := 0; row < h; row++ {
		line := frame.Cells[row]
		if len(line) != w {
			return renderFramePlain(frame)
		}
		for col := 0; col < w; col++ {
			c := line[col]
			ch := string(c.Rune)
			if opts.DisableColor {
				b.WriteString(ch)
				continue
			}
			st := lipgloss.NewStyle().Foreground(rgbHex(c.R, c.G, c.B))
			b.WriteString(st.Render(ch))
		}
		b.WriteByte('\n')
	}
	return b.String()
}

func rgbHex(r, g, b uint8) color.Color {
	return lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", r, g, b))
}

func renderFramePlain(frame anim.Frame) string {
	var b strings.Builder
	for _, row := range frame.Cells {
		for _, c := range row {
			r := c.Rune
			if r == 0 {
				r = ' '
			}
			b.WriteRune(r)
		}
		b.WriteByte('\n')
	}
	return b.String()
}
