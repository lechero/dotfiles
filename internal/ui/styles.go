package ui

import (
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
)

var (
	colAccent = lipgloss.AdaptiveColor{Light: "#7D3AED", Dark: "#A78BFA"}
	colGreen  = lipgloss.AdaptiveColor{Light: "#15803D", Dark: "#4ADE80"}
	colYellow = lipgloss.AdaptiveColor{Light: "#B45309", Dark: "#FBBF24"}
	colRed    = lipgloss.AdaptiveColor{Light: "#B91C1C", Dark: "#F87171"}
	colDim    = lipgloss.AdaptiveColor{Light: "#6B7280", Dark: "#8B8F98"}
	colFaint  = lipgloss.AdaptiveColor{Light: "#D1D5DB", Dark: "#3A3F4B"}
	colText   = lipgloss.AdaptiveColor{Light: "#111827", Dark: "#E5E7EB"}

	sTitle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFFFFF")).Background(lipgloss.Color("#7D3AED")).Padding(0, 1)
	sDryBadge = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#1F2937")).Background(lipgloss.Color("#FBBF24")).Padding(0, 1)
	sTab      = lipgloss.NewStyle().Foreground(colDim).Padding(0, 1)
	sTabOn    = lipgloss.NewStyle().Bold(true).Foreground(colAccent).Underline(true).Padding(0, 1)
	sHeading  = lipgloss.NewStyle().Bold(true).Foreground(colText)
	sDim      = lipgloss.NewStyle().Foreground(colDim)
	sAccent   = lipgloss.NewStyle().Foreground(colAccent)
	sGreen    = lipgloss.NewStyle().Foreground(colGreen)
	sYellow   = lipgloss.NewStyle().Foreground(colYellow)
	sRed      = lipgloss.NewStyle().Foreground(colRed)
	sBold     = lipgloss.NewStyle().Bold(true)
	sCursor   = lipgloss.NewStyle().Foreground(colAccent).Bold(true)
	sBox      = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(colAccent).Padding(0, 1)
)

var eighths = []string{"", "▏", "▎", "▍", "▌", "▋", "▊", "▉"}

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

// volumeBar draws used against free across width cells, colored by how full it is.
func volumeBar(used, total int64, width int) string {
	if total <= 0 || width <= 0 {
		return ""
	}
	full := float64(used) / float64(total)
	u := min(width, int(math.Round(full*float64(width))))
	style := sGreen
	if full > 0.85 {
		style = sRed
	} else if full > 0.70 {
		style = sYellow
	}
	return style.Render(strings.Repeat("█", u)) +
		lipgloss.NewStyle().Foreground(colFaint).Render(strings.Repeat("░", width-u))
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
