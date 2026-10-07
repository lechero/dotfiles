package tui

import (
	"cmp"
	"fmt"
	"io"
	"slices"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/glamour/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/lechero/dotfiles/apps/internal/disk/human"
	"github.com/lechero/dotfiles/apps/internal/dotui/work"
)

// pageChrome is the lines of an item's page around its scrolling content:
// two of header, a blank line, and the footer.
const pageChrome = 4

// maxNotes is how much of an item's activity its page shows.
const maxNotes = 20

func (st *styles) tone(t work.Tone) lipgloss.Style {
	switch t {
	case work.Info:
		return st.info
	case work.Good:
		return st.good
	case work.Warn:
		return st.warn
	case work.Bad:
		return st.bad
	}
	return st.dim
}

// workRow is an item in a list.
type workRow struct{ work.Item }

func (r workRow) FilterValue() string {
	return strings.Join([]string{r.Ref, r.Title, r.Where, r.Next, r.Author}, " ")
}

// workDelegate draws an item in two lines: its reference, title and age,
// then why it needs you, where it is and its state.
type workDelegate struct {
	v         *serviceView
	followUps bool // the follow-up list leads with why
}

func (d workDelegate) Height() int                         { return 2 }
func (d workDelegate) Spacing() int                        { return 1 }
func (d workDelegate) Update(tea.Msg, *list.Model) tea.Cmd { return nil }

func (d workDelegate) Render(w io.Writer, m list.Model, index int, li list.Item) {
	r, ok := li.(workRow)
	if !ok {
		return
	}
	st := d.v.st
	width := m.Width()
	cursor, title := "  ", st.text.Render(r.Title)
	if index == m.Index() {
		cursor, title = st.accent.Render("› "), st.accent.Render(r.Title)
	}
	age := ""
	if !r.Updated.IsZero() {
		age = st.dim.Render(human.Ago(d.v.now(), r.Updated))
	}
	line1 := cursor + st.accent.Render(r.Ref) + " " + title
	if room := width - lipgloss.Width(age) - 1; lipgloss.Width(line1) > room {
		line1 = ansi.Truncate(line1, max(0, room), "…")
	}
	if gap := width - lipgloss.Width(line1) - lipgloss.Width(age); gap > 0 {
		line1 += strings.Repeat(" ", gap) + age
	}

	var parts []string
	if d.followUps && r.Next != "" {
		parts = append(parts, st.tone(r.NextTone).Render(r.Next))
	}
	if r.Where != "" {
		parts = append(parts, st.dim.Render(r.Where))
	}
	if me, _, _ := strings.Cut(d.v.account, " "); r.Author != "" && r.Author != me {
		parts = append(parts, st.dim.Render("by "+r.Author))
	}
	for _, b := range r.Badges {
		if !d.followUps || b.Text != r.Next {
			parts = append(parts, st.tone(b.Tone).Render(b.Text))
		}
	}
	line2 := ansi.Truncate("    "+strings.Join(parts, st.dim.Render(" · ")), width, "…")
	_, _ = fmt.Fprint(w, line1+"\n"+line2)
}

// helpKeys are the keys for the selected item, for the list's help.
func (v *serviceView) helpKeys() []key.Binding {
	bindings := []key.Binding{
		key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "open")),
		key.NewBinding(key.WithKeys("o"), key.WithHelp("o", "browser")),
		key.NewBinding(key.WithKeys("y"), key.WithHelp("y", "copy link")),
	}
	if it, ok := v.selected(); ok {
		for _, a := range v.svc.Actions(it) {
			bindings = append(bindings, key.NewBinding(key.WithKeys(a.Key), key.WithHelp(a.Key, a.Help)))
		}
	}
	return append(bindings,
		key.NewBinding(key.WithKeys("1"), key.WithHelp(fmt.Sprintf("1-%d", len(v.lists)), "views")),
		key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")))
}

// overviewHeight is the lines of section totals above the follow-up list.
func (v *serviceView) overviewHeight() int {
	if v.dash == nil {
		return 0
	}
	return len(v.dash.Sections) + 2 // a blank line and the list's heading
}

