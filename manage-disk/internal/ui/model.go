// Package ui is the manage-disk terminal interface.
package ui

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/stopwatch"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"manage-disk/internal/clean"
	"manage-disk/internal/docker"
	"manage-disk/internal/human"
	"manage-disk/internal/scan"
)

// Options are set from the command line.
type Options struct {
	DryRun         bool  // delete and run nothing; show what would happen
	IncludePrivate bool  // also scan Desktop, Documents, Downloads… (macOS may ask)
	DockerTarget   int64 // free space the Docker tab plans for (docker.DefaultTarget when 0)
}

const (
	tabOverview = iota
	tabMap
	tabExplorer
	tabClean
	tabWorktrees
	tabDocker
	tabCount
)

var tabNames = []string{"1 Overview", "2 Map", "3 Explorer", "4 Clean", "5 Worktrees", "6 Docker"}

// taskState is a cleanup task plus what discovery found for it.
type taskState struct {
	task    *clean.Task
	enabled bool // part of the next run
	items   []clean.Item
	loading bool
	waiting bool // a heavy task waiting for the scan tree
	err     error
}

// ready sums what a run would clean.
func (t *taskState) ready() (size int64, n int) {
	for _, it := range t.items {
		if it.Ready() {
			size += it.Size
			n++
		}
	}
	return size, n
}

// blocked counts chosen items that cannot be cleaned right now.
func (t *taskState) blocked() int {
	n := 0
	for _, it := range t.items {
		if it.Selected && it.Blocked != "" {
			n++
		}
	}
	return n
}

type (
	tickMsg time.Time
	volMsg  struct {
		vol   scan.Volume
		trend []freeSample
	}
	historyMsg  struct{ last *clean.HistoryEntry }
	scanDoneMsg struct {
		gen int
		res *scan.Result
		err error
	}
	discoverMsg struct {
		gen, idx int
		items    []clean.Item
		err      error
	}
	runEventMsg  struct{ ev clean.Event }
	runDoneMsg   struct{ sum clean.Summary }
	remeasureMsg struct {
		path string
		node *scan.Node // nil when the path is gone
	}
	revealMsg struct{ err error }
)

type tabHit struct{ x0, x1, tab int }

type app struct {
	env  *clean.Env
	home string
	opts Options
	w, h int
	tab  int

	keys    keyMap
	helpBar help.Model
	flash   string
	headerH int
	tabHits []tabHit

	vol      scan.Volume
	volOK    bool
	gauge    progress.Model
	trend    []freeSample
	last     *clean.HistoryEntry
	lastScan scanStats

	scanning   bool
	scanGen    int
	scanCancel context.CancelFunc
	prog       *scan.Progress
	scanStart  time.Time
	scanBar    progress.Model
	res        *scan.Result
	scanErr    error
	cats       []scan.Category
	hot        []*scan.Node
	spots      table.Model

	discGen int
	tasks   []*taskState

	spin     spinner.Model
	spinning bool
	exp      explorerState
	mp       mapState
	cl       cleanState
	wt       wtState
	dk       dkState
	run      runPanel
}

// Model is the Bubble Tea model. It wraps a pointer so helpers can mutate state.
type Model struct{ a *app }

// New builds the interface for env.
func New(env *clean.Env, opts Options) Model {
	if opts.DockerTarget <= 0 {
		opts.DockerTarget = docker.DefaultTarget
	}
	a := &app{
		env:      env,
		home:     env.Home,
		opts:     opts,
		keys:     newKeyMap(),
		helpBar:  help.New(),
		spin:     spinner.New(spinner.WithSpinner(spinner.MiniDot), spinner.WithStyle(sAccent)),
		gauge:    progress.New(progress.WithScaledGradient("#4ADE80", "#F87171"), progress.WithoutPercentage()),
		scanBar:  progress.New(progress.WithDefaultGradient()),
		lastScan: loadScanStats(env.StateDir),
		spots:    newSpotsTable(),
	}
	for _, t := range clean.Catalog() {
		a.tasks = append(a.tasks, &taskState{task: t, enabled: t.Tier == clean.Tier1})
	}
	a.exp = newExplorerState(env.Home)
	a.mp.nested = true
	a.cl = newCleanState()
	a.wt = newWtState()
	a.dk = newDkState()
	a.run = newRunPanel()
	return Model{a}
}

