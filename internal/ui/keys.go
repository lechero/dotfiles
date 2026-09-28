package ui

import (
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
)

// keyMap is every binding, so a key means one thing in one place and the help
// footer can never drift from what the handlers do.
type keyMap struct {
	Overview, Map, Explorer, Clean key.Binding
	Worktrees                      key.Binding
	NextTab, PrevTab               key.Binding
	Help, Quit                     key.Binding
	Rescan, Private                key.Binding

	Up, Down, Left, Right       key.Binding
	PageUp, PageDown, Home, End key.Binding

	MapOpen, MapBack, Nested, NextBlock, PrevBlock key.Binding
	ExpOpen, ExpBack, Filter, Reveal               key.Binding
	SpotOpen                                       key.Binding

	Toggle, AllTier1, None, DryRun, Refresh, CleanNow, Details key.Binding
	Yes, No, Stop, Done                                        key.Binding

	WtAll, WtBranches, WtFetch, WtRecheck, WtRemove key.Binding
}

func newKeyMap() keyMap {
	b := func(keys []string, helpKey, desc string) key.Binding {
		return key.NewBinding(key.WithKeys(keys...), key.WithHelp(helpKey, desc))
	}
	return keyMap{
		Overview:  b([]string{"1"}, "1", "overview"),
		Map:       b([]string{"2"}, "2", "map"),
		Explorer:  b([]string{"3"}, "3", "explorer"),
		Clean:     b([]string{"4"}, "4", "clean"),
		Worktrees: b([]string{"5"}, "5", "worktrees"),
		NextTab:   b([]string{"tab"}, "tab", "next tab"),
		PrevTab:   b([]string{"shift+tab"}, "⇧tab", "previous tab"),
		Help:      b([]string{"?"}, "?", "all keys"),
		Quit:      b([]string{"q", "ctrl+c"}, "q", "quit"),
		Rescan:    b([]string{"r"}, "r", "rescan"),
		Private:   b([]string{"p"}, "p", "private folders on/off"),

		Up:       b([]string{"up", "k"}, "↑/k", "up"),
		Down:     b([]string{"down", "j"}, "↓/j", "down"),
		Left:     b([]string{"left", "h"}, "←/h", "left"),
		Right:    b([]string{"right", "l"}, "→/l", "right"),
		PageUp:   b([]string{"pgup", "ctrl+u"}, "pgup", "page up"),
		PageDown: b([]string{"pgdown", "ctrl+d"}, "pgdn", "page down"),
		Home:     b([]string{"home", "g"}, "g", "first"),
		End:      b([]string{"end", "G"}, "G", "last"),

		MapOpen:   b([]string{"enter", " "}, "enter", "zoom in"),
		MapBack:   b([]string{"backspace", "esc", "u"}, "⌫/esc", "zoom out"),
		Nested:    b([]string{"v"}, "v", "nested blocks on/off"),
		NextBlock: b([]string{"]", "n"}, "]", "next block"),
		PrevBlock: b([]string{"[", "N"}, "[", "previous block"),

		ExpOpen: b([]string{"enter", "right", "l"}, "enter/→", "open"),
		ExpBack: b([]string{"backspace", "left", "h", "esc"}, "←/⌫", "up a folder"),
		Filter:  b([]string{"/"}, "/", "filter"),
		Reveal:  b([]string{"o"}, "o", "reveal in Finder"),

		SpotOpen: b([]string{"enter"}, "enter", "open spot in map"),

		Toggle:   b([]string{" ", "x"}, "space", "switch on/off"),
		AllTier1: b([]string{"a"}, "a", "all"),
		None:     b([]string{"n"}, "n", "none"),
		DryRun:   b([]string{"d"}, "d", "dry run on/off"),
		Refresh:  b([]string{"r"}, "r", "measure again"),
		CleanNow: b([]string{"c"}, "c", "clean…"),
		Details:  b([]string{"enter", "right", "l"}, "enter", "items"),

		WtAll:      b([]string{"a"}, "a", "pick everything verified"),
		WtBranches: b([]string{"b"}, "b", "merged branches too on/off"),
		WtFetch:    b([]string{"f"}, "f", "fetch, then check"),
		WtRecheck:  b([]string{"r"}, "r", "check again"),
		WtRemove:   b([]string{"c"}, "c", "remove…"),

		Yes:  b([]string{"y", "enter"}, "y", "go ahead"),
		No:   b([]string{"n", "esc", "q"}, "n/esc", "cancel"),
		Stop: b([]string{"esc", "ctrl+c"}, "esc", "stop after this item"),
		Done: b([]string{"enter", "esc", "q"}, "enter", "back"),
	}
}

// keyHelp is one screen's bindings, in the shape bubbles/help wants.
type keyHelp struct {
	short []key.Binding
	full  [][]key.Binding
}

func (k keyHelp) ShortHelp() []key.Binding  { return k.short }
func (k keyHelp) FullHelp() [][]key.Binding { return k.full }

var _ help.KeyMap = keyHelp{}

// helpKeys is what the footer shows for the screen in front of you.
func (a *app) helpKeys() keyHelp {
	k := a.keys
	tabs := []key.Binding{k.Overview, k.Map, k.Explorer, k.Clean, k.Worktrees, k.NextTab}
	general := []key.Binding{k.Rescan, k.Private, k.Help, k.Quit}
	switch {
	case a.tab == tabWorktrees:
		return a.wtHelpKeys(tabs)
	case a.tab == tabOverview:
		return keyHelp{
			short: []key.Binding{k.Up, k.Down, k.SpotOpen, k.Map, k.Clean, k.Rescan, k.Help, k.Quit},
			full:  [][]key.Binding{{k.Up, k.Down, k.SpotOpen}, tabs, general},
		}
	case a.tab == tabMap:
		move := []key.Binding{k.Up, k.Down, k.Left, k.Right, k.NextBlock, k.PrevBlock}
		return keyHelp{
			short: []key.Binding{k.Left, k.Right, k.MapOpen, k.MapBack, k.Nested, k.Reveal, k.Help, k.Quit},
			full:  [][]key.Binding{move, {k.MapOpen, k.MapBack, k.Nested, k.Reveal}, tabs, general},
		}
	case a.tab == tabExplorer && a.exp.filtering:
		return keyHelp{short: []key.Binding{
			key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "keep filter")),
			key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "clear filter")),
		}}
	case a.tab == tabExplorer:
		move := []key.Binding{k.Up, k.Down, k.PageUp, k.PageDown, k.Home, k.End}
		return keyHelp{
			short: []key.Binding{k.Up, k.Down, k.ExpOpen, k.ExpBack, k.Filter, k.Reveal, k.Help, k.Quit},
			full:  [][]key.Binding{move, {k.ExpOpen, k.ExpBack, k.Filter, k.Reveal}, tabs, general},
		}
	}
	return a.cleanHelpKeys(tabs)
}
