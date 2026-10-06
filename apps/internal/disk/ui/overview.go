package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/lechero/dotfiles/apps/internal/disk/clean"
	"github.com/lechero/dotfiles/apps/internal/disk/human"
	"github.com/lechero/dotfiles/apps/internal/disk/scan"
)

func newSpotsTable() table.Model {
	s := table.DefaultStyles()
	s.Header = s.Header.BorderStyle(lipgloss.NormalBorder()).BorderForeground(colFaint).BorderBottom(true).Bold(true)
	s.Selected = s.Selected.Foreground(lipgloss.Color("#FFFFFF")).Background(lipgloss.Color("#6D28D9")).Bold(false)
	return table.New(table.WithFocused(true), table.WithStyles(s))
}

// refreshSpots fills the biggest-spots table for the current width.
func (a *app) refreshSpots() {
	sizeW, shareW, tagW := 10, 7, min(24, max(10, a.w/5))
	whereW := max(16, a.w-sizeW-shareW-tagW-8)
	a.spots.SetColumns([]table.Column{
		{Title: "Size", Width: sizeW}, {Title: "Share", Width: shareW},
		{Title: "Where", Width: whereW}, {Title: "Cleanable", Width: tagW},
	})
	tags := a.cleanTags()
	var total int64 = 1
	if a.res != nil {
		total = max(1, a.res.Root.Size)
	}
	rows := make([]table.Row, 0, len(a.hot))
	for _, n := range a.hot {
		rows = append(rows, table.Row{
			fmt.Sprintf("%*s", sizeW, human.Bytes(n.Size)),
			fmt.Sprintf("%5.1f%%", 100*float64(n.Size)/float64(total)),
			truncLeft(tildePath(a.home, n.Path()), whereW),
			truncRight(tags[n.Path()], tagW),
		})
	}
	a.spots.SetRows(rows)
}

func (a *app) overviewKey(msg tea.KeyMsg) tea.Cmd {
	if key.Matches(msg, a.keys.SpotOpen) {
		a.openSpot()
		return nil
	}
	var cmd tea.Cmd
	a.spots, cmd = a.spots.Update(msg)
	return cmd
}

// openSpot shows the selected spot in the map, inside its parent folder.
func (a *app) openSpot() {
	i := a.spots.Cursor()
	if a.res == nil || i < 0 || i >= len(a.hot) || a.hot[i].Parent == nil {
		return
	}
	spot := a.hot[i]
	a.exp.enter(spot.Parent.Path())
	a.exp.selectName(spot.Parent, spot.Name)
	a.mp.sel = spot.Name
	a.tab = tabMap
}

func (a *app) overviewMouse(msg tea.MouseMsg) tea.Cmd {
	if msg.Action != tea.MouseActionPress {
		return nil
	}
	switch msg.Button {
	case tea.MouseButtonWheelUp:
		a.spots.MoveUp(1)
	case tea.MouseButtonWheelDown:
		a.spots.MoveDown(1)
	}
	return nil
}

func (a *app) scanStatus() string {
	if a.scanning {
		p := a.prog
		elapsed := human.Duration(time.Since(a.scanStart))
		var line string
		if a.lastScan.Files > 0 {
			// The previous scan's file count makes a fair yardstick for this one.
			frac := min(0.99, float64(p.Files())/float64(a.lastScan.Files))
			a.scanBar.Width = min(40, max(10, a.w/3))
			line = "Scanning " + a.scanBar.ViewAs(frac) + sDim.Render(fmt.Sprintf("  %s files · %s · %s",
				human.Count(p.Files()), human.Bytes(p.Bytes()), elapsed))
		} else {
			line = fmt.Sprintf("%s Scanning… %s files · %s · %s", a.spin.View(),
				human.Count(p.Files()), human.Bytes(p.Bytes()), elapsed)
		}
		if cur := p.Current(); cur != "" {
			line += "  " + sDim.Render(truncLeft(tildePath(a.home, cur), max(10, a.w-lipgloss.Width(line)-2)))
		}
		if wait := p.Waiting(); wait != "" {
			line += "\n" + sYellow.Render("Waiting on "+truncLeft(tildePath(a.home, wait), max(10, a.w-90))+
				" — if macOS shows a permission dialog, either answer is fine; the scan moves on after 15s.")
		}
		return lipgloss.NewStyle().MaxWidth(a.w).Render(line)
	}
	if a.scanErr != nil {
		return sRed.Render("Scan failed: " + a.scanErr.Error())
	}
	if a.res == nil {
		return ""
	}
	return sDim.Render(fmt.Sprintf("Scanned %s files (%s) in %s", human.Count(a.res.Root.Files),
		human.Bytes(a.res.Root.Size), human.Duration(a.res.Took)))
}

