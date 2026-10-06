package ui

import (
	"context"
	"fmt"
	"io"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/lechero/dotfiles/apps/internal/disk/clean"
	"github.com/lechero/dotfiles/apps/internal/disk/docker"
	"github.com/lechero/dotfiles/apps/internal/disk/human"
)

const (
	dkList      = iota
	dkPreparing // looking at Docker again before asking to confirm
	dkConfirm
	dkRunning
	dkDone
)

type dkState struct {
	mode    int
	list    list.Model
	rep     docker.Report
	index   map[string]int  // resource key → position in rep.Resources
	chosen  map[string]bool // by key, so a re-check keeps your picks
	seen    map[string]bool // judged before: a re-check never re-picks what you unpicked
	loading bool
	gen     int
	before  docker.Disk // Docker's disk as the last run started
	ranAt   time.Time
	plan    docker.Plan // the cheapest way to the target free space
	// partial holds, by key, the build cache s picked to prune only in part;
	// picking it by hand prunes it all again.
	partial map[string]docker.Pick
}

type dkAuditMsg struct {
	gen int
	rep docker.Report
}

func newDkState() dkState {
	return dkState{list: newPickList(dkDelegate{}, "resource", "resources"), chosen: map[string]bool{}, seen: map[string]bool{},
		partial: map[string]docker.Pick{}}
}

// auditDocker looks at everything Docker keeps, in the background.
func (a *app) auditDocker() tea.Cmd {
	d := &a.dk
	d.gen++
	d.loading = true
	gen, env := d.gen, a.env
	return tea.Batch(a.wake(), func() tea.Msg {
		return dkAuditMsg{gen: gen, rep: clean.AuditDocker(context.Background(), env)}
	})
}

func (a *app) onDockerAudit(msg dkAuditMsg) tea.Cmd {
	d := &a.dk
	if msg.gen != d.gen {
		return nil
	}
	d.loading, d.rep = false, msg.rep
	d.plan = docker.Suggest(d.rep, a.opts.DockerTarget)
	d.index = make(map[string]int, len(d.rep.Resources))
	for i, r := range d.rep.Resources {
		d.index[r.Key()] = i
	}
	for _, r := range d.rep.Resources {
		k := r.Key()
		if !d.seen[k] {
			d.seen[k] = true
			d.chosen[k] = r.Verdict.Preselect()
		}
		if !r.Verdict.Removable() {
			d.chosen[k] = false
		}
	}
	d.settle()

	var keep string
	if e, ok := d.list.SelectedItem().(dkEntry); ok {
		keep = e.r.Key()
	}
	entries := make([]list.Item, len(d.rep.Resources))
	for i, r := range d.rep.Resources {
		entries[i] = dkEntry{r: r, a: a}
	}
	cmd := d.list.SetItems(entries)
	if i, ok := d.index[keep]; ok {
		d.list.Select(i)
	}
	if d.mode == dkPreparing {
		if n, _ := a.dkChosen(); n == 0 {
			d.mode = dkList
			a.flash = "Nothing to remove: what you picked is in use or gone since."
		} else {
			d.mode = dkConfirm
		}
	}
	return cmd
}

func (d *dkState) get(k string) (docker.Resource, bool) {
	if i, ok := d.index[k]; ok {
		return d.rep.Resources[i], true
	}
	return docker.Resource{}, false
}

// settle keeps picks whole after a fresh look: a pick brings along what must
// go first (the stopped containers pinning an image), and is dropped when one
// of those can't go any more.
func (d *dkState) settle() {
	for changed := true; changed; {
		changed = false
		for _, r := range d.rep.Resources {
			if !d.chosen[r.Key()] {
				continue
			}
			for _, k := range r.Implies {
				x, ok := d.get(k)
				switch {
				case !ok || !x.Verdict.Removable():
					d.chosen[r.Key()], changed = false, true
				case !d.chosen[k]:
					d.chosen[k], changed = true, true
				}
			}
		}
	}
}

// pick switches one resource on or off, with what it depends on: picking an
// image picks the stopped containers that pin it, and unpicking one of those
// unpicks the image.
func (d *dkState) pick(k string, on bool) {
	if d.chosen[k] == on {
		return
	}
	d.chosen[k] = on
	if on {
		r, _ := d.get(k)
		for _, dep := range r.Implies {
			d.pick(dep, true)
		}
		return
	}
	for _, r := range d.rep.Resources {
		if d.chosen[r.Key()] && slices.Contains(r.Implies, k) {
			d.pick(r.Key(), false)
		}
	}
}

