package tui

import (
	"context"
	"slices"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/lechero/dotfiles/apps/internal/dotui/brew"
	"github.com/lechero/dotfiles/apps/internal/dotui/catalog"
	"github.com/lechero/dotfiles/apps/internal/dotui/chezmoi"
	"github.com/lechero/dotfiles/apps/internal/dotui/machine"
	"github.com/lechero/dotfiles/apps/internal/dotui/work"
)

// dashboardModel is dotui on its dashboard, with every reader answering
// and one service, everything loaded.
func dashboardModel(t *testing.T) Model {
	t.Helper()
	c, err := catalog.Parse([]byte(`
packages:
  - {name: fish, prio: 1}
  - {name: tmux, prio: 1}
  - {name: jq, prio: 2}
  - {name: stow, prio: 3}
`))
	if err != nil {
		t.Fatal(err)
	}
	const gib = 1 << 30
	m := New(Config{
		Catalog: c, File: "packages.yaml", Self: "/bin/dotui",
		NewDisk:  func() Disk { return &fakeDisk{} },
		Services: []work.Service{&fakeService{status: work.Connected, dash: forgeDash()}},
		Outdated: func(context.Context) ([]brew.Update, error) {
			return []brew.Update{{Name: "glab"}, {Name: "node"}}, nil
		},
		SourceState: func(context.Context, string) (chezmoi.Source, error) {
			return chezmoi.Source{Branch: "main", Head: "177c1e6", Subject: "fix: something", When: testNow.Add(-2 * time.Hour),
				Dirty: 1, Remote: chezmoi.RemoteNewer}, nil
		},
		Machine: func(context.Context) machine.Info {
			return machine.Info{Name: "Test Mac", User: "Miguel Fuentes", MacOS: "26.5", Booted: testNow.Add(-72 * time.Hour),
				Load: 2.5, CPUs: 8, Battery: 15}
		},
		DiskStatus: func(context.Context) (DiskStatus, error) {
			return DiskStatus{Free: 25 * gib, Total: 100 * gib, LastClean: testNow.Add(-48 * time.Hour), Freed: gib,
				TrendAt: []time.Time{testNow.Add(-3 * 24 * time.Hour), testNow.Add(-24 * time.Hour), testNow},
				Trend:   []int64{27 * gib, 26 * gib, 25 * gib}}, nil
		},
	})
	m.dash.now = func() time.Time { return testNow }
	v := m.services()[0]
	v.now = m.dash.now
	m, _ = update(m, tea.WindowSizeMsg{Width: 130, Height: 50})
	m, _ = update(m, brewLoadedMsg{inv: brew.Inventory{Formulae: map[string]bool{"fish": true, "jq": true}}})
	m, _ = update(m, chezmoiLoadedMsg{changes: []chezmoi.Change{
		{Path: ".config/fish/config.fish", Local: ' ', Apply: 'M'},
		{Path: ".zshrc", Local: 'M', Apply: 'M'},
	}})
	m = run(t, m, m.loadDashboard())
	return run(t, m, v.check())
}

func TestDashboardIsFirstAndSumsItAllUp(t *testing.T) {
	m := dashboardModel(t)
	if m.kind() != dashboardTab {
		t.Fatalf("dotui should open on the dashboard, got kind %d", m.kind())
	}
	out := ansi.Strip(m.View().Content)
	for _, want := range []string{
		"Good afternoon, Miguel · Wednesday 7 October",
		"Test Mac · macOS 26.5 · up 3 days · load 2.5 on 8 cores · battery 15%",
		// Cards, numbered with the key that opens their tab.
		"Packages  2", "P1 1/2 · P2 1/1 · P3 0/1 · P4 —", "✗ 1 missing: tmux · i installs", "↑ 2 updates · U upgrades", "glab, node",
		"Dotfiles  3", "2 to apply · 1 changed outside chezmoi", "main · 177c1e6 · 2 hours ago",
		"↓ origin is ahead · 1 uncommitted",
		"Disk  4", "25.0 GiB free", "of 100 GiB · 75% full", "↓ 2.0 GiB since 3 days ago", "Last clean 2 days ago, freed 1.0 GiB",
		"Forge  5", "● me on forge.dev", "1 Reviews · 12 PRs", "1 review requested · 1 checks failing",
		// Every service's follow-ups, most urgent first.
		"Next up across Forge",
		"Forge   #9 Please look",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("dashboard is missing %q:\n%s", want, out)
		}
	}
	if strings.Index(out, "#9 Please look") > strings.Index(out, "#1 Red build") {
		t.Error("the review should come before the failing checks")
	}
}

