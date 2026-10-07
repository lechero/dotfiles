package tui

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"

	"github.com/lechero/dotfiles/apps/internal/dotui/work"
)

const (
	// serviceTop is the lines above a service tab's body: dotui's tab line,
	// the account line and the sub-tab line.
	serviceTop = 3
	// workTimeout bounds one call to a service's CLI.
	workTimeout = time.Minute
	// refreshEvery is how often the dashboards of connected services reload.
	refreshEvery = 5 * time.Minute
)

type (
	checkedMsg struct {
		v       *serviceView
		status  work.Status
		account string
		err     error
	}
	dashLoadedMsg struct {
		v    *serviceView
		dash work.Dashboard
		err  error
	}
	detailLoadedMsg struct {
		v      *serviceView
		id     string
		detail work.Detail
		err    error
	}
	// actionDoneMsg comes back from a command that had the terminal.
	actionDoneMsg struct {
		v      *serviceView
		what   string
		err    error
		signIn bool // it was the sign-in or the install: check again
	}
	composedMsg struct {
		v      *serviceView
		item   work.Item
		action work.Action
		file   string
		err    error
	}
	flashMsg struct {
		v    *serviceView
		text string
		bad  bool
	}
	refreshMsg struct{}
)

// serviceView is a GitHub, GitLab or Jira tab: a dashboard of what needs
// following up, a list per section, and a page for one item.
type serviceView struct {
	svc   work.Service
	st    *styles
	cache work.Cache
	// self and file run `dotui _exec` and `dotui install`.
	self, file string

	status   work.Status
	account  string
	checkErr error
	checking bool

	dash      *work.Dashboard
	fromCache bool // dash is the last run's, until a fresh one loads
	dashErr   error
	loading   bool

	view  int          // 0 is the follow-ups, i+1 is section i
	lists []list.Model // one per view
	page  *itemPage    // the open item, or nil

	confirm     *work.Action // waiting for y
	confirmItem work.Item
	choose      *work.Action // waiting for a number, one of its choices
	chooseItem  work.Item
	flash       string
	flashBad    bool

	width, height int

	// Replaced in tests.
	execute func(cmd *exec.Cmd, done tea.ExecCallback) tea.Cmd
	open    func(url string) error
	copy    func(text string) error
	now     func() time.Time
}

// itemPage is one item's page: its facts, checks, description and activity.
type itemPage struct {
	item     work.Item
	detail   *work.Detail
	err      error
	loading  bool
	vp       viewport.Model
	rendered int // the width the content was rendered for
}

func newServiceView(svc work.Service, st *styles, cache work.Cache, self, file string) *serviceView {
	v := &serviceView{
		svc: svc, st: st, cache: cache, self: self, file: file,
		execute: tea.ExecProcess,
		open:    func(url string) error { return exec.Command("open", url).Run() },
		copy: func(text string) error {
			cmd := exec.Command("pbcopy")
			cmd.Stdin = strings.NewReader(text)
			return cmd.Run()
		},
		now: time.Now,
	}
	if d, ok := cache.Load(svc.Name()); ok {
		v.dash, v.fromCache = &d, true
	}
	v.buildLists()
	return v
}

// buildLists fills one list per view. Lists that exist keep their filter,
// and their selection where the same item still is.
func (v *serviceView) buildLists() tea.Cmd {
	var sections []work.Section
	var followUps []work.Item
	if v.dash != nil {
		sections = v.dash.Sections
		followUps = v.dash.FollowUps()
	}
	views := make([][]work.Item, 0, len(sections)+1)
	views = append(views, followUps)
	for _, s := range sections {
		views = append(views, s.Items)
	}
	if len(v.lists) != len(views) {
		v.lists = make([]list.Model, len(views))
		for i := range views {
			v.lists[i] = v.newList(i == 0)
		}
	}
	var cmds []tea.Cmd
	for i, items := range views {
		l := &v.lists[i]
		var keep string
		if r, ok := l.SelectedItem().(workRow); ok {
			keep = r.ID
		}
		rows := make([]list.Item, len(items))
		selected := -1
		for j, it := range items {
			rows[j] = workRow{it}
			if it.ID == keep {
				selected = j
			}
		}
		cmds = append(cmds, l.SetItems(rows))
		if selected >= 0 && l.FilterState() == list.Unfiltered {
			l.Select(selected)
		}
	}
	if v.view >= len(v.lists) {
		v.view = 0
	}
	v.resize()
	return tea.Batch(cmds...)
}