func (m Model) Init() tea.Cmd                           { return m.a.init() }
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) { return m, m.a.update(msg) }
func (m Model) View() string                            { return m.a.view() }

func (a *app) init() tea.Cmd {
	return tea.Batch(a.loadVolume(), a.loadHistory(), a.startScan(), a.discoverAll(), a.auditWorktrees(false), a.auditDocker())
}

// busy reports whether anything on screen is still in motion.
func (a *app) busy() bool {
	if a.scanning || a.run.active || a.cl.mode == modePreparing || a.wt.loading || a.dk.loading {
		return true
	}
	for _, ts := range a.tasks {
		if ts.loading || ts.waiting {
			return true
		}
	}
	return false
}

// wake restarts the spinner when work begins. A second tick chain is
// harmless: the spinner drops ticks carrying an old tag.
func (a *app) wake() tea.Cmd {
	if a.spinning {
		return nil
	}
	a.spinning = true
	return a.spin.Tick
}

func tick() tea.Cmd {
	return tea.Tick(150*time.Millisecond, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func (a *app) update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.w, a.h = msg.Width, msg.Height
		a.helpBar.Width = msg.Width
	case spinner.TickMsg:
		if !a.busy() { // nothing moving: let the spinner sleep instead of redrawing ten times a second
			a.spinning = false
			return nil
		}
		var cmd tea.Cmd
		a.spin, cmd = a.spin.Update(msg)
		return cmd
	case progress.FrameMsg:
		m, cmd := a.run.bar.Update(msg)
		a.run.bar = m.(progress.Model)
		return cmd
	case stopwatch.TickMsg, stopwatch.StartStopMsg, stopwatch.ResetMsg:
		var cmd tea.Cmd
		a.run.watch, cmd = a.run.watch.Update(msg)
		return cmd
	case list.FilterMatchesMsg: // lists filter in the background; the one on screen asked
		var cmd tea.Cmd
		switch a.tab {
		case tabWorktrees:
			a.wt.list, cmd = a.wt.list.Update(msg)
		case tabDocker:
			a.dk.list, cmd = a.dk.list.Update(msg)
		default:
			a.cl.items, cmd = a.cl.items.Update(msg)
		}
		return cmd
	case wtAuditMsg:
		return a.onWorktreeAudit(msg)
	case dkAuditMsg:
		return a.onDockerAudit(msg)
	case tickMsg:
		if a.scanning || a.run.active {
			return tick()
		}
	case volMsg:
		a.vol, a.volOK = msg.vol, true
		if msg.trend != nil {
			a.trend = msg.trend
		}
	case historyMsg:
		a.last = msg.last
	case scanDoneMsg:
		return a.onScanDone(msg)
	case discoverMsg:
		return a.onDiscover(msg)
	case runEventMsg:
		return a.onRunEvent(msg.ev)
	case runDoneMsg:
		return a.onRunDone(msg.sum)
	case remeasureMsg:
		a.env.EditTree(func(root *scan.Node) {
			if msg.node == nil {
				root.Remove(msg.path)
			} else {
				root.Replace(msg.path, msg.node)
			}
		})
		a.refreshInsights()
	case revealMsg:
		if msg.err != nil {
			a.flash = "couldn't reveal in Finder: " + msg.err.Error()
		}
	case tea.MouseMsg:
		return a.onMouse(msg)
	case tea.KeyMsg:
		return a.onKey(msg)
	}
	return nil
}

