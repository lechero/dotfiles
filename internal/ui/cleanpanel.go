package ui

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/stopwatch"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"manage-disk/internal/clean"
	"manage-disk/internal/human"
	"manage-disk/internal/scan"
)

const (
	modeList = iota
	modeDetail
	modePreparing // re-checking what runs before asking to confirm
	modeConfirm
	modeRunning
	modeDone
)

type runState struct {
	cur               clean.Event
	done, total       int
	skipped, failed   int
	bytes, totalBytes int64
	log               []string
	stopping          bool
}

type cleanState struct {
	mode    int
	cursor  int // task under the cursor
	task    int // task whose items are open
	rowTask map[int]int

	items list.Model // the open task's items, filterable

	run    runState
	bar    progress.Model
	watch  stopwatch.Model
	log    viewport.Model
	cancel context.CancelFunc
	ch     chan tea.Msg

	summary *clean.Summary
}

func newCleanState() cleanState {
	l := list.New(nil, itemDelegate{}, 0, 0)
	l.SetShowTitle(false)
	l.SetShowHelp(false)
	l.SetFilteringEnabled(true)
	l.SetStatusBarItemName("item", "items")
	l.DisableQuitKeybindings()
	l.KeyMap.PrevPage = key.NewBinding(key.WithKeys("pgup"))
	l.KeyMap.NextPage = key.NewBinding(key.WithKeys("pgdown"))
	l.KeyMap.ShowFullHelp.SetEnabled(false)
	l.KeyMap.CloseFullHelp.SetEnabled(false)

	vp := viewport.New(0, 0)
	return cleanState{
		items: l,
		bar:   progress.New(progress.WithDefaultGradient()),
		watch: stopwatch.NewWithInterval(time.Second),
		log:   vp,
	}
}

// modal modes take every key.
func (c *cleanState) modal() bool { return c.mode >= modePreparing }

// itemEntry is one row of the item list; it reads the item live from its
// task, so toggling a checkbox never has to rebuild the list.
type itemEntry struct {
	ts  *taskState
	idx int
}

func (e itemEntry) item() (clean.Item, bool) {
	if e.idx < len(e.ts.items) {
		return e.ts.items[e.idx], true
	}
	return clean.Item{}, false
}

func (e itemEntry) FilterValue() string {
	it, _ := e.item()
	return it.Label + " " + it.Note
}

type itemDelegate struct{}

func (itemDelegate) Height() int                         { return 1 }
func (itemDelegate) Spacing() int                        { return 0 }
func (itemDelegate) Update(tea.Msg, *list.Model) tea.Cmd { return nil }

func (itemDelegate) Render(w io.Writer, m list.Model, index int, li list.Item) {
	e := li.(itemEntry)
	it, ok := e.item()
	if !ok {
		return
	}
	width := m.Width()
	labelW := 8
	for _, x := range e.ts.items {
		labelW = max(labelW, len([]rune(x.Label)))
	}
	labelW = min(labelW, max(16, width/2))
	noteW := max(10, width-19-labelW)

	marker, box := "  ", "[ ]"
	if it.Selected {
		box = "[" + sGreen.Render("x") + "]"
	}
	label := pad(truncRight(it.Label, labelW), labelW)
	if index == m.Index() {
		marker, label = sCursor.Render("› "), sCursor.Render(label)
	}
	extra := sDim.Render(truncRight(it.Note, noteW))
	if it.Blocked != "" {
		extra = sYellow.Render(truncRight("blocked: "+it.Blocked, noteW))
	}
	fmt.Fprint(w, marker+box+" "+padLeft(human.Bytes(it.Size), 9)+"  "+label+"  "+extra)
}

// fillItems loads the open task's items into the list.
func (c *cleanState) fillItems(a *app) {
	ts := a.tasks[c.task]
	entries := make([]list.Item, len(ts.items))
	for i := range ts.items {
		entries[i] = itemEntry{ts: ts, idx: i}
	}
	c.items.SetItems(entries)
}

func (c *cleanState) selectedEntry() (itemEntry, bool) {
	e, ok := c.items.SelectedItem().(itemEntry)
	if !ok || e.idx >= len(e.ts.items) {
		return itemEntry{}, false
	}
	return e, true
}

