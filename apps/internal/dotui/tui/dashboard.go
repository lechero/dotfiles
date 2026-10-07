package tui

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"

	"github.com/lechero/dotfiles/apps/internal/dotui/brew"
	"github.com/lechero/dotfiles/apps/internal/dotui/chezmoi"
	"github.com/lechero/dotfiles/apps/internal/dotui/machine"
	"github.com/lechero/dotfiles/apps/internal/dotui/work"
)

// DiskStatus is what the dashboard's Disk card shows: the volume, and from
// manage-disk's records, the last clean and free space over time.
type DiskStatus struct {
	Free, Total int64
	LastClean   time.Time
	Freed       int64
	TrendAt     []time.Time
	Trend       []int64 // free space at each TrendAt
}

// dashPrio is how far the dashboard checks packages: core and daily, like
// `task deps:check`.
const dashPrio = 2

type (
	outdatedMsg struct {
		updates []brew.Update
		err     error
	}
	sourceMsg struct {
		source chezmoi.Source
		err    error
	}
	machineMsg struct{ info machine.Info }
	diskMsg    struct {
		status DiskStatus
		err    error
	}
)

// dashboard is the first tab's own state; the rest it reads from the other
// tabs.
type dashboard struct {
	updates     []brew.Update
	updatesErr  error
	updatesRead bool

	source    *chezmoi.Source
	sourceErr error

	machine *machine.Info

	disk    *DiskStatus
	diskErr error

	next list.Model // follow-ups from every service, most urgent first
	now  func() time.Time
}

// nextRow is a follow-up on the dashboard, with the tab it belongs to.
type nextRow struct {
	work.Item
	service string
	tab     int
}

func (r nextRow) FilterValue() string { return r.Title }

func newDashboard(st *styles, tabs int) *dashboard {
	d := &dashboard{now: time.Now}
	l := list.New(nil, nextDelegate{st: st}, 0, 0)
	l.SetShowTitle(false)
	l.SetFilteringEnabled(false)
	l.SetStatusBarItemName("follow-up", "follow-ups")
	l.DisableQuitKeybindings()
	l.StatusMessageLifetime = 4 * time.Second
	help := func() []key.Binding {
		return []key.Binding{
			key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "open")),
			key.NewBinding(key.WithKeys("1"), key.WithHelp(fmt.Sprintf("1-%d", tabs), "go to tab")),
			key.NewBinding(key.WithKeys("i"), key.WithHelp("i", "install missing")),
			key.NewBinding(key.WithKeys("u"), key.WithHelp("u", "update dotfiles")),
			key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh all")),
		}
	}
	l.AdditionalShortHelpKeys = help
	l.AdditionalFullHelpKeys = help
	d.next = l
	return d
}

// loadDashboard reads what only the dashboard shows, with the readers the
// command gave; a missing reader leaves its part out.
func (m Model) loadDashboard() tea.Cmd {
	var cmds []tea.Cmd
	if f := m.cfg.Outdated; f != nil {
		cmds = append(cmds, func() tea.Msg {
			ctx, cancel := context.WithTimeout(context.Background(), loadTimeout)
			defer cancel()
			updates, err := f(ctx)
			return outdatedMsg{updates: updates, err: err}
		})
	}
	if f := m.cfg.SourceState; f != nil {
		source := m.cfg.Source
		cmds = append(cmds, func() tea.Msg {
			ctx, cancel := context.WithTimeout(context.Background(), loadTimeout)
			defer cancel()
			s, err := f(ctx, source)
			return sourceMsg{source: s, err: err}
		})
	}
	if f := m.cfg.Machine; f != nil {
		cmds = append(cmds, func() tea.Msg {
			ctx, cancel := context.WithTimeout(context.Background(), loadTimeout)
			defer cancel()
			return machineMsg{info: f(ctx)}
		})
	}
	if f := m.cfg.DiskStatus; f != nil {
		cmds = append(cmds, func() tea.Msg {
			ctx, cancel := context.WithTimeout(context.Background(), loadTimeout)
			defer cancel()
			s, err := f(ctx)
			return diskMsg{status: s, err: err}
		})
	}
	return tea.Batch(cmds...)
}

