// Package ui is the manage-disk terminal interface.
package ui

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"manage-disk/internal/clean"
	"manage-disk/internal/human"
	"manage-disk/internal/scan"
)

// Options are set from the command line.
type Options struct {
	DryRun         bool // delete and run nothing; show what would happen
	IncludePrivate bool // also scan Desktop, Documents, Downloads… (macOS may ask)
}

const (
	tabOverview = iota
	tabExplorer
	tabClean
)

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
	tickMsg     time.Time
	volMsg      struct{ vol scan.Volume }
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

type app struct {
	env  *clean.Env
	home string
	opts Options
	w, h int
	tab  int
	help bool

	flash string

	vol   scan.Volume
	volOK bool
	last  *clean.HistoryEntry

	scanning   bool
	scanGen    int
	scanCancel context.CancelFunc
	prog       *scan.Progress
	scanStart  time.Time
	res        *scan.Result
	scanErr    error
	cats       []scan.Category
	hot        []*scan.Node

	discGen int
	tasks   []*taskState

	spin spinner.Model
	exp  explorerState
	cl   cleanState
}

// Model is the Bubble Tea model. It wraps a pointer so helpers can mutate state.
type Model struct{ a *app }

// New builds the interface for env.
func New(env *clean.Env, opts Options) Model {
	a := &app{
		env:  env,
		home: env.Home,
		opts: opts,
		spin: spinner.New(spinner.WithSpinner(spinner.MiniDot), spinner.WithStyle(sAccent)),
	}
	for _, t := range clean.Catalog() {
		a.tasks = append(a.tasks, &taskState{task: t, enabled: t.Tier == clean.Tier1})
	}
	a.exp.path = env.Home
	return Model{a}
}

func (m Model) Init() tea.Cmd                           { return m.a.init() }
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) { return m, m.a.update(msg) }
func (m Model) View() string                            { return m.a.view() }

func (a *app) init() tea.Cmd {
	return tea.Batch(a.spin.Tick, a.loadVolume(), a.loadHistory(), a.startScan(), a.discoverAll())
}

func tick() tea.Cmd {
	return tea.Tick(150*time.Millisecond, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func (a *app) update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.w, a.h = msg.Width, msg.Height
	case spinner.TickMsg:
		var cmd tea.Cmd
		a.spin, cmd = a.spin.Update(msg)
		return cmd
	case tickMsg:
		if a.scanning || a.cl.mode == modeRunning {
			return tick()
		}
	case volMsg:
		a.vol, a.volOK = msg.vol, true
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
	case tea.KeyMsg:
		return a.onKey(msg.String())
	}
	return nil
}

func (a *app) onKey(key string) tea.Cmd {
	a.flash = ""
	if a.tab == tabClean && a.cl.modal() {
		return a.cleanKey(key)
	}
	switch key {
	case "ctrl+c", "q":
		if a.scanCancel != nil {
			a.scanCancel()
		}
		return tea.Quit
	case "?":
		a.help = !a.help
		return nil
	case "1":
		a.tab = tabOverview
		return nil
	case "2":
		a.tab = tabExplorer
		return nil
	case "3":
		a.tab = tabClean
		return nil
	case "tab":
		a.tab = (a.tab + 1) % 3
		return nil
	case "shift+tab":
		a.tab = (a.tab + 2) % 3
		return nil
	}
	a.help = false
	switch a.tab {
	case tabOverview:
		return a.overviewKey(key)
	case tabExplorer:
		return a.explorerKey(key)
	default:
		return a.cleanKey(key)
	}
}

func (a *app) loadVolume() tea.Cmd {
	home := a.home
	return func() tea.Msg {
		v, err := scan.VolumeOf(home)
		if err != nil {
			return nil
		}
		return volMsg{v}
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
	return tea.Batch(tick(), func() tea.Msg {
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
	// Heavy tasks size their items off the tree, so they start now.
	var cmds []tea.Cmd
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
}

// discoverAll re-finds every task's items with a fresh process snapshot.
func (a *app) discoverAll() tea.Cmd {
	a.env.ResetProcs()
	a.discGen++
	var cmds []tea.Cmd
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
	a.cl.clampDetail(a)
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
	bodyH := max(3, a.h-lipgloss.Height(header)-lipgloss.Height(footer))
	var body string
	switch {
	case a.help:
		body = a.helpView()
	case a.tab == tabOverview:
		body = a.overviewView(bodyH)
	case a.tab == tabExplorer:
		body = a.explorerView(bodyH)
	default:
		body = a.cleanView(bodyH)
	}
	body = lipgloss.NewStyle().Width(a.w).Height(bodyH).MaxHeight(bodyH).MaxWidth(a.w).Render(body)
	return lipgloss.JoinVertical(lipgloss.Left, header, body, footer)
}

func (a *app) headerView() string {
	line1 := sTitle.Render("manage-disk")
	if a.volOK {
		barW := min(40, max(10, a.w/4))
		line1 += "  " + volumeBar(a.vol.Used(), a.vol.Total, barW) + "  " +
			sBold.Render(human.Bytes(a.vol.Free)) + sDim.Render(" free of "+human.Bytes(a.vol.Total))
	}
	if a.opts.DryRun {
		line1 += "  " + sDryBadge.Render("DRY RUN")
	}

	names := []string{"1 Overview", "2 Explorer", "3 Clean"}
	var tabs []string
	for i, n := range names {
		if i == a.tab {
			tabs = append(tabs, sTabOn.Render(n))
		} else {
			tabs = append(tabs, sTab.Render(n))
		}
	}
	line2 := strings.Join(tabs, " ")
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
	var hints string
	switch {
	case a.help:
		hints = "? close help"
	case a.tab == tabOverview:
		hints = "r rescan · p " + privateWord(a.opts.IncludePrivate) + " private folders · 2 explore · 3 clean · ? help · q quit"
	case a.tab == tabExplorer:
		hints = "↑↓ move · enter/→ open · ←/backspace up · o reveal in Finder · r rescan · q quit"
	default:
		hints = a.cleanHints()
	}
	return sDim.Render(truncRight(hints, a.w))
}

func privateWord(on bool) string {
	if on {
		return "skip"
	}
	return "include"
}

func (a *app) helpView() string {
	lines := []string{
		sHeading.Render("manage-disk"),
		"",
		"Overview   where your space goes, by category and by biggest spots",
		"Explorer   drill into any folder, largest first (read-only)",
		"Clean      re-run the cleanups: Tier 1 caches are on by default,",
		"           Tier 2 (rebuilds, re-downloads) you switch on yourself",
		"",
		"Sizes are allocated bytes, like du; sparse files count what they use.",
		"Folders macOS guards with consent dialogs (Desktop, Documents, Downloads…)",
		"are skipped unless you press p or start with --include-private.",
		"",
		"Before every clean the tool re-checks what is running: browser caches of",
		"open browsers, a running dev server's .next, npx folders an MCP server",
		"runs from, and worktrees something works in are all left alone.",
		"",
		"Runs are logged to ~/Library/Logs/manage-disk/, and --dry-run (or d in",
		"the Clean tab) shows what would happen without deleting anything.",
	}
	return strings.Join(lines, "\n")
}