// typing reports whether a text field has the keyboard, so letters go to it
// instead of switching tabs or quitting.
func (a *app) typing() bool {
	switch a.tab {
	case tabExplorer:
		return a.exp.filtering
	case tabClean:
		return a.cl.mode == modeDetail && a.cl.items.SettingFilter()
	case tabWorktrees:
		return a.wt.mode == wtList && a.wt.list.SettingFilter()
	case tabDocker:
		return a.dk.mode == dkList && a.dk.list.SettingFilter()
	}
	return false
}

// modal reports whether the tab in front is in a flow (confirming, running…)
// that takes every key and keeps you on it.
func (a *app) modal() bool {
	switch a.tab {
	case tabClean:
		return a.cl.modal()
	case tabWorktrees:
		return a.wt.mode != wtList
	case tabDocker:
		return a.dk.mode != dkList
	}
	return false
}

func (a *app) onKey(msg tea.KeyMsg) tea.Cmd {
	a.flash = ""
	k := a.keys
	switch {
	case a.modal() || a.typing():
		switch a.tab {
		case tabExplorer:
			return a.explorerKey(msg)
		case tabWorktrees:
			return a.wtKey(msg)
		case tabDocker:
			return a.dkKey(msg)
		}
		return a.cleanKey(msg)
	case key.Matches(msg, k.Quit):
		if a.scanCancel != nil {
			a.scanCancel()
		}
		return tea.Quit
	case key.Matches(msg, k.Help):
		a.helpBar.ShowAll = !a.helpBar.ShowAll
		return nil
	case key.Matches(msg, k.Overview):
		a.setTab(tabOverview)
		return nil
	case key.Matches(msg, k.Map):
		a.setTab(tabMap)
		return nil
	case key.Matches(msg, k.Explorer):
		a.setTab(tabExplorer)
		return nil
	case key.Matches(msg, k.Clean):
		a.setTab(tabClean)
		return nil
	case key.Matches(msg, k.Worktrees):
		a.setTab(tabWorktrees)
		return nil
	case key.Matches(msg, k.Docker):
		a.setTab(tabDocker)
		return nil
	case key.Matches(msg, k.NextTab):
		a.setTab((a.tab + 1) % tabCount)
		return nil
	case key.Matches(msg, k.PrevTab):
		a.setTab((a.tab + tabCount - 1) % tabCount)
		return nil
	case a.tab == tabClean || a.tab == tabWorktrees || a.tab == tabDocker:
		// these tabs give r and p their own meaning
	case key.Matches(msg, k.Rescan):
		return tea.Batch(a.startScan(), a.discoverAll())
	case key.Matches(msg, k.Private):
		a.opts.IncludePrivate = !a.opts.IncludePrivate
		if a.opts.IncludePrivate {
			a.flash = "Rescanning with Desktop, Documents, Downloads, app containers… — macOS may ask for permission"
		}
		return a.startScan()
	}
	switch a.tab {
	case tabOverview:
		return a.overviewKey(msg)
	case tabMap:
		return a.mapKey(msg)
	case tabExplorer:
		return a.explorerKey(msg)
	case tabWorktrees:
		return a.wtKey(msg)
	case tabDocker:
		return a.dkKey(msg)
	default:
		return a.cleanKey(msg)
	}
}

// setTab switches tabs; the map and the explorer hand over what is selected,
// since both look at the same folder.
func (a *app) setTab(t int) {
	if a.res != nil {
		n := a.exp.current(a.res.Root)
		switch {
		case a.tab == tabExplorer && t == tabMap:
			if rows := a.exp.rows(n); a.exp.cursor < len(rows) {
				a.mp.sel = rows[a.exp.cursor].Name
			}
		case a.tab == tabMap && t == tabExplorer:
			a.exp.selectName(n, a.mp.sel)
		}
	}
	a.tab = t
}