func (a *app) selectedTotals() (size int64, n, blocked int) {
	for _, ts := range a.tasks {
		if !ts.enabled {
			continue
		}
		s, k := ts.ready()
		size += s
		n += k
		blocked += ts.blocked()
	}
	return size, n, blocked
}

func (a *app) cleanKey(msg tea.KeyMsg) tea.Cmd {
	c, k := &a.cl, a.keys
	switch c.mode {
	case modeList:
		return a.listKey(msg)
	case modeDetail:
		return a.detailKey(msg)
	case modePreparing:
		if msg.Type == tea.KeyEsc || msg.Type == tea.KeyCtrlC || msg.String() == "n" {
			c.mode = modeList
		}
	case modeConfirm:
		switch {
		case key.Matches(msg, k.Yes):
			return a.startRun()
		case key.Matches(msg, k.No):
			c.mode = modeList
		case msg.Type == tea.KeyCtrlC:
			return tea.Quit
		}
	case modeRunning:
		if key.Matches(msg, k.Stop) && c.cancel != nil {
			c.cancel()
			c.run.stopping = true
			return nil
		}
		var cmd tea.Cmd
		c.log, cmd = c.log.Update(msg)
		return cmd
	case modeDone:
		switch {
		case msg.Type == tea.KeyCtrlC:
			return tea.Quit
		case key.Matches(msg, k.Done):
			c.mode, c.summary = modeList, nil
		default:
			var cmd tea.Cmd
			c.log, cmd = c.log.Update(msg)
			return cmd
		}
	}
	return nil
}

func (a *app) listKey(msg tea.KeyMsg) tea.Cmd {
	c, k := &a.cl, a.keys
	switch {
	case key.Matches(msg, k.Up):
		c.cursor = max(0, c.cursor-1)
	case key.Matches(msg, k.Down):
		c.cursor = min(len(a.tasks)-1, c.cursor+1)
	case key.Matches(msg, k.Toggle):
		a.tasks[c.cursor].enabled = !a.tasks[c.cursor].enabled
	case key.Matches(msg, k.Details):
		a.openDetail(c.cursor)
	case key.Matches(msg, k.AllTier1):
		for _, ts := range a.tasks {
			ts.enabled = ts.enabled || ts.task.Tier == clean.Tier1
		}
	case key.Matches(msg, k.None):
		for _, ts := range a.tasks {
			ts.enabled = false
		}
	case key.Matches(msg, k.DryRun):
		a.opts.DryRun = !a.opts.DryRun
	case key.Matches(msg, k.Refresh):
		return a.discoverAll()
	case key.Matches(msg, k.CleanNow):
		return a.prepareRun()
	}
	return nil
}

func (a *app) openDetail(i int) {
	c := &a.cl
	c.task, c.mode = i, modeDetail
	c.items.ResetFilter()
	c.fillItems(a)
	c.items.Select(0)
}

func (a *app) detailKey(msg tea.KeyMsg) tea.Cmd {
	c, k := &a.cl, a.keys
	if c.items.SettingFilter() {
		var cmd tea.Cmd
		c.items, cmd = c.items.Update(msg)
		return cmd
	}
	ts := a.tasks[c.task]
	switch {
	case msg.Type == tea.KeyEsc && c.items.IsFiltered():
		c.items.ResetFilter()
	case msg.Type == tea.KeyEsc || msg.Type == tea.KeyBackspace || msg.Type == tea.KeyLeft:
		c.mode = modeList
	case key.Matches(msg, k.Toggle):
		if e, ok := c.selectedEntry(); ok {
			ts.items[e.idx].Selected = !ts.items[e.idx].Selected
			ts.enabled = ts.enabled || ts.items[e.idx].Selected
		}
	case key.Matches(msg, k.AllTier1):
		for i := range ts.items {
			ts.items[i].Selected = true
		}
		ts.enabled = true
	case key.Matches(msg, k.None):
		for i := range ts.items {
			ts.items[i].Selected = false
		}
	case key.Matches(msg, k.Reveal):
		if e, ok := c.selectedEntry(); ok {
			return reveal(ts.items[e.idx].Path())
		}
	case key.Matches(msg, k.CleanNow):
		return a.prepareRun()
	default:
		var cmd tea.Cmd
		c.items, cmd = c.items.Update(msg)
		return cmd
	}
	return nil
}

