package ui

import (
	"context"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/lechero/dotfiles/apps/internal/disk/clean"
	"github.com/lechero/dotfiles/apps/internal/disk/scan"
)

func mapTree(home string) *scan.Node {
	return node(home, 100<<20,
		node("big", 60<<20, node("x", 40<<20), node("y", 20<<20)),
		node("mid", 30<<20, node("z", 30<<20)),
		&scan.Node{Name: "file.bin", Size: 10 << 20, Files: 1},
	)
}

func click(a *app, x, y int, b tea.MouseButton) tea.Cmd {
	if b == tea.MouseWheelUp || b == tea.MouseWheelDown {
		return a.update(tea.MouseWheelMsg{X: x, Y: y, Button: b})
	}
	return a.update(tea.MouseClickMsg{X: x, Y: y, Button: b})
}

func TestMapShowsBlocksMovesAndZooms(t *testing.T) {
	a := testApp(t, Options{})
	h := a.home
	giveTree(a, mapTree(h))
	press(t, a, "2")
	out := screen(a)
	for _, want := range []string{"big/", "mid/", "file.bin", "▸ big/", "60.0 MiB"} {
		if !strings.Contains(out, want) {
			t.Errorf("map is missing %q", want)
		}
	}
	if a.mp.sel != "big" {
		t.Fatalf("the largest block should start selected, got %q", a.mp.sel)
	}
	press(t, a, "right")
	if a.mp.sel == "big" {
		t.Error("→ should move to a block on the right")
	}
	press(t, a, "left")
	if a.mp.sel != "big" {
		t.Errorf("← should come back to big, got %q", a.mp.sel)
	}

	press(t, a, "enter")
	if a.exp.path != h+"/big" {
		t.Fatalf("enter should zoom into big, path %s", a.exp.path)
	}
	if out := screen(a); !strings.Contains(out, "~ › big") {
		t.Errorf("breadcrumb should show where we are:\n%s", out)
	}
	press(t, a, "esc")
	if a.exp.path != h || a.mp.sel != "big" {
		t.Errorf("esc should zoom out onto big: path %s sel %q", a.exp.path, a.mp.sel)
	}

	a.mp.sel = "file.bin"
	press(t, a, "enter")
	if a.exp.path != h || a.flash == "" {
		t.Error("a file cannot be zoomed into; say so instead")
	}
}

func TestMapMouseSelectsThenZooms(t *testing.T) {
	a := testApp(t, Options{})
	giveTree(a, mapTree(a.home))
	press(t, a, "2")
	screen(a)
	var mid mapBlock
	for _, b := range a.mp.blocks {
		if b.name == "mid" {
			mid = b
		}
	}
	x, y := mid.cell.X+1, a.mp.top+mid.cell.Y+1
	drive(t, a, click(a, x, y, tea.MouseLeft))
	if a.mp.sel != "mid" {
		t.Fatalf("a click should select mid, got %q", a.mp.sel)
	}
	drive(t, a, click(a, x, y, tea.MouseLeft))
	if a.exp.path != a.home+"/mid" {
		t.Errorf("clicking the selection should zoom in, path %s", a.exp.path)
	}
	drive(t, a, click(a, 0, a.mp.top, tea.MouseRight))
	if a.exp.path != a.home {
		t.Errorf("right-click should zoom out, path %s", a.exp.path)
	}
}

func TestTabsAreClickable(t *testing.T) {
	a := testApp(t, Options{})
	screen(a)
	hit := a.tabHits[tabExplorer]
	drive(t, a, click(a, hit.x0+1, 1, tea.MouseLeft))
	if a.tab != tabExplorer {
		t.Errorf("clicking the Explorer tab should open it, tab = %d", a.tab)
	}
}

func TestExplorerFilter(t *testing.T) {
	a := testApp(t, Options{})
	giveTree(a, mapTree(a.home))
	press(t, a, "3", "/", "f", "i")
	if !a.exp.filtering || a.exp.filter.Value() != "fi" {
		t.Fatalf("typing after / should fill the filter, got %q", a.exp.filter.Value())
	}
	press(t, a, "q") // a letter, not quit, while typing
	if a.exp.filter.Value() != "fiq" {
		t.Fatalf("q should type while filtering, got %q", a.exp.filter.Value())
	}
	press(t, a, "backspace", "enter")
	rows := a.exp.rows(a.exp.current(a.res.Root))
	if len(rows) != 1 || rows[0].Name != "file.bin" || a.exp.filtering {
		t.Errorf("filter fi should keep file.bin only, got %d rows, filtering %v", len(rows), a.exp.filtering)
	}
	if out := screen(a); !strings.Contains(out, "1 of 3") {
		t.Errorf("the filter line should count matches:\n%s", out)
	}
	press(t, a, "esc")
	if a.exp.filter.Value() != "" {
		t.Error("esc should clear a kept filter first")
	}
}

