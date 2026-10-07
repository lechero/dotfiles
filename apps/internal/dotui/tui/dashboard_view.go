package tui

import (
	"cmp"
	"fmt"
	"io"
	"slices"
	"strings"
	"time"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/lechero/dotfiles/apps/internal/disk/human"
	"github.com/lechero/dotfiles/apps/internal/dotui/catalog"
	"github.com/lechero/dotfiles/apps/internal/dotui/chezmoi"
	"github.com/lechero/dotfiles/apps/internal/dotui/work"
)

const (
	// cardLines is the lines inside every card, below its title, so the
	// grid's height doesn't depend on what's in it.
	cardLines = 4
	// cardHeight is a card with its title and border.
	cardHeight = cardLines + 3
	// dashTop is the lines above the cards: dotui's tab line, the greeting,
	// the Mac, and a blank line.
	dashTop = 4
	// nextHeading is the blank line and the heading above the follow-ups.
	nextHeading = 2
)

// cardColumns fits the cards to the width.
func cardColumns(width int) int {
	switch {
	case width >= 108:
		return 3
	case width >= 72:
		return 2
	}
	return 1
}

// cardCount is how many cards the dashboard has: packages, dotfiles, disk
// if there's a Disk tab, and a card per service.
func (m Model) cardCount() int {
	n := 2 + len(m.services())
	if m.cfg.NewDisk != nil {
		n++
	}
	return n
}

// layoutDashboard sizes the follow-up list to what the cards leave.
func (m *Model) layoutDashboard() {
	cols := cardColumns(m.width)
	rows := (m.cardCount() + cols - 1) / cols
	h := m.height - dashTop - rows*cardHeight - nextHeading
	m.dash.next.SetSize(m.width, max(3, h))
}

func (m Model) dashboardView() string {
	parts := []string{m.tabsLine(), m.greetingLine(), m.machineLine(), "", m.cardGrid(), "", m.nextHeadingLine()}
	if len(m.dash.next.Items()) == 0 {
		parts = append(parts, m.nextEmpty())
	} else {
		parts = append(parts, m.dash.next.View())
	}
	return strings.Join(parts, "\n")
}

func (m Model) greetingLine() string {
	st := m.styles
	if c := m.dash.confirm; c != nil {
		return ansi.Truncate(" "+st.warn.Render(c.question)+st.accent.Render(" y/n"), m.width, "…")
	}
	now := m.dash.now()
	hello := greeting(now)
	if mi := m.dash.machine; mi != nil && mi.FirstName() != "" {
		hello += ", " + mi.FirstName()
	}
	return " " + st.accent.Render(hello) + st.dim.Render(" · "+now.Format("Monday 2 January"))
}

func (m Model) machineLine() string {
	st := m.styles
	mi := m.dash.machine
	if mi == nil {
		return " " + st.dim.Render("Reading this Mac…")
	}
	var parts []string
	for _, s := range []string{mi.Name, mi.MacOS} {
		if s != "" {
			parts = append(parts, s)
		}
	}
	if mi.MacOS != "" {
		parts[len(parts)-1] = "macOS " + mi.MacOS
	}
	if !mi.Booted.IsZero() {
		parts = append(parts, "up "+uptime(m.dash.now(), mi.Booted))
	}
	line := st.dim.Render(strings.Join(parts, " · "))
	if mi.Load > 0 {
		load := fmt.Sprintf("load %.1f", mi.Load)
		if mi.CPUs > 0 {
			load += fmt.Sprintf(" on %d cores", mi.CPUs)
		}
		style := st.dim
		if mi.CPUs > 0 && mi.Load > float64(mi.CPUs) {
			style = st.warn
		}
		line += st.dim.Render(" · ") + style.Render(load)
	}
	if mi.Battery >= 0 {
		battery := fmt.Sprintf("battery %d%%", mi.Battery)
		switch {
		case mi.Charging:
			battery += ", charging"
		case mi.OnPower:
			battery += ", on power"
		}
		style := st.dim
		if mi.Battery <= 20 && !mi.OnPower {
			style = st.bad
		}
		line += st.dim.Render(" · ") + style.Render(battery)
	}
	return ansi.Truncate(" "+line, m.width, "…")
}

