package ui

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/lechero/dotfiles/apps/internal/disk/clean"
	"github.com/lechero/dotfiles/apps/internal/disk/human"
	"github.com/lechero/dotfiles/apps/internal/disk/worktrees"
)

const (
	wtList      = iota
	wtPreparing // checking everything again before asking to confirm
	wtConfirm
	wtRunning
	wtDone
)

type wtState struct {
	mode     int
	list     list.Model
	all      []worktrees.Worktree
	chosen   map[string]bool // by path, so a re-check keeps your picks
	seen     map[string]bool // judged before: a re-check never re-picks what you unpicked
	loading  bool
	fetching bool
	gen      int
	branches bool // also delete the branches of merged worktrees
	checked  time.Time
}

type wtAuditMsg struct {
	gen int
	all []worktrees.Worktree
}

func newWtState() wtState {
	return wtState{list: newPickList(wtDelegate{}, "worktree", "worktrees"), chosen: map[string]bool{}, seen: map[string]bool{}}
}

// auditWorktrees gathers the evidence in the background; with fetch, each
// repo first learns what its origin looks like today.
func (a *app) auditWorktrees(fetch bool) tea.Cmd {
	w := &a.wt
	w.gen++
	w.loading, w.fetching = true, fetch
	gen, env := w.gen, a.env
	return tea.Batch(a.wake(), func() tea.Msg {
		return wtAuditMsg{gen: gen, all: clean.AuditWorktrees(context.Background(), env, fetch)}
	})
}

func (a *app) onWorktreeAudit(msg wtAuditMsg) tea.Cmd {
	w := &a.wt
	if msg.gen != w.gen {
		return nil
	}
	w.loading, w.fetching, w.all, w.checked = false, false, msg.all, time.Now()
	for _, x := range w.all {
		if !w.seen[x.Path] {
			w.seen[x.Path] = true
			w.chosen[x.Path] = x.Verdict.Preselect()
		}
		if !x.Verdict.Removable() {
			w.chosen[x.Path] = false
		}
	}
	var keep string
	if e, ok := w.list.SelectedItem().(wtEntry); ok {
		keep = e.w.Path
	}
	entries := make([]list.Item, len(w.all))
	for i, x := range w.all {
		entries[i] = wtEntry{w: x, a: a}
	}
	cmd := w.list.SetItems(entries)
	for i, x := range w.all {
		if x.Path == keep {
			w.list.Select(i)
		}
	}
	if w.mode == wtPreparing {
		if n, _ := a.wtChosen(); n == 0 {
			w.mode = wtList
			a.flash = "Nothing to remove: what you picked is in use or changed since."
		} else {
			w.mode = wtConfirm
		}
	}
	return cmd
}

// wtChosen counts what a removal would take, with its size.
func (a *app) wtChosen() (n int, size int64) {
	for _, x := range a.wt.all {
		if a.wt.chosen[x.Path] && x.Verdict.Removable() {
			n++
			size += a.worktreeSize(x.Path)
		}
	}
	return n, size
}

func (a *app) worktreeSize(path string) int64 {
	if a.res == nil {
		return 0
	}
	if n := a.res.Root.Find(path); n != nil {
		return n.Size
	}
	return 0
}

// wtEntry is a row of the list; it reads picks and sizes from the app live.
type wtEntry struct {
	w worktrees.Worktree
	a *app
}

func (e wtEntry) FilterValue() string {
	return e.w.RepoName() + " " + e.w.Name() + " " + e.w.Branch + " " + e.w.Verdict.String()
}

type wtDelegate struct{}

func (wtDelegate) Height() int                         { return 2 }
func (wtDelegate) Spacing() int                        { return 0 }
func (wtDelegate) Update(tea.Msg, *list.Model) tea.Cmd { return nil }

func (wtDelegate) Render(out io.Writer, m list.Model, index int, li list.Item) {
	e := li.(wtEntry)
	w, a := e.w, e.a
	width := m.Width()
	marker := "  "
	if index == m.Index() {
		marker = sCursor.Render("› ")
	}
	box := "   "
	if w.Verdict.Removable() {
		box = "[ ]"
		if a.wt.chosen[w.Path] {
			box = "[" + sGreen.Render("x") + "]"
		}
	}
	size := ""
	if s := a.worktreeSize(w.Path); s > 0 {
		size = human.Bytes(s)
	}
	nameW := max(10, width-26)
	name := pad(truncRight(w.RepoName()+" › "+w.Name(), nameW), nameW)
	if index == m.Index() {
		name = sCursor.Render(name)
	}
	fmt.Fprint(out, marker+box+" "+verdictBadge(w.Verdict)+" "+name+padLeft(size, 10)+"\n"+
		"      "+sDim.Render(truncRight(wtSummary(w), max(10, width-8))))
}

