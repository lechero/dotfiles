package tui

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/lechero/dotfiles/apps/internal/dotui/catalog"
	"github.com/lechero/dotfiles/apps/internal/dotui/work"
)

var testNow = time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC)

// fakeService is a service whose answers the test sets.
type fakeService struct {
	status   work.Status
	checkErr error
	dash     work.Dashboard
	detail   work.Detail
	actions  []work.Action
	loads    int
	details  []string
}

func (f *fakeService) Name() string      { return "Forge" }
func (f *fakeService) CLI() string       { return "forge" }
func (f *fakeService) Package() string   { return "forge-cli" }
func (f *fakeService) Connect() []string { return []string{"forge", "login"} }
func (f *fakeService) Check(context.Context) (work.Status, string, error) {
	return f.status, "me on forge.dev", f.checkErr
}
func (f *fakeService) Dashboard(context.Context) (work.Dashboard, error) {
	f.loads++
	return f.dash, nil
}
func (f *fakeService) Detail(_ context.Context, it work.Item) (work.Detail, error) {
	f.details = append(f.details, it.ID)
	return f.detail, nil
}
func (f *fakeService) Actions(work.Item) []work.Action { return f.actions }

func forgeDash() work.Dashboard {
	return work.Dashboard{
		Account: "me on forge.dev",
		Fetched: testNow.Add(-time.Minute),
		Sections: []work.Section{
			{Key: "Reviews", Title: "Waiting for you", Total: 1, Items: []work.Item{
				{Kind: work.PullRequest, ID: "team/svc#9", Ref: "#9", Title: "Please look", Where: "team/svc", Author: "pat",
					URL: "https://forge.dev/team/svc/pull/9", Updated: testNow.Add(-2 * time.Hour),
					Next: "review requested", NextTone: work.Warn, Urgency: 90},
			}},
			{Key: "PRs", Title: "Yours", Total: 12, Items: []work.Item{
				{Kind: work.PullRequest, ID: "me/app#1", Ref: "#1", Title: "Red build", Where: "me/app", Author: "me",
					Updated: testNow.Add(-time.Hour), Next: "checks failing", NextTone: work.Bad, Urgency: 80,
					Badges: []work.Badge{{Text: "checks failing", Tone: work.Bad}, {Text: "approved", Tone: work.Good}}},
				{Kind: work.PullRequest, ID: "me/app#2", Ref: "#2", Title: "Quiet", Where: "me/app", Author: "me", Updated: testNow},
			}},
		},
	}
}

// executed records what the tab handed the terminal to.
type executed struct{ cmds []*exec.Cmd }

func (e *executed) execute(cmd *exec.Cmd, done tea.ExecCallback) tea.Cmd {
	e.cmds = append(e.cmds, cmd)
	return func() tea.Msg { return done(nil) }
}

func (e *executed) last() []string {
	if len(e.cmds) == 0 {
		return nil
	}
	return e.cmds[len(e.cmds)-1].Args
}

// serviceModel is dotui with only the fake service's tab after its own, in
// front, with the service checked and its dashboard loaded.
func serviceModel(t *testing.T, svc *fakeService) (Model, *serviceView, *executed) {
	t.Helper()
	c, err := catalog.Parse([]byte("packages:\n  - {name: fish, prio: 1}\n"))
	if err != nil {
		t.Fatal(err)
	}
	m := New(Config{Catalog: c, File: "packages.yaml", Self: "/bin/dotui", Services: []work.Service{svc}})
	v := m.tabs[len(m.tabs)-1].service
	ex := &executed{}
	v.execute = ex.execute
	v.now = func() time.Time { return testNow }
	m, _ = update(m, tea.WindowSizeMsg{Width: 120, Height: 40})
	m = press(t, m, shiftTabKey) // the last tab
	m = run(t, m, v.check())
	return m, v, ex
}