// card draws a box with a title, its tab's number, and cardLines lines.
func (m Model) card(title string, tab int, lines []string, width int) string {
	st := m.styles
	inner := max(10, width-4) // border and padding
	head := st.accent.Render(title) + st.dim.Render(fmt.Sprintf("  %d", tab+1))
	body := make([]string, cardLines)
	for i := range body {
		if i < len(lines) {
			body[i] = ansi.Truncate(lines[i], inner, "…")
		}
	}
	return lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(st.dim.GetForeground()).
		Padding(0, 1).Width(width).Render(strings.Join(append([]string{head}, body...), "\n"))
}

func (m Model) cardGrid() string {
	cols := cardColumns(m.width)
	width := (m.width - (cols - 1)) / cols
	inner := max(10, width-4)
	var cards []string
	for t, entry := range m.tabs {
		var lines []string
		switch entry.kind {
		case packagesTab:
			lines = m.packagesCard()
		case dotfilesTab:
			lines = m.dotfilesCard(inner)
		case diskTab:
			lines = m.diskCard(inner)
		case serviceTab:
			lines = m.serviceCard(entry.service, inner)
		default:
			continue
		}
		cards = append(cards, m.card(entry.name, t, lines, width))
	}
	var rows []string
	for i := 0; i < len(cards); i += cols {
		row := cards[i:min(i+cols, len(cards))]
		rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, intersperse(row, " ")...))
	}
	return strings.Join(rows, "\n")
}

func intersperse(items []string, sep string) []string {
	out := make([]string, 0, 2*len(items))
	for i, it := range items {
		if i > 0 {
			out = append(out, sep)
		}
		out = append(out, it)
	}
	return out
}

func (m Model) packagesCard() []string {
	st := m.styles
	switch {
	case m.brewErr != nil:
		return []string{st.bad.Render(m.brewErr.Error())}
	case m.inv == nil:
		return []string{st.dim.Render("Asking Homebrew…")}
	}
	tallies := prioTallies(m.cfg.Catalog, m.inv)
	counts := make([]string, 0, catalog.MaxPrio)
	for prio := catalog.MinPrio; prio <= catalog.MaxPrio; prio++ {
		t := tallies[prio]
		if t.total == 0 {
			counts = append(counts, st.dim.Render(fmt.Sprintf("P%d —", prio)))
			continue
		}
		style := st.good
		if t.installed < t.total {
			style = st.warn
			if prio <= dashPrio {
				style = st.bad
			}
		}
		counts = append(counts, st.dim.Render(fmt.Sprintf("P%d ", prio))+style.Render(fmt.Sprintf("%d/%d", t.installed, t.total)))
	}
	lines := []string{strings.Join(counts, st.dim.Render(" · "))}
	if missing := m.missingUpTo(dashPrio); len(missing) > 0 {
		names := make([]string, len(missing))
		for i, p := range missing {
			names[i] = p.pkg.Name
		}
		lines = append(lines, st.bad.Render(fmt.Sprintf("✗ %d missing: %s", len(missing), shorten(names, 2)))+st.dim.Render(" · i installs"))
	} else {
		lines = append(lines, st.good.Render(fmt.Sprintf("✓ Everything up to P%d is installed", dashPrio)))
	}
	d := m.dash
	switch {
	case d.updatesErr != nil:
		lines = append(lines, st.dim.Render("Couldn't check for updates"))
	case !d.updatesRead:
		lines = append(lines, st.dim.Render("Checking for updates…"))
	case len(d.updates) == 0:
		lines = append(lines, st.good.Render("✓ Nothing to update"))
	default:
		names := make([]string, len(d.updates))
		for i, u := range d.updates {
			names[i] = u.Name
		}
		lines = append(lines,
			st.warn.Render(fmt.Sprintf("↑ %d %s", len(d.updates), plural(len(d.updates), "update", "updates")))+st.dim.Render(" · U upgrades"),
			st.dim.Render(shorten(names, 3)))
	}
	return lines
}