func (a *app) onMouse(msg tea.MouseMsg) tea.Cmd {
	if msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft && msg.Y == 1 {
		for _, h := range a.tabHits {
			if msg.X >= h.x0 && msg.X < h.x1 && !a.modal() {
				a.setTab(h.tab)
				return nil
			}
		}
	}
	switch a.tab {
	case tabMap:
		return a.mapMouse(msg)
	case tabExplorer:
		return a.explorerMouse(msg)
	case tabClean:
		return a.cleanMouse(msg)
	case tabWorktrees:
		return a.wtMouse(msg)
	case tabDocker:
		return a.dkMouse(msg)
	default:
		return a.overviewMouse(msg)
	}
}

func (a *app) loadVolume() tea.Cmd {
	home, dir := a.home, a.env.StateDir
	return func() tea.Msg {
		v, err := scan.VolumeOf(home)
		if err != nil {
			return nil
		}
		return volMsg{vol: v, trend: recordFree(dir, v.Free, time.Now())}
	}
}

func (a *app) loadHistory() tea.Cmd {
	env := a.env
	return func() tea.Msg { return historyMsg{clean.LastRun(env)} }
}

func (a *app) scanOptions() scan.Options {
	opts := scan.DefaultOptions(a.home)
	opts.IncludePrivate = a.opts.IncludePrivate
	return opts
}

// startScan (re)measures the home folder; the old tree stays on screen meanwhile.
func (a *app) startScan() tea.Cmd {
	if a.scanCancel != nil {
		a.scanCancel()
	}
	a.scanGen++
	gen := a.scanGen
	ctx, cancel := context.WithCancel(context.Background())
	a.scanCancel = cancel
	a.prog = &scan.Progress{}
	a.scanning = true
	a.scanStart = time.Now()
	home, opts, prog := a.home, a.scanOptions(), a.prog
	return tea.Batch(tick(), a.wake(), func() tea.Msg {
		res, err := scan.Scan(ctx, home, opts, prog)
		return scanDoneMsg{gen: gen, res: res, err: err}
	})
}

func (a *app) onScanDone(msg scanDoneMsg) tea.Cmd {
	if msg.gen != a.scanGen {
		return nil
	}
	a.scanning = false
	if msg.res == nil || errors.Is(msg.err, context.Canceled) {
		a.scanErr = msg.err
		return nil
	}
	a.res, a.scanErr = msg.res, nil
	a.env.SetTree(msg.res.Root)
	a.refreshInsights()
	stats := scanStats{Files: msg.res.Root.Files, Bytes: msg.res.Root.Size, Took: msg.res.Took, At: time.Now()}
	a.lastScan = stats
	dir := a.env.StateDir
	cmds := []tea.Cmd{func() tea.Msg { saveScanStats(dir, stats); return nil }}
	// Heavy tasks size their items off the tree, so they start now.
	for i, ts := range a.tasks {
		if ts.task.Heavy {
			cmds = append(cmds, a.discover(i))
		}
	}
	return tea.Batch(cmds...)
}

func (a *app) refreshInsights() {
	if a.res == nil {
		return
	}
	a.cats = scan.Categories(a.res.Root)
	a.hot = scan.Hotspots(a.res.Root, 15, 256<<20)
	a.refreshSpots()
}

// discoverAll re-finds every task's items with a fresh process snapshot.
func (a *app) discoverAll() tea.Cmd {
	a.env.ResetProcs()
	a.discGen++
	cmds := []tea.Cmd{a.wake()}
	for i, ts := range a.tasks {
		if ts.task.Heavy && a.res == nil {
			ts.waiting = true
			continue
		}
		cmds = append(cmds, a.discover(i))
	}
	return tea.Batch(cmds...)
}

func (a *app) discover(i int) tea.Cmd {
	ts := a.tasks[i]
	ts.loading, ts.waiting = true, false
	gen, env, task := a.discGen, a.env, ts.task
	return func() tea.Msg {
		items, err := task.Discover(context.Background(), env)
		return discoverMsg{gen: gen, idx: i, items: items, err: err}
	}
}

