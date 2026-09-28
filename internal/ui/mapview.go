package ui

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"

	"manage-disk/internal/human"
	"manage-disk/internal/scan"
	"manage-disk/internal/treemap"
)

// mapState is the treemap tab. It shares the folder on screen with the
// explorer (a.exp.path), so both tabs always look at the same place.
type mapState struct {
	sel    string // name of the selected block; names survive a relayout, indexes don't
	nested bool   // draw each big block's own contents inside it
	blocks []mapBlock
	top    int // screen row the map starts on, for mouse hits
}

type mapBlock struct {
	node *scan.Node // nil for the "smaller items" block
	name string
	size int64
	cell treemap.Cell
	hue  int // -1 for grey
}

// mapItems picks what gets its own block: the biggest entries, while each
// still earns a visible share. The rest become one "smaller items" block.
func mapItems(n *scan.Node, limit int) (shown []*scan.Node, rest, restN int64) {
	rest, restN = n.Small, n.SmallN
	for i, c := range n.Children {
		switch {
		case c.Size <= 0:
		case i < limit && float64(c.Size) >= 0.003*float64(n.Size):
			shown = append(shown, c)
		default:
			rest += c.Size
			restN++
		}
	}
	return shown, rest, restN
}

func layoutValues(shown []*scan.Node, rest int64) []float64 {
	values := make([]float64, 0, len(shown)+1)
	for _, c := range shown {
		values = append(values, float64(c.Size))
	}
	if rest > 0 {
		values = append(values, float64(rest))
	}
	return values
}

func (a *app) layoutMap(n *scan.Node, w, h int) []mapBlock {
	shown, rest, restN := mapItems(n, 48)
	cells := treemap.Layout(layoutValues(shown, rest), 0, 0, w, h)
	blocks := make([]mapBlock, 0, len(cells))
	for i, c := range shown {
		blocks = append(blocks, mapBlock{node: c, name: c.Name, size: c.Size, cell: cells[i], hue: i})
	}
	if rest > 0 {
		blocks = append(blocks, mapBlock{name: "… " + human.Count(restN) + " smaller", size: rest, cell: cells[len(shown)], hue: -1})
	}
	return blocks
}

func (m *mapState) selected() int {
	for i, b := range m.blocks {
		if b.name == m.sel && !b.cell.Empty() {
			return i
		}
	}
	for i, b := range m.blocks {
		if !b.cell.Empty() {
			m.sel = b.name
			return i
		}
	}
	return -1
}

// move selects the nearest block in direction (dx, dy), measured between
// block centres in square units (a cell is about twice as tall as wide).
func (m *mapState) move(dx, dy int) {
	cur := m.selected()
	if cur < 0 {
		return
	}
	cx, cy := center(m.blocks[cur].cell)
	best, bestScore := -1, math.Inf(1)
	for i, b := range m.blocks {
		if i == cur || b.cell.Empty() {
			continue
		}
		bx, by := center(b.cell)
		ddx, ddy := bx-cx, by-cy
		var along, across float64
		if dx != 0 {
			along, across = ddx*float64(dx), math.Abs(ddy)
		} else {
			along, across = ddy*float64(dy), math.Abs(ddx)
		}
		if along <= 0 {
			continue
		}
		if score := along + 2*across; score < bestScore {
			best, bestScore = i, score
		}
	}
	if best >= 0 {
		m.sel = m.blocks[best].name
	}
}

func center(c treemap.Cell) (x, y float64) {
	return float64(c.X) + float64(c.W)/2, (float64(c.Y) + float64(c.H)/2) * 2
}

// step moves the selection to the next or previous block in size order.
func (m *mapState) step(delta int) {
	cur := m.selected()
	if cur < 0 {
		return
	}
	for i := 1; i <= len(m.blocks); i++ {
		j := ((cur+delta*i)%len(m.blocks) + len(m.blocks)) % len(m.blocks)
		if !m.blocks[j].cell.Empty() {
			m.sel = m.blocks[j].name
			return
		}
	}
}

func (a *app) mapKey(msg tea.KeyMsg) tea.Cmd {
	if a.res == nil {
		return nil
	}
	k, m := a.keys, &a.mp
	switch {
	case key.Matches(msg, k.Left):
		m.move(-1, 0)
	case key.Matches(msg, k.Right):
		m.move(1, 0)
	case key.Matches(msg, k.Up):
		m.move(0, -1)
	case key.Matches(msg, k.Down):
		m.move(0, 1)
	case key.Matches(msg, k.NextBlock):
		m.step(1)
	case key.Matches(msg, k.PrevBlock):
		m.step(-1)
	case key.Matches(msg, k.MapOpen):
		a.mapZoomIn()
	case key.Matches(msg, k.MapBack):
		a.mapZoomOut()
	case key.Matches(msg, k.Nested):
		m.nested = !m.nested
	case key.Matches(msg, k.Reveal):
		if i := m.selected(); i >= 0 && m.blocks[i].node != nil {
			return reveal(m.blocks[i].node.Path())
		}
	}
	return nil
}

