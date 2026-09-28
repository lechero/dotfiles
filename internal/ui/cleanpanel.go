package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

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
	start             time.Time
	cur               clean.Event
	done, total       int
	skipped, failed   int
	bytes, totalBytes int64
	log               []string
	stopping          bool
}

func (r *runState) addLog(s string) {
	r.log = append(r.log, s)
	if len(r.log) > 400 {
		r.log = r.log[len(r.log)-400:]
	}
}

type cleanState struct {
	mode             int
	cursor           int // task under the cursor
	task             int // task whose items are shown
	dcursor, doffset int
	run              runState
	cancel           context.CancelFunc
	ch               chan tea.Msg
	summary          *clean.Summary
}

// modal modes take every key.
func (c *cleanState) modal() bool { return c.mode >= modePreparing }

func (c *cleanState) clampDetail(a *app) {
	if c.mode == modeDetail {
		c.dcursor = max(0, min(c.dcursor, len(a.tasks[c.task].items)-1))
	}
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

func (a *app) cleanKey(key string) tea.Cmd {
	c := &a.cl
	switch c.mode {
	case modeList:
		return a.listKey(key)
	case modeDetail:
		return a.detailKey(key)
	case modePreparing:
		if key == "esc" || key == "ctrl+c" || key == "n" {
			c.mode = modeList
		}
	case modeConfirm:
		switch key {
		case "y", "enter":
			return a.startRun()
		case "n", "esc", "q":
			c.mode = modeList
		case "ctrl+c":
			return tea.Quit
		}
	case modeRunning:
		if (key == "esc" || key == "ctrl+c") && c.cancel != nil {
			c.cancel()
			c.run.stopping = true
		}
	case modeDone:
		if key == "ctrl+c" {
			return tea.Quit
		}
		c.mode, c.summary = modeList, nil
		switch key {
		case "1":
			a.tab = tabOverview
		case "2":
			a.tab = tabExplorer
		}
	}
	return nil
}

func (a *app) listKey(key string) tea.Cmd {
	c := &a.cl
	switch key {
	case "up", "k":
		c.cursor = max(0, c.cursor-1)
	case "down", "j":
		c.cursor = min(len(a.tasks)-1, c.cursor+1)
	case " ", "space", "x":
		ts := a.tasks[c.cursor]
		ts.enabled = !ts.enabled
	case "enter", "right", "l":
		c.task, c.dcursor, c.doffset, c.mode = c.cursor, 0, 0, modeDetail
	case "a":
		for _, ts := range a.tasks {
			ts.enabled = ts.enabled || ts.task.Tier == clean.Tier1
		}
	case "n":
		for _, ts := range a.tasks {
			ts.enabled = false
		}
	case "d":
		a.opts.DryRun = !a.opts.DryRun
	case "r":
		return a.discoverAll()
	case "c":
		return a.prepareRun()
	}
	return nil
}

func (a *app) detailKey(key string) tea.Cmd {
	c := &a.cl
	ts := a.tasks[c.task]
	n := len(ts.items)
	switch key {
	case "esc", "left", "h", "backspace":
		c.mode = modeList
	case "up", "k":
		c.dcursor--
	case "down", "j":
		c.dcursor++
	case "pgup", "ctrl+u":
		c.dcursor -= 10
	case "pgdown", "ctrl+d":
		c.dcursor += 10
	case "home", "g":
		c.dcursor = 0
	case "end", "G":
		c.dcursor = n - 1
	case " ", "space", "x":
		if c.dcursor < n {
			ts.items[c.dcursor].Selected = !ts.items[c.dcursor].Selected
			if ts.items[c.dcursor].Selected {
				ts.enabled = true
			}
		}
	case "a":
		for i := range ts.items {
			ts.items[i].Selected = true
		}
		ts.enabled = true
	case "n":
		for i := range ts.items {
			ts.items[i].Selected = false
		}
	case "o":
		if c.dcursor < n {
			return reveal(ts.items[c.dcursor].Path())
		}
	case "c":
		return a.prepareRun()
	}
	c.dcursor = max(0, min(c.dcursor, n-1))
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
	var cmds []tea.Cmd
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
	r := runState{start: time.Now()}
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
	a.cl.mode, a.cl.run = modeRunning, r
	ctx, cancel := context.WithCancel(context.Background())
	a.cl.cancel = cancel
	ch := make(chan tea.Msg, 512)
	a.cl.ch = ch
	env, dry := a.env, a.opts.DryRun
	go func() {
		sum := clean.Run(ctx, env, sels, dry, func(e clean.Event) { ch <- runEventMsg{e} })
		ch <- runDoneMsg{sum}
		close(ch)
	}()
	return tea.Batch(waitFor(ch), tick())
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
	r := &a.cl.run
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
		r.addLog(sGreen.Render("✓ ") + label + sDim.Render("  "+human.Bytes(ev.Bytes)+note))
	case clean.EventSkip:
		r.skipped++
		r.addLog(sYellow.Render("– ") + label + sDim.Render("  skipped: "+ev.Msg))
	case clean.EventFail:
		r.failed++
		r.addLog(sRed.Render("✗ ") + label + sRed.Render("  "+ev.Msg))
	case clean.EventLog:
		r.addLog(sDim.Render("    " + ev.Msg))
	}
	return waitFor(a.cl.ch)
}

