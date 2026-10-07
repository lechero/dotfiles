// Package tui is dotui's interactive view: a dashboard of it all, the
// Homebrew packages with their priorities and install state, what
// `chezmoi apply` would change, where the disk space goes, and what needs
// following up on GitHub, GitLab and Jira.
package tui

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/lechero/dotfiles/apps/internal/dotui/brew"
	"github.com/lechero/dotfiles/apps/internal/dotui/catalog"
	"github.com/lechero/dotfiles/apps/internal/dotui/chezmoi"
	"github.com/lechero/dotfiles/apps/internal/dotui/machine"
	"github.com/lechero/dotfiles/apps/internal/dotui/work"
)

// Config is what the view needs from the command line.
type Config struct {
	Catalog catalog.Catalog
	// File is the package list, passed on to `dotui install`.
	File string
	// Source is chezmoi's source directory, or "" for chezmoi's own.
	Source string
	// Self is this executable, which runs installs and waits afterwards.
	Self string
	// NewDisk starts the Disk tab's analyzer when the tab first opens; nil
	// leaves the tab out.
	NewDisk func() Disk
	// Services get a tab each, after the others.
	Services []work.Service
	// Cache keeps the services' last dashboards.
	Cache work.Cache

	// The dashboard's readers; each left nil leaves its part out.
	Outdated    func(ctx context.Context) ([]brew.Update, error)
	SourceState func(ctx context.Context, source string) (chezmoi.Source, error)
	Machine     func(ctx context.Context) machine.Info
	DiskStatus  func(ctx context.Context) (DiskStatus, error)
}

type tabKind int

const (
	dashboardTab tabKind = iota
	packagesTab
	dotfilesTab
	diskTab
	serviceTab
)

// tabEntry is one of the tabs along the top.
type tabEntry struct {
	kind    tabKind
	name    string
	service *serviceView // a service tab's
}

// headerHeight is the number of lines above the list.
const headerHeight = 2

// loadTimeout bounds `brew list` and `chezmoi status`.
const loadTimeout = time.Minute

type (
	brewLoadedMsg struct {
		inv brew.Inventory
		err error
	}
	chezmoiLoadedMsg struct {
		changes []chezmoi.Change
		err     error
	}
	execDoneMsg struct {
		what string
		err  error
	}
)

// Model is the Bubble Tea model.
type Model struct {
	cfg    Config
	keys   keyMap
	styles *styles

	tabs          []tabEntry
	tab           int // the one in front, in tabs
	width, height int
	maxPrio       int
	missingOnly   bool

	inv         *brew.Inventory
	brewErr     error
	brewLoading bool

	changes        []chezmoi.Change
	chezmoiErr     error
	chezmoiLoading bool

	packages list.Model
	dotfiles list.Model
	spinner  spinner.Model

	disk Disk // nil until the Disk tab first opens
	dash *dashboard

	// execute hands the terminal to a command; tests replace it.
	execute func(cmd *exec.Cmd, done tea.ExecCallback) tea.Cmd
}

// New builds the view. It starts loading when the program runs it.
func New(cfg Config) Model {
	st := newStyles(true)
	m := Model{
		cfg:            cfg,
		keys:           newKeyMap(),
		styles:         &st,
		maxPrio:        catalog.MaxPrio,
		brewLoading:    true,
		chezmoiLoading: true,
		spinner:        spinner.New(spinner.WithSpinner(spinner.Dot)),
		execute:        tea.ExecProcess,
	}
	m.packages = m.newList("package", "packages", m.keys.packageKeys)
	m.dotfiles = m.newList("change", "changes", m.keys.dotfileKeys)
	m.dotfiles.Title = "Changes to apply"
	m.tabs = []tabEntry{{kind: dashboardTab, name: "Dashboard"}, {kind: packagesTab, name: "Packages"}, {kind: dotfilesTab, name: "Dotfiles"}}
	if cfg.NewDisk != nil {
		m.tabs = append(m.tabs, tabEntry{kind: diskTab, name: "Disk"})
	}
	for _, svc := range cfg.Services {
		v := newServiceView(svc, m.styles, cfg.Cache, cfg.Self, cfg.File)
		m.tabs = append(m.tabs, tabEntry{kind: serviceTab, name: svc.Name(), service: v})
	}
	m.dash = newDashboard(m.styles, len(m.tabs))
	m.applyStyles(true)
	m.showPackages()
	m.rebuildNext() // from the services' cached dashboards
	return m
}