func (a *app) onDiscover(msg discoverMsg) tea.Cmd {
	if msg.gen != a.discGen {
		return nil
	}
	ts := a.tasks[msg.idx]
	// Keep choices made in the item list across refreshes.
	prev := make(map[string]bool, len(ts.items))
	for _, it := range ts.items {
		prev[it.Path()] = it.Selected
	}
	for i := range msg.items {
		if sel, ok := prev[msg.items[i].Path()]; ok {
			msg.items[i].Selected = sel
		}
	}
	ts.loading, ts.items, ts.err = false, msg.items, msg.err
	if a.cl.mode == modeDetail && a.cl.task == msg.idx {
		a.cl.fillItems(a)
	}
	if a.cl.mode == modePreparing {
		return a.maybeConfirm()
	}
	return nil
}

func (a *app) remeasure(path string) tea.Cmd {
	opts := a.scanOptions()
	return func() tea.Msg {
		res, err := scan.Scan(context.Background(), path, opts, nil)
		if err != nil || res == nil {
			return remeasureMsg{path: path}
		}
		return remeasureMsg{path: path, node: res.Root}
	}
}

func reveal(path string) tea.Cmd {
	return func() tea.Msg { return revealMsg{exec.Command("open", "-R", path).Run()} }
}

func (a *app) view() string {
	if a.w == 0 {
		return "starting…"
	}
	header := a.headerView()
	footer := a.footerView()
	a.headerH = lipgloss.Height(header)
	bodyH := max(3, a.h-a.headerH-lipgloss.Height(footer))
	var body string
	switch a.tab {
	case tabOverview:
		body = a.overviewView(bodyH)
	case tabMap:
		body = a.mapView(bodyH)
	case tabExplorer:
		body = a.explorerView(bodyH)
	case tabWorktrees:
		body = a.worktreesView(bodyH)
	case tabDocker:
		body = a.dockerView(bodyH)
	default:
		body = a.cleanView(bodyH)
	}
	body = lipgloss.NewStyle().Width(a.w).Height(bodyH).MaxHeight(bodyH).MaxWidth(a.w).Render(body)
	return lipgloss.JoinVertical(lipgloss.Left, header, body, footer)
}

func (a *app) headerView() string {
	line1 := sTitle.Render("manage-disk")
	if a.volOK && a.vol.Total > 0 {
		a.gauge.Width = min(40, max(10, a.w/4))
		used := float64(a.vol.Used()) / float64(a.vol.Total)
		line1 += "  " + a.gauge.ViewAs(used) + "  " + sBold.Render(human.Bytes(a.vol.Free)) +
			sDim.Render(" free of "+human.Bytes(a.vol.Total))
	}
	if a.opts.DryRun {
		line1 += "  " + sDryBadge.Render("DRY RUN")
	}

	var line2 string
	a.tabHits = a.tabHits[:0]
	for i, n := range tabNames {
		if i > 0 {
			line2 += " "
		}
		style := sTab
		if i == a.tab {
			style = sTabOn
		}
		x0 := lipgloss.Width(line2)
		line2 += style.Render(n)
		a.tabHits = append(a.tabHits, tabHit{x0: x0, x1: lipgloss.Width(line2), tab: i})
	}
	if a.last != nil {
		right := sDim.Render("last clean " + human.Ago(time.Now(), a.last.Time) + " · freed " + human.Bytes(a.last.Freed))
		if gap := a.w - lipgloss.Width(line2) - lipgloss.Width(right); gap > 1 {
			line2 += strings.Repeat(" ", gap) + right
		}
	}
	fit := lipgloss.NewStyle().MaxWidth(a.w)
	return lipgloss.JoinVertical(lipgloss.Left, fit.Render(line1), fit.Render(line2), sDim.Render(strings.Repeat("─", a.w)))
}

func (a *app) footerView() string {
	if a.flash != "" {
		return sYellow.Render(truncRight(a.flash, a.w))
	}
	return a.helpBar.View(a.helpKeys())
}