// rebuildNext gathers every service's follow-ups into the dashboard's list,
// keeping the selection on the same item.
func (m *Model) rebuildNext() tea.Cmd {
	var rows []nextRow
	for t, entry := range m.tabs {
		v := entry.service
		if v == nil || !v.showsDash() {
			continue
		}
		for _, it := range v.lists[0].Items() {
			rows = append(rows, nextRow{Item: it.(workRow).Item, service: entry.name, tab: t})
		}
	}
	// Each service's are in order already; the most urgent go first.
	slices.SortStableFunc(rows, func(a, b nextRow) int { return cmp.Compare(b.Urgency, a.Urgency) })
	l := &m.dash.next
	keep := ""
	if r, ok := l.SelectedItem().(nextRow); ok {
		keep = r.ID
	}
	items := make([]list.Item, len(rows))
	selected := 0
	for i, r := range rows {
		items[i] = r
		if r.ID == keep {
			selected = i
		}
	}
	cmd := l.SetItems(items)
	l.Select(selected)
	d := nextDelegate{st: m.styles}
	for _, r := range rows {
		d.refW = max(d.refW, len([]rune(r.Ref)))
		d.reasonW = max(d.reasonW, len([]rune(r.reason())))
	}
	l.SetDelegate(d)
	return cmd
}

// missingUpTo lists the packages up to prio that aren't installed.
func (m Model) missingUpTo(prio int) []pkgItem {
	if m.inv == nil {
		return nil
	}
	return visiblePackages(m.cfg.Catalog, m.inv, prio, true)
}

// dashboardKey handles a key on the dashboard.
func (m *Model) dashboardKey(msg tea.KeyPressMsg) tea.Cmd {
	k := msg.String()
	if n := int(k[0] - '0'); len(k) == 1 && n >= 1 && n <= len(m.tabs) {
		return m.setTab(n - 1)
	}
	switch k {
	case "enter":
		r, ok := m.dash.next.SelectedItem().(nextRow)
		if !ok {
			return nil
		}
		cmd := m.setTab(r.tab)
		v := m.tabs[r.tab].service
		v.view = 0
		return tea.Batch(cmd, v.openPage(r.Item))
	case "i":
		missing := m.missingUpTo(dashPrio)
		if len(missing) == 0 {
			return nil
		}
		return m.install(missing)
	case "u":
		return m.runAndWait("update", chezmoi.Command(context.Background(), m.cfg.Source, "update").Args)
	case "r":
		cmds := []tea.Cmd{m.reload(), m.loadDashboard()}
		for _, v := range m.services() {
			if v.status == work.Connected {
				cmds = append(cmds, v.load())
			} else if !v.checking {
				cmds = append(cmds, v.check())
			}
		}
		return tea.Batch(cmds...)
	}
	var cmd tea.Cmd
	m.dash.next, cmd = m.dash.next.Update(msg)
	return cmd
}

// updateDashboard handles the dashboard's own messages.
func (m *Model) updateDashboard(msg tea.Msg) {
	d := m.dash
	switch msg := msg.(type) {
	case outdatedMsg:
		d.updates, d.updatesErr, d.updatesRead = msg.updates, msg.err, true
	case sourceMsg:
		d.sourceErr = msg.err
		if msg.err == nil {
			d.source = &msg.source
		}
	case machineMsg:
		d.machine = &msg.info
	case diskMsg:
		d.diskErr = msg.err
		if msg.err == nil {
			d.disk = &msg.status
		}
	}
}

// uptime says how long since t, coarsely: "3 days", "10 hours".
func uptime(now, t time.Time) string {
	d := now.Sub(t)
	switch {
	case d >= 48*time.Hour:
		return fmt.Sprintf("%d days", int(d.Hours()/24))
	case d >= 2*time.Hour:
		return fmt.Sprintf("%d hours", int(d.Hours()))
	case d >= time.Hour:
		return "1 hour"
	}
	return fmt.Sprintf("%d minutes", max(1, int(d.Minutes())))
}

// greeting fits the time of day.
func greeting(now time.Time) string {
	switch h := now.Hour(); {
	case h < 5:
		return "Up late"
	case h < 12:
		return "Good morning"
	case h < 18:
		return "Good afternoon"
	}
	return "Good evening"
}

// shorten keeps the first n of names, and says how many more there are.
func shorten(names []string, n int) string {
	if len(names) <= n {
		return strings.Join(names, ", ")
	}
	return strings.Join(names[:n], ", ") + fmt.Sprintf(" and %d more", len(names)-n)
}