// dkPicked lists what a removal would take, in the order it goes: containers
// before the images and volumes they pin, the build cache last so it also
// frees the layers of images removed just before it.
func (a *app) dkPicked() []docker.Resource {
	var out []docker.Resource
	for _, r := range a.dk.rep.Resources {
		if a.dk.chosen[r.Key()] && r.Verdict.Removable() {
			out = append(out, a.dkEffective(r))
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Kind != out[j].Kind {
			return out[i].Kind < out[j].Kind
		}
		return out[i].Until > out[j].Until // stale cache before the rest of it
	})
	return out
}

// dkEffective is r as a removal would take it: build cache a plan prunes only
// in part shows, and removes, just that part, measured again from the latest
// look since the cut is a moment, not a list.
func (a *app) dkEffective(r docker.Resource) docker.Resource {
	pk, ok := a.dk.partial[r.Key()]
	if !ok || !a.dk.chosen[r.Key()] {
		return r
	}
	size, n := r.Cut(pk.Before)
	r.Name += fmt.Sprintf(" (the oldest %d of %d records)", n, r.Records)
	r.Reasons = []string{fmt.Sprintf("only what no build used since %s: the builds since keep their cache", human.Ago(time.Now(), pk.Before)),
		fmt.Sprintf("the oldest %d of %d records", n, r.Records)}
	r.Before, r.Size, r.Records, r.LastUsed = pk.Before, size, n, pk.Before
	return r
}

func (a *app) dkChosen() (n int, size int64) {
	for _, r := range a.dkPicked() {
		n++
		size += r.Size
	}
	return n, size
}

// dkEntry is a row of the list; it reads picks from the app live.
type dkEntry struct {
	r docker.Resource
	a *app
}

func (e dkEntry) FilterValue() string {
	return e.r.Kind.String() + " " + e.r.Name + " " + e.r.Verdict.String() + " " + e.r.Project + " " + strings.Join(e.r.Tags, " ")
}

type dkDelegate struct{}

func (dkDelegate) Height() int                         { return 2 }
func (dkDelegate) Spacing() int                        { return 0 }
func (dkDelegate) Update(tea.Msg, *list.Model) tea.Cmd { return nil }

func (dkDelegate) Render(out io.Writer, m list.Model, index int, li list.Item) {
	e := li.(dkEntry)
	a := e.a
	r := a.dkEffective(e.r)
	width := m.Width()
	marker := "  "
	if index == m.Index() {
		marker = sCursor.Render("› ")
	}
	box := "   "
	if r.Verdict.Removable() {
		box = "[ ]"
		if a.dk.chosen[r.Key()] {
			box = "[" + sGreen.Render("x") + "]"
		}
	}
	nameW := max(10, width-38)
	name := pad(truncRight(r.Name, nameW), nameW)
	if index == m.Index() {
		name = sCursor.Render(name)
	}
	fmt.Fprint(out, marker+box+" "+dkBadge(r.Verdict)+" "+sDim.Render(pad(kindLabel(r.Kind), 9))+" "+name+padLeft(human.Bytes(r.Size), 10)+"\n"+
		"      "+sDim.Render(truncRight(strings.Join(r.Reasons, " · "), max(10, width-8))))
}

func kindLabel(k docker.Kind) string {
	if k == docker.BuildCache {
		return "cache"
	}
	return k.String()
}

func dkBadge(v docker.Verdict) string {
	switch v {
	case docker.Unused:
		return sGreen.Render("✓ unused")
	case docker.Orphan:
		return sDim.Render("∅ orphan")
	case docker.Old:
		return lipgloss.NewStyle().Foreground(hsl(190, 0.6, 0.55)).Render("◷ old   ")
	case docker.Review:
		return sYellow.Render("! review")
	case docker.Data:
		return lipgloss.NewStyle().Foreground(hsl(275, 0.6, 0.65)).Render("◆ data  ")
	}
	return sRed.Render("✗ in use")
}

