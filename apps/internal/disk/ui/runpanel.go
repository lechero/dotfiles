package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/progress"
	"charm.land/bubbles/v2/stopwatch"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/lechero/dotfiles/apps/internal/disk/clean"
	"github.com/lechero/dotfiles/apps/internal/disk/human"
	"github.com/lechero/dotfiles/apps/internal/disk/scan"
)

// runPanel is the one run in flight — a clean, a worktree or Docker removal —
// shown by the tab that started it.
type runPanel struct {
	owner   int
	active  bool
	state   runState
	bar     progress.Model
	watch   stopwatch.Model
	log     viewport.Model
	cancel  context.CancelFunc
	ch      chan tea.Msg
	summary *clean.Summary
}

type runState struct {
	cur               clean.Event
	done, total       int
	skipped, failed   int
	bytes, totalBytes int64
	lines             []string
	stopping          bool
}

func newRunPanel() runPanel {
	return runPanel{
		bar:   progress.New(progress.WithDefaultBlend()),
		watch: stopwatch.New(stopwatch.WithInterval(time.Second)),
		log:   viewport.New(),
	}
}

// startRun runs sels in the background; owner is the tab that shows it.
func (a *app) startRun(owner int, sels []clean.Selection) tea.Cmd {
	r := &a.run
	st := runState{}
	for _, s := range sels {
		for _, it := range s.Items {
			st.totalBytes += it.Size
			st.total++
		}
	}
	r.owner, r.state, r.summary, r.active = owner, st, nil, true
	r.log.SetContent("")
	ctx, cancel := context.WithCancel(context.Background())
	r.cancel = cancel
	ch := make(chan tea.Msg, 512)
	r.ch = ch
	env, dry := a.env, a.opts.DryRun
	go func() {
		sum := clean.Run(ctx, env, sels, dry, func(e clean.Event) { ch <- runEventMsg{e} })
		ch <- runDoneMsg{sum}
		close(ch)
	}()
	return tea.Batch(waitFor(ch), tick(), a.wake(), r.bar.SetPercent(0), r.watch.Reset(), r.watch.Start())
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
	r := &a.run
	st := &r.state
	label := ev.Task + ": " + ev.Item
	switch ev.Kind {
	case clean.EventStart:
		st.cur = ev
	case clean.EventDone:
		st.done++
		st.bytes += ev.Bytes
		note := ""
		if ev.Msg != "" {
			note = " (" + ev.Msg + ")"
		}
		r.addLog(sGreen.Render("✓ ") + label + sDim.Render("  "+human.Bytes(ev.Bytes)+note))
	case clean.EventSkip:
		st.skipped++
		r.addLog(sYellow.Render("– ") + label + sDim.Render("  skipped: "+ev.Msg))
	case clean.EventFail:
		st.failed++
		r.addLog(sRed.Render("✗ ") + label + sRed.Render("  "+ev.Msg))
	case clean.EventLog:
		r.addLog(sDim.Render("    " + ev.Msg))
	}
	return tea.Batch(waitFor(r.ch), r.bar.SetPercent(st.fraction()))
}

func (st *runState) fraction() float64 {
	switch {
	case st.totalBytes > 0:
		return min(1, float64(st.bytes)/float64(st.totalBytes))
	case st.total > 0:
		return float64(st.done+st.skipped+st.failed) / float64(st.total)
	}
	return 0
}

// addLog appends to the run log, following the newest line unless you have
// scrolled up to read something.
func (r *runPanel) addLog(s string) {
	follow := r.log.AtBottom()
	r.state.lines = append(r.state.lines, s)
	if len(r.state.lines) > 2000 {
		r.state.lines = r.state.lines[len(r.state.lines)-2000:]
	}
	r.log.SetContent(strings.Join(r.state.lines, "\n"))
	if follow {
		r.log.GotoBottom()
	}
}