func (a *app) overviewView(h int) string {
	var b strings.Builder
	line := func(s string) { b.WriteString(s + "\n") }
	status := a.scanStatus()
	if t := a.trendLine(); t != "" {
		if gap := a.w - lipgloss.Width(status) - lipgloss.Width(t); gap > 2 && !strings.Contains(status, "\n") {
			status += strings.Repeat(" ", gap) + t
		}
	}
	line(status)
	line("")
	if a.res == nil {
		line(sDim.Render("Measuring your home folder. Big homes take a minute; everything else works meanwhile."))
		line("")
		a.writeCleanable(line)
		return b.String()
	}

	total := a.res.Root.Size
	line(sHeading.Render("Where your space goes"))
	line(a.stackedBar(total, a.w))
	nameW, sizeW := 24, 10
	barW := max(8, min(40, a.w-nameW-sizeW-12))
	for _, c := range a.cats {
		name := c.Name
		if c.Partial {
			name += "*"
		}
		dot := lipgloss.NewStyle().Foreground(categoryColor(c.Name)).Render("■")
		if c.Size < 1<<20 {
			if c.Partial {
				line(fmt.Sprintf(" %s %s %s  %s", dot, pad(name, nameW), padLeft("—", sizeW), sDim.Render("not scanned, see * below")))
			}
			continue
		}
		frac := float64(c.Size) / float64(max(1, total))
		line(fmt.Sprintf(" %s %s %s  %s %s", dot, pad(name, nameW), padLeft(human.Bytes(c.Size), sizeW),
			bar(frac, barW, lipgloss.NewStyle().Foreground(categoryColor(c.Name))), sDim.Render(fmt.Sprintf("%3.0f%%", frac*100))))
	}
	line("")

	// The spots table gets whatever room is left above the cleanable summary.
	used := strings.Count(b.String(), "\n")
	room := h - used - 9
	if room >= 4 && len(a.hot) > 0 {
		line(sHeading.Render("Biggest spots") + sDim.Render("  ↑↓ choose · enter shows it in the map"))
		a.refreshSpots()
		a.spots.SetWidth(a.w)
		a.spots.SetHeight(min(room-1, len(a.hot)+2)) // +2: the header and its rule
		line(a.spots.View())
		line("")
	}
	a.writeCleanable(line)
	if nr := a.res.NoResponse; len(nr) > 0 {
		var names []string
		for _, p := range nr {
			names = append(names, tildePath(a.home, p))
		}
		line(sYellow.Render("! Never answered, so not measured (macOS may have asked for permission): " + strings.Join(names, ", ")))
	}
	if len(a.res.Private) > 0 && !a.opts.IncludePrivate {
		line(sDim.Render("* Not scanned, macOS guards them: " + strings.Join(scan.SummarizePrivate(a.res.Private), ", ") +
			". Press p to include them (macOS may ask for permission)."))
	}
	return b.String()
}

// stackedBar is the whole scan in one line, one coloured run per category.
func (a *app) stackedBar(total int64, width int) string {
	if total <= 0 || width <= 0 {
		return ""
	}
	var b strings.Builder
	used := 0
	for i, c := range a.cats {
		cells := int(float64(c.Size) / float64(total) * float64(width))
		if i == len(a.cats)-1 {
			cells = width - used
		}
		cells = min(cells, width-used)
		if cells <= 0 {
			continue
		}
		b.WriteString(lipgloss.NewStyle().Foreground(categoryColor(c.Name)).Render(strings.Repeat("█", cells)))
		used += cells
	}
	return b.String()
}

// trendLine is free space over the recorded samples, oldest to newest.
func (a *app) trendLine() string {
	if len(a.trend) < 2 {
		return ""
	}
	first, last := a.trend[0], a.trend[len(a.trend)-1]
	span := last.At.Sub(first.At)
	label := "free space"
	if span >= 24*time.Hour {
		label += fmt.Sprintf(", %d days", int(span.Hours()/24))
	}
	return sDim.Render(label+" ") + sAccent.Render(sparkline(a.trend, 30)) + sDim.Render(" "+human.Bytes(last.Free))
}

func (a *app) writeCleanable(line func(string)) {
	var t1, t2 int64
	var n1, n2, blocked1 int
	loading := false
	for _, ts := range a.tasks {
		loading = loading || ts.loading || ts.waiting
		size, n := ts.ready()
		if ts.task.Tier == clean.Tier1 {
			if ts.enabled {
				t1 += size
				n1 += n
				blocked1 += ts.blocked()
			}
		} else {
			t2 += size
			n2 += n
		}
	}
	head := sHeading.Render("Cleanable now")
	if loading {
		head += "  " + a.spin.View() + sDim.Render(" still measuring")
	}
	line(head)
	t1s := fmt.Sprintf(" Tier 1  %s in %s", sGreen.Render(human.Bytes(t1)), plural(n1, "item"))
	if blocked1 > 0 {
		t1s += sYellow.Render(fmt.Sprintf("  (%d blocked by running apps)", blocked1))
	}
	line(t1s + sDim.Render("  — caches, on by default"))
	line(fmt.Sprintf(" Tier 2  %s in %s", sYellow.Render(human.Bytes(t2)), plural(n2, "item")) +
		sDim.Render("  — costs a rebuild or re-download, off by default"))
	line(sDim.Render(" Press 4 to review and clean."))
}

func tildePath(home, p string) string {
	if p == home {
		return "~"
	}
	if rel, ok := strings.CutPrefix(p, home+"/"); ok {
		return "~/" + rel
	}
	return p
}

// cleanTags maps paths that a cleanup would delete to a short label.
func (a *app) cleanTags() map[string]string {
	tags := map[string]string{}
	for _, ts := range a.tasks {
		if ts.task.Run != nil {
			continue // commands decide what they delete
		}
		for _, it := range ts.items {
			for _, p := range it.Paths {
				tags[p] = fmt.Sprintf("T%d %s", ts.task.Tier, ts.task.Title)
			}
		}
	}
	return tags
}