// run runs cmd and feeds what it sends back into m, as Bubble Tea would.
func run(t *testing.T, m Model, cmd tea.Cmd) Model {
	t.Helper()
	queue := []tea.Cmd{cmd}
	for steps := 0; len(queue) > 0; steps++ {
		if steps > 100 {
			t.Fatal("run: too many steps")
		}
		c := queue[0]
		queue = queue[1:]
		if c == nil {
			continue
		}
		switch msg := c().(type) {
		case nil:
		case tea.BatchMsg:
			queue = append(queue, msg...)
		default:
			var next tea.Cmd
			m, next = update(m, msg)
			queue = append(queue, next)
		}
	}
	return m
}

// keys presses keys and runs what they start.
func keys(t *testing.T, m Model, keys ...tea.KeyPressMsg) Model {
	t.Helper()
	for _, k := range keys {
		var cmd tea.Cmd
		m, cmd = update(m, k)
		m = run(t, m, cmd)
	}
	return m
}

func screen(m Model) string { return ansi.Strip(m.View().Content) }

func TestServiceDashboard(t *testing.T) {
	svc := &fakeService{status: work.Connected, dash: forgeDash()}
	m, _, _ := serviceModel(t, svc)
	out := screen(m)
	for _, want := range []string{
		"Forge 2", // the tab's name says how much needs you
		"● me on forge.dev · updated 1 minute ago",
		"1 Follow-ups 2", "2 Reviews 1", "3 PRs 12",
		"Reviews  1  1 review requested",
		"PRs     12  1 checks failing",
		"#9 Please look", "review requested · team/svc · by pat",
		"#1 Red build", "checks failing · me/app · approved", // the reason isn't repeated as a badge
	} {
		if !strings.Contains(out, want) {
			t.Errorf("dashboard is missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "Quiet") {
		t.Error("an item that needs nothing isn't a follow-up")
	}

	m = keys(t, m, char('3'))
	if out := screen(m); !strings.Contains(out, "Quiet") || strings.Contains(out, "Please look") {
		t.Errorf("3 should show the PRs:\n%s", out)
	}
}

func TestServiceItemPage(t *testing.T) {
	svc := &fakeService{status: work.Connected, dash: forgeDash(), detail: work.Detail{
		Fields:   []work.Field{{Label: "Branch", Value: "fix → main"}},
		Checks:   []work.Check{{Name: "unit", State: "failure", Tone: work.Bad}, {Name: "lint", State: "success", Tone: work.Good}},
		Body:     "Fixes the **crash**.",
		Activity: []work.Note{{Author: "pat", When: testNow.Add(-time.Hour), What: "approved"}},
	}, actions: []work.Action{{Key: "d", Help: "diff", Args: []string{"forge", "diff", "9"}, Pause: true}}}
	m, v, ex := serviceModel(t, svc)

	m = keys(t, m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if v.page == nil || !slices.Equal(svc.details, []string{"team/svc#9"}) {
		t.Fatalf("enter should open the first follow-up's page and load it: details %v", svc.details)
	}
	out := screen(m)
	for _, want := range []string{
		"#9 Please look", "team/svc · by pat · updated 2 hours ago",
		"Branch  fix → main", "Checks  1 failing · 1 passing", "✗ unit", "✓ lint",
		"Description", "crash", "pat approved · 1 hour ago",
		"d diff · o browser · y copy link · r refresh · esc back",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("page is missing %q:\n%s", want, out)
		}
	}

	m = keys(t, m, char('d'))
	if want := []string{"/bin/dotui", "_exec", "forge", "diff", "9"}; !slices.Equal(ex.last(), want) {
		t.Errorf("d ran %q, want %q: through dotui, which waits for Enter", ex.last(), want)
	}
	if svc.loads != 2 || len(svc.details) != 2 {
		t.Errorf("after an action the dashboard and page reload: loads %d, details %v", svc.loads, svc.details)
	}

	m = keys(t, m, tea.KeyPressMsg{Code: tea.KeyEscape})
	if v.page != nil {
		t.Error("esc should close the page")
	}
	if !strings.Contains(screen(m), "Follow up") {
		t.Error("closing the page goes back to the dashboard")
	}
}

func TestServiceAsksBeforeActionsThatDontAsk(t *testing.T) {
	svc := &fakeService{status: work.Connected, dash: forgeDash(), actions: []work.Action{
		{Key: "v", Help: "approve", Args: []string{"forge", "approve", "9"}, Confirm: "Approve #9?", Pause: true},
	}}
	m, v, ex := serviceModel(t, svc)

	m = keys(t, m, char('v'))
	if !strings.Contains(screen(m), "Approve #9? y/n") || len(ex.cmds) != 0 {
		t.Fatalf("v should ask first:\n%s", screen(m))
	}
	m = keys(t, m, char('n'))
	if len(ex.cmds) != 0 || !strings.Contains(screen(m), "Left alone.") {
		t.Errorf("n should leave it alone, ran %v", ex.last())
	}
	m = keys(t, m, char('v'), char('y'))
	if want := []string{"/bin/dotui", "_exec", "forge", "approve", "9"}; !slices.Equal(ex.last(), want) {
		t.Errorf("y ran %q, want %q", ex.last(), want)
	}
	if !strings.Contains(screen(m), "approve finished.") {
		t.Errorf("the result should show:\n%s", screen(m))
	}
	if v.confirm != nil {
		t.Error("the question should be gone")
	}
}

func TestServiceComposesAComment(t *testing.T) {
	editor := filepath.Join(t.TempDir(), "editor")
	script := "#!/bin/sh\nprintf 'Looks good to me.\\n' > \"$1\"\n"
	if err := os.WriteFile(editor, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("VISUAL", editor)
	svc := &fakeService{status: work.Connected, dash: forgeDash(), actions: []work.Action{
		{Key: "C", Help: "comment", Args: []string{"forge", "comment", "9", "--body", work.BodyArg}, Compose: true, Pause: true},
	}}
	m, v, ex := serviceModel(t, svc)
	// The editor really runs; the comment command is only recorded.
	v.execute = func(cmd *exec.Cmd, done tea.ExecCallback) tea.Cmd {
		if cmd.Path == editor {
			err := cmd.Run()
			return func() tea.Msg { return done(err) }
		}
		return ex.execute(cmd, done)
	}

	keys(t, m, char('C'))
	if want := []string{"/bin/dotui", "_exec", "forge", "comment", "9", "--body", "Looks good to me."}; !slices.Equal(ex.last(), want) {
		t.Errorf("posted with %q, want %q", ex.last(), want)
	}

	if err := os.WriteFile(editor, []byte("#!/bin/sh\n: > \"$1\"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	ex.cmds = nil
	m = keys(t, m, char('C'))
	if len(ex.cmds) != 0 || !strings.Contains(screen(m), "Nothing written, so nothing posted.") {
		t.Errorf("an empty comment shouldn't be posted: ran %v\n%s", ex.last(), screen(m))
	}
}

func TestServiceSignInAndInstall(t *testing.T) {
	svc := &fakeService{status: work.SignedOut, dash: forgeDash()}
	m, v, ex := serviceModel(t, svc)
	if out := screen(m); !strings.Contains(out, "forge isn't signed in to Forge.") || !strings.Contains(out, "forge login") {
		t.Errorf("signed out:\n%s", out)
	}
	if strings.Contains(ansi.Strip(m.tabsLine()), "Forge 2") {
		t.Error("a signed-out tab shouldn't count follow-ups")
	}
	svc.status = work.Connected // the sign-in works
	m = keys(t, m, char('c'))
	if want := []string{"/bin/dotui", "_exec", "forge", "login"}; !slices.Equal(ex.cmds[0].Args, want) {
		t.Errorf("c ran %q, want %q", ex.cmds[0].Args, want)
	}
	if v.status != work.Connected || !strings.Contains(screen(m), "Follow up") {
		t.Errorf("after signing in it checks again and loads:\n%s", screen(m))
	}

	svc2 := &fakeService{status: work.NotInstalled}
	m, _, ex = serviceModel(t, svc2)
	if !strings.Contains(screen(m), "which isn't installed") {
		t.Errorf("not installed:\n%s", screen(m))
	}
	keys(t, m, char('i'))
	if want := []string{"/bin/dotui", "_exec", "/bin/dotui", "install", "--file", "packages.yaml", "forge-cli"}; !slices.Equal(ex.last(), want) {
		t.Errorf("i ran %q, want %q", ex.last(), want)
	}
}

func TestServiceShowsTheLastRunWhileChecking(t *testing.T) {
	cache := work.Cache{Dir: t.TempDir()}
	cache.Save("Forge", forgeDash())
	c, _ := catalog.Parse([]byte("packages:\n  - {name: fish, prio: 1}\n"))
	m := New(Config{Catalog: c, Services: []work.Service{&fakeService{}}, Cache: cache})
	m.tabs[len(m.tabs)-1].service.now = func() time.Time { return testNow }
	m, _ = update(m, tea.WindowSizeMsg{Width: 120, Height: 40})
	m = press(t, m, shiftTabKey)
	out := screen(m)
	for _, want := range []string{"asking forge…", "showing the last run's, from 1 minute ago", "Please look"} {
		if !strings.Contains(out, want) {
			t.Errorf("before the check, the cached dashboard should show %q:\n%s", want, out)
		}
	}
}

func TestServiceTabKeys(t *testing.T) {
	svc := &fakeService{status: work.Connected, dash: forgeDash()}
	m, v, _ := serviceModel(t, svc)
	if _, cmd := update(m, char('q')); cmd == nil || cmd() != tea.Quit() {
		t.Error("q should quit")
	}
	m = keys(t, m, char('/'), char('q'))
	if !v.takesKeys() {
		t.Fatal("/ should start a filter")
	}
	if _, cmd := update(m, char('x')); cmd != nil && cmd() == tea.Quit() {
		t.Error("letters type into the filter")
	}
	m = keys(t, m, tea.KeyPressMsg{Code: tea.KeyEscape}, tabKey)
	if m.kind() != packagesTab {
		t.Errorf("tab from the last tab should wrap to Packages, got kind %d", m.kind())
	}
}

func TestServiceRefreshKeepsTheSelection(t *testing.T) {
	svc := &fakeService{status: work.Connected, dash: forgeDash()}
	m, v, _ := serviceModel(t, svc)
	m = keys(t, m, char('3'), tea.KeyPressMsg{Code: tea.KeyDown})
	if it, _ := v.selected(); it.ID != "me/app#2" {
		t.Fatalf("selected %q", it.ID)
	}
	// A new PR arrives at the top.
	d := forgeDash()
	d.Sections[1].Items = append([]work.Item{{ID: "me/app#3", Ref: "#3", Title: "New"}}, d.Sections[1].Items...)
	svc.dash = d
	keys(t, m, char('r'))
	if it, _ := v.selected(); it.ID != "me/app#2" {
		t.Errorf("after a refresh the selection should stay on me/app#2, got %q", it.ID)
	}
}

func TestServicePicksAChoiceFirst(t *testing.T) {
	svc := &fakeService{status: work.Connected, dash: forgeDash(), actions: []work.Action{
		{Key: "t", Help: "move", Choices: []string{"To Do", "In Progress", "Done"}, Pause: true,
			Args: []string{"forge", "move", "--status", work.ChoiceArg}},
	}}
	m, v, ex := serviceModel(t, svc)

	m = keys(t, m, char('t'))
	if out := screen(m); !strings.Contains(out, "Move #9 to: 1 To Do · 2 In Progress · 3 Done") || !v.takesKeys() {
		t.Fatalf("t should offer the statuses:\n%s", out)
	}
	m = keys(t, m, char('7'))
	if len(ex.cmds) != 0 || !strings.Contains(screen(m), "Left alone.") {
		t.Errorf("a number with no choice should cancel, ran %v", ex.last())
	}
	keys(t, m, char('t'), char('2'))
	if want := []string{"/bin/dotui", "_exec", "forge", "move", "--status", "In Progress"}; !slices.Equal(ex.last(), want) {
		t.Errorf("2 ran %q, want %q", ex.last(), want)
	}
}