func verdictBadge(v worktrees.Verdict) string {
	switch v {
	case worktrees.Merged:
		return sGreen.Render("✓ merged")
	case worktrees.Old:
		return lipgloss.NewStyle().Foreground(hsl(190, 0.6, 0.55)).Render("◷ old   ")
	case worktrees.Stale:
		return sDim.Render("∅ stale ")
	case worktrees.Review:
		return sYellow.Render("! review")
	}
	return sRed.Render("✗ keep  ")
}

// wtSummary is the one-line evidence under a worktree's name.
func wtSummary(w worktrees.Worktree) string {
	var parts []string
	switch {
	case w.Branch != "":
		parts = append(parts, w.Branch)
	case len(w.Head) >= 7:
		parts = append(parts, "detached "+w.Head[:7])
	}
	if w.PR != nil {
		parts = append(parts, fmt.Sprintf("PR #%d %s", w.PR.Number, strings.ToLower(w.PR.State)))
	}
	if !w.LastCommit.IsZero() {
		parts = append(parts, human.Ago(time.Now(), w.LastCommit))
	}
	if len(w.Reasons) > 0 && !strings.HasPrefix(w.Reasons[0], "PR #") { // the PR is already named
		parts = append(parts, w.Reasons[0])
	}
	return strings.Join(parts, " · ")
}

func (a *app) wtKey(msg tea.KeyPressMsg) tea.Cmd {
	w, k := &a.wt, a.keys
	switch w.mode {
	case wtPreparing:
		if msg.String() == "esc" || msg.String() == "ctrl+c" {
			w.mode = wtList
		}
		return nil
	case wtConfirm:
		switch {
		case key.Matches(msg, k.Yes):
			return a.wtStartRun()
		case key.Matches(msg, k.No):
			w.mode = wtList
		case msg.String() == "ctrl+c":
			return tea.Quit
		}
		return nil
	case wtRunning:
		return a.runKey(msg)
	case wtDone:
		back, cmd := a.doneKey(msg)
		if back {
			w.mode = wtList
		}
		return cmd
	}

	if w.list.SettingFilter() {
		var cmd tea.Cmd
		w.list, cmd = w.list.Update(msg)
		return cmd
	}
	e, hasSel := w.list.SelectedItem().(wtEntry)
	switch {
	case msg.String() == "esc" && w.list.IsFiltered():
		w.list.ResetFilter()
	case key.Matches(msg, k.Toggle):
		switch {
		case !hasSel:
		case e.w.Verdict.Removable():
			w.chosen[e.w.Path] = !w.chosen[e.w.Path]
		default:
			a.flash = "Kept: " + strings.Join(e.w.Reasons, "; ")
		}
	case key.Matches(msg, k.PickVerified):
		for _, x := range w.all {
			if x.Verdict.Preselect() {
				w.chosen[x.Path] = true
			}
		}
	case key.Matches(msg, k.None):
		for p := range w.chosen {
			w.chosen[p] = false
		}
	case key.Matches(msg, k.WtBranches):
		w.branches = !w.branches
	case key.Matches(msg, k.WtFetch):
		a.flash = "Fetching origin for each repo, then checking again…"
		return a.auditWorktrees(true)
	case key.Matches(msg, k.Recheck):
		a.env.ResetProcs()
		return a.auditWorktrees(false)
	case key.Matches(msg, k.Reveal):
		if hasSel {
			return reveal(e.w.Path)
		}
	case key.Matches(msg, k.Remove):
		return a.wtPrepare()
	default:
		var cmd tea.Cmd
		w.list, cmd = w.list.Update(msg)
		return cmd
	}
	return nil
}

// wtPrepare checks everything again — what runs, what changed — so nothing
// picked earlier is removed after it came back into use.
func (a *app) wtPrepare() tea.Cmd {
	if n, _ := a.wtChosen(); n == 0 {
		a.flash = "Pick a worktree first: space, or a for everything verified."
		return nil
	}
	a.env.ResetProcs()
	a.wt.mode = wtPreparing
	return a.auditWorktrees(false)
}

func (a *app) wtStartRun() tea.Cmd {
	var items []clean.Item
	for _, x := range a.wt.all {
		if a.wt.chosen[x.Path] && x.Verdict.Removable() {
			items = append(items, clean.WorktreeItem(x, a.worktreeSize(x.Path)))
		}
	}
	a.wt.mode = wtRunning
	return a.startRun(tabWorktrees, []clean.Selection{{Task: clean.WorktreeTask(a.wt.branches), Items: items}})
}