func (a *app) onRunDone(sum clean.Summary) tea.Cmd {
	a.cl.mode, a.cl.summary, a.cl.cancel = modeDone, &sum, nil
	var cmds []tea.Cmd
	if !sum.DryRun {
		// Drop what was deleted from the tree; re-measure what commands cleaned.
		a.env.EditTree(func(root *scan.Node) {
			for _, c := range sum.Cleaned {
				if !c.Command {
					for _, p := range c.Item.Paths {
						root.Remove(p)
					}
				}
			}
		})
		for _, c := range sum.Cleaned {
			// Only paths the scan read: re-measuring Docker's disk image would
			// reach into a folder macOS guards with a dialog.
			if c.Command && a.res != nil && a.res.Root.Find(c.Item.Path()) != nil {
				cmds = append(cmds, a.remeasure(c.Item.Path()))
			}
		}
		a.refreshInsights()
	}
	cmds = append(cmds, a.loadVolume(), a.loadHistory(), a.discoverAll())
	return tea.Batch(cmds...)
}

func (a *app) cleanHints() string {
	switch a.cl.mode {
	case modeDetail:
		return "↑↓ move · space toggle · a all · n none · o reveal · esc back · c clean"
	case modePreparing:
		return "esc cancel"
	case modeConfirm:
		return "y clean · n cancel"
	case modeRunning:
		return "esc stop after the current item"
	case modeDone:
		return "any key to go back"
	}
	return "↑↓ move · space toggle · enter items · c clean · d dry run · a all tier 1 · n none · r refresh · q quit"
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
	line := func(s string) { b.WriteString(s + "\n") }
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

	listH := max(3, h-lipgloss.Height(b.String())-2)
	if c.dcursor < c.doffset {
		c.doffset = c.dcursor
	}
	if c.dcursor >= c.doffset+listH {
		c.doffset = c.dcursor - listH + 1
	}
	// Size the label column to the longest label, and give the note what is left.
	labelW := 8
	for _, it := range ts.items {
		labelW = max(labelW, len([]rune(it.Label)))
	}
	labelW = min(labelW, max(16, a.w/2))
	noteW := max(10, a.w-19-labelW)
	for i := c.doffset; i < min(len(ts.items), c.doffset+listH); i++ {
		it := ts.items[i]
		marker, box := "  ", "[ ]"
		if it.Selected {
			box = "[" + sGreen.Render("x") + "]"
		}
		label := pad(truncRight(it.Label, labelW), labelW)
		if i == c.dcursor {
			marker, label = sCursor.Render("› "), sCursor.Render(label)
		}
		extra := sDim.Render(truncRight(it.Note, noteW))
		if it.Blocked != "" {
			extra = sYellow.Render(truncRight("blocked: "+it.Blocked, noteW))
		}
		b.WriteString(marker + box + " " + padLeft(human.Bytes(it.Size), 9) + "  " + label + "  " + extra + "\n")
	}
	size, n := ts.ready()
	b.WriteString("\n" + sDim.Render(fmt.Sprintf("%d items · %s chosen in %d", len(ts.items), human.Bytes(size), n)))
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
	r := &a.cl.run
	title := " Cleaning…"
	if a.opts.DryRun {
		title = " Dry run…"
	}
	if r.stopping {
		title = " Stopping after the current item…"
	}
	frac := 0.0
	if r.totalBytes > 0 {
		frac = float64(r.bytes) / float64(r.totalBytes)
	} else if r.total > 0 {
		frac = float64(r.done) / float64(r.total)
	}
	var b strings.Builder
	b.WriteString(a.spin.View() + sBold.Render(title) + sDim.Render("  "+human.Duration(time.Since(r.start))) + "\n\n")
	b.WriteString(bar(frac, max(10, min(60, a.w-40)), sGreen) + fmt.Sprintf("  %s / %s · %d of %d items\n",
		human.Bytes(r.bytes), human.Bytes(r.totalBytes), r.done+r.skipped+r.failed, r.total))
	if r.cur.Item != "" {
		b.WriteString(sDim.Render("now: "+truncRight(r.cur.Task+" — "+r.cur.Item, max(10, a.w-6))) + "\n")
	}
	b.WriteString("\n")
	b.WriteString(tail(r.log, max(1, h-6)))
	return b.String()
}

func (a *app) doneView(h int) string {
	s := a.cl.summary
	r := &a.cl.run
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
	b.WriteString("\n" + tail(r.log, max(1, h-8)))
	return b.String()
}

func plural(n int, word string) string {
	if n == 1 {
		return "1 " + word
	}
	return fmt.Sprintf("%d %ss", n, word)
}

func tail(lines []string, n int) string {
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return strings.Join(lines, "\n")
}