func (a *app) cleanMouse(msg tea.MouseMsg) tea.Cmd {
	c := &a.cl
	switch c.mode {
	case modeRunning, modeDone:
		var cmd tea.Cmd
		c.log, cmd = c.log.Update(msg)
		return cmd
	case modeDetail:
		if msg.Action == tea.MouseActionPress {
			switch msg.Button {
			case tea.MouseButtonWheelUp:
				c.items.CursorUp()
			case tea.MouseButtonWheelDown:
				c.items.CursorDown()
			}
		}
	case modeList:
		if msg.Action != tea.MouseActionPress {
			return nil
		}
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			c.cursor = max(0, c.cursor-1)
		case tea.MouseButtonWheelDown:
			c.cursor = min(len(a.tasks)-1, c.cursor+1)
		case tea.MouseButtonLeft:
			if i, ok := c.rowTask[msg.Y]; ok {
				if i == c.cursor {
					a.openDetail(i)
				} else {
					c.cursor = i
				}
			}
		}
	}
	return nil
}

// prepareRun re-discovers the chosen tasks with a fresh look at running
// processes, so nothing started since the last check gets its files deleted.
func (a *app) prepareRun() tea.Cmd {
	anyEnabled := false
	for _, ts := range a.tasks {
		anyEnabled = anyEnabled || ts.enabled
	}
	if !anyEnabled {
		a.flash = "Nothing selected: switch a task on with space."
		return nil
	}
	// What is cleanable is decided by the fresh look below, not by what was on screen.
	a.cl.mode = modePreparing
	a.env.ResetProcs()
	a.discGen++
	cmds := []tea.Cmd{a.wake()}
	for i, ts := range a.tasks {
		if ts.task.Heavy && a.res == nil && !ts.enabled {
			ts.waiting = true
			continue
		}
		cmds = append(cmds, a.discover(i))
	}
	return tea.Batch(cmds...)
}

func (a *app) maybeConfirm() tea.Cmd {
	for _, ts := range a.tasks {
		if ts.enabled && (ts.loading || ts.waiting) {
			return nil
		}
	}
	if _, n, _ := a.selectedTotals(); n == 0 {
		a.cl.mode = modeList
		a.flash = "Nothing to clean right now: what you chose is blocked or already gone."
		return nil
	}
	a.cl.mode = modeConfirm
	return nil
}

func (a *app) startRun() tea.Cmd {
	var sels []clean.Selection
	r := runState{}
	for _, ts := range a.tasks {
		if !ts.enabled {
			continue
		}
		var items []clean.Item
		for _, it := range ts.items {
			if it.Ready() {
				items = append(items, it)
				r.totalBytes += it.Size
				r.total++
			}
		}
		if len(items) > 0 {
			sels = append(sels, clean.Selection{Task: ts.task, Items: items})
		}
	}
	c := &a.cl
	c.mode, c.run = modeRunning, r
	c.log.SetContent("")
	ctx, cancel := context.WithCancel(context.Background())
	c.cancel = cancel
	ch := make(chan tea.Msg, 512)
	c.ch = ch
	env, dry := a.env, a.opts.DryRun
	go func() {
		sum := clean.Run(ctx, env, sels, dry, func(e clean.Event) { ch <- runEventMsg{e} })
		ch <- runDoneMsg{sum}
		close(ch)
	}()
	return tea.Batch(waitFor(ch), tick(), a.wake(), c.bar.SetPercent(0), c.watch.Reset(), c.watch.Start())
}

func waitFor(ch <-chan tea.Msg) tea.Cmd {
	return func() tea.Msg {
		msg, ok := <-ch
		if !ok {
			return nil
		}
		return msg
	}
}