func (a *app) dkKey(msg tea.KeyMsg) tea.Cmd {
	d, k := &a.dk, a.keys
	switch d.mode {
	case dkPreparing:
		if msg.Type == tea.KeyEsc || msg.Type == tea.KeyCtrlC {
			d.mode = dkList
		}
		return nil
	case dkConfirm:
		switch {
		case key.Matches(msg, k.Yes):
			return a.dkStartRun()
		case key.Matches(msg, k.No):
			d.mode = dkList
		case msg.Type == tea.KeyCtrlC:
			return tea.Quit
		}
		return nil
	case dkRunning:
		return a.runKey(msg)
	case dkDone:
		back, cmd := a.doneKey(msg)
		if back {
			d.mode = dkList
		}
		return cmd
	}

	if d.list.SettingFilter() {
		var cmd tea.Cmd
		d.list, cmd = d.list.Update(msg)
		return cmd
	}
	e, hasSel := d.list.SelectedItem().(dkEntry)
	switch {
	case msg.Type == tea.KeyEsc && d.list.IsFiltered():
		d.list.ResetFilter()
	case key.Matches(msg, k.Toggle):
		switch {
		case !hasSel:
		case e.r.Verdict.Removable():
			delete(d.partial, e.r.Key()) // by hand, a pick is all of it
			d.pick(e.r.Key(), !d.chosen[e.r.Key()])
		default:
			a.flash = "In use: " + strings.Join(e.r.Reasons, "; ")
		}
	case key.Matches(msg, k.PickVerified):
		for _, r := range d.rep.Resources {
			if r.Verdict.Preselect() {
				d.pick(r.Key(), true)
			}
		}
	case key.Matches(msg, k.None):
		for p := range d.chosen {
			d.chosen[p] = false
		}
		clear(d.partial)
	case key.Matches(msg, k.Suggest):
		a.dkSuggest()
	case key.Matches(msg, k.Recheck):
		a.env.ResetProcs()
		return a.auditDocker()
	case key.Matches(msg, k.Reveal):
		if hasSel && e.r.ProjectDir != "" && !e.r.ProjectGone {
			return reveal(e.r.ProjectDir)
		}
		a.flash = "Only a compose project's folder can be shown in Finder; Docker keeps the rest inside its VM."
	case key.Matches(msg, k.Remove):
		return a.dkPrepare()
	default:
		var cmd tea.Cmd
		d.list, cmd = d.list.Update(msg)
		return cmd
	}
	return nil
}

// dkSuggest replaces your picks with the plan for the target, for you to look
// over before c.
func (a *app) dkSuggest() {
	d, p := &a.dk, a.dk.plan
	target := human.Bytes(p.Target)
	switch {
	case p.Err != "":
		a.flash = "No plan: " + p.Err
		return
	case p.Reached():
		a.flash = "Docker's disk already has the " + target + " target free."
		return
	}
	for k := range d.chosen {
		d.chosen[k] = false
	}
	clear(d.partial)
	for _, pk := range p.Picks {
		d.chosen[pk.Key] = true
		if !pk.Before.IsZero() {
			d.partial[pk.Key] = pk
		}
	}
	d.settle()
	if p.Short == 0 {
		a.flash = fmt.Sprintf("Picked the cheapest way to %s free: %s, about %s. c removes them.",
			target, plural(len(p.Picks), "resource"), human.Bytes(p.Frees))
		return
	}
	var vols []string
	for _, k := range p.Volumes[:min(3, len(p.Volumes))] {
		if r, ok := d.get(k); ok {
			vols = append(vols, r.Name+" "+human.Bytes(r.Size))
		}
	}
	a.flash = fmt.Sprintf("Picked everything but volumes, about %s: still %s short of %s.", human.Bytes(p.Frees), human.Bytes(p.Short), target)
	if len(vols) > 0 {
		a.flash += " Volumes are yours to pick (" + strings.Join(vols, ", ") + "), or raise Docker's disk limit."
	} else {
		a.flash += " Raise Docker's disk limit: Settings → Resources."
	}
}

// dkPrepare looks at Docker again, so nothing picked earlier is removed after
// a container started using it.
func (a *app) dkPrepare() tea.Cmd {
	if n, _ := a.dkChosen(); n == 0 {
		a.flash = "Pick something first: space, or a for everything verified."
		return nil
	}
	a.env.ResetProcs()
	a.dk.mode = dkPreparing
	return a.auditDocker()
}

func (a *app) dkStartRun() tea.Cmd {
	var items []clean.Item
	for _, r := range a.dkPicked() {
		items = append(items, clean.DockerItem(r))
	}
	a.dk.mode, a.dk.before, a.dk.ranAt = dkRunning, a.dk.rep.Disk, time.Now()
	return a.startRun(tabDocker, []clean.Selection{{Task: clean.DockerTask(), Items: items}})
}

