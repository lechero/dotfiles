// Package treemap lays sizes out as nested rectangles (squarified treemaps,
// Bruls, Huizing & van Wijk, 2000) and snaps them to terminal cells.
package treemap

import "math"

// Rect is a rectangle in layout space.
type Rect struct{ X, Y, W, H float64 }

// Cell is a rectangle of terminal cells.
type Cell struct{ X, Y, W, H int }

// Empty reports whether the cell rectangle covers nothing.
func (c Cell) Empty() bool { return c.W <= 0 || c.H <= 0 }

// Contains reports whether the cell at (x, y) lies inside c.
func (c Cell) Contains(x, y int) bool {
	return x >= c.X && x < c.X+c.W && y >= c.Y && y < c.Y+c.H
}

// Squarify lays values out inside r, each rectangle's area proportional to its
// value, keeping every rectangle as close to square as the algorithm can. Pass
// values largest first; zero or negative values get empty rectangles.
func Squarify(values []float64, r Rect) []Rect {
	out := make([]Rect, len(values))
	var total float64
	for _, v := range values {
		if v > 0 {
			total += v
		}
	}
	if total <= 0 || r.W <= 0 || r.H <= 0 {
		return out
	}
	scale := r.W * r.H / total
	idx := make([]int, 0, len(values)) // positive values, in order
	areas := make([]float64, 0, len(values))
	for i, v := range values {
		if v > 0 {
			idx = append(idx, i)
			areas = append(areas, v*scale)
		}
	}

	for i := 0; i < len(areas); {
		short := math.Min(r.W, r.H)
		j, sum := i+1, areas[i]
		for j < len(areas) && worst(areas[i:j+1], sum+areas[j], short) <= worst(areas[i:j], sum, short) {
			sum += areas[j]
			j++
		}
		if j == len(areas) { // the last row takes whatever is left, absorbing float drift
			sum = r.W * r.H
			if sum <= 0 {
				break
			}
			rescale := sum / sumOf(areas[i:j])
			for k := i; k < j; k++ {
				areas[k] *= rescale
			}
		}
		if r.W >= r.H { // a column on the left
			w := sum / r.H
			y := r.Y
			for k := i; k < j; k++ {
				h := areas[k] / w
				out[idx[k]] = Rect{r.X, y, w, h}
				y += h
			}
			r.X += w
			r.W -= w
		} else { // a row on top
			h := sum / r.W
			x := r.X
			for k := i; k < j; k++ {
				w := areas[k] / h
				out[idx[k]] = Rect{x, r.Y, w, h}
				x += w
			}
			r.Y += h
			r.H -= h
		}
		i = j
	}
	return out
}

// worst is the largest aspect ratio in a row of areas laid along side.
func worst(row []float64, sum, side float64) float64 {
	lo, hi := row[0], row[0]
	for _, a := range row[1:] {
		lo = math.Min(lo, a)
		hi = math.Max(hi, a)
	}
	s2, w2 := sum*sum, side*side
	return math.Max(w2*hi/s2, s2/(w2*lo))
}

func sumOf(xs []float64) float64 {
	var s float64
	for _, x := range xs {
		s += x
	}
	return s
}

// Layout lays values out in a box of w×h terminal cells. Cells are about twice
// as tall as they are wide, so the layout runs in a space where a cell is two
// units tall; that keeps the blocks looking square rather than tall. Shared
// edges round the same way, so neighbours never overlap or leave a gap.
func Layout(values []float64, x, y, w, h int) []Cell {
	const cellAspect = 2.0
	rects := Squarify(values, Rect{0, 0, float64(w), float64(h) * cellAspect})
	out := make([]Cell, len(rects))
	for i, r := range rects {
		if r.W <= 0 || r.H <= 0 {
			continue
		}
		x0, x1 := round(r.X), round(r.X+r.W)
		y0, y1 := round(r.Y/cellAspect), round((r.Y+r.H)/cellAspect)
		x1, y1 = min(x1, w), min(y1, h)
		out[i] = Cell{X: x + x0, Y: y + y0, W: x1 - x0, H: y1 - y0}
	}
	return out
}

func round(f float64) int { return int(math.Floor(f + 0.5)) }