func (a *app) mapZoomIn() {
	m := &a.mp
	i := m.selected()
	if i < 0 {
		return
	}
	b := m.blocks[i]
	switch {
	case b.node == nil:
		a.flash = "These are the folder's small entries, lumped together; the explorer lists them."
	case b.node.Skipped != "":
		a.flash = b.name + " was not scanned: " + b.node.Skipped
	case !b.node.IsDir || len(b.node.Children) == 0:
		a.flash = b.name + " has nothing big enough inside to map."
	default:
		a.exp.path, m.sel = b.node.Path(), ""
		a.exp.cursor, a.exp.offset = 0, 0
	}
}

func (a *app) mapZoomOut() {
	n := a.exp.current(a.res.Root)
	if n.Parent == nil {
		return
	}
	a.exp.path, a.mp.sel = n.Parent.Path(), n.Name
}

// mapMouse: click selects, clicking the selection zooms in, right-click or
// the wheel going up zooms out.
func (a *app) mapMouse(msg tea.MouseMsg) tea.Cmd {
	if a.res == nil || msg.Action != tea.MouseActionPress {
		return nil
	}
	m := &a.mp
	switch msg.Button {
	case tea.MouseButtonLeft:
		for _, b := range m.blocks {
			if b.cell.Contains(msg.X, msg.Y-m.top) {
				if b.name == m.sel {
					a.mapZoomIn()
				} else {
					m.sel = b.name
				}
				return nil
			}
		}
	case tea.MouseButtonRight, tea.MouseButtonWheelUp:
		a.mapZoomOut()
	case tea.MouseButtonWheelDown:
		m.step(1)
	}
	return nil
}

func (a *app) mapView(h int) string {
	if a.res == nil {
		return a.scanStatus() + "\n\n" + sDim.Render("The map draws once the first scan finishes.")
	}
	n := a.exp.current(a.res.Root)
	mapH := max(3, h-3)
	m := &a.mp
	m.blocks = a.layoutMap(n, a.w, mapH)
	m.top = a.headerH + 1
	sel := m.selected()

	c := newCanvas(a.w, mapH)
	tags := a.cleanTags()
	for i, b := range m.blocks {
		a.drawBlock(c, b, i == sel, tags)
	}
	return a.breadcrumb(n) + "\n" + c.String() + "\n" + a.mapInfo(n, sel, tags)
}

func (a *app) breadcrumb(n *scan.Node) string {
	parts := []string{"~"}
	if rel := n.Rel(); rel != "" {
		parts = append(parts, strings.Split(rel, "/")...)
	}
	crumb := strings.Join(parts, " › ")
	right := sDim.Render(fmt.Sprintf("  %s · %s files", human.Bytes(n.Size), human.Count(n.Files)))
	if a.mp.nested {
		right += sDim.Render(" · nested")
	}
	if a.scanning {
		right += "  " + a.spin.View() + sDim.Render(" rescanning")
	}
	room := max(10, a.w-lipgloss.Width(right))
	return sHeading.Render(truncLeft(crumb, room)) + right
}

func (a *app) mapInfo(n *scan.Node, sel int, tags map[string]string) string {
	if sel < 0 {
		return sDim.Render("(empty)") + "\n"
	}
	b := a.mp.blocks[sel]
	name := b.name
	var files int64
	var path string
	if b.node != nil {
		files, path = b.node.Files, b.node.Path()
		if b.node.IsDir {
			name += "/"
		}
	}
	share := 100 * float64(b.size) / float64(max(1, n.Size))
	line := sCursor.Render("▸ "+name) + sDim.Render(fmt.Sprintf("  %s · %.1f%% of this folder", human.Bytes(b.size), share))
	if files > 0 {
		line += sDim.Render(" · " + human.Count(files) + " files")
	}
	if t := tags[path]; t != "" {
		line += "  " + sGreen.Render(t)
	}
	where := ""
	if path != "" {
		where = sDim.Render(truncLeft(tildePath(a.home, path), a.w))
	}
	return lipgloss.NewStyle().MaxWidth(a.w).Render(line) + "\n" + where
}

