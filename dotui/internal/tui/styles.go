package tui

import (
	"charm.land/lipgloss/v2"

	"github.com/lechero/dotfiles/dotui/internal/catalog"
)

// styles are Catppuccin colors, Mocha on dark terminals and Latte on light
// ones, to match kitty and tmux.
type styles struct {
	title       lipgloss.Style
	tabActive   lipgloss.Style
	tabInactive lipgloss.Style
	dim         lipgloss.Style
	text        lipgloss.Style
	accent      lipgloss.Style
	good        lipgloss.Style
	bad         lipgloss.Style
	warn        lipgloss.Style
	info        lipgloss.Style
	prio        [catalog.MaxPrio + 1]lipgloss.Style
}

func newStyles(isDark bool) styles {
	pick := lipgloss.LightDark(isDark)
	var (
		mauve   = pick(lipgloss.Color("#8839ef"), lipgloss.Color("#cba6f7"))
		red     = pick(lipgloss.Color("#d20f39"), lipgloss.Color("#f38ba8"))
		peach   = pick(lipgloss.Color("#fe640b"), lipgloss.Color("#fab387"))
		yellow  = pick(lipgloss.Color("#df8e1d"), lipgloss.Color("#f9e2af"))
		green   = pick(lipgloss.Color("#40a02b"), lipgloss.Color("#a6e3a1"))
		teal    = pick(lipgloss.Color("#179299"), lipgloss.Color("#94e2d5"))
		blue    = pick(lipgloss.Color("#1e66f5"), lipgloss.Color("#89b4fa"))
		text    = pick(lipgloss.Color("#4c4f69"), lipgloss.Color("#cdd6f4"))
		overlay = pick(lipgloss.Color("#8c8fa1"), lipgloss.Color("#7f849c"))
		base    = pick(lipgloss.Color("#eff1f5"), lipgloss.Color("#1e1e2e"))
	)
	badge := lipgloss.NewStyle().Foreground(base).Bold(true).Padding(0, 1)
	return styles{
		title:       lipgloss.NewStyle().Foreground(base).Background(mauve).Bold(true).Padding(0, 1),
		tabActive:   lipgloss.NewStyle().Foreground(mauve).Bold(true).Underline(true).Padding(0, 1),
		tabInactive: lipgloss.NewStyle().Foreground(overlay).Padding(0, 1),
		dim:         lipgloss.NewStyle().Foreground(overlay),
		text:        lipgloss.NewStyle().Foreground(text),
		accent:      lipgloss.NewStyle().Foreground(mauve).Bold(true),
		good:        lipgloss.NewStyle().Foreground(green),
		bad:         lipgloss.NewStyle().Foreground(red),
		warn:        lipgloss.NewStyle().Foreground(yellow),
		info:        lipgloss.NewStyle().Foreground(blue),
		prio: [...]lipgloss.Style{
			1: badge.Background(peach),
			2: badge.Background(blue),
			3: badge.Background(teal),
			4: badge.Background(overlay),
		},
	}
}
