package ui

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"

	"manage-disk/internal/clean"
	"manage-disk/internal/scan"
)

func testApp(t *testing.T, opts Options) *app {
	t.Helper()
	home := t.TempDir()
	env := clean.NewEnv(home)
	env.SnapshotFn = func(context.Context) (clean.Procs, error) { return nil, nil }
	env.LookPath = func(string) (string, error) { return "", errors.New("not in tests") }
	a := New(env, opts).a
	a.update(tea.WindowSizeMsg{Width: 120, Height: 40})
	return a
}

func node(name string, size int64, kids ...*scan.Node) *scan.Node {
	n := &scan.Node{Name: name, IsDir: len(kids) > 0, Size: size, Children: kids, Files: 1}
	for _, k := range kids {
		k.Parent = n
	}
	return n
}

// giveTree hands the app a finished scan, as if Scan had returned it.
func giveTree(a *app, root *scan.Node, private ...string) {
	a.update(scanDoneMsg{gen: a.scanGen, res: &scan.Result{Root: root, Took: time.Second, Private: private}})
}

func key(s string) tea.Msg {
	switch s {
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEscape}
	case "left":
		return tea.KeyMsg{Type: tea.KeyLeft}
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
	case " ":
		return tea.KeyMsg{Type: tea.KeySpace}
	}
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

// drive runs cmd and feeds what it produces back into the app, like the
// Bubble Tea runtime would — minus timers, which would loop forever.
func drive(t *testing.T, a *app, cmd tea.Cmd) {
	t.Helper()
	queue := []tea.Cmd{cmd}
	for steps := 0; len(queue) > 0; steps++ {
		if steps > 500 {
			t.Fatal("drive: too many steps")
		}
		c := queue[0]
		queue = queue[1:]
		if c == nil {
			continue
		}
		switch msg := c().(type) {
		case nil, tickMsg, spinner.TickMsg:
		case tea.BatchMsg:
			queue = append(queue, msg...)
		default:
			queue = append(queue, a.update(msg))
		}
	}
}

func press(t *testing.T, a *app, keys ...string) {
	t.Helper()
	for _, k := range keys {
		drive(t, a, a.update(key(k)))
	}
}

func TestOverviewShowsCategoriesHotspotsAndPrivacy(t *testing.T) {
	a := testApp(t, Options{})
	h := a.home
	containers := &scan.Node{Name: "Containers", IsDir: true, Skipped: scan.SkipPrivacy}
	root := node(h, 30<<30,
		node("projects", 20<<30, node("app", 20<<30, node("a", 8<<30), node("b", 7<<30), node("c", 5<<30))),
		node("Library", 10<<30, node("Caches", 10<<30, &scan.Node{Name: "huge.bin", Size: 10 << 30, Files: 1}), containers),
	)
	giveTree(a, root, "Desktop", "Downloads", "Library/Containers", "Library/Group Containers", "Library/Mail")
	out := a.view()
	for _, want := range []string{"Where your space goes", "Projects & worktrees", "App & tool caches",
		"Docker VM*", "not scanned, see * below", "Biggest spots", "~/projects/app", "huge.bin",
		"Desktop, Downloads, app containers (Docker's VM disk too), 1 more in Library", "Cleanable now"} {
		if !strings.Contains(out, want) {
			t.Errorf("overview is missing %q", want)
		}
	}
	if strings.Contains(out, "Library/Mail") {
		t.Error("places only Full Disk Access opens should not be offered")
	}
}

func TestExplorerDrillsInAndOut(t *testing.T) {
	a := testApp(t, Options{})
	h := a.home
	giveTree(a, node(h, 30<<20,
		node("big", 20<<20, node("inner", 20<<20, node("deep", 20<<20))),
		&scan.Node{Name: "file.bin", Size: 10 << 20, Files: 1},
	))
	press(t, a, "2")
	if !strings.Contains(a.view(), "big/") {
		t.Fatal("explorer should list the root's children")
	}
	press(t, a, "enter")
	if a.exp.path != h+"/big" {
		t.Fatalf("path = %s after enter", a.exp.path)
	}
	press(t, a, "left")
	if a.exp.path != h || a.exp.cursor != 0 {
		t.Errorf("back up should land on big: path=%s cursor=%d", a.exp.path, a.exp.cursor)
	}
	press(t, a, "down", "enter") // a file: nothing to open
	if a.exp.path != h {
		t.Errorf("entering a file should not navigate, path=%s", a.exp.path)
	}

	// The folder on screen vanishes (deleted by a clean): fall back to its parent.
	a.exp.path = h + "/big/inner/deep"
	a.env.EditTree(func(root *scan.Node) { root.Remove(h + "/big/inner") })
	a.view()
	if a.exp.path != h+"/big" {
		t.Errorf("explorer should fall back to the nearest folder left, got %s", a.exp.path)
	}
}