// drawBlock paints one block, leaving a one-cell gap on its right and bottom
// so neighbours read as separate tiles. Big folders show their own contents
// as a second level when nested view is on.
func (a *app) drawBlock(c *canvas, b mapBlock, selected bool, tags map[string]string) {
	body := b.cell
	if body.W >= 2 {
		body.W--
	}
	if body.H >= 2 {
		body.H--
	}
	if body.Empty() {
		return
	}
	sh := shadeFor(b.hue)
	base := sh.base
	if selected {
		base = sh.hi
	}
	c.fill(body, c.style(base, blockText, false))

	label := b.name
	if b.node != nil && b.node.IsDir {
		label += "/"
	}
	if b.node != nil && tags[b.node.Path()] != "" {
		label += " ♻"
	}
	if selected {
		label = "▸ " + label
	}
	canNest := a.mp.nested && b.node != nil && b.node.IsDir && len(b.node.Children) > 0 && body.W >= 14 && body.H >= 6
	if body.W >= 3 {
		head := " " + label
		if canNest || body.H == 1 {
			head += "  " + human.Bytes(b.size)
		}
		c.text(body.X, body.Y, head, body.W, c.style(base, blockText, true))
		if !canNest && body.H >= 2 && body.W >= 6 {
			c.text(body.X, body.Y+1, " "+human.Bytes(b.size), body.W, c.style(base, blockSubtext, false))
		}
	}
	if !canNest {
		return
	}
	inner := treemap.Cell{X: body.X + 1, Y: body.Y + 1, W: body.W - 1, H: body.H - 1}
	kids, rest, restN := mapItems(b.node, 24)
	cells := treemap.Layout(layoutValues(kids, rest), inner.X, inner.Y, inner.W, inner.H)
	for j, kc := range cells {
		if kc.W >= 2 {
			kc.W--
		}
		if kc.H >= 2 {
			kc.H--
		}
		if kc.Empty() {
			continue
		}
		tone, name, size := sh.a, "", rest
		if j%2 == 1 {
			tone = sh.b
		}
		if j < len(kids) {
			name, size = kids[j].Name, kids[j].Size
			if kids[j].IsDir {
				name += "/"
			}
		} else {
			tone, name = greyShade.a, "… "+human.Count(restN)
		}
		c.fill(kc, c.style(tone, blockText, false))
		if kc.W >= 4 {
			c.text(kc.X, kc.Y, " "+name, kc.W, c.style(tone, blockText, false))
			if kc.H >= 2 && kc.W >= 7 {
				c.text(kc.X, kc.Y+1, " "+human.Bytes(size), kc.W, c.style(tone, blockSubtext, false))
			}
		}
	}
}

// canvas is a grid of cells with interned styles, rendered one styled run at
// a time so a full-screen map stays cheap to redraw.
type canvas struct {
	w, h   int
	runes  []rune
	ids    []int
	styles []lipgloss.Style
	index  map[string]int
}

func newCanvas(w, h int) *canvas {
	c := &canvas{w: w, h: h, runes: make([]rune, w*h), ids: make([]int, w*h), index: map[string]int{}}
	for i := range c.runes {
		c.runes[i] = ' '
	}
	c.styles = []lipgloss.Style{lipgloss.NewStyle()}
	return c
}

func (c *canvas) style(bg, fg lipgloss.Color, bold bool) int {
	k := string(bg) + "|" + string(fg) + "|" + strconv.FormatBool(bold)
	if id, ok := c.index[k]; ok {
		return id
	}
	c.styles = append(c.styles, lipgloss.NewStyle().Background(bg).Foreground(fg).Bold(bold))
	c.index[k] = len(c.styles) - 1
	return len(c.styles) - 1
}

func (c *canvas) fill(r treemap.Cell, id int) {
	for y := max(0, r.Y); y < min(c.h, r.Y+r.H); y++ {
		for x := max(0, r.X); x < min(c.w, r.X+r.W); x++ {
			c.runes[y*c.w+x], c.ids[y*c.w+x] = ' ', id
		}
	}
}

// text writes s from (x, y), clipped to maxW cells. A wide rune takes two
// cells; the second holds 0 and is skipped when rendering.
func (c *canvas) text(x, y int, s string, maxW, id int) {
	if y < 0 || y >= c.h {
		return
	}
	end := min(c.w, x+maxW)
	for _, r := range truncRight(s, maxW) {
		rw := runewidth.RuneWidth(r)
		if rw == 0 || x+rw > end {
			break
		}
		c.runes[y*c.w+x], c.ids[y*c.w+x] = r, id
		if rw == 2 {
			c.runes[y*c.w+x+1], c.ids[y*c.w+x+1] = 0, id
		}
		x += rw
	}
}

func (c *canvas) String() string {
	var b strings.Builder
	run := make([]rune, 0, c.w)
	for y := 0; y < c.h; y++ {
		row := y * c.w
		start := 0
		for x := 1; x <= c.w; x++ {
			if x < c.w && c.ids[row+x] == c.ids[row+start] {
				continue
			}
			run = run[:0]
			for _, r := range c.runes[row+start : row+x] {
				if r != 0 {
					run = append(run, r)
				}
			}
			b.WriteString(c.styles[c.ids[row+start]].Render(string(run)))
			start = x
		}
		if y < c.h-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}
