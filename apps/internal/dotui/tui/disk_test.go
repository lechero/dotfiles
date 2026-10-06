package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

// fakeDisk records what the Disk tab hands the analyzer.
type fakeDisk struct {
	inits     int
	got       []tea.Msg
	takesKeys bool
}

func (d *fakeDisk) Init() tea.Cmd { d.inits++; return nil }

func (d *fakeDisk) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	d.got = append(d.got, msg)
	if k, ok := msg.(tea.KeyPressMsg); ok && k.String() == "q" {
		return d, tea.Quit
	}
	return d, nil
}

func (d *fakeDisk) View() tea.View  { return tea.NewView(d.Content()) }
func (d *fakeDisk) Content() string { return "the disk analyzer" }
func (d *fakeDisk) TakesKeys() bool { return d.takesKeys }

// keys lists the keys the analyzer got.
func (d *fakeDisk) keys() []string {
	var out []string
	for _, msg := range d.got {
		if k, ok := msg.(tea.KeyPressMsg); ok {
			out = append(out, k.String())
		}
	}
	return out
}

func diskModel(t *testing.T) (Model, *fakeDisk, *int) {
	t.Helper()
	disk, created := &fakeDisk{}, 0
	m := testModelWith(t, func() Disk { created++; return disk })
	return m, disk, &created
}

func update(m Model, msg tea.Msg) (Model, tea.Cmd) {
	model, cmd := m.Update(msg)
	return model.(Model), cmd
}

var (
	tabKey      = tea.KeyPressMsg{Code: tea.KeyTab}
	shiftTabKey = tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift}
)

func TestDiskTabStartsTheAnalyzerWhenFirstOpened(t *testing.T) {
	m, disk, created := diskModel(t)
	if *created != 0 {
		t.Fatal("the analyzer started before its tab opened")
	}
	if line := ansi.Strip(m.tabsLine()); !strings.Contains(line, "Disk") {
		t.Errorf("tab line is missing Disk: %q", line)
	}

	m = press(t, m, tabKey, tabKey)
	if m.tab != diskTab || *created != 1 || disk.inits != 1 {
		t.Fatalf("two tabs on: tab %d, created %d, inits %d", m.tab, *created, disk.inits)
	}
	if got, want := disk.got[0], (tea.WindowSizeMsg{Width: 100, Height: 29}); got != want {
		t.Errorf("first message %#v, want %#v: the screen below the tab line", got, want)
	}
	v := m.View()
	if !strings.Contains(v.Content, "the disk analyzer") || v.MouseMode != tea.MouseModeCellMotion {
		t.Errorf("disk view: mouse mode %v\n%s", v.MouseMode, v.Content)
	}

	m = press(t, m, tabKey)
	if m.tab != packagesTab || m.View().MouseMode != tea.MouseModeNone {
		t.Errorf("tab from Disk should wrap to Packages with the mouse off, tab %d", m.tab)
	}
	m = press(t, m, shiftTabKey)
	if m.tab != diskTab || *created != 1 || disk.inits != 1 {
		t.Errorf("reopening Disk should reuse the analyzer: tab %d, created %d, inits %d", m.tab, *created, disk.inits)
	}

	m, _ = update(m, tea.WindowSizeMsg{Width: 80, Height: 20})
	if got, want := disk.got[len(disk.got)-1], (tea.WindowSizeMsg{Width: 80, Height: 19}); got != want {
		t.Errorf("resize passed %#v, want %#v", got, want)
	}
}

func TestDiskTabGetsTheKeysAndTheMouse(t *testing.T) {
	m, disk, _ := diskModel(t)
	m = press(t, m, shiftTabKey) // back from Packages wraps to Disk
	m = press(t, m, char('1'), char('r'))
	if got := strings.Join(disk.keys(), " "); got != "1 r" {
		t.Errorf("analyzer got keys %q, want \"1 r\"", got)
	}
	if m.maxPrio != 4 || m.brewLoading {
		t.Errorf("1 and r are the analyzer's on its tab: maxPrio %d, reloading %v", m.maxPrio, m.brewLoading)
	}
	if _, cmd := update(m, char('q')); cmd == nil || cmd() != tea.Quit() {
		t.Error("q should reach the analyzer, which quits")
	}

	m, _ = update(m, tea.MouseClickMsg{X: 7, Y: 5, Button: tea.MouseLeft})
	if click, ok := disk.got[len(disk.got)-1].(tea.MouseClickMsg); !ok || click.X != 7 || click.Y != 4 {
		t.Errorf("click should reach the analyzer one line up, got %#v", disk.got[len(disk.got)-1])
	}

	disk.takesKeys = true
	m = press(t, m, tabKey)
	if m.tab != diskTab || disk.keys()[len(disk.keys())-1] != "tab" {
		t.Error("while the analyzer has the keyboard, tab is its key")
	}
	x := strings.Index(ansi.Strip(m.tabsLine()), "Packages")
	m, _ = update(m, tea.MouseClickMsg{X: x, Y: 0, Button: tea.MouseLeft})
	if m.tab != diskTab {
		t.Error("clicking another tab shouldn't leave a flow the analyzer is in")
	}

	disk.takesKeys = false
	m, _ = update(m, tea.MouseClickMsg{X: x, Y: 0, Button: tea.MouseLeft})
	if m.tab != packagesTab {
		t.Errorf("clicking Packages in the tab line should open it, tab %d", m.tab)
	}
}

type scanProgressMsg struct{}

func TestDiskKeepsWorkingInTheBackground(t *testing.T) {
	m, disk, _ := diskModel(t)
	m = press(t, m, shiftTabKey, tabKey) // open Disk, then back to Packages
	before := len(disk.keys())
	m, _ = update(m, scanProgressMsg{})
	if _, ok := disk.got[len(disk.got)-1].(scanProgressMsg); !ok {
		t.Error("the analyzer should get its messages while another tab is open")
	}
	m = press(t, m, char('1'))
	if len(disk.keys()) != before || m.maxPrio != 1 {
		t.Errorf("on Packages, 1 is a priority filter: analyzer keys %v, maxPrio %d", disk.keys(), m.maxPrio)
	}
}

func TestNoDiskTabWithoutAnAnalyzer(t *testing.T) {
	m := testModel(t)
	if strings.Contains(ansi.Strip(m.tabsLine()), "Disk") {
		t.Error("no analyzer, no Disk tab")
	}
	if m = press(t, m, tabKey, tabKey); m.tab != packagesTab {
		t.Errorf("two tabs should wrap back to Packages, tab %d", m.tab)
	}
}