func (a *app) onRunEvent(ev clean.Event) tea.Cmd {
	c := &a.cl
	r := &c.run
	label := ev.Task + ": " + ev.Item
	switch ev.Kind {
	case clean.EventStart:
		r.cur = ev
	case clean.EventDone:
		r.done++
		r.bytes += ev.Bytes
		note := ""
		if ev.Msg != "" {
			note = " (" + ev.Msg + ")"
		}
		c.addLog(sGreen.Render("✓ ") + label + sDim.Render("  "+human.Bytes(ev.Bytes)+note))
	case clean.EventSkip:
		r.skipped++
		c.addLog(sYellow.Render("– ") + label + sDim.Render("  skipped: "+ev.Msg))
	case clean.EventFail:
		r.failed++
		c.addLog(sRed.Render("✗ ") + label + sRed.Render("  "+ev.Msg))
	case clean.EventLog:
		c.addLog(sDim.Render("    " + ev.Msg))
	}
	return tea.Batch(waitFor(c.ch), c.bar.SetPercent(r.fraction()))
}

func (r *runState) fraction() float64 {
	switch {
	case r.totalBytes > 0:
		return min(1, float64(r.bytes)/float64(r.totalBytes))
	case r.total > 0:
		return float64(r.done+r.skipped+r.failed) / float64(r.total)
	}
	return 0
}

// addLog appends to the run log, following the newest line unless you have
// scrolled up to read something.
func (c *cleanState) addLog(s string) {
	follow := c.log.AtBottom()
	c.run.log = append(c.run.log, s)
	if len(c.run.log) > 2000 {
		c.run.log = c.run.log[len(c.run.log)-2000:]
	}
	c.log.SetContent(strings.Join(c.run.log, "\n"))
	if follow {
		c.log.GotoBottom()
	}
}

func (a *app) onRunDone(sum clean.Summary) tea.Cmd {
	c := &a.cl
	c.mode, c.summary, c.cancel = modeDone, &sum, nil
	cmds := []tea.Cmd{c.watch.Stop(), c.bar.SetPercent(1)}
	if !sum.DryRun {
		// Drop what was deleted from the tree; re-measure what commands cleaned.
		a.env.EditTree(func(root *scan.Node) {
			for _, cl := range sum.Cleaned {
				if !cl.Command {
					for _, p := range cl.Item.Paths {
						root.Remove(p)
					}
				}
			}
		})
		for _, cl := range sum.Cleaned {
			// Only paths the scan read: re-measuring Docker's disk image would
			// reach into a folder macOS guards with a dialog.
			if cl.Command && a.res != nil && a.res.Root.Find(cl.Item.Path()) != nil {
				cmds = append(cmds, a.remeasure(cl.Item.Path()))
			}
		}
		a.refreshInsights()
	}
	cmds = append(cmds, a.loadVolume(), a.loadHistory(), a.discoverAll())
	return tea.Batch(cmds...)
}

func (a *app) cleanHelpKeys(tabs []key.Binding) keyHelp {
	k := a.keys
	scroll := key.NewBinding(key.WithKeys("up", "down"), key.WithHelp("↑↓/wheel", "scroll the log"))
	switch a.cl.mode {
	case modeDetail:
		back := key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "back"))
		return keyHelp{short: []key.Binding{k.Up, k.Down, k.Toggle, k.Filter, k.AllTier1, k.None, k.Reveal, back, k.CleanNow}}
	case modePreparing:
		return keyHelp{short: []key.Binding{key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel"))}}
	case modeConfirm:
		return keyHelp{short: []key.Binding{k.Yes, k.No}}
	case modeRunning:
		return keyHelp{short: []key.Binding{k.Stop, scroll}}
	case modeDone:
		return keyHelp{short: []key.Binding{k.Done, scroll}}
	}
	return keyHelp{
		short: []key.Binding{k.Up, k.Down, k.Toggle, k.Details, k.CleanNow, k.DryRun, k.Help, k.Quit},
		full: [][]key.Binding{
			{k.Up, k.Down, k.Toggle, k.Details},
			{k.AllTier1, k.None, k.Refresh, k.DryRun, k.CleanNow},
			tabs, {k.Help, k.Quit},
		},
	}
}

func (a *app) cleanView(h int) string {
	switch a.cl.mode {
	case modeDetail:
		return a.detailView(h)
	case modePreparing:
		return a.spin.View() + " Checking what's running and measuring again before anything is deleted…"
	case modeConfirm:
		return a.confirmView()
	case modeRunning:
		return a.runningView(h)
	case modeDone:
		return a.doneView(h)
	}
	return a.listView()
}

