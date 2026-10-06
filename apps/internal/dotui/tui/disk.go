package tui

import (
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
)

// Disk is the analyzer on the Disk tab: manage-disk's interface, drawn below
// dotui's tab line.
type Disk interface {
	tea.Model
	// Content draws it as text, in the size of the last tea.WindowSizeMsg.
	Content() string
	// TakesKeys reports whether it wants every key, tab included: while it
	// confirms or runs something, or while a text field has the keyboard.
	TakesKeys() bool
}

// diskTop is the screen line the Disk tab's content starts on, below the
// tab line.
const diskTop = 1

// openDisk starts the analyzer the first time the Disk tab opens, so its
// scan of the home folder only runs when someone wants to see it.
func (m *Model) openDisk() tea.Cmd {
	if m.disk != nil || m.cfg.NewDisk == nil {
		return nil
	}
	m.disk = m.cfg.NewDisk()
	return tea.Batch(m.disk.Init(), m.updateDisk(m.diskSize()))
}

// diskSize is the room the analyzer gets: the screen below the tab line.
func (m Model) diskSize() tea.WindowSizeMsg {
	return tea.WindowSizeMsg{Width: m.width, Height: max(0, m.height-diskTop)}
}

// updateDisk passes msg to the analyzer, once it runs.
func (m *Model) updateDisk(msg tea.Msg) tea.Cmd {
	if m.disk == nil {
		return nil
	}
	model, cmd := m.disk.Update(msg)
	m.disk = model.(Disk)
	return cmd
}

// diskKey handles a key on the Disk tab. tab switches views, unless the
// analyzer has the keyboard; every other key is the analyzer's, q and ctrl+c
// included, so it can stop its scan before quitting.
func (m *Model) diskKey(msg tea.KeyPressMsg) tea.Cmd {
	if m.disk != nil && !m.disk.TakesKeys() && key.Matches(msg, m.keys.SwitchTab) {
		return m.switchTab(msg.String() == "shift+tab")
	}
	return m.updateDisk(msg)
}

// diskMouse handles the mouse on the Disk tab: a click on a tab name opens
// that tab, and the rest goes to the analyzer, moved up past the tab line.
func (m *Model) diskMouse(msg tea.MouseMsg) tea.Cmd {
	if msg.Mouse().Y >= diskTop {
		return m.updateDisk(shiftMouse(msg, diskTop))
	}
	click, ok := msg.(tea.MouseClickMsg)
	if !ok || click.Button != tea.MouseLeft || (m.disk != nil && m.disk.TakesKeys()) {
		return nil
	}
	if t, ok := m.tabAt(click.X); ok {
		return m.setTab(t)
	}
	return nil
}

// shiftMouse moves a mouse event dy lines up.
func shiftMouse(msg tea.MouseMsg, dy int) tea.Msg {
	switch msg := msg.(type) {
	case tea.MouseClickMsg:
		msg.Y -= dy
		return msg
	case tea.MouseReleaseMsg:
		msg.Y -= dy
		return msg
	case tea.MouseWheelMsg:
		msg.Y -= dy
		return msg
	case tea.MouseMotionMsg:
		msg.Y -= dy
		return msg
	}
	return msg
}