// kind is what the tab in front shows.
func (m Model) kind() tabKind { return m.tabs[m.tab].kind }

// services are the service tabs' views.
func (m Model) services() []*serviceView {
	var out []*serviceView
	for _, t := range m.tabs {
		if t.service != nil {
			out = append(out, t.service)
		}
	}
	return out
}

func (m Model) newList(singular, plural string, keys func() []key.Binding) list.Model {
	l := list.New(nil, rowDelegate{st: m.styles}, 0, 0)
	l.SetStatusBarItemName(singular, plural)
	l.DisableQuitKeybindings()
	l.AdditionalShortHelpKeys = keys
	l.AdditionalFullHelpKeys = keys
	l.StatusMessageLifetime = 4 * time.Second
	return l
}

// Init starts loading brew and chezmoi state, and checks the services.
func (m Model) Init() tea.Cmd {
	cmds := []tea.Cmd{m.spinner.Tick, loadBrew, loadChezmoi(m.cfg.Source), tea.RequestBackgroundColor}
	for _, v := range m.services() {
		cmds = append(cmds, v.check())
	}
	if len(m.cfg.Services) > 0 {
		cmds = append(cmds, refreshTick())
	}
	cmds = append(cmds, m.loadDashboard())
	return tea.Batch(cmds...)
}

func loadBrew() tea.Msg {
	ctx, cancel := context.WithTimeout(context.Background(), loadTimeout)
	defer cancel()
	inv, err := brew.List(ctx)
	return brewLoadedMsg{inv: inv, err: err}
}

func loadChezmoi(source string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), loadTimeout)
		defer cancel()
		changes, err := chezmoi.Status(ctx, source)
		return chezmoiLoadedMsg{changes: changes, err: err}
	}
}

// Update handles a message. The analyzer on the Disk tab gets everything
// that isn't dotui's own, even while another tab is open, so a scan keeps
// going in the background.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var diskCmd tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.packages.SetSize(msg.Width, msg.Height-headerHeight)
		m.dotfiles.SetSize(msg.Width, msg.Height-headerHeight)
		for _, v := range m.services() {
			v.setSize(msg.Width, msg.Height)
		}
		m.layoutDashboard()
		cmd := m.updateDisk(m.diskSize())
		return m, cmd

	case tea.BackgroundColorMsg:
		m.applyStyles(msg.IsDark())
		cmd := m.updateDisk(msg)
		return m, cmd

	case spinner.TickMsg:
		diskCmd = m.updateDisk(msg)
		if !m.brewLoading && !m.chezmoiLoading {
			return m, diskCmd
		}
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, tea.Batch(cmd, diskCmd)

	case brewLoadedMsg:
		m.brewLoading = false
		m.brewErr = msg.err
		if msg.err == nil {
			m.inv = &msg.inv
		}
		cmd := m.showPackages()
		return m, cmd

	case chezmoiLoadedMsg:
		m.chezmoiLoading = false
		m.chezmoiErr = msg.err
		m.changes = msg.changes
		cmd := m.showChanges()
		return m, cmd

	case checkedMsg:
		cmd := msg.v.update(msg)
		return m, tea.Batch(cmd, m.rebuildNext())
	case dashLoadedMsg:
		cmd := msg.v.update(msg)
		return m, tea.Batch(cmd, m.rebuildNext())
	case outdatedMsg, sourceMsg, machineMsg, diskMsg:
		m.updateDashboard(msg)
		return m, nil
	case detailLoadedMsg:
		return m, msg.v.update(msg)
	case actionDoneMsg:
		return m, msg.v.update(msg)
	case composedMsg:
		return m, msg.v.update(msg)
	case flashMsg:
		return m, msg.v.update(msg)
	case refreshMsg:
		cmds := []tea.Cmd{refreshTick(), m.loadDashboard()}
		for _, v := range m.services() {
			cmds = append(cmds, v.load())
		}
		return m, tea.Batch(cmds...)

	case execDoneMsg:
		status := m.styles.good.Render(msg.what + " finished")
		if msg.err != nil {
			status = m.styles.bad.Render(fmt.Sprintf("%s failed: %v", msg.what, msg.err))
		}
		cmd := tea.Batch(m.reload(), m.loadDashboard(), m.notify(status))
		return m, cmd

	case tea.KeyPressMsg:
		switch m.kind() {
		case dashboardTab:
			switch {
			case msg.String() == "ctrl+c":
				return m, tea.Quit
			case m.dash.confirm != nil: // the answer to its question
			case key.Matches(msg, m.keys.Quit):
				return m, tea.Quit
			case key.Matches(msg, m.keys.SwitchTab):
				cmd := m.switchTab(msg.String() == "shift+tab")
				return m, cmd
			}
			cmd := m.dashboardKey(msg)
			return m, cmd
		case diskTab:
			cmd := m.diskKey(msg)
			return m, cmd
		case serviceTab:
			cmd := m.serviceKey(m.tabs[m.tab].service, msg)
			return m, cmd
		}
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		// While typing a filter, every key belongs to the list.
		if m.active().FilterState() != list.Filtering {
			if model, cmd, handled := m.handleKey(msg); handled {
				return model, cmd
			}
		}
		cmd := m.updateList(m.kind(), msg)
		return m, cmd

	case tea.MouseMsg: // the mouse is only on for the Disk tab
		if m.kind() == diskTab {
			cmd := m.diskMouse(msg)
			return m, cmd
		}
		return m, nil

	case list.FilterMatchesMsg: // for the list being filtered, on the tab in front
		switch m.kind() {
		case diskTab:
			cmd := m.updateDisk(msg)
			return m, cmd
		case serviceTab:
			v := m.tabs[m.tab].service
			var cmd tea.Cmd
			*v.list(), cmd = v.list().Update(msg)
			return m, cmd
		}
		cmd := m.updateList(m.kind(), msg)
		return m, cmd

	default:
		diskCmd = m.updateDisk(msg)
	}

	// Anything else, like a status message timing out, may be for either list.
	cmd := tea.Batch(diskCmd, m.updateList(packagesTab, msg), m.updateList(dotfilesTab, msg))
	return m, cmd
}

