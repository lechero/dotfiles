package ui

import (
	"image/color"
	"math"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/mattn/go-runewidth"
)

// The colours and styles below suit a dark terminal until setTheme hears the
// terminal's real background colour.
var (
	colAccent, colGreen, colYellow, colRed, colDim, colFaint, colText color.Color

	sTab, sTabOn, sHeading, sDim, sAccent, sGreen, sYellow, sRed, sCursor, sBox lipgloss.Style

	sTitle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFFFFF")).Background(lipgloss.Color("#7D3AED")).Padding(0, 1)
	sDryBadge = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#1F2937")).Background(lipgloss.Color("#FBBF24")).Padding(0, 1)
	sBold     = lipgloss.NewStyle().Bold(true)
)

func init() { setTheme(true) }

// setTheme picks the colours for a dark or a light terminal background.
func setTheme(isDark bool) {
	pick := lipgloss.LightDark(isDark)
	colAccent = pick(lipgloss.Color("#7D3AED"), lipgloss.Color("#A78BFA"))
	colGreen = pick(lipgloss.Color("#15803D"), lipgloss.Color("#4ADE80"))
	colYellow = pick(lipgloss.Color("#B45309"), lipgloss.Color("#FBBF24"))
	colRed = pick(lipgloss.Color("#B91C1C"), lipgloss.Color("#F87171"))
	colDim = pick(lipgloss.Color("#6B7280"), lipgloss.Color("#8B8F98"))
	colFaint = pick(lipgloss.Color("#D1D5DB"), lipgloss.Color("#3A3F4B"))
	colText = pick(lipgloss.Color("#111827"), lipgloss.Color("#E5E7EB"))

	sTab = lipgloss.NewStyle().Foreground(colDim).Padding(0, 1)
	sTabOn = lipgloss.NewStyle().Bold(true).Foreground(colAccent).Underline(true).Padding(0, 1)
	sHeading = lipgloss.NewStyle().Bold(true).Foreground(colText)
	sDim = lipgloss.NewStyle().Foreground(colDim)
	sAccent = lipgloss.NewStyle().Foreground(colAccent)
	sGreen = lipgloss.NewStyle().Foreground(colGreen)
	sYellow = lipgloss.NewStyle().Foreground(colYellow)
	sRed = lipgloss.NewStyle().Foreground(colRed)
	sCursor = lipgloss.NewStyle().Foreground(colAccent).Bold(true)
	sBox = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(colAccent).Padding(0, 1)
}

var eighths = []string{"", "▏", "▎", "▍", "▌", "▋", "▊", "▉"}

// barStyle colours an explorer bar like that entry's block on the map.
func barStyle(hue int) lipgloss.Style {
	if hue < 0 {
		return sDim
	}
	return lipgloss.NewStyle().Foreground(shadeFor(hue).hi)
}

// bar draws frac (0–1) of width cells, with eighth-cell precision.
func bar(frac float64, width int, style lipgloss.Style) string {
	if width <= 0 {
		return ""
	}
	frac = math.Max(0, math.Min(1, frac))
	cells := frac * float64(width)
	full := int(cells)
	part := int((cells - float64(full)) * 8)
	s := strings.Repeat("█", full)
	used := full
	if part > 0 && full < width {
		s += eighths[part]
		used++
	}
	return style.Render(s) + lipgloss.NewStyle().Foreground(colFaint).Render(strings.Repeat("·", width-used))
}

// truncLeft keeps the end of s, which is the telling part of a path.
func truncLeft(s string, width int) string {
	if width <= 0 {
		return ""
	}
	if runewidth.StringWidth(s) <= width {
		return s
	}
	rs := []rune(s)
	w := 1 // the ellipsis
	i := len(rs)
	for i > 0 && w+runewidth.RuneWidth(rs[i-1]) <= width {
		i--
		w += runewidth.RuneWidth(rs[i])
	}
	return "…" + string(rs[i:])
}

// truncRight keeps the start of s.
func truncRight(s string, width int) string {
	if width <= 0 {
		return ""
	}
	if runewidth.StringWidth(s) <= width {
		return s
	}
	return runewidth.Truncate(s, width, "…")
}

// pad fits s, which may carry styles, into exactly width cells.
func pad(s string, width int) string {
	if w := lipgloss.Width(s); w < width {
		return s + strings.Repeat(" ", width-w)
	}
	return s
}

// padLeft right-aligns s in width cells.
func padLeft(s string, width int) string {
	if w := lipgloss.Width(s); w < width {
		return strings.Repeat(" ", width-w) + s
	}
	return s
}