// render draws the tab below dotui's tab line.
func (v *serviceView) render() string {
	return lipgloss.JoinVertical(lipgloss.Left, v.accountLine(), v.subTabs(), v.body())
}

func (v *serviceView) accountLine() string {
	st := v.st
	var line string
	switch v.status {
	case work.Connected:
		line = " " + st.good.Render("●") + " " + v.account
		if v.dash != nil {
			line += st.dim.Render(" · updated " + human.Ago(v.now(), v.dash.Fetched))
		}
	case work.SignedOut:
		line = " " + st.bad.Render("●") + " " + v.svc.CLI() + " isn't signed in"
	case work.NotInstalled:
		line = " " + st.bad.Render("●") + " " + v.svc.CLI() + " isn't installed"
	default:
		line = " " + st.dim.Render("● asking "+v.svc.CLI()+"…")
	}
	if v.fromCache && v.dash != nil {
		line += st.dim.Render(" · showing the last run's, from " + human.Ago(v.now(), v.dash.Fetched))
	}
	if v.loading {
		line += st.dim.Render(" · refreshing…")
	}
	switch {
	case v.choose != nil:
		picks := make([]string, len(v.choose.Choices))
		for i, c := range v.choose.Choices {
			picks[i] = st.accent.Render(fmt.Sprint(i+1)) + " " + c
		}
		line += "  " + st.warn.Render(capitalize(v.choose.Help)+" "+v.chooseItem.Ref+" to:") + " " +
			strings.Join(picks, st.dim.Render(" · ")) + st.dim.Render("  (esc: cancel)")
	case v.confirm != nil:
		line += "  " + st.warn.Render(v.confirm.Confirm+" y/n")
	case v.flash != "" && v.flashBad:
		line += "  " + st.bad.Render(v.flash)
	case v.flash != "":
		line += "  " + st.good.Render(v.flash)
	case v.dashErr != nil:
		line += "  " + st.bad.Render("Couldn't refresh: "+v.dashErr.Error())
	}
	return ansi.Truncate(line, v.width, "…")
}

func (v *serviceView) subTabs() string {
	if !v.showsDash() {
		return ""
	}
	names := []string{"Follow-ups"}
	counts := []int{len(v.lists[0].Items())}
	for _, s := range v.dash.Sections {
		names = append(names, s.Key)
		counts = append(counts, s.Total)
	}
	tabs := make([]string, len(names))
	for i, name := range names {
		label := fmt.Sprintf("%d %s %d", i+1, name, counts[i])
		if i == v.view && v.page == nil {
			tabs[i] = v.st.tabActive.Render(label)
		} else {
			tabs[i] = v.st.tabInactive.Render(label)
		}
	}
	return ansi.Truncate(strings.Join(tabs, ""), v.width, "…")
}

func (v *serviceView) body() string {
	st := v.st
	cli := st.accent.Render(v.svc.CLI())
	switch {
	case v.status == work.NotInstalled:
		return fmt.Sprintf("\n %s's tab reads your work through %s, which isn't installed.\n\n Press %s to install it with Homebrew.",
			v.svc.Name(), cli, st.accent.Render("i"))
	case v.status == work.SignedOut:
		return fmt.Sprintf("\n %s isn't signed in to %s.\n\n Press %s to sign in: dotui hands this terminal to %s.",
			cli, v.svc.Name(), st.accent.Render("c"), st.accent.Render(strings.Join(v.svc.Connect(), " ")))
	case v.status == work.Unknown && v.checkErr != nil:
		return fmt.Sprintf("\n %s\n\n Press %s to try again.", st.bad.Render(v.checkErr.Error()), st.accent.Render("r"))
	case !v.showsDash() && v.dashErr != nil:
		return fmt.Sprintf("\n %s\n\n Press %s to try again.", st.bad.Render(v.dashErr.Error()), st.accent.Render("r"))
	case !v.showsDash():
		return st.dim.Render("\n Loading " + v.svc.Name() + "…")
	case v.page != nil:
		return v.pageView()
	case v.view == 0:
		return lipgloss.JoinVertical(lipgloss.Left, v.overview(), v.followUpList())
	}
	s := v.dash.Sections[v.view-1]
	if len(s.Items) == 0 {
		return st.dim.Render("\n " + s.Empty)
	}
	return v.list().View()
}