// serviceKey handles a key on a service's tab. tab switches tabs and q
// quits, unless the tab is asking something or a filter is being typed.
func (m *Model) serviceKey(v *serviceView, msg tea.KeyPressMsg) tea.Cmd {
	if msg.String() == "ctrl+c" {
		return tea.Quit
	}
	if !v.takesKeys() {
		switch {
		case key.Matches(msg, m.keys.SwitchTab):
			return m.switchTab(msg.String() == "shift+tab")
		case key.Matches(msg, m.keys.Quit):
			return tea.Quit
		}
	}
	return v.key(msg)
}

// updateList passes msg to the list on tab t, if it has one.
func (m *Model) updateList(t tabKind, msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	switch t {
	case packagesTab:
		m.packages, cmd = m.packages.Update(msg)
	case dotfilesTab:
		m.dotfiles, cmd = m.dotfiles.Update(msg)
	}
	return cmd
}

// switchTab moves to the next tab, or the previous one.
func (m *Model) switchTab(back bool) tea.Cmd {
	n := len(m.tabs)
	next := (m.tab + 1) % n
	if back {
		next = (m.tab + n - 1) % n
	}
	return m.setTab(next)
}

func (m *Model) setTab(t int) tea.Cmd {
	m.tab = t
	if m.kind() == diskTab {
		return m.openDisk()
	}
	return nil
}

func (m Model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd, bool) {
	switch {
	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit, true
	case key.Matches(msg, m.keys.SwitchTab):
		cmd := m.switchTab(msg.String() == "shift+tab")
		return m, cmd, true
	case key.Matches(msg, m.keys.Refresh):
		cmd := m.reload()
		return m, cmd, true
	}

	if m.kind() == packagesTab {
		switch {
		case key.Matches(msg, m.keys.Prio):
			m.maxPrio = int(msg.String()[0] - '0')
			cmd := m.showPackages()
			return m, cmd, true
		case key.Matches(msg, m.keys.Missing):
			m.missingOnly = !m.missingOnly
			cmd := m.showPackages()
			return m, cmd, true
		case key.Matches(msg, m.keys.Install):
			it, ok := m.packages.SelectedItem().(pkgItem)
			if !ok {
				return m, nil, true
			}
			cmd := m.install([]pkgItem{it})
			return m, cmd, true
		case key.Matches(msg, m.keys.InstallAll):
			var items []pkgItem
			for _, li := range m.packages.VisibleItems() {
				items = append(items, li.(pkgItem))
			}
			cmd := m.install(items)
			return m, cmd, true
		}
		return m, nil, false
	}

	switch {
	case key.Matches(msg, m.keys.Diff):
		if len(m.changes) == 0 {
			cmd := m.notify("Nothing to diff: the dotfiles are in sync.")
			return m, cmd, true
		}
		cmd := chezmoi.Command(context.Background(), m.cfg.Source, "diff")
		return m, tea.ExecProcess(cmd, func(err error) tea.Msg { return execDoneMsg{what: "diff", err: err} }), true
	case key.Matches(msg, m.keys.Apply):
		if len(m.changes) == 0 {
			cmd := m.notify("Nothing to apply: the dotfiles are in sync.")
			return m, cmd, true
		}
		return m, m.runAndWait("apply", chezmoi.Command(context.Background(), m.cfg.Source, "apply").Args), true
	case key.Matches(msg, m.keys.Update):
		return m, m.runAndWait("update", chezmoi.Command(context.Background(), m.cfg.Source, "update").Args), true
	}
	return m, nil, false
}