func (a *app) taskStatus(ts *taskState) string {
	switch {
	case ts.loading:
		return a.spin.View() + sDim.Render(" measuring…")
	case ts.waiting:
		return sDim.Render("waits for the scan")
	case ts.err != nil:
		return sRed.Render(ts.err.Error())
	case len(ts.items) == 0:
		return sDim.Render("nothing to clean")
	}
	size, n := ts.ready()
	s := padLeft(human.Bytes(size), 9) + sDim.Render(fmt.Sprintf("  %d of %d items", n, len(ts.items)))
	if b := ts.blocked(); b > 0 {
		s += sYellow.Render(fmt.Sprintf("  %d blocked", b))
	}
	return s
}

func (a *app) listView() string {
	var b strings.Builder
	row := 0
	line := func(s string) {
		b.WriteString(s + "\n")
		row++
	}
	a.cl.rowTask = map[int]int{}
	line(sHeading.Render("Clean") + sDim.Render("  space switches a task on or off · enter shows its items · c cleans"))
	titleW := min(34, max(20, a.w/3))
	for _, tier := range []clean.Tier{clean.Tier1, clean.Tier2} {
		line("")
		if tier == clean.Tier1 {
			line(sGreen.Render("Tier 1") + sDim.Render(" · caches that regenerate on their own"))
		} else {
			line(sYellow.Render("Tier 2") + sDim.Render(" · you pay with a re-download or a rebuild"))
		}
		for i, ts := range a.tasks {
			if ts.task.Tier != tier {
				continue
			}
			marker, box := "  ", "[ ]"
			if ts.enabled {
				box = "[" + sGreen.Render("x") + "]"
			}
			title := pad(truncRight(ts.task.Title, titleW), titleW)
			if i == a.cl.cursor {
				marker, title = sCursor.Render("› "), sCursor.Render(title)
			}
			a.cl.rowTask[a.headerH+row] = i
			line(marker + box + " " + title + " " + a.taskStatus(ts))
		}
	}
	line("")
	size, n, blocked := a.selectedTotals()
	sum := fmt.Sprintf("Selected: %s in %s", sBold.Render(human.Bytes(size)), plural(n, "item"))
	if blocked > 0 {
		sum += sYellow.Render(fmt.Sprintf(" · %d blocked, will be skipped", blocked))
	}
	if a.opts.DryRun {
		sum += "  " + sDryBadge.Render("DRY RUN") + sDim.Render(" nothing will be deleted")
	}
	line(sum)
	if a.cl.cursor < len(a.tasks) {
		t := a.tasks[a.cl.cursor].task
		wrap := lipgloss.NewStyle().Width(max(20, a.w-2))
		line("")
		line(wrap.Render(sDim.Render(t.About)))
		if t.Cost != "" {
			line(wrap.Render(sYellow.Render("Cost: ") + sDim.Render(t.Cost)))
		}
	}
	return b.String()
}

func (a *app) detailView(h int) string {
	c := &a.cl
	ts := a.tasks[c.task]
	wrap := lipgloss.NewStyle().Width(max(20, a.w-2))
	var b strings.Builder
	head := sHeading.Render(ts.task.Title) + sDim.Render(fmt.Sprintf(" · Tier %d", ts.task.Tier))
	if !ts.enabled {
		head += sDim.Render(" · off (selecting an item switches it on)")
	}
	b.WriteString(head + "\n")
	b.WriteString(wrap.Render(sDim.Render(ts.task.About)) + "\n")
	if ts.task.Cost != "" {
		b.WriteString(wrap.Render(sYellow.Render("Cost: ")+sDim.Render(ts.task.Cost)) + "\n")
	}
	b.WriteString("\n")
	if ts.loading || ts.waiting || len(ts.items) == 0 {
		b.WriteString(a.taskStatus(ts) + "\n")
		return b.String()
	}
	size, n := ts.ready()
	foot := sDim.Render(fmt.Sprintf("%s chosen in %s", human.Bytes(size), plural(n, "item")))
	c.items.SetSize(a.w, max(4, h-lipgloss.Height(b.String())-1))
	b.WriteString(c.items.View() + "\n" + foot)
	return b.String()
}