func (v *serviceView) newList(followUps bool) list.Model {
	l := list.New(nil, workDelegate{v: v, followUps: followUps}, 0, 0)
	l.SetShowTitle(false)
	l.SetFilteringEnabled(true)
	l.SetStatusBarItemName("item", "items")
	l.DisableQuitKeybindings()
	// Page keys that don't clash with the actions.
	l.KeyMap.PrevPage = key.NewBinding(key.WithKeys("pgup", "left"), key.WithHelp("pgup", "prev page"))
	l.KeyMap.NextPage = key.NewBinding(key.WithKeys("pgdown", "right"), key.WithHelp("pgdn", "next page"))
	l.AdditionalShortHelpKeys = v.helpKeys
	l.AdditionalFullHelpKeys = v.helpKeys
	l.Styles = list.DefaultStyles(v.st.isDark)
	return l
}

func (v *serviceView) setSize(w, h int) {
	v.width, v.height = w, h
	v.resize()
}

func (v *serviceView) resize() {
	body := max(1, v.height-serviceTop)
	for i := range v.lists {
		h := body
		if i == 0 {
			h -= v.overviewHeight()
		}
		v.lists[i].SetSize(v.width, max(1, h))
	}
	if v.page != nil {
		v.page.vp.SetWidth(v.width)
		v.page.vp.SetHeight(max(1, body-pageChrome))
		if v.page.rendered != v.width {
			v.renderPage()
		}
	}
}

// check asks the service whether its CLI is there and signed in.
func (v *serviceView) check() tea.Cmd {
	v.checking = true
	svc := v.svc
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), workTimeout)
		defer cancel()
		status, account, err := svc.Check(ctx)
		return checkedMsg{v: v, status: status, account: account, err: err}
	}
}

// load fetches a fresh dashboard, unless one is on its way.
func (v *serviceView) load() tea.Cmd {
	if v.loading || v.status != work.Connected {
		return nil
	}
	v.loading = true
	svc := v.svc
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), workTimeout)
		defer cancel()
		d, err := svc.Dashboard(ctx)
		return dashLoadedMsg{v: v, dash: d, err: err}
	}
}

func (v *serviceView) loadDetail(it work.Item) tea.Cmd {
	svc := v.svc
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), workTimeout)
		defer cancel()
		d, err := svc.Detail(ctx, it)
		return detailLoadedMsg{v: v, id: it.ID, detail: d, err: err}
	}
}

// update handles the view's own messages.
func (v *serviceView) update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case checkedMsg:
		v.checking = false
		v.status, v.account, v.checkErr = msg.status, msg.account, msg.err
		return v.load()
	case dashLoadedMsg:
		v.loading = false
		if msg.err != nil {
			v.dashErr = msg.err
			return nil
		}
		v.dash, v.fromCache, v.dashErr = &msg.dash, false, nil
		v.cache.Save(v.svc.Name(), msg.dash)
		return v.buildLists()
	case detailLoadedMsg:
		if p := v.page; p != nil && p.item.ID == msg.id {
			p.loading = false
			p.err = msg.err
			if msg.err == nil {
				p.detail = &msg.detail
			}
			v.renderPage()
		}
	case actionDoneMsg:
		if msg.err != nil {
			v.setFlash(msg.what+" failed: "+msg.err.Error(), true)
		} else {
			v.setFlash(msg.what+" finished.", false)
		}
		if msg.signIn {
			return v.check()
		}
		return v.refresh()
	case composedMsg:
		return v.onComposed(msg)
	case flashMsg:
		v.setFlash(msg.text, msg.bad)
	}
	return nil
}