// install runs `dotui install` for the missing packages among items.
func (m *Model) install(items []pkgItem) tea.Cmd {
	if m.inv == nil {
		return m.notify("Still checking what's installed.")
	}
	var names []string
	for _, it := range items {
		if it.known && it.state == brew.Missing {
			names = append(names, it.pkg.Name)
		}
	}
	if len(names) == 0 {
		return m.notify("Nothing to install: everything shown is installed.")
	}
	args := append([]string{m.cfg.Self, "install", "--file", m.cfg.File}, names...)
	return m.runAndWait("install", args)
}

// runAndWait hands the terminal to a command, through `dotui _exec`, which
// waits for Enter afterwards so the output can be read.
func (m Model) runAndWait(what string, args []string) tea.Cmd {
	cmd := exec.Command(m.cfg.Self, append([]string{"_exec"}, args...)...)
	return m.execute(cmd, func(err error) tea.Msg { return execDoneMsg{what: what, err: err} })
}

func (m *Model) reload() tea.Cmd {
	m.brewLoading, m.chezmoiLoading = true, true
	return tea.Batch(m.spinner.Tick, loadBrew, loadChezmoi(m.cfg.Source))
}

func (m *Model) notify(status string) tea.Cmd {
	if m.kind() == dashboardTab {
		m.dash.flash = status
		return nil
	}
	if m.kind() == packagesTab {
		return m.packages.NewStatusMessage(status)
	}
	return m.dotfiles.NewStatusMessage(status)
}

func (m *Model) showPackages() tea.Cmd {
	m.packages.Title = filterTitle(m.maxPrio, m.missingOnly)
	rows := visiblePackages(m.cfg.Catalog, m.inv, m.maxPrio, m.missingOnly)
	items := make([]list.Item, len(rows))
	for i, r := range rows {
		items[i] = r
	}
	return m.packages.SetItems(items)
}

func (m *Model) showChanges() tea.Cmd {
	items := make([]list.Item, len(m.changes))
	for i, c := range m.changes {
		items[i] = changeItem{change: c}
	}
	return m.dotfiles.SetItems(items)
}

func (m Model) active() list.Model {
	if m.kind() == packagesTab {
		return m.packages
	}
	return m.dotfiles
}

// View renders the tab line, then the Disk tab's analyzer or a summary line
// and the active list. The mouse is on for the Disk tab only, where it picks
// blocks on the map; elsewhere the terminal keeps selecting text.
func (m Model) View() tea.View {
	var content string
	switch m.kind() {
	case dashboardTab:
		content = m.dashboardView()
	case diskTab:
		body := m.styles.dim.Render(" Starting the disk analyzer…")
		if m.disk != nil {
			body = m.disk.Content()
		}
		content = lipgloss.JoinVertical(lipgloss.Left, m.tabsLine(), body)
	case serviceTab:
		content = lipgloss.JoinVertical(lipgloss.Left, m.tabsLine(), m.tabs[m.tab].service.render())
	default:
		content = lipgloss.JoinVertical(lipgloss.Left, m.tabsLine(), m.summaryLine(), m.active().View())
	}
	v := tea.NewView(content)
	v.AltScreen = true
	v.WindowTitle = "dotui"
	if m.kind() == diskTab {
		v.MouseMode = tea.MouseModeCellMotion
	}
	return v
}

// tabText draws tab t's name, and for a service how many items need you.
// The count is drawn beside the tab's style, not inside it: styling inside
// an underlined style garbles.
func (m Model) tabText(t int) string {
	style := m.tabStyle(t)
	if v := m.tabs[t].service; v != nil {
		if n := v.followUps(); n > 0 {
			return style.PaddingRight(0).Render(m.tabs[t].name) + " " + m.styles.warn.Render(fmt.Sprint(n)) + " "
		}
	}
	return style.Render(m.tabs[t].name)
}

