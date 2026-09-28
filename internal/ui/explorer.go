package ui

import (
	"fmt"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"manage-disk/internal/human"
	"manage-disk/internal/scan"
)

type explorerState struct {
	path   string // folder on screen; kept as a path so rescans and deletions can't strand it
	cursor int
	offset int
	page   int
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

func explorerRows(n *scan.Node) int {
	rows := len(n.Children)
	if n.SmallN > 0 {
		rows++
	}
	return rows
}

func (a *app) explorerKey(key string) tea.Cmd {
	switch key {
	case "r":
		return tea.Batch(a.startScan(), a.discoverAll())
	case "p":
		return a.overviewKey("p")
	}
	if a.res == nil {
		return nil
	}
	e := &a.exp
	n := e.current(a.res.Root)
	page := max(1, e.page)
	switch key {
	case "up", "k":
		e.cursor--
	case "down", "j":
		e.cursor++
	case "pgup", "ctrl+u":
		e.cursor -= page
	case "pgdown", "ctrl+d":
		e.cursor += page
	case "home", "g":
		e.cursor = 0
	case "end", "G":
		e.cursor = explorerRows(n) - 1
	case "enter", "right", "l":
		if e.cursor < len(n.Children) {
			c := n.Children[e.cursor]
			switch {
			case c.Skipped != "":
				a.flash = c.Name + " was not scanned: " + c.Skipped
			case c.IsDir && len(c.Children) > 0:
				e.path, e.cursor, e.offset = c.Path(), 0, 0
			case c.IsDir:
				a.flash = fmt.Sprintf("%s holds only small items (%s in %s)", c.Name, human.Bytes(c.Small), human.Count(c.SmallN))
			}
		}
	case "left", "h", "backspace", "esc":
		if n.Parent != nil {
			from := n
			e.path, e.cursor, e.offset = n.Parent.Path(), 0, 0
			for i, c := range n.Parent.Children {
				if c == from {
					e.cursor = i
				}
			}
		}
	case "o":
		target := n.Path()
		if e.cursor < len(n.Children) {
			target = n.Children[e.cursor].Path()
		}
		return reveal(target)
	}
	e.cursor = max(0, min(e.cursor, explorerRows(n)-1))
	return nil
}

func (a *app) explorerView(h int) string {
	if a.res == nil {
		return a.scanStatus() + "\n\n" + sDim.Render("The explorer opens when the first scan finishes.")
	}
	e := &a.exp
	n := e.current(a.res.Root)
	head := sHeading.Render(tildePath(a.home, n.Path())) +
		sDim.Render(fmt.Sprintf("  %s · %s files", human.Bytes(n.Size), human.Count(n.Files)))
	if a.scanning {
		head += "  " + a.spin.View() + sDim.Render(" rescanning")
	}
	listH := max(1, h-2)
	e.page = listH
	rows := explorerRows(n)
	e.cursor = max(0, min(e.cursor, rows-1))
	if e.cursor < e.offset {
		e.offset = e.cursor
	}
	if e.cursor >= e.offset+listH {
		e.offset = e.cursor - listH + 1
	}

	tags := a.cleanTags()
	nameW := max(10, a.w-36)
	var b strings.Builder
	b.WriteString(head + "\n\n")
	for i := e.offset; i < min(rows, e.offset+listH); i++ {
		var size int64
		var name string
		if i < len(n.Children) {
			c := n.Children[i]
			size = c.Size
			name = c.Name
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
			sDim.Render(fmt.Sprintf("%5.1f%%", frac*100)), bar(frac, 12, sAccent), name)
	}
	if rows == 0 {
		b.WriteString(sDim.Render("  (empty)") + "\n")
	}
	return b.String()
}
