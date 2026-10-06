package ui

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/lucasb-eyer/go-colorful"
)

// shade is one colour family: a block, two alternating tones for the blocks
// nested inside it, and a brighter tone for when it is selected.
type shade struct {
	base, a, b, hi lipgloss.Color
}

// hues are ordered so that neighbouring indexes are far apart on the wheel.
var hues = []float64{210, 145, 28, 275, 340, 175, 48, 0, 235, 95, 305, 190}

var shades = func() []shade {
	out := make([]shade, len(hues))
	for i, h := range hues {
		out[i] = shade{base: hsl(h, 0.42, 0.30), a: hsl(h, 0.38, 0.38), b: hsl(h, 0.38, 0.45), hi: hsl(h, 0.62, 0.48)}
	}
	return out
}()

var greyShade = shade{base: "#3F434C", a: "#4B505A", b: "#555B66", hi: "#6B7280"}

// Text on blocks: light on every shade above, dimmer for secondary lines.
const (
	blockText    = lipgloss.Color("#F8FAFC")
	blockSubtext = lipgloss.Color("#CBD5E1")
)

func shadeFor(i int) shade {
	if i < 0 {
		return greyShade
	}
	return shades[i%len(shades)]
}

func hsl(h, s, l float64) lipgloss.Color { return lipgloss.Color(colorful.Hsl(h, s, l).Hex()) }

// categoryHue colours the overview's categories; the map reuses the same
// hues for the folders those categories are made of.
var categoryHue = map[string]float64{
	"Projects & worktrees":   210,
	"Docker VM":              190,
	"App & tool caches":      48,
	"Node & JS toolchains":   145,
	"Xcode & simulators":     235,
	"Android, JVM, Go, Rust": 28,
	"AI models & agents":     305,
	"Other app data":         175,
	"Media & personal":       340,
}

func categoryColor(name string) lipgloss.Color {
	if h, ok := categoryHue[name]; ok {
		return hsl(h, 0.62, 0.55)
	}
	return "#6B7280"
}