func TestDashboardKeys(t *testing.T) {
	m := dashboardModel(t)
	if m = keys(t, m, char('3')); m.kind() != dotfilesTab {
		t.Errorf("3 should open the Dotfiles tab, got kind %d", m.kind())
	}

	m = goTo(t, m, dashboardTab)
	m = keys(t, m, tea.KeyPressMsg{Code: tea.KeyDown}, tea.KeyPressMsg{Code: tea.KeyEnter})
	v := m.services()[0]
	if m.kind() != serviceTab || v.page == nil || v.page.item.ID != "me/app#1" {
		t.Errorf("enter should open the second follow-up on its tab: kind %d, page %v", m.kind(), v.page)
	}

	m = goTo(t, m, dashboardTab)
	if _, cmd := update(m, char('i')); cmd == nil {
		t.Error("i should install what's missing")
	}
	if _, cmd := update(m, char('u')); cmd == nil {
		t.Error("u should pull and apply the dotfiles")
	}
	if _, cmd := update(m, char('q')); cmd == nil || cmd() != tea.Quit() {
		t.Error("q should quit")
	}
}

func TestDashboardWithoutReaders(t *testing.T) {
	m := testModelWith(t, nil)
	m = goTo(t, m, dashboardTab)
	out := ansi.Strip(m.View().Content)
	for _, want := range []string{"Reading this Mac…", "Packages  2", "Checking for updates…", "No services to follow up."} {
		if !strings.Contains(out, want) {
			t.Errorf("before anything is read, the dashboard should say %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "Disk  4") {
		t.Error("without a Disk tab, there's no Disk card")
	}
}

func TestDashboardUpgradeAsksFirst(t *testing.T) {
	m := dashboardModel(t)
	m.dash.updates = append(m.dash.updates, brew.Update{Name: "python@3.11", Pinned: true})
	ex := &executed{}
	m.execute = ex.execute

	m = keys(t, m, char('U'))
	out := ansi.Strip(m.View().Content)
	if !strings.Contains(out, "Upgrade 2 packages with brew upgrade? glab, node y/n") {
		t.Fatalf("U should ask first, without the pinned package:\n%s", out)
	}
	m, _ = update(m, char('q'))
	if m.dash.confirm != nil || len(ex.cmds) != 0 {
		t.Error("any key but y should leave it alone, q too")
	}
	if !strings.Contains(ansi.Strip(m.View().Content), "Left alone.") {
		t.Error("it should say it left it alone")
	}

	m = keys(t, m, char('U'))
	// Only the hand-off is checked: what it starts afterwards reloads brew.
	update(m, char('y'))
	if want := []string{"/bin/dotui", "_exec", "brew", "upgrade"}; !slices.Equal(ex.last(), want) {
		t.Errorf("y ran %q, want %q: through dotui, which waits for Enter", ex.last(), want)
	}
}

func TestDashboardNothingToUpgrade(t *testing.T) {
	m := dashboardModel(t)
	m.dash.updates = nil
	m = keys(t, m, char('U'))
	if m.dash.confirm != nil || !strings.Contains(ansi.Strip(m.View().Content), "Nothing to upgrade.") {
		t.Errorf("with nothing to upgrade, U should say so:\n%s", ansi.Strip(m.View().Content))
	}
}