func (a *app) onRunDone(sum clean.Summary) tea.Cmd {
	r := &a.run
	r.summary, r.cancel, r.active = &sum, nil, false
	cmds := []tea.Cmd{r.watch.Stop(), r.bar.SetPercent(1)}
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
	switch r.owner {
	case tabClean:
		a.cl.mode = modeDone
	case tabWorktrees:
		a.wt.mode = wtDone
		cmds = append(cmds, a.auditWorktrees(false))
	case tabDocker:
		a.dk.mode = dkDone
		cmds = append(cmds, a.auditDocker())
	}
	return tea.Batch(cmds...)
}

// runKey handles keys while a run is in flight.
func (a *app) runKey(msg tea.KeyPressMsg) tea.Cmd {
	r := &a.run
	if key.Matches(msg, a.keys.Stop) && r.cancel != nil {
		r.cancel()
		r.state.stopping = true
		return nil
	}
	var cmd tea.Cmd
	r.log, cmd = r.log.Update(msg)
	return cmd
}

// doneKey handles keys on a run's summary; back reports that it is dismissed.
func (a *app) doneKey(msg tea.KeyPressMsg) (back bool, cmd tea.Cmd) {
	switch {
	case msg.String() == "ctrl+c":
		return false, tea.Quit
	case key.Matches(msg, a.keys.Done):
		a.run.summary = nil
		return true, nil
	}
	a.run.log, cmd = a.run.log.Update(msg)
	return false, cmd
}

func (a *app) runMouse(msg tea.MouseMsg) tea.Cmd {
	var cmd tea.Cmd
	a.run.log, cmd = a.run.log.Update(msg)
	return cmd
}

func (a *app) runningView(h int) string {
	r := &a.run
	st := &r.state
	title := " Working…"
	if a.opts.DryRun {
		title = " Dry run…"
	}
	if st.stopping {
		title = " Stopping after the current item…"
	}
	r.bar.SetWidth(max(10, min(60, a.w-40)))
	var b strings.Builder
	b.WriteString(a.spin.View() + sBold.Render(title) + sDim.Render("  "+r.watch.View()) + "\n\n")
	b.WriteString(r.bar.View() + fmt.Sprintf("  %s / %s · %d of %d items\n",
		human.Bytes(st.bytes), human.Bytes(st.totalBytes), st.done+st.skipped+st.failed, st.total))
	if st.cur.Item != "" {
		b.WriteString(sDim.Render("now: "+truncRight(st.cur.Task+" — "+st.cur.Item, max(10, a.w-6))) + "\n")
	}
	b.WriteString("\n")
	r.log.SetWidth(a.w)
	r.log.SetHeight(max(1, h-6))
	b.WriteString(r.log.View())
	return b.String()
}

func (a *app) doneView(h int) string {
	r := &a.run
	s := r.summary
	if s == nil {
		return ""
	}
	var b strings.Builder
	took := human.Duration(s.Ended.Sub(s.Started))
	if s.DryRun {
		b.WriteString(sBold.Render("Dry run finished in "+took) + "\n")
		fmt.Fprintf(&b, "Would handle %s, about %s. Nothing was changed.\n", plural(len(s.Cleaned), "item"), human.Bytes(s.Estimated))
	} else {
		b.WriteString(sBold.Render("Done in "+took) + "\n")
		if r.owner == tabDocker { // what changed is inside Docker's VM; the Mac sees it later
			b.WriteString(a.dkFreedLine(s.DryRun) + "\n")
		} else {
			fmt.Fprintf(&b, "Free space %s → %s  %s\n", human.Bytes(s.FreeBefore), human.Bytes(s.FreeAfter),
				sGreen.Render("(+"+human.Bytes(max(0, s.Freed()))+")"))
		}
	}
	verb := "done"
	if s.DryRun {
		verb = "would be done"
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
	if !s.DryRun && s.Freed() < s.Estimated/2 && r.owner != tabDocker {
		b.WriteString(sDim.Render("Less space came back than measured: pnpm's cloned files share blocks, and Docker returns space a few minutes later.") + "\n")
	}
	if s.LogPath != "" {
		b.WriteString(sDim.Render("Log: "+tildePath(a.home, s.LogPath)) + "\n")
	}
	b.WriteString("\n")
	r.log.SetWidth(a.w)
	r.log.SetHeight(max(1, h-lipgloss.Height(b.String())))
	b.WriteString(r.log.View())
	return b.String()
}