func (m Model) dotfilesCard(width int) []string {
	st := m.styles
	var lines []string
	switch {
	case m.chezmoiErr != nil:
		lines = append(lines, st.bad.Render(m.chezmoiErr.Error()))
	case m.chezmoiLoading && m.changes == nil:
		lines = append(lines, st.dim.Render("Asking chezmoi…"))
	case len(m.changes) == 0:
		lines = append(lines, st.good.Render("✓ In sync with the source"))
	default:
		conflicts := 0
		for _, c := range m.changes {
			if c.Conflict() {
				conflicts++
			}
		}
		if conflicts > 0 {
			lines = append(lines, fit([]string{
				st.warn.Render(fmt.Sprintf("%d to apply", len(m.changes))),
				st.bad.Render(fmt.Sprintf("%d changed outside chezmoi", conflicts)),
			}, st.dim.Render(" · "), width))
		} else {
			lines = append(lines, st.warn.Render(fmt.Sprintf("%d %s to apply", len(m.changes), plural(len(m.changes), "change", "changes"))))
		}
	}

	d := m.dash
	s := d.source
	switch {
	case d.sourceErr != nil:
		return append(lines, st.bad.Render(d.sourceErr.Error()))
	case s == nil:
		return append(lines, st.dim.Render("Reading the source…"))
	}
	where := s.Branch + " · " + s.Head
	if !s.When.IsZero() {
		where += " · " + human.Ago(d.now(), s.When)
	}
	lines = append(lines, st.dim.Render(where), ansi.Truncate(s.Subject, width, "…"))
	// Facts first; the hint goes when there's no room for it.
	var state []string
	hint := ""
	switch s.Remote {
	case chezmoi.RemoteSame:
		state = append(state, st.good.Render("✓ Up to date with origin"))
	case chezmoi.RemoteNewer:
		state = append(state, st.warn.Render("↓ origin is ahead"))
		hint = st.dim.Render("u updates")
	case chezmoi.RemoteOlder:
		state = append(state, st.warn.Render("↑ not pushed"))
	default:
		state = append(state, st.dim.Render("Couldn't ask origin"))
	}
	if s.Dirty > 0 {
		state = append(state, st.warn.Render(fmt.Sprintf("%d uncommitted", s.Dirty)))
	}
	if hint != "" {
		state = append(state, hint)
	}
	return append(lines, fit(state, st.dim.Render(" · "), width))
}

func (m Model) diskCard(width int) []string {
	st := m.styles
	d := m.dash
	switch {
	case d.diskErr != nil:
		return []string{st.bad.Render(d.diskErr.Error())}
	case d.disk == nil:
		return []string{st.dim.Render("Measuring…")}
	}
	ds := d.disk
	used := 0.0
	if ds.Total > 0 {
		used = float64(ds.Total-ds.Free) / float64(ds.Total)
	}
	fullness := st.good
	switch {
	case used >= 0.9:
		fullness = st.bad
	case used >= 0.8:
		fullness = st.warn
	}
	free := st.text.Bold(true).Render(human.Bytes(ds.Free)) + st.dim.Render(" free")
	barW := max(6, width-lipgloss.Width(free)-1)
	filled := int(used*float64(barW) + 0.5)
	bar := fullness.Render(strings.Repeat("▌", filled)) + st.dim.Render(strings.Repeat("░", barW-filled))
	lines := []string{
		bar + " " + free,
		st.dim.Render(fmt.Sprintf("of %s · ", human.Bytes(ds.Total))) + fullness.Render(fmt.Sprintf("%.0f%% full", used*100)),
		m.trendLine(width),
	}
	if ds.LastClean.IsZero() {
		lines = append(lines, st.dim.Render("Never cleaned with manage-disk"))
	} else {
		lines = append(lines, st.dim.Render("Last clean "+human.Ago(d.now(), ds.LastClean)+", freed "+human.Bytes(ds.Freed)))
	}
	return lines
}