// refresh reloads the dashboard, and the open page.
func (v *serviceView) refresh() tea.Cmd {
	cmds := []tea.Cmd{v.load()}
	if v.page != nil {
		v.page.loading = true
		cmds = append(cmds, v.loadDetail(v.page.item))
	}
	return tea.Batch(cmds...)
}

func (v *serviceView) setFlash(text string, bad bool) { v.flash, v.flashBad = text, bad }

// takesKeys reports whether every key belongs to the view: while a filter
// is typed or a question waits for its answer.
func (v *serviceView) takesKeys() bool {
	return v.confirm != nil || v.choose != nil || (v.page == nil && v.list().FilterState() == list.Filtering)
}

func (v *serviceView) list() *list.Model { return &v.lists[v.view] }

// selected is the item a key acts on: the open page's, or the list's.
func (v *serviceView) selected() (work.Item, bool) {
	if v.page != nil {
		return v.page.item, true
	}
	r, ok := v.list().SelectedItem().(workRow)
	return r.Item, ok
}

// key handles a key on the tab.
func (v *serviceView) key(msg tea.KeyPressMsg) tea.Cmd {
	k := msg.String()
	if v.confirm != nil {
		a, it := *v.confirm, v.confirmItem
		v.confirm = nil
		if k == "y" {
			v.flash = ""
			return v.run(it, a)
		}
		v.setFlash("Left alone.", false)
		return nil
	}
	if v.choose != nil {
		a, it := *v.choose, v.chooseItem
		v.choose = nil
		n := int(k[0] - '1')
		if len(k) != 1 || n < 0 || n >= len(a.Choices) {
			v.setFlash("Left alone.", false)
			return nil
		}
		args := make([]string, len(a.Args))
		for i, arg := range a.Args {
			args[i] = strings.ReplaceAll(arg, work.ChoiceArg, a.Choices[n])
		}
		a.Args, a.Choices = args, nil
		return v.start(it, a)
	}
	if v.page == nil && v.list().FilterState() == list.Filtering {
		var cmd tea.Cmd
		*v.list(), cmd = v.list().Update(msg)
		return cmd
	}
	v.flash = ""

	switch v.status {
	case work.Connected:
	case work.NotInstalled:
		if k == "i" {
			args := []string{v.self, "install", "--file", v.file, v.svc.Package()}
			return v.handOff("install", args, "", true, true)
		}
		return nil
	case work.SignedOut:
		if k == "c" {
			return v.handOff("sign-in", v.svc.Connect(), "", true, true)
		}
		return nil
	default:
		if k == "r" && !v.checking {
			return v.check()
		}
		return nil
	}

	if k == "r" {
		return v.refresh()
	}
	if it, ok := v.selected(); ok {
		switch k {
		case "o":
			return v.openURL(it.URL)
		case "y":
			return v.copyURL(it.URL)
		}
		for _, a := range v.svc.Actions(it) {
			if a.Key == k {
				return v.start(it, a)
			}
		}
	}
	if v.page != nil {
		return v.pageKey(msg)
	}
	if n := int(k[0] - '0'); len(k) == 1 && n >= 1 && n <= len(v.lists) {
		v.view = n - 1
		return nil
	}
	if k == "enter" {
		if it, ok := v.selected(); ok {
			return v.openPage(it)
		}
		return nil
	}
	var cmd tea.Cmd
	*v.list(), cmd = v.list().Update(msg)
	return cmd
}

func (v *serviceView) pageKey(msg tea.KeyPressMsg) tea.Cmd {
	switch msg.String() {
	case "esc", "backspace", "left", "h":
		v.page = nil
		return nil
	}
	var cmd tea.Cmd
	v.page.vp, cmd = v.page.vp.Update(msg)
	return cmd
}

func (v *serviceView) openPage(it work.Item) tea.Cmd {
	v.page = &itemPage{item: it, loading: true, vp: viewport.New()}
	v.resize()
	v.renderPage()
	return v.loadDetail(it)
}

// start begins an action: with its choices first, if it has them, then its
// question, if it has one.
func (v *serviceView) start(it work.Item, a work.Action) tea.Cmd {
	switch {
	case len(a.Choices) > 0:
		v.choose, v.chooseItem = &a, it
		return nil
	case a.Confirm != "":
		v.confirm, v.confirmItem = &a, it
		return nil
	}
	return v.run(it, a)
}