func (a *app) dkMouse(msg tea.MouseMsg) tea.Cmd {
	switch a.dk.mode {
	case dkRunning, dkDone:
		return a.runMouse(msg)
	case dkList:
		if msg.Action == tea.MouseActionPress {
			switch msg.Button {
			case tea.MouseButtonWheelUp:
				a.dk.list.CursorUp()
			case tea.MouseButtonWheelDown:
				a.dk.list.CursorDown()
			}
		}
	}
	return nil
}

func (a *app) dkHelpKeys(tabs []key.Binding) keyHelp {
	k := a.keys
	switch a.dk.mode {
	case dkPreparing:
		return keyHelp{short: []key.Binding{key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel"))}}
	case dkConfirm:
		return keyHelp{short: []key.Binding{k.Yes, k.No}}
	case dkRunning:
		return keyHelp{short: []key.Binding{k.Stop}}
	case dkDone:
		return keyHelp{short: []key.Binding{k.Done}}
	}
	return keyHelp{
		short: []key.Binding{k.Up, k.Down, k.Toggle, k.Suggest, k.PickVerified, k.Remove, k.Filter, k.Help},
		full: [][]key.Binding{
			{k.Up, k.Down, k.Toggle, k.PickVerified, k.None, k.Filter},
			{k.Suggest, k.Recheck, k.Reveal, k.Remove},
			tabs, {k.Help, k.Quit},
		},
	}
}

func (a *app) dockerView(h int) string {
	d := &a.dk
	switch d.mode {
	case dkPreparing:
		return a.spin.View() + " Looking at Docker again — what runs, what uses what — before anything is removed…"
	case dkConfirm:
		return a.dkConfirmView()
	case dkRunning:
		return a.runningView(h)
	case dkDone:
		return a.doneView(h)
	}
	if d.rep.At.IsZero() {
		return a.spin.View() + " Asking Docker what it keeps: images, containers, volumes, build cache, and how full its disk is…"
	}
	if d.rep.Err != "" {
		msg := d.rep.Err + "."
		switch {
		case d.rep.Down:
			msg = "Docker isn't running. Start Docker Desktop, then press r."
		case d.rep.Missing:
			msg = "Docker isn't installed."
		}
		if d.loading {
			msg = a.spin.View() + " " + msg
		}
		return sDim.Render(msg)
	}

	head := a.dkHeader()
	listW, detailW := a.w, 0
	if a.w >= 120 {
		detailW = min(64, a.w*2/5)
		listW = a.w - detailW - 1
	}
	bodyH := max(4, h-lipgloss.Height(head)-1)
	var detail string
	if e, ok := d.list.SelectedItem().(dkEntry); ok {
		if detailW > 0 {
			detail = a.dkDetail(a.dkEffective(e.r), detailW, bodyH)
		} else {
			detail = a.dkDetail(a.dkEffective(e.r), a.w, 12)
			bodyH -= lipgloss.Height(detail)
		}
	}
	d.list.SetSize(listW, max(4, bodyH))
	body := d.list.View()
	if detailW > 0 {
		body = lipgloss.JoinHorizontal(lipgloss.Top, lipgloss.NewStyle().Width(listW).Render(body), " ", detail)
	} else {
		body += "\n" + detail
	}
	return head + "\n\n" + body
}

// dkHeader shows Docker's own disk — what a build runs out of — what each
// verdict holds, and where your picks would leave it.
func (a *app) dkHeader() string {
	d := &a.dk
	disk := d.rep.Disk
	line := sHeading.Render("Docker's disk") + "  "
	if disk.OK() {
		a.gauge.Width = min(40, max(10, a.w/4))
		line += a.gauge.ViewAs(float64(disk.Used())/float64(disk.Total)) + "  " + sBold.Render(human.Bytes(disk.Free)) +
			sDim.Render(" free of "+human.Bytes(disk.Total)+" inside Docker's VM")
	} else {
		line += sDim.Render("not measured: " + disk.Err)
	}
	if d.loading {
		line += "  " + a.spin.View() + sDim.Render(" checking again…")
	}

	counts := map[docker.Verdict]int{}
	sizes := map[docker.Verdict]int64{}
	for _, r := range d.rep.Resources {
		counts[r.Verdict]++
		sizes[r.Verdict] += r.Size
	}
	var parts []string
	for _, v := range []docker.Verdict{docker.Unused, docker.Orphan, docker.Old, docker.Review, docker.Data, docker.InUse} {
		if counts[v] > 0 {
			parts = append(parts, dkBadge(v)+sDim.Render(fmt.Sprintf(" %d · %s", counts[v], human.Bytes(sizes[v]))))
		}
	}
	second := strings.Join(parts, "   ")

	n, size := a.dkChosen()
	third := fmt.Sprintf("Picked: %s", plural(n, "resource"))
	if size > 0 {
		third += ", about " + human.Bytes(size)
		if disk.OK() {
			third += sDim.Render(" → about ") + sBold.Render(human.Bytes(disk.Free+size)) + sDim.Render(" free after")
		}
	}
	if d.rep.SharedCache > 0 {
		third += sDim.Render(fmt.Sprintf(" · %s cache shared with images", human.Bytes(d.rep.SharedCache)))
	}
	if a.opts.DryRun {
		third += "  " + sDryBadge.Render("DRY RUN")
	}
	fit := lipgloss.NewStyle().MaxWidth(a.w)
	out := fit.Render(line) + "\n" + fit.Render(second) + "\n" + fit.Render(third)
	if t := a.dkTargetLine(size); t != "" {
		out += "\n" + fit.Render(t)
	}
	return out
}

// dkTargetLine says how far Docker's disk is from the target, and what s would
// pick to get there.
func (a *app) dkTargetLine(picked int64) string {
	p := a.dk.plan
	if p.Err != "" {
		return ""
	}
	line := sHeading.Render("Target") + sDim.Render(" "+human.Bytes(p.Target)+" free: ")
	if p.Reached() {
		return line + sGreen.Render("met")
	}
	line += sBold.Render(human.Bytes(p.Need())) + sDim.Render(" to go")
	switch {
	case picked >= p.Need():
		line += sGreen.Render(" · your picks get there")
	case p.Short > 0:
		line += sDim.Render(" · everything but volumes frees about "+human.Bytes(p.Frees)+", ") +
			sYellow.Render(human.Bytes(p.Short)+" short") + sDim.Render(" · s picks all of it")
	default:
		line += sDim.Render(fmt.Sprintf(" · s picks the cheapest way there: %s, about %s", plural(len(p.Picks), "resource"), human.Bytes(p.Frees)))
	}
	return line
}

// dkDetail lays out all the evidence about one resource.
func (a *app) dkDetail(r docker.Resource, width, h int) string {
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
	now := time.Now()
	rows = append(rows, sBold.Render(truncRight(r.Name, inner)))
	sub := r.Kind.String()
	if r.Kind == docker.Container || r.Kind == docker.Image {
		sub += " " + r.ShortID()
	}
	rows = append(rows, sDim.Render(sub), "")
	row("Verdict", dkBadge(r.Verdict)+"  "+strings.Join(r.Reasons, "; "))
	size := human.Bytes(r.Size)
	if r.Shared > 0 {
		size += sDim.Render(" (+" + human.Bytes(r.Shared) + " shared with other images, which stays)")
	}
	row("Frees", size)
	switch r.Kind {
	case docker.Container:
		row("State", r.State)
		row("Image", r.Image)
		if len(r.Mounts) > 0 {
			row("Volumes", strings.Join(r.Mounts, ", ")+sDim.Render(" (they stay)"))
		}
	case docker.Image:
		if len(r.Tags) > 1 {
			row("Tags", strings.Join(r.Tags, ", "))
		}
		if len(r.Tags) > 0 { // nothing names an untagged image, so nothing pulls it back
			registry := sYellow.Render("no: built here and never pushed")
			if r.InRegistry {
				registry = "yes: docker pull brings it back"
			}
			row("Registry", registry)
		}
	case docker.BuildCache:
		row("Records", fmt.Sprint(r.Records))
		if !r.LastUsed.IsZero() {
			row("Used", human.Ago(now, r.LastUsed))
		}
	}
	if r.Kind == docker.Image || r.Kind == docker.Volume {
		var users []string
		for _, u := range r.Users {
			state := "stopped"
			if u.Running {
				state = "running"
			}
			users = append(users, u.Name+sDim.Render(" ("+state+")"))
		}
		value := sDim.Render("none")
		if len(users) > 0 {
			value = strings.Join(users, ", ")
		}
		row("Used by", value)
	}
	if r.Project != "" {
		project := r.Project
		if r.Service != "" {
			project += sDim.Render(" · service " + r.Service)
		}
		row("Project", project)
		switch {
		case r.ProjectGone:
			row("", sRed.Render(tildePath(a.home, r.ProjectDir)+" is gone"))
		case r.ProjectDir != "":
			row("", sDim.Render(tildePath(a.home, r.ProjectDir)))
		}
	}
	if !r.Created.IsZero() && r.Kind != docker.BuildCache {
		row("Created", human.Ago(now, r.Created))
	}
	if len(r.Implies) > 0 {
		var first []string
		for _, k := range r.Implies {
			if x, ok := a.dk.get(k); ok {
				first = append(first, x.Name)
			}
		}
		row("First", strings.Join(first, ", ")+sDim.Render(" (picked along with it)"))
	}
	if r.Verdict.Removable() {
		cmd := strings.ReplaceAll(strings.Join(r.Command(), " "), r.ID, r.ShortID())
		row("Runs", sDim.Render("docker "+cmd))
	}
	if len(rows) > h-2 {
		rows = rows[:max(1, h-2)]
	}
	return lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(colFaint).
		Padding(0, 1).Width(width - 2).Render(strings.Join(rows, "\n"))
}

func (a *app) dkConfirmView() string {
	picked := a.dkPicked()
	n, size := a.dkChosen()
	title := fmt.Sprintf("Remove %s, about %s?", plural(n, "resource"), human.Bytes(size))
	if a.opts.DryRun {
		title = fmt.Sprintf("Dry run: %s, about %s. Nothing will be removed.", plural(n, "resource"), human.Bytes(size))
	}
	lines := []string{sBold.Render(title), ""}
	volumes, builtHere, cache := 0, 0, false
	for i, r := range picked {
		switch {
		case r.Kind == docker.Volume && r.Size > 0:
			volumes++
		case r.Kind == docker.Image && r.BuiltHere:
			builtHere++
		case r.Kind == docker.BuildCache:
			cache = true
		}
		if i >= 14 {
			continue
		}
		lines = append(lines, fmt.Sprintf(" %s %s %s %s  %s", sDim.Render("•"), dkBadge(r.Verdict), sDim.Render(pad(kindLabel(r.Kind), 9)),
			pad(truncRight(r.Name, 44), 44), padLeft(human.Bytes(r.Size), 9)))
	}
	if len(picked) > 14 {
		lines = append(lines, sDim.Render(fmt.Sprintf("   … and %d more", len(picked)-14)))
	}
	lines = append(lines, "")
	if disk := a.dk.rep.Disk; disk.OK() {
		line := fmt.Sprintf("Docker's disk: about %s → %s free.", human.Bytes(disk.Free), human.Bytes(disk.Free+size))
		switch target := a.opts.DockerTarget; {
		case disk.Free >= target:
		case disk.Free+size >= target:
			line += sGreen.Render(" That meets the " + human.Bytes(target) + " target.")
		default:
			line += sYellow.Render(" Short of the " + human.Bytes(target) + " target.")
		}
		lines = append(lines, line)
	}
	if volumes > 0 {
		lines = append(lines, sRed.Render(fmt.Sprintf("%s: the data in them is gone for good.", plural(volumes, "volume"))))
	}
	if builtHere > 0 && !cache {
		what := "1 image was"
		if builtHere > 1 {
			what = fmt.Sprintf("%d images were", builtHere)
		}
		lines = append(lines, sYellow.Render(what+" built here: their layers may stay in the build cache unless you pick that too."))
	}
	lines = append(lines, sDim.Render("Nothing is forced: docker refuses whatever a container started using since the check."),
		"", sGreen.Render("y")+" remove   "+sDim.Render("n / esc")+" cancel")
	return sBox.Render(strings.Join(lines, "\n"))
}

// dkFreedLine is the done view's account of Docker's disk, which is what a
// Docker run changes; the Mac gets its space back minutes later, if at all.
func (a *app) dkFreedLine(dry bool) string {
	d := &a.dk
	if dry || !d.before.OK() {
		return ""
	}
	if d.loading || d.rep.At.Before(d.ranAt) {
		return a.spin.View() + sDim.Render(" measuring Docker's disk again…")
	}
	after := d.rep.Disk
	if !after.OK() {
		return sDim.Render("Docker's disk: couldn't measure it again (" + after.Err + ")")
	}
	return fmt.Sprintf("Docker's disk %s → %s free  %s", human.Bytes(d.before.Free), human.Bytes(after.Free),
		sGreen.Render("(+"+human.Bytes(max(0, after.Free-d.before.Free))+")"))
}
