package tui

import "charm.land/bubbles/v2/key"

type keyMap struct {
	Quit       key.Binding
	SwitchTab  key.Binding
	Refresh    key.Binding
	Prio       key.Binding
	Missing    key.Binding
	Install    key.Binding
	InstallAll key.Binding
	Diff       key.Binding
	Apply      key.Binding
	Update     key.Binding
}

func newKeyMap() keyMap {
	return keyMap{
		Quit:       key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
		SwitchTab:  key.NewBinding(key.WithKeys("tab", "shift+tab"), key.WithHelp("tab/⇧tab", "switch view")),
		Refresh:    key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh")),
		Prio:       key.NewBinding(key.WithKeys("1", "2", "3", "4"), key.WithHelp("1-4", "priorities up to")),
		Missing:    key.NewBinding(key.WithKeys("m"), key.WithHelp("m", "missing only")),
		Install:    key.NewBinding(key.WithKeys("i"), key.WithHelp("i", "install")),
		InstallAll: key.NewBinding(key.WithKeys("I"), key.WithHelp("I", "install all shown")),
		Diff:       key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "diff")),
		Apply:      key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "apply")),
		Update:     key.NewBinding(key.WithKeys("u"), key.WithHelp("u", "pull and apply")),
	}
}

func (k keyMap) packageKeys() []key.Binding {
	return []key.Binding{k.Prio, k.Missing, k.Install, k.InstallAll, k.Refresh, k.SwitchTab, k.Quit}
}

func (k keyMap) dotfileKeys() []key.Binding {
	return []key.Binding{k.Diff, k.Apply, k.Update, k.Refresh, k.SwitchTab, k.Quit}
}