func fakeTask(dir string) *clean.Task {
	return &clean.Task{
		ID: "fake", Title: "Fake cache", Tier: clean.Tier1, About: "test",
		Discover: func(ctx context.Context, env *clean.Env) ([]clean.Item, error) {
			if _, err := os.Stat(dir); err != nil {
				return nil, nil
			}
			return []clean.Item{{Paths: []string{dir}, Label: "cache", Size: 2 << 20, Selected: true}}, nil
		},
	}
}

func cleanFlow(t *testing.T, dryRun bool) (*app, string) {
	t.Helper()
	a := testApp(t, Options{DryRun: dryRun})
	dir := filepath.Join(a.home, ".cache", "fake")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(dir+"/blob", bytes.Repeat([]byte{1}, 2<<20), 0o644)
	a.tasks = []*taskState{{task: fakeTask(dir), enabled: true}}
	root := node(a.home, 2<<20, node(".cache", 2<<20, node("fake", 2<<20)))
	giveTree(a, root)

	press(t, a, "3", "c") // re-checks, then asks
	if a.cl.mode != modeConfirm {
		t.Fatalf("after c the mode is %d, want confirm", a.cl.mode)
	}
	if !strings.Contains(a.view(), " 1 item, about") { // "Clean 1 item, about…" or "Dry run: 1 item, about…"
		t.Errorf("confirm should count the item:\n%s", a.view())
	}
	press(t, a, "y")
	if a.cl.mode != modeDone || a.cl.summary == nil {
		t.Fatalf("run did not finish: mode %d", a.cl.mode)
	}
	return a, dir
}

func TestCleanFlowDeletesAndUpdatesTheTree(t *testing.T) {
	a, dir := cleanFlow(t, false)
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatal("the cache should be deleted")
	}
	if a.res.Root.Find(dir) != nil || a.res.Root.Size != 0 {
		t.Errorf("the tree should drop the deleted folder, root size %d", a.res.Root.Size)
	}
	if out := a.view(); !strings.Contains(out, "Done in") || !strings.Contains(out, "1 cleaned") {
		t.Errorf("done view:\n%s", out)
	}
	if a.last == nil {
		t.Error("a real run should land in history")
	}
	press(t, a, "enter")
	if a.cl.mode != modeList {
		t.Error("any key should leave the summary")
	}
}

func TestDryRunDeletesNothing(t *testing.T) {
	a, dir := cleanFlow(t, true)
	if _, err := os.Stat(dir); err != nil {
		t.Fatal("a dry run deleted the cache")
	}
	if a.res.Root.Find(dir) == nil {
		t.Error("a dry run must not touch the tree")
	}
	if out := a.view(); !strings.Contains(out, "Dry run finished") {
		t.Errorf("done view:\n%s", out)
	}
}

func TestBlockedItemsNeverRun(t *testing.T) {
	a := testApp(t, Options{})
	dir := filepath.Join(a.home, ".cache", "busy")
	os.MkdirAll(dir, 0o755)
	task := &clean.Task{
		ID: "busy", Title: "Busy", Tier: clean.Tier1,
		Discover: func(context.Context, *clean.Env) ([]clean.Item, error) {
			return []clean.Item{{Paths: []string{dir}, Label: "busy", Size: 1 << 20, Selected: true, Blocked: "Chrome is running"}}, nil
		},
	}
	a.tasks = []*taskState{{task: task, enabled: true}}
	press(t, a, "3", "c")
	if a.cl.mode != modeList || !strings.Contains(a.flash, "Nothing to clean") {
		t.Errorf("mode %d flash %q: a blocked-only selection must not reach confirm", a.cl.mode, a.flash)
	}
	if _, err := os.Stat(dir); err != nil {
		t.Error("blocked folder was deleted")
	}
}