// overview is a line per section: its total, then how many need you, and
// why, most urgent first.
func (v *serviceView) overview() string {
	st := v.st
	keyW, totalW := 0, 0
	for _, s := range v.dash.Sections {
		keyW = max(keyW, lipgloss.Width(s.Key))
		totalW = max(totalW, len(fmt.Sprint(s.Total)))
	}
	lines := make([]string, 0, len(v.dash.Sections)+2)
	for _, s := range v.dash.Sections {
		type reason struct {
			next    string
			tone    work.Tone
			urgency int
			count   int
		}
		var reasons []*reason
		for _, it := range s.Items {
			if it.Next == "" {
				continue
			}
			i := slices.IndexFunc(reasons, func(r *reason) bool { return r.next == it.Next })
			if i < 0 {
				reasons = append(reasons, &reason{next: it.Next, tone: it.NextTone, urgency: it.Urgency})
				i = len(reasons) - 1
			}
			reasons[i].count++
		}
		slices.SortStableFunc(reasons, func(a, b *reason) int { return cmp.Compare(b.urgency, a.urgency) })
		parts := make([]string, len(reasons))
		for i, r := range reasons {
			parts[i] = st.tone(r.tone).Render(fmt.Sprintf("%d %s", r.count, r.next))
		}
		detail := strings.Join(parts, st.dim.Render(" · "))
		if detail == "" {
			detail = st.dim.Render("—")
		}
		line := fmt.Sprintf(" %-*s %*d  %s", keyW, s.Key, totalW, s.Total, detail)
		lines = append(lines, ansi.Truncate(line, v.width, "…"))
	}
	return strings.Join(append(lines, "", " "+st.accent.Render("Follow up")), "\n")
}

func (v *serviceView) followUpList() string {
	if len(v.lists[0].Items()) == 0 {
		return v.st.good.Render("\n Nothing needs you right now.")
	}
	return v.lists[0].View()
}

func (v *serviceView) pageView() string {
	st := v.st
	p := v.page
	it := p.item
	header := " " + st.accent.Render(it.Ref) + " " + st.text.Bold(true).Render(it.Title)
	var sub []string
	for _, s := range []string{it.Where, it.Branch} {
		if s != "" {
			sub = append(sub, s)
		}
	}
	if it.Author != "" {
		sub = append(sub, "by "+it.Author)
	}
	if !it.Updated.IsZero() {
		sub = append(sub, "updated "+human.Ago(v.now(), it.Updated))
	}
	footer := v.pageFooter()
	return strings.Join([]string{
		ansi.Truncate(header, v.width, "…"),
		ansi.Truncate(" "+st.dim.Render(strings.Join(sub, " · ")), v.width, "…"),
		"",
		p.vp.View(),
		footer,
	}, "\n")
}

func (v *serviceView) pageFooter() string {
	st := v.st
	var parts []string
	for _, a := range v.svc.Actions(v.page.item) {
		parts = append(parts, st.accent.Render(a.Key)+" "+st.dim.Render(a.Help))
	}
	parts = append(parts, st.accent.Render("o")+" "+st.dim.Render("browser"), st.accent.Render("y")+" "+st.dim.Render("copy link"),
		st.accent.Render("r")+" "+st.dim.Render("refresh"), st.accent.Render("esc")+" "+st.dim.Render("back"))
	return ansi.Truncate(" "+strings.Join(parts, st.dim.Render(" · ")), v.width, "…")
}

// renderPage fills the open page's viewport for the current width.
func (v *serviceView) renderPage() {
	p := v.page
	if p == nil {
		return
	}
	p.rendered = v.width
	st := v.st
	var b strings.Builder
	switch {
	case p.err != nil:
		b.WriteString(" " + st.bad.Render(p.err.Error()) + "\n\n Press r to try again, or o to open it in the browser.")
	case p.detail == nil:
		b.WriteString(st.dim.Render(" Loading…"))
	default:
		v.writeDetail(&b, *p.detail)
	}
	p.vp.SetContent(b.String())
}