func TestSwitchingMapAndExplorerKeepsTheSelection(t *testing.T) {
	a := testApp(t, Options{})
	giveTree(a, mapTree(a.home))
	press(t, a, "3", "down") // explorer: select mid
	press(t, a, "2")
	if a.mp.sel != "mid" {
		t.Errorf("the map should select what the explorer had, got %q", a.mp.sel)
	}
	a.mp.sel = "file.bin"
	press(t, a, "3")
	if rows := a.exp.rows(a.exp.current(a.res.Root)); rows[a.exp.cursor].Name != "file.bin" {
		t.Errorf("the explorer should select what the map had, got %s", rows[a.exp.cursor].Name)
	}
}

func TestSpotOpensInTheMap(t *testing.T) {
	a := testApp(t, Options{})
	h := a.home
	giveTree(a, node(h, 30<<30,
		node("projects", 20<<30, node("app", 20<<30, node("a", 8<<30), node("b", 7<<30), node("c", 5<<30))),
		node("Library", 10<<30, node("Caches", 10<<30, &scan.Node{Name: "huge.bin", Size: 10 << 30, Files: 1})),
	))
	screen(a)
	press(t, a, "down", "enter") // second spot: huge.bin
	if a.tab != tabMap || a.exp.path != h+"/Library/Caches" || a.mp.sel != "huge.bin" {
		t.Errorf("enter should open the spot in the map: tab %d path %s sel %q", a.tab, a.exp.path, a.mp.sel)
	}
}

func threeItems(context.Context, *clean.Env) ([]clean.Item, error) {
	return []clean.Item{
		{Paths: []string{"/x/alpha"}, Label: "alpha", Size: 3 << 20, Selected: true},
		{Paths: []string{"/x/beta"}, Label: "beta", Size: 2 << 20, Selected: true},
		{Paths: []string{"/x/gamma"}, Label: "gamma", Size: 1 << 20, Selected: true},
	}, nil
}

func TestItemListTogglesAndFilters(t *testing.T) {
	a := testApp(t, Options{})
	a.tasks = []*taskState{{task: &clean.Task{ID: "t", Title: "Three", Tier: clean.Tier1, Discover: threeItems}, enabled: true}}
	drive(t, a, a.discoverAll())
	press(t, a, "4", "enter")
	if a.cl.mode != modeDetail || len(a.cl.items.Items()) != 3 {
		t.Fatalf("enter should open the item list: mode %d, %d items", a.cl.mode, len(a.cl.items.Items()))
	}
	press(t, a, " ")
	if a.tasks[0].items[0].Selected {
		t.Error("space should switch the first item off")
	}
	press(t, a, "/", "b", "e", "t", "enter")
	if n := len(a.cl.items.VisibleItems()); n != 1 {
		t.Fatalf("filtering bet should leave beta alone, got %d", n)
	}
	if out := screen(a); !strings.Contains(out, "beta") || strings.Contains(out, "gamma") {
		t.Errorf("filtered list:\n%s", out)
	}
	press(t, a, "esc")
	if a.cl.items.IsFiltered() || a.cl.mode != modeDetail {
		t.Error("esc should clear the filter first")
	}
	press(t, a, "esc")
	if a.cl.mode != modeList {
		t.Error("a second esc should go back to the task list")
	}
}

func TestFreeSpaceTrend(t *testing.T) {
	dir := t.TempDir()
	t0 := time.Date(2026, 9, 28, 9, 0, 0, 0, time.UTC)
	recordFree(dir, 60<<30, t0)
	s := recordFree(dir, 61<<30, t0.Add(time.Minute)) // too soon: updates the last point
	if len(s) != 1 || s[0].Free != 61<<30 {
		t.Fatalf("a reading within 10 minutes should replace the last one: %+v", s)
	}
	recordFree(dir, 80<<30, t0.Add(time.Hour))
	s = recordFree(dir, 107<<30, t0.Add(2*time.Hour))
	if len(s) != 3 || len(loadFree(dir)) != 3 {
		t.Fatalf("want 3 samples, got %d", len(s))
	}
	if got := sparkline(s, 10); got != "▁▃█" {
		t.Errorf("sparkline = %q", got)
	}
}

func TestEmbeddedLeavesTabAndItsNameToTheHost(t *testing.T) {
	a := testApp(t, Options{Embedded: true})
	m := Model{a}
	if out := ansi.Strip(m.Content()); strings.Contains(out, "manage-disk") {
		t.Errorf("an embedded view shouldn't name itself:\n%s", out)
	}
	press(t, a, "tab")
	if a.tab != tabOverview {
		t.Errorf("tab belongs to the host when embedded, but it moved to tab %d", a.tab)
	}
	if strings.Contains(screen(a), "next tab") {
		t.Error("the help shouldn't offer tab when embedded")
	}

	giveTree(a, mapTree(a.home))
	if m.TakesKeys() {
		t.Error("nothing has the keyboard yet")
	}
	press(t, a, "3", "/")
	if !m.TakesKeys() {
		t.Error("the explorer's filter field has the keyboard")
	}
}