func (m Model) tabStyle(t int) lipgloss.Style {
	if t == m.tab {
		return m.styles.tabActive
	}
	return m.styles.tabInactive
}

func (m Model) tabsLine() string {
	tabs := []string{m.styles.title.Render("dotui")}
	for t := range m.tabs {
		tabs = append(tabs, m.tabText(t))
	}
	line := strings.Join(tabs, " ")
	if m.brewLoading || m.chezmoiLoading {
		line += "  " + m.spinner.View() + m.styles.dim.Render(" checking…")
	}
	// The analyzer's own help can't mention tab, which is dotui's.
	if m.kind() == diskTab {
		hint := m.styles.dim.Render("tab switch view ")
		if gap := m.width - lipgloss.Width(line) - lipgloss.Width(hint); gap > 1 {
			line += strings.Repeat(" ", gap) + hint
		}
	}
	return line
}

// tabAt returns the tab whose name is at column x of the tab line.
func (m Model) tabAt(x int) (int, bool) {
	pos := lipgloss.Width(m.styles.title.Render("dotui"))
	for t := range m.tabs {
		pos++ // the space before each name
		w := lipgloss.Width(m.tabText(t))
		if x >= pos && x < pos+w {
			return t, true
		}
		pos += w
	}
	return 0, false
}

func (m Model) summaryLine() string {
	if m.kind() == dotfilesTab {
		return " " + m.dotfilesSummary()
	}
	if m.brewErr != nil {
		return " " + m.styles.bad.Render(m.brewErr.Error())
	}
	tallies := prioTallies(m.cfg.Catalog, m.inv)
	var parts []string
	for prio := catalog.MinPrio; prio <= catalog.MaxPrio; prio++ {
		t := tallies[prio]
		part := fmt.Sprintf("%s %s", m.styles.prio[prio].Render(fmt.Sprintf("P%d", prio)), catalog.PrioNames[prio])
		if m.inv != nil {
			count := fmt.Sprintf("%d/%d", t.installed, t.total)
			if t.installed < t.total {
				count = m.styles.bad.Render(count)
			} else {
				count = m.styles.good.Render(count)
			}
			part += " " + count
		}
		if prio > m.maxPrio {
			part = m.styles.dim.Render(fmt.Sprintf("P%d %s", prio, catalog.PrioNames[prio]))
		}
		parts = append(parts, part)
	}
	return " " + strings.Join(parts, "  ")
}

func (m Model) dotfilesSummary() string {
	source := m.cfg.Source
	if source == "" {
		source = "chezmoi's source"
	} else if home, err := os.UserHomeDir(); err == nil {
		source = strings.Replace(source, home, "~", 1)
	}
	switch {
	case m.chezmoiErr != nil:
		return m.styles.bad.Render(m.chezmoiErr.Error())
	case m.chezmoiLoading && m.changes == nil:
		return m.styles.dim.Render("Asking chezmoi…")
	case len(m.changes) == 0:
		return m.styles.good.Render("In sync with " + source)
	}
	conflicts := 0
	for _, c := range m.changes {
		if c.Conflict() {
			conflicts++
		}
	}
	line := fmt.Sprintf("%d %s to apply from %s", len(m.changes), plural(len(m.changes), "change", "changes"), source)
	if conflicts > 0 {
		line += m.styles.bad.Render(fmt.Sprintf(", %d changed outside chezmoi", conflicts))
	}
	return line
}

// applyStyles sets the colors for a dark or light terminal, including the
// lists' own.
func (m *Model) applyStyles(isDark bool) {
	*m.styles = newStyles(isDark)
	for _, l := range []*list.Model{&m.packages, &m.dotfiles} {
		l.Styles = list.DefaultStyles(isDark)
		l.Styles.Title = m.styles.accent
	}
	for _, v := range m.services() {
		for i := range v.lists {
			v.lists[i].Styles = list.DefaultStyles(isDark)
		}
		v.renderPage()
	}
	if m.dash != nil {
		m.dash.next.Styles = list.DefaultStyles(isDark)
	}
}

// filterTitle describes which packages are shown.
func filterTitle(maxPrio int, missingOnly bool) string {
	title := "All priorities"
	switch maxPrio {
	case catalog.MinPrio:
		title = "Priority 1"
	case catalog.MaxPrio:
	default:
		title = fmt.Sprintf("Priorities 1–%d", maxPrio)
	}
	if missingOnly {
		title += ", missing only"
	}
	return title
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}