func (v *serviceView) writeDetail(b *strings.Builder, d work.Detail) {
	st := v.st
	labelW := 0
	for _, f := range d.Fields {
		labelW = max(labelW, lipgloss.Width(f.Label))
	}
	for _, f := range d.Fields {
		fmt.Fprintf(b, " %s  %s\n", st.dim.Render(fmt.Sprintf("%-*s", labelW, f.Label)), st.tone(f.Tone).Render(f.Value))
	}

	if len(d.Checks) > 0 {
		fmt.Fprintf(b, "\n %s  %s\n", st.accent.Render(cmp.Or(d.ChecksTitle, "Checks")), v.checkSummary(d))
		nameW := 0
		for _, c := range d.Checks {
			nameW = max(nameW, lipgloss.Width(c.Name))
		}
		nameW = min(nameW, max(10, v.width-20))
		icons := map[work.Tone]string{work.Good: "✓", work.Bad: "✗", work.Warn: "◷", work.Info: "•", work.Dim: "·"}
		for _, c := range d.Checks {
			name := ansi.Truncate(c.Name, nameW, "…")
			fmt.Fprintf(b, "  %s %-*s  %s\n", st.tone(c.Tone).Render(icons[c.Tone]), nameW, name, st.dim.Render(c.State))
		}
	}

	if body := strings.TrimSpace(d.Body); body != "" {
		fmt.Fprintf(b, "\n %s\n%s", st.accent.Render("Description"), v.markdown(body))
	}

	if len(d.Activity) > 0 {
		notes := d.Activity
		heading := "Activity"
		if len(notes) > maxNotes {
			heading = fmt.Sprintf("Activity, the last %d of %d", maxNotes, len(notes))
			notes = notes[len(notes)-maxNotes:]
		}
		fmt.Fprintf(b, "\n %s\n", st.accent.Render(heading))
		for _, n := range notes {
			fmt.Fprintf(b, "\n %s %s %s\n", st.text.Bold(true).Render(n.Author), n.What, st.dim.Render("· "+human.Ago(v.now(), n.When)))
			if text := strings.TrimSpace(n.Text); text != "" {
				b.WriteString(v.markdown(text))
			}
		}
	}
}

// checkSummary counts CI checks as failing, running, passing or not run,
// and other checks, like subtasks, by their state.
func (v *serviceView) checkSummary(d work.Detail) string {
	st := v.st
	var sum []string
	if d.ChecksTitle != "" {
		var states []string
		count := map[string]int{}
		tone := map[string]work.Tone{}
		for _, c := range d.Checks {
			if count[c.State] == 0 {
				states = append(states, c.State)
			}
			count[c.State]++
			tone[c.State] = c.Tone
		}
		for _, s := range states {
			sum = append(sum, st.tone(tone[s]).Render(fmt.Sprintf("%d %s", count[s], s)))
		}
		return strings.Join(sum, st.dim.Render(" · "))
	}
	tally := map[work.Tone]int{}
	for _, c := range d.Checks {
		tally[c.Tone]++
	}
	for _, t := range []struct {
		tone work.Tone
		word string
	}{{work.Bad, "failing"}, {work.Warn, "running"}, {work.Good, "passing"}, {work.Dim, "not run"}} {
		if n := tally[t.tone]; n > 0 {
			sum = append(sum, st.tone(t.tone).Render(fmt.Sprintf("%d %s", n, t.word)))
		}
	}
	return strings.Join(sum, st.dim.Render(" · "))
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// markdown renders text for the terminal, or wraps it as it is when it
// won't render.
func (v *serviceView) markdown(text string) string {
	style := "light"
	if v.st.isDark {
		style = "dark"
	}
	width := max(20, v.width-2)
	r, err := glamour.NewTermRenderer(glamour.WithStandardStyle(style), glamour.WithWordWrap(width))
	if err == nil {
		if out, err := r.Render(text); err == nil {
			return strings.TrimRight(out, "\n") + "\n"
		}
	}
	return lipgloss.NewStyle().Width(width).PaddingLeft(2).Render(text) + "\n"
}