// trendLine is free space over the last week: a sparkline, and the change.
func (m Model) trendLine(width int) string {
	st := m.styles
	ds := m.dash.disk
	since := m.dash.now().Add(-7 * 24 * time.Hour)
	var free []int64
	var first time.Time
	for i, at := range ds.TrendAt {
		if at.After(since) && i < len(ds.Trend) {
			if first.IsZero() {
				first = at
			}
			free = append(free, ds.Trend[i])
		}
	}
	if len(free) < 2 {
		return st.dim.Render("No free-space trend yet")
	}
	change := free[len(free)-1] - free[0]
	text := "steady"
	style := st.dim
	switch {
	case change <= -(1 << 30):
		text, style = "↓ "+human.Bytes(-change), st.warn
	case change >= 1<<30:
		text, style = "↑ "+human.Bytes(change), st.good
	}
	label := style.Render(text) + st.dim.Render(" since "+human.Ago(m.dash.now(), first))
	return sparkline(free, max(4, width-lipgloss.Width(label)-1)) + " " + label
}

var sparks = []rune("▁▂▃▄▅▆▇█")

// sparkline draws the last width values between their own low and high.
func sparkline(values []int64, width int) string {
	if len(values) > width {
		values = values[len(values)-width:]
	}
	lo, hi := slices.Min(values), slices.Max(values)
	var b strings.Builder
	for _, v := range values {
		i := 0
		if hi > lo {
			i = int(float64(v-lo) / float64(hi-lo) * float64(len(sparks)-1))
		}
		b.WriteRune(sparks[i])
	}
	return b.String()
}

func (m Model) serviceCard(v *serviceView, width int) []string {
	st := m.styles
	cli := v.svc.CLI()
	switch v.status {
	case work.NotInstalled:
		return []string{st.bad.Render(cli + " isn't installed"), st.dim.Render("i on its tab installs it")}
	case work.SignedOut:
		return []string{st.bad.Render(cli + " isn't signed in"), st.dim.Render("c on its tab signs in")}
	}
	var lines []string
	switch {
	case v.status == work.Connected:
		lines = append(lines, st.good.Render("● ")+v.account)
	case v.checkErr != nil:
		return []string{st.bad.Render(v.checkErr.Error())}
	default:
		lines = append(lines, st.dim.Render("● asking "+cli+"…"))
	}
	if !v.showsDash() {
		if v.dashErr != nil {
			return append(lines, st.bad.Render(v.dashErr.Error()))
		}
		return append(lines, st.dim.Render("Loading…"))
	}
	totals := make([]string, len(v.dash.Sections))
	for i, s := range v.dash.Sections {
		totals[i] = st.text.Bold(true).Render(fmt.Sprint(s.Total)) + st.dim.Render(" "+s.Key)
	}
	lines = append(lines, strings.Join(totals, st.dim.Render(" · ")))

	type reason struct {
		next    string
		tone    work.Tone
		urgency int
		count   int
	}
	var reasons []*reason
	for _, it := range v.lists[0].Items() {
		r := it.(workRow)
		i := slices.IndexFunc(reasons, func(x *reason) bool { return x.next == r.Next })
		if i < 0 {
			reasons = append(reasons, &reason{next: r.Next, tone: r.NextTone, urgency: r.Urgency})
			i = len(reasons) - 1
		}
		reasons[i].count++
	}
	if len(reasons) == 0 {
		return append(lines, st.good.Render("✓ Nothing needs you"))
	}
	slices.SortStableFunc(reasons, func(a, b *reason) int { return cmp.Compare(b.urgency, a.urgency) })
	parts := make([]string, len(reasons))
	for i, r := range reasons {
		parts[i] = st.tone(r.tone).Render(fmt.Sprintf("%d %s", r.count, r.next))
	}
	return append(lines, fill(parts, st.dim.Render(" · "), width, cardLines-len(lines))...)
}

