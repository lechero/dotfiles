package tui

import (
	"fmt"
	"io"
	"strings"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/lechero/dotfiles/apps/internal/dotui/brew"
	"github.com/lechero/dotfiles/apps/internal/dotui/catalog"
	"github.com/lechero/dotfiles/apps/internal/dotui/chezmoi"
)

// pkgItem is a package row. known is false until brew has been asked.
type pkgItem struct {
	pkg   catalog.Package
	state brew.State
	known bool
}

func (i pkgItem) FilterValue() string { return i.pkg.Name + " " + i.pkg.Note }

// changeItem is a `chezmoi status` row.
type changeItem struct {
	change chezmoi.Change
}

func (i changeItem) FilterValue() string { return i.change.Path }

// rowDelegate renders one-line rows with the shared styles.
type rowDelegate struct {
	st *styles
}

func (d rowDelegate) Height() int                         { return 1 }
func (d rowDelegate) Spacing() int                        { return 0 }
func (d rowDelegate) Update(tea.Msg, *list.Model) tea.Cmd { return nil }

func (d rowDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	cursor := "  "
	if index == m.Index() {
		cursor = d.st.accent.Render("▌ ")
	}
	var row string
	switch it := item.(type) {
	case pkgItem:
		row = d.packageRow(it, m.Width()-2)
	case changeItem:
		row = d.changeRow(it, m.Width()-2)
	}
	fmt.Fprint(w, cursor+row)
}

const nameWidth = 34

func (d rowDelegate) packageRow(it pkgItem, width int) string {
	icon := d.st.dim.Render("…")
	if it.known {
		switch it.state {
		case brew.Installed:
			icon = d.st.good.Render("✔")
		case brew.Outside:
			icon = d.st.warn.Render("◆")
		default:
			icon = d.st.bad.Render("✘")
		}
	}
	badge := d.st.prio[it.pkg.Prio].Render(fmt.Sprintf("P%d", it.pkg.Prio))
	name := lipgloss.NewStyle().Width(nameWidth).Render(ansi.Truncate(it.pkg.Name, nameWidth-1, "…"))
	kind := "    "
	if it.pkg.Cask {
		kind = "cask"
	}
	left := strings.Join([]string{icon, badge, d.st.text.Render(name), d.st.dim.Render(kind)}, " ")
	room := width - lipgloss.Width(left) - 2
	if room < 1 || it.pkg.Note == "" {
		return left
	}
	return left + "  " + d.st.dim.Render(ansi.Truncate(it.pkg.Note, room, "…"))
}

func (d rowDelegate) changeRow(it changeItem, width int) string {
	c := it.change
	var icon string
	switch c.Apply {
	case 'A':
		icon = d.st.good.Render("+")
	case 'D':
		icon = d.st.bad.Render("-")
	case 'R':
		icon = d.st.info.Render("▶")
	default:
		icon = d.st.warn.Render("~")
	}
	desc := c.Describe()
	descStyle := d.st.dim
	if c.Conflict() {
		icon = d.st.bad.Render("!")
		descStyle = d.st.bad
	}
	path := ansi.Truncate(c.Path, max(width-lipgloss.Width(desc)-6, 10), "…")
	return icon + " " + d.st.text.Render(path) + "  " + descStyle.Render(desc)
}
