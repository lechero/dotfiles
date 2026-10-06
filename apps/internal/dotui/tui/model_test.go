package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/lechero/dotfiles/apps/internal/dotui/brew"
	"github.com/lechero/dotfiles/apps/internal/dotui/catalog"
	"github.com/lechero/dotfiles/apps/internal/dotui/chezmoi"
)

func testModel(t *testing.T) Model {
	t.Helper()
	return testModelWith(t, nil)
}

// testModelWith is testModel with newDisk behind the Disk tab.
func testModelWith(t *testing.T, newDisk func() Disk) Model {
	t.Helper()
	c, err := catalog.Parse([]byte(`
packages:
  - {name: fish, prio: 1, note: Shell}
  - {name: tmux, prio: 1, note: Multiplexer}
  - {name: jq, prio: 2, note: JSON}
  - {name: rancher, cask: true, prio: 2}
  - {name: stow, prio: 4, note: Old}
`))
	if err != nil {
		t.Fatal(err)
	}
	var model tea.Model = New(Config{Catalog: c, File: "packages.yaml", Self: "dotui", NewDisk: newDisk})
	model, _ = model.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	model, _ = model.Update(brewLoadedMsg{inv: brew.Inventory{
		Formulae: map[string]bool{"fish": true, "jq": true},
		Casks:    map[string]bool{"rancher": true},
	}})
	return model.(Model)
}

func press(t *testing.T, m Model, keys ...tea.KeyPressMsg) Model {
	t.Helper()
	var model tea.Model = m
	for _, k := range keys {
		model, _ = model.Update(k)
	}
	return model.(Model)
}

func char(r rune) tea.KeyPressMsg { return tea.KeyPressMsg{Code: r, Text: string(r)} }

func names(m Model) []string {
	var out []string
	for _, it := range m.packages.VisibleItems() {
		out = append(out, it.(pkgItem).pkg.Name)
	}
	return out
}

func TestPriorityAndMissingFilters(t *testing.T) {
	m := testModel(t)
	if got := strings.Join(names(m), " "); got != "fish tmux jq rancher stow" {
		t.Errorf("all = %q", got)
	}
	m = press(t, m, char('1'))
	if got := strings.Join(names(m), " "); got != "fish tmux" {
		t.Errorf("up to 1 = %q", got)
	}
	m = press(t, m, char('4'), char('m'))
	if got := strings.Join(names(m), " "); got != "tmux stow" {
		t.Errorf("missing only = %q", got)
	}
	m = press(t, m, char('m'))
	if got := len(names(m)); got != 5 {
		t.Errorf("after turning missing-only off, %d packages shown, want 5", got)
	}
}

func TestHeaderCounts(t *testing.T) {
	view := strings.Join(strings.Fields(ansi.Strip(testModel(t).View().Content)), " ")
	for _, want := range []string{"P1 core 1/2", "P2 daily 2/2", "P4 rarely 0/1", "All priorities"} {
		if !strings.Contains(view, want) {
			t.Errorf("view is missing %q:\n%s", want, view)
		}
	}
}

func TestInstallOnlyWhatsMissing(t *testing.T) {
	m := testModel(t)
	// fish is installed: no command to run, just a status message.
	if got := m.install([]pkgItem{{pkg: catalog.Package{Name: "fish"}, state: brew.Installed, known: true}}); got == nil {
		t.Error("expected a status message command")
	}
	if !strings.Contains(ansi.Strip(m.packages.View()), "Nothing to install") {
		t.Errorf("no status message shown:\n%s", ansi.Strip(m.packages.View()))
	}
}

func TestSwitchTabShowsChanges(t *testing.T) {
	m := testModel(t)
	var model tea.Model = press(t, m, tea.KeyPressMsg{Code: tea.KeyTab})
	model, _ = model.Update(chezmoiLoadedMsg{changes: []chezmoi.Change{
		{Path: ".config/fish/config.fish", Local: ' ', Apply: 'M'},
		{Path: ".zshrc", Local: 'M', Apply: 'M'},
	}})
	view := ansi.Strip(model.(Model).View().Content)
	for _, want := range []string{"2 changes to apply", "1 changed outside chezmoi", ".config/fish/config.fish", "will be updated"} {
		if !strings.Contains(view, want) {
			t.Errorf("view is missing %q:\n%s", want, view)
		}
	}
}

func TestQuitKeysExceptWhileFiltering(t *testing.T) {
	m := testModel(t)
	if _, cmd := m.Update(char('q')); cmd == nil || cmd() != tea.Quit() {
		t.Error("q didn't quit")
	}
	// Start a filter: now q is text, not quit.
	m = press(t, m, char('/'))
	if _, cmd := m.Update(char('q')); cmd != nil && cmd() == tea.Quit() {
		t.Error("q quit while typing a filter")
	}
}