// run does an action: asks for a comment first if it needs one, then hands
// the terminal to the command.
func (v *serviceView) run(it work.Item, a work.Action) tea.Cmd {
	if a.Compose {
		return v.compose(it, a)
	}
	return v.handOff(a.Help, a.Args, a.Dir, a.Pause, false)
}

// handOff gives the terminal to args. With pause, it goes through
// `dotui _exec`, which waits for Enter so the output can be read.
func (v *serviceView) handOff(what string, args []string, dir string, pause, signIn bool) tea.Cmd {
	if pause {
		args = append([]string{v.self, "_exec"}, args...)
	}
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir = dir
	return v.execute(cmd, func(err error) tea.Msg { return actionDoneMsg{v: v, what: what, err: err, signIn: signIn} })
}

// compose opens $EDITOR on an empty file for a comment.
func (v *serviceView) compose(it work.Item, a work.Action) tea.Cmd {
	f, err := os.CreateTemp("", "dotui-comment-*.md")
	if err != nil {
		v.setFlash("Couldn't make a file to write the comment in: "+err.Error(), true)
		return nil
	}
	name := f.Name()
	_ = f.Close() // empty; the editor fills it
	editor := strings.Fields(os.Getenv("VISUAL"))
	if len(editor) == 0 {
		editor = strings.Fields(os.Getenv("EDITOR"))
	}
	if len(editor) == 0 {
		editor = []string{"vi"}
	}
	cmd := exec.Command(editor[0], append(editor[1:], name)...)
	return v.execute(cmd, func(err error) tea.Msg { return composedMsg{v: v, item: it, action: a, file: name, err: err} })
}

// onComposed posts what was written, or nothing if it's empty.
func (v *serviceView) onComposed(msg composedMsg) tea.Cmd {
	data, readErr := os.ReadFile(msg.file)
	_ = os.Remove(msg.file) // a temporary file; nothing else uses it
	body := strings.TrimSpace(string(data))
	switch {
	case msg.err != nil:
		v.setFlash("The editor failed: "+msg.err.Error(), true)
		return nil
	case readErr != nil:
		v.setFlash("Couldn't read the comment back: "+readErr.Error(), true)
		return nil
	case body == "":
		v.setFlash("Nothing written, so nothing posted.", false)
		return nil
	}
	args := make([]string, len(msg.action.Args))
	for i, arg := range msg.action.Args {
		args[i] = strings.ReplaceAll(arg, work.BodyArg, body)
	}
	return v.handOff(msg.action.Help, args, msg.action.Dir, msg.action.Pause, false)
}

func (v *serviceView) openURL(url string) tea.Cmd {
	if url == "" {
		v.setFlash("It has no link to open.", true)
		return nil
	}
	return func() tea.Msg {
		if err := v.open(url); err != nil {
			return flashMsg{v: v, text: "Couldn't open it: " + err.Error(), bad: true}
		}
		return flashMsg{v: v, text: "Opened in your browser."}
	}
}

func (v *serviceView) copyURL(url string) tea.Cmd {
	if url == "" {
		v.setFlash("It has no link to copy.", true)
		return nil
	}
	return func() tea.Msg {
		if err := v.copy(url); err != nil {
			return flashMsg{v: v, text: "Couldn't copy it: " + err.Error(), bad: true}
		}
		return flashMsg{v: v, text: "Copied " + url}
	}
}

// showsDash reports whether there's a dashboard to show: a fresh one, or
// the last run's while the CLI is asked who's signed in.
func (v *serviceView) showsDash() bool {
	return v.dash != nil && (v.status == work.Connected || v.status == work.Unknown && v.checkErr == nil)
}

// followUps counts what needs you, for the tab's name.
func (v *serviceView) followUps() int {
	if !v.showsDash() {
		return 0
	}
	return len(v.lists[0].Items())
}

func refreshTick() tea.Cmd {
	return tea.Tick(refreshEvery, func(time.Time) tea.Msg { return refreshMsg{} })
}