// fit joins as many of parts as fit in width, in order, by sep: the first
// always, cut short if it must.
func fit(parts []string, sep string, width int) string {
	line := ""
	for _, p := range parts {
		switch {
		case line == "":
			line = p
		case lipgloss.Width(line+sep+p) <= width:
			line += sep + p
		default:
			return line
		}
	}
	return line
}

// fill puts parts on up to n lines of width, in order, joined by sep.
func fill(parts []string, sep string, width, n int) []string {
	var lines []string
	line := ""
	for _, p := range parts {
		switch {
		case line == "":
			line = p
		case lipgloss.Width(line+sep+p) <= width:
			line += sep + p
		default:
			lines = append(lines, line)
			line = p
		}
	}
	if line != "" {
		lines = append(lines, line)
	}
	if len(lines) > n {
		lines = lines[:n]
	}
	return lines
}

func (m Model) nextHeadingLine() string {
	st := m.styles
	var names []string
	for _, v := range m.services() {
		names = append(names, v.svc.Name())
	}
	line := " " + st.accent.Render("Next up")
	if len(names) > 0 {
		line += st.dim.Render(" across " + joinAnd(names))
	}
	if m.dash.flash != "" {
		line += "  " + m.dash.flash
	}
	return ansi.Truncate(line, m.width, "…")
}

func (m Model) nextEmpty() string {
	if len(m.services()) == 0 {
		return m.styles.dim.Render(" No services to follow up.")
	}
	return m.styles.good.Render(" Nothing needs you right now.")
}

// joinAnd joins names as a sentence: "GitHub, GitLab and Jira".
func joinAnd(names []string) string {
	if len(names) < 2 {
		return strings.Join(names, "")
	}
	return strings.Join(names[:len(names)-1], ", ") + " and " + names[len(names)-1]
}

// nextDelegate draws a follow-up on one line, in columns: its service,
// reference and title, then why it needs you and where it is.
type nextDelegate struct {
	st *styles
	// refW and reasonW are the widest reference and reason, so the columns
	// line up.
	refW, reasonW int
}

// reason is the right-hand column of a follow-up.
func (r nextRow) reason() string {
	if r.Where == "" {
		return r.Next
	}
	return r.Next + " · " + r.Where
}

func (d nextDelegate) Height() int                         { return 1 }
func (d nextDelegate) Spacing() int                        { return 0 }
func (d nextDelegate) Update(tea.Msg, *list.Model) tea.Cmd { return nil }

func (d nextDelegate) Render(w io.Writer, l list.Model, index int, li list.Item) {
	r, ok := li.(nextRow)
	if !ok {
		return
	}
	st := d.st
	cursor, title := "  ", st.text.Render(r.Title)
	if index == l.Index() {
		cursor, title = st.accent.Render("› "), st.accent.Render(r.Title)
	}
	left := cursor + st.dim.Render(fmt.Sprintf("%-7s", r.service)) + " " + st.accent.Render(fmt.Sprintf("%-*s", d.refW, r.Ref)) + " "
	rightW := min(d.reasonW, l.Width()*2/5)
	right := st.tone(r.NextTone).Render(r.Next)
	if r.Where != "" {
		right += st.dim.Render(" · " + r.Where)
	}
	right = ansi.Truncate(right, rightW, "…")
	right += strings.Repeat(" ", max(0, rightW-lipgloss.Width(right)))
	room := l.Width() - lipgloss.Width(left) - rightW - 2
	title = ansi.Truncate(title, max(5, room), "…")
	gap := max(2, l.Width()-lipgloss.Width(left)-lipgloss.Width(title)-rightW)
	_, _ = fmt.Fprint(w, left+title+strings.Repeat(" ", gap)+right)
}
