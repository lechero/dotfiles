package ui

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/lechero/dotfiles/apps/internal/disk/human"
	"github.com/lechero/dotfiles/apps/internal/disk/scan"
)

type explorerState struct {
	path   string // folder on screen; kept as a path so rescans and deletions can't strand it
	cursor int
	offset int
	page   int
	top    int // screen row of the first list row, for mouse hits

	filtering bool // the filter field has the keyboard
	filter    textinput.Model
}

func newExplorerState(home string) explorerState {
	ti := textinput.New()
	ti.Prompt = "/ "
	ti.Placeholder = "filter by name"
	ti.CharLimit = 80
	return explorerState{path: home, filter: ti}
}

// current returns the folder on screen, or its nearest ancestor still in the tree.
func (e *explorerState) current(root *scan.Node) *scan.Node {
	for p := e.path; ; p = filepath.Dir(p) {
		if n := root.Find(p); n != nil && n.IsDir {
			e.path = p
			return n
		}
		if p == root.Name || !strings.HasPrefix(p, root.Name) || p == "/" {
			e.path = root.Name
			return root
		}
	}
}

// rows are the folder's entries that pass the filter.
func (e *explorerState) rows(n *scan.Node) []*scan.Node {
	q := strings.ToLower(e.filter.Value())
	if q == "" {
		return n.Children
	}
	var out []*scan.Node
	for _, c := range n.Children {
		if strings.Contains(strings.ToLower(c.Name), q) {
			out = append(out, c)
		}
	}
	return out
}

// rowCount includes the "… smaller items" line when no filter hides it.
func (e *explorerState) rowCount(n *scan.Node) int {
	rows := len(e.rows(n))
	if n.SmallN > 0 && e.filter.Value() == "" {
		rows++
	}
	return rows
}

func (e *explorerState) selectName(n *scan.Node, name string) {
	for i, c := range e.rows(n) {
		if c.Name == name {
			e.cursor = i
			return
		}
	}
}

func (e *explorerState) enter(path string) {
	e.path, e.cursor, e.offset = path, 0, 0
	e.filter.SetValue("")
}

func (a *app) explorerKey(msg tea.KeyMsg) tea.Cmd {
	e := &a.exp
	if e.filtering {
		switch msg.Type {
		case tea.KeyEsc:
			e.filtering = false
			e.filter.SetValue("")
			e.filter.Blur()
			return nil
		case tea.KeyEnter:
			e.filtering = false
			e.filter.Blur()
			return nil
		}
		var cmd tea.Cmd
		e.filter, cmd = e.filter.Update(msg)
		e.cursor, e.offset = 0, 0
		return cmd
	}
	if a.res == nil {
		return nil
	}
	k := a.keys
	n := e.current(a.res.Root)
	rows := e.rows(n)
	page := max(1, e.page)
	switch {
	case key.Matches(msg, k.Filter):
		e.filtering = true
		return e.filter.Focus()
	case key.Matches(msg, k.Up):
		e.cursor--
	case key.Matches(msg, k.Down):
		e.cursor++
	case key.Matches(msg, k.PageUp):
		e.cursor -= page
	case key.Matches(msg, k.PageDown):
		e.cursor += page
	case key.Matches(msg, k.Home):
		e.cursor = 0
	case key.Matches(msg, k.End):
		e.cursor = e.rowCount(n) - 1
	case key.Matches(msg, k.ExpOpen):
		a.explorerOpen(n, rows)
	case msg.Type == tea.KeyEsc && e.filter.Value() != "":
		e.filter.SetValue("") // esc first clears a kept filter, then goes up
	case key.Matches(msg, k.ExpBack):
		if n.Parent != nil {
			from := n.Name
			e.enter(n.Parent.Path())
			e.selectName(n.Parent, from)
		}
	case key.Matches(msg, k.Reveal):
		target := n.Path()
		if e.cursor < len(rows) {
			target = rows[e.cursor].Path()
		}
		return reveal(target)
	}
	e.cursor = max(0, min(e.cursor, e.rowCount(n)-1))
	return nil
}