func (a *app) confirmView() string {
	size, n, blocked := a.selectedTotals()
	title := fmt.Sprintf("Clean %s, about %s?", plural(n, "item"), human.Bytes(size))
	if a.opts.DryRun {
		title = fmt.Sprintf("Dry run: %s, about %s. Nothing will be deleted.", plural(n, "item"), human.Bytes(size))
	}
	var lines []string
	lines = append(lines, sBold.Render(title), "")
	for _, ts := range a.tasks {
		if !ts.enabled {
			continue
		}
		s, k := ts.ready()
		if k == 0 {
			continue
		}
		lines = append(lines, fmt.Sprintf(" %s %s  %s", sDim.Render("•"), pad(ts.task.Title, 32),
			padLeft(human.Bytes(s), 9)+sDim.Render("  "+plural(k, "item"))))
	}
	if blocked > 0 {
		lines = append(lines, "", sYellow.Render(fmt.Sprintf("%d chosen items are in use and will be skipped.", blocked)))
	}
	lines = append(lines, "", sGreen.Render("y")+" clean   "+sDim.Render("n / esc")+" cancel")
	return sBox.Render(strings.Join(lines, "\n"))
}

func (a *app) runningView(h int) string {
	c := &a.cl
	r := &c.run
	title := " Cleaning…"
	if a.opts.DryRun {
		title = " Dry run…"
	}
	if r.stopping {
		title = " Stopping after the current item…"
	}
	c.bar.Width = max(10, min(60, a.w-40))
	var b strings.Builder
	b.WriteString(a.spin.View() + sBold.Render(title) + sDim.Render("  "+c.watch.View()) + "\n\n")
	b.WriteString(c.bar.View() + fmt.Sprintf("  %s / %s · %d of %d items\n",
		human.Bytes(r.bytes), human.Bytes(r.totalBytes), r.done+r.skipped+r.failed, r.total))
	if r.cur.Item != "" {
		b.WriteString(sDim.Render("now: "+truncRight(r.cur.Task+" — "+r.cur.Item, max(10, a.w-6))) + "\n")
	}
	b.WriteString("\n")
	c.log.Width, c.log.Height = a.w, max(1, h-6)
	b.WriteString(c.log.View())
	return b.String()
}

func (a *app) doneView(h int) string {
	c := &a.cl
	s := c.summary
	var b strings.Builder
	took := human.Duration(s.Ended.Sub(s.Started))
	if s.DryRun {
		b.WriteString(sBold.Render("Dry run finished in "+took) + "\n")
		b.WriteString(fmt.Sprintf("Would clean %s, about %s. Nothing was deleted.\n", plural(len(s.Cleaned), "item"), human.Bytes(s.Estimated)))
	} else {
		b.WriteString(sBold.Render("Done in "+took) + "\n")
		b.WriteString(fmt.Sprintf("Free space %s → %s  %s\n", human.Bytes(s.FreeBefore), human.Bytes(s.FreeAfter),
			sGreen.Render("(+"+human.Bytes(max(0, s.Freed()))+")")))
	}
	verb := "cleaned"
	if s.DryRun {
		verb = "would be cleaned"
	}
	counts := sGreen.Render(fmt.Sprintf("✓ %d %s", len(s.Cleaned), verb))
	if s.Skipped > 0 {
		counts += "   " + sYellow.Render(fmt.Sprintf("– %d skipped", s.Skipped))
	}
	if s.Failed > 0 {
		counts += "   " + sRed.Render(fmt.Sprintf("✗ %d failed", s.Failed))
	}
	b.WriteString(counts + "\n")
	if s.Cancelled {
		b.WriteString(sYellow.Render("Stopped early, as you asked.") + "\n")
	}
	if !s.DryRun && s.Freed() < s.Estimated/2 {
		b.WriteString(sDim.Render("Less space came back than measured: pnpm's cloned files share blocks, and Docker returns space a few minutes later.") + "\n")
	}
	if s.LogPath != "" {
		b.WriteString(sDim.Render("Log: "+tildePath(a.home, s.LogPath)) + "\n")
	}
	b.WriteString("\n")
	c.log.Width, c.log.Height = a.w, max(1, h-lipgloss.Height(b.String()))
	b.WriteString(c.log.View())
	return b.String()
}

func plural(n int, word string) string {
	if n == 1 {
		return "1 " + word
	}
	return fmt.Sprintf("%d %ss", n, word)
}
