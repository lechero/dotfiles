// Package tui is dotui's interactive view: the Homebrew packages with their
// priorities and install state, and what `chezmoi apply` would change.
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
}

type tab int

const (
	packagesTab tab = iota
	dotfilesTab
)

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

	tab         tab
	maxPrio     int
	missingOnly bool

	inv         *brew.Inventory
	brewErr     error
	brewLoading bool

	changes        []chezmoi.Change
	chezmoiErr     error
	chezmoiLoading bool

	packages list.Model
	dotfiles list.Model
	spinner  spinner.Model
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
	}
	m.packages = m.newList("package", "packages", m.keys.packageKeys)
	m.dotfiles = m.newList("change", "changes", m.keys.dotfileKeys)
	m.dotfiles.Title = "Changes to apply"
	m.applyStyles(true)
	m.showPackages()
	return m
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

// Init starts loading brew and chezmoi state.
func (m Model) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, loadBrew, loadChezmoi(m.cfg.Source), tea.RequestBackgroundColor)
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

// Update handles a message.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.packages.SetSize(msg.Width, msg.Height-headerHeight)
		m.dotfiles.SetSize(msg.Width, msg.Height-headerHeight)
		return m, nil

	case tea.BackgroundColorMsg:
		m.applyStyles(msg.IsDark())
		return m, nil

	case spinner.TickMsg:
		if !m.brewLoading && !m.chezmoiLoading {
			return m, nil
		}
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

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

	case execDoneMsg:
		status := m.styles.good.Render(msg.what + " finished")
		if msg.err != nil {
			status = m.styles.bad.Render(fmt.Sprintf("%s failed: %v", msg.what, msg.err))
		}
		cmd := tea.Batch(m.reload(), m.notify(status))
		return m, cmd

	case tea.KeyPressMsg:
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		// While typing a filter, every key belongs to the list.
		if m.active().FilterState() != list.Filtering {
			if model, cmd, handled := m.handleKey(msg); handled {
				return model, cmd
			}
		}
	}

	var cmd tea.Cmd
	if m.tab == packagesTab {
		m.packages, cmd = m.packages.Update(msg)
	} else {
		m.dotfiles, cmd = m.dotfiles.Update(msg)
	}
	return m, cmd
}

func (m Model) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd, bool) {
	switch {
	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit, true
	case key.Matches(msg, m.keys.SwitchTab):
		m.tab = 1 - m.tab
		return m, nil, true
	case key.Matches(msg, m.keys.Refresh):
		cmd := m.reload()
		return m, cmd, true
	}

	if m.tab == packagesTab {
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
	return tea.ExecProcess(cmd, func(err error) tea.Msg { return execDoneMsg{what: what, err: err} })
}

func (m *Model) reload() tea.Cmd {
	m.brewLoading, m.chezmoiLoading = true, true
	return tea.Batch(m.spinner.Tick, loadBrew, loadChezmoi(m.cfg.Source))
}

func (m *Model) notify(status string) tea.Cmd {
	if m.tab == packagesTab {
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
	if m.tab == packagesTab {
		return m.packages
	}
	return m.dotfiles
}

// View renders the header and the active list.
func (m Model) View() tea.View {
	header := lipgloss.JoinVertical(lipgloss.Left, m.tabsLine(), m.summaryLine())
	v := tea.NewView(lipgloss.JoinVertical(lipgloss.Left, header, m.active().View()))
	v.AltScreen = true
	v.WindowTitle = "dotui"
	return v
}

func (m Model) tabsLine() string {
	tabs := []string{m.styles.title.Render("dotui")}
	for t, name := range []string{"Packages", "Dotfiles"} {
		if tab(t) == m.tab {
			tabs = append(tabs, m.styles.tabActive.Render(name))
		} else {
			tabs = append(tabs, m.styles.tabInactive.Render(name))
		}
	}
	line := strings.Join(tabs, " ")
	if m.brewLoading || m.chezmoiLoading {
		line += "  " + m.spinner.View() + m.styles.dim.Render(" checking…")
	}
	return line
}

func (m Model) summaryLine() string {
	if m.tab == dotfilesTab {
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