func (a *app) explorerOpen(n *scan.Node, rows []*scan.Node) {
	e := &a.exp
	if e.cursor >= len(rows) {
		return
	}
	c := rows[e.cursor]
	switch {
	case c.Skipped != "":
		a.flash = c.Name + " was not scanned: " + c.Skipped
	case c.IsDir && len(c.Children) > 0:
		e.enter(c.Path())
	case c.IsDir:
		a.flash = fmt.Sprintf("%s holds only small items (%s in %s)", c.Name, human.Bytes(c.Small), human.Count(c.SmallN))
	}
}

// explorerMouse: wheel scrolls, a click selects, clicking the selection opens it.
func (a *app) explorerMouse(msg tea.MouseMsg) tea.Cmd {
	if a.res == nil || msg.Action != tea.MouseActionPress {
		return nil
	}
	e := &a.exp
	n := e.current(a.res.Root)
	switch msg.Button {
	case tea.MouseButtonWheelUp:
		e.cursor = max(0, e.cursor-3)
	case tea.MouseButtonWheelDown:
		e.cursor = min(e.rowCount(n)-1, e.cursor+3)
	case tea.MouseButtonRight:
		if n.Parent != nil {
			from := n.Name
			e.enter(n.Parent.Path())
			e.selectName(n.Parent, from)
		}
	case tea.MouseButtonLeft:
		row := e.offset + msg.Y - e.top
		if row < 0 || row >= e.rowCount(n) {
			return nil
		}
		if row == e.cursor {
			a.explorerOpen(n, e.rows(n))
		} else {
			e.cursor = row
		}
	}
	return nil
}

func (a *app) explorerView(h int) string {
	if a.res == nil {
		return a.scanStatus() + "\n\n" + sDim.Render("The explorer opens when the first scan finishes.")
	}
	e := &a.exp
	n := e.current(a.res.Root)
	rows := e.rows(n)
	head := sHeading.Render(tildePath(a.home, n.Path())) +
		sDim.Render(fmt.Sprintf("  %s · %s files", human.Bytes(n.Size), human.Count(n.Files)))
	if a.scanning {
		head += "  " + a.spin.View() + sDim.Render(" rescanning")
	}
	second := ""
	if e.filtering || e.filter.Value() != "" {
		e.filter.Width = max(10, a.w/3)
		second = e.filter.View() + sDim.Render(fmt.Sprintf("  %d of %d", len(rows), len(n.Children)))
	}

	listH := max(1, h-2)
	e.page = listH
	e.top = a.headerH + 2
	total := e.rowCount(n)
	e.cursor = max(0, min(e.cursor, total-1))
	if e.cursor < e.offset {
		e.offset = e.cursor
	}
	if e.cursor >= e.offset+listH {
		e.offset = e.cursor - listH + 1
	}

	tags := a.cleanTags()
	nameW := max(10, a.w-36)
	var b strings.Builder
	b.WriteString(head + "\n" + second + "\n")
	for i := e.offset; i < min(total, e.offset+listH); i++ {
		var size int64
		var name string
		hue := -1
		if i < len(rows) {
			c := rows[i]
			size, name = c.Size, c.Name
			if c.IsDir {
				name += "/"
			}
			name = truncRight(name, nameW)
			if i == e.cursor {
				name = sCursor.Render(name)
			}
			switch {
			case c.Skipped != "":
				name += sDim.Render("  not scanned: " + c.Skipped)
			case tags[c.Path()] != "":
				name += "  " + sGreen.Render(tags[c.Path()])
			}
			hue = indexOf(n.Children, c)
		} else {
			size = n.Small
			name = sDim.Render(fmt.Sprintf("… %s smaller items", human.Count(n.SmallN)))
		}
		frac := float64(size) / float64(max(1, n.Size))
		marker := "  "
		if i == e.cursor {
			marker = sCursor.Render("› ")
		}
		fmt.Fprintf(&b, "%s%s %s %s  %s\n", marker, padLeft(human.Bytes(size), 9),
			sDim.Render(fmt.Sprintf("%5.1f%%", frac*100)), bar(frac, 12, barStyle(hue)), name)
	}
	if total == 0 {
		b.WriteString(sDim.Render("  (nothing here matches)") + "\n")
	}
	return b.String()
}

func indexOf(nodes []*scan.Node, x *scan.Node) int {
	for i, n := range nodes {
		if n == x {
			return i
		}
	}
	return -1
}