func (a *app) wtMouse(msg tea.MouseMsg) tea.Cmd {
	switch a.wt.mode {
	case wtRunning, wtDone:
		return a.runMouse(msg)
	case wtList:
		if pressed(msg) {
			switch msg.Mouse().Button {
			case tea.MouseWheelUp:
				a.wt.list.CursorUp()
			case tea.MouseWheelDown:
				a.wt.list.CursorDown()
			}
		}
	}
	return nil
}

func (a *app) wtHelpKeys(tabs []key.Binding) keyHelp {
	k := a.keys
	switch a.wt.mode {
	case wtPreparing:
		return keyHelp{short: []key.Binding{key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel"))}}
	case wtConfirm:
		return keyHelp{short: []key.Binding{k.Yes, k.No}}
	case wtRunning:
		return keyHelp{short: []key.Binding{k.Stop}}
	case wtDone:
		return keyHelp{short: []key.Binding{k.Done}}
	}
	return keyHelp{
		short: []key.Binding{k.Up, k.Down, k.Toggle, k.PickVerified, k.WtBranches, k.Remove, k.Filter, k.Help},
		full: [][]key.Binding{
			{k.Up, k.Down, k.Toggle, k.PickVerified, k.None, k.Filter},
			{k.WtBranches, k.Recheck, k.WtFetch, k.Reveal, k.Remove},
			tabs, {k.Help, k.Quit},
		},
	}
}

func (a *app) worktreesView(h int) string {
	w := &a.wt
	switch w.mode {
	case wtPreparing:
		return a.spin.View() + " Checking every picked worktree again — what runs in it, what changed — before anything is removed…"
	case wtConfirm:
		return a.wtConfirmView()
	case wtRunning:
		return a.runningView(h)
	case wtDone:
		return a.doneView(h)
	}
	if len(w.all) == 0 {
		if w.loading {
			return a.spin.View() + " Checking the worktrees of every repo under ~/projects: git status, what is merged, PRs via gh…"
		}
		return sDim.Render("No linked worktrees under ~/projects.")
	}

	head := a.wtHeader()
	listW, detailW := a.w, 0
	if a.w >= 120 {
		detailW = min(64, a.w*2/5)
		listW = a.w - detailW - 1
	}
	bodyH := max(4, h-lipgloss.Height(head)-1)
	var detail string
	if e, ok := w.list.SelectedItem().(wtEntry); ok {
		if detailW > 0 {
			detail = a.wtDetail(e.w, detailW, bodyH)
		} else {
			detail = a.wtDetail(e.w, a.w, 12)
			bodyH -= lipgloss.Height(detail)
		}
	}
	w.list.SetSize(listW, max(4, bodyH))
	body := w.list.View()
	if detailW > 0 {
		body = lipgloss.JoinHorizontal(lipgloss.Top, lipgloss.NewStyle().Width(listW).Render(body), " ", detail)
	} else {
		body += "\n" + detail
	}
	return head + "\n\n" + body
}

func (a *app) wtHeader() string {
	w := &a.wt
	counts := map[worktrees.Verdict]int{}
	repos := map[string]bool{}
	for _, x := range w.all {
		counts[x.Verdict]++
		repos[x.Repo] = true
	}
	line := sHeading.Render(fmt.Sprintf("%d worktrees in %d repos", len(w.all), len(repos)))
	for _, v := range []worktrees.Verdict{worktrees.Merged, worktrees.Old, worktrees.Stale, worktrees.Review, worktrees.Keep} {
		if counts[v] > 0 {
			line += "  " + verdictBadge(v) + sDim.Render(fmt.Sprintf(" %d", counts[v]))
		}
	}
	if w.loading {
		label := " checking again…"
		if w.fetching {
			label = " fetching, then checking…"
		}
		line += "  " + a.spin.View() + sDim.Render(label)
	}
	n, size := a.wtChosen()
	second := fmt.Sprintf("Picked: %s", plural(n, "worktree"))
	if size > 0 {
		second += ", " + human.Bytes(size)
	}
	if w.branches {
		second += sYellow.Render(" · merged branches go too")
	} else {
		second += sDim.Render(" · branches stay (b to delete merged ones too)")
	}
	if f := oldestFetch(w.all); !f.IsZero() && time.Since(f) > 24*time.Hour {
		second += sDim.Render(" · a repo last fetched " + human.Ago(time.Now(), f) + " (f fetches)")
	}
	if a.opts.DryRun {
		second += "  " + sDryBadge.Render("DRY RUN")
	}
	fit := lipgloss.NewStyle().MaxWidth(a.w)
	return fit.Render(line) + "\n" + fit.Render(second)
}

func oldestFetch(all []worktrees.Worktree) time.Time {
	var oldest time.Time
	for _, x := range all {
		if !x.Fetched.IsZero() && (oldest.IsZero() || x.Fetched.Before(oldest)) {
			oldest = x.Fetched
		}
	}
	return oldest
}

// wtDetail lays out all the evidence about one worktree.
func (a *app) wtDetail(w worktrees.Worktree, width, h int) string {
	inner := max(10, width-4)
	labelW := 9
	valueW := max(10, inner-labelW-1)
	var rows []string
	row := func(label, value string) {
		wrapped := lipgloss.NewStyle().Width(valueW).Render(value)
		for i, l := range strings.Split(wrapped, "\n") {
			if i == 0 {
				rows = append(rows, sDim.Render(pad(label, labelW))+" "+l)
			} else {
				rows = append(rows, strings.Repeat(" ", labelW+1)+l)
			}
		}
	}
	rows = append(rows, sBold.Render(truncRight(w.RepoName()+" › "+w.Name(), inner)))
	rows = append(rows, sDim.Render(truncLeft(tildePath(a.home, w.Path), inner)), "")
	row("Verdict", verdictBadge(w.Verdict)+"  "+strings.Join(w.Reasons, "; "))
	switch {
	case w.Branch != "":
		branch := w.Branch
		if a.wt.branches && w.DeletableBranch() {
			branch += sYellow.Render("  (deleted with it)")
		}
		row("Branch", branch)
	case len(w.Head) >= 7:
		row("Branch", "detached at "+w.Head[:7])
	}
	switch {
	case w.PR != nil:
		row("PR", fmt.Sprintf("#%d %s · %s", w.PR.Number, strings.ToLower(w.PR.State), w.PR.Title))
		row("", sDim.Render(w.PR.URL))
	case w.PRError != "":
		row("PR", sDim.Render(w.PRError))
	case w.Branch != "":
		row("PR", sDim.Render("none found for this branch"))
	}
	if w.Base != "" && !w.Prunable {
		row("Commits", fmt.Sprintf("%d ahead, %d behind %s · %d not in it by content · %d held only here",
			w.Ahead, w.Behind, w.Base, w.Unique, w.Unpushed))
	}
	if !w.Prunable {
		changes := "clean"
		if w.Changed+w.Untracked > 0 {
			changes = fmt.Sprintf("%s, %s", plural(w.Changed, "changed file"), plural(w.Untracked, "untracked file"))
		}
		row("Changes", changes)
		ignored := sDim.Render("only build output")
		if len(w.Ignored) > 0 {
			ignored = strings.Join(w.Ignored, ", ")
		}
		row("Ignored", ignored)
		inUse := sDim.Render("no")
		if w.InUse != "" {
			inUse = sRed.Render(w.InUse)
		}
		row("In use", inUse)
	}
	if !w.LastCommit.IsZero() {
		row("Last", human.Ago(time.Now(), w.LastCommit)+" — "+w.Subject)
	}
	if s := a.worktreeSize(w.Path); s > 0 {
		row("Size", human.Bytes(s))
	}
	if len(rows) > h-2 {
		rows = rows[:max(1, h-2)]
	}
	return lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(colFaint).
		Padding(0, 1).Width(width - 2).Render(strings.Join(rows, "\n"))
}

func (a *app) wtConfirmView() string {
	n, size := a.wtChosen()
	title := fmt.Sprintf("Remove %s, about %s?", plural(n, "worktree"), human.Bytes(size))
	if a.opts.DryRun {
		title = fmt.Sprintf("Dry run: %s, about %s. Nothing will be removed.", plural(n, "worktree"), human.Bytes(size))
	}
	lines := []string{sBold.Render(title), ""}
	shown, branches := 0, 0
	for _, x := range a.wt.all {
		if !a.wt.chosen[x.Path] || !x.Verdict.Removable() {
			continue
		}
		if a.wt.branches && x.DeletableBranch() {
			branches++
		}
		if shown++; shown > 14 {
			continue
		}
		lines = append(lines, fmt.Sprintf(" %s %s %s  %s", sDim.Render("•"), verdictBadge(x.Verdict),
			pad(truncRight(x.RepoName()+" › "+x.Name(), 48), 48), padLeft(human.Bytes(a.worktreeSize(x.Path)), 9)))
	}
	if shown > 14 {
		lines = append(lines, sDim.Render(fmt.Sprintf("   … and %d more", shown-14)))
	}
	lines = append(lines, "")
	if branches > 0 {
		lines = append(lines, sYellow.Render(fmt.Sprintf("%s of merged work will be deleted too.", plural(branches, "branch"))))
	} else {
		lines = append(lines, sDim.Render("Branches stay; only the checkouts go."))
	}
	lines = append(lines, sDim.Render("git worktree remove refuses any worktree with changes — nothing is forced."),
		"", sGreen.Render("y")+" remove   "+sDim.Render("n / esc")+" cancel")
	return sBox.Render(strings.Join(lines, "\n"))
}
