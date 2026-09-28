package treemap

import (
	"math"
	"testing"
)

func TestSquarifyKeepsAreasProportionalAndInside(t *testing.T) {
	// The example from the paper: seven values in a 6×4 box.
	values := []float64{6, 6, 4, 3, 2, 2, 1}
	box := Rect{0, 0, 6, 4}
	rects := Squarify(values, box)
	for i, r := range rects {
		if got := r.W * r.H; math.Abs(got-values[i]) > 1e-9 {
			t.Errorf("rect %d area %.6f, want %.0f", i, got, values[i])
		}
		if r.X < -1e-9 || r.Y < -1e-9 || r.X+r.W > 6+1e-9 || r.Y+r.H > 4+1e-9 {
			t.Errorf("rect %d = %+v leaves the box", i, r)
		}
		if aspect := math.Max(r.W/r.H, r.H/r.W); aspect > 3 {
			t.Errorf("rect %d aspect %.2f, squarify should keep it near 1", i, aspect)
		}
	}
	for i := range rects {
		for j := i + 1; j < len(rects); j++ {
			a, b := rects[i], rects[j]
			ox := math.Min(a.X+a.W, b.X+b.W) - math.Max(a.X, b.X)
			oy := math.Min(a.Y+a.H, b.Y+b.H) - math.Max(a.Y, b.Y)
			if ox > 1e-9 && oy > 1e-9 {
				t.Errorf("rects %d and %d overlap", i, j)
			}
		}
	}
}

func TestLayoutTilesTheBoxExactly(t *testing.T) {
	cases := [][]float64{
		{50, 30, 10, 5, 3, 2},
		{1000, 1, 1, 1},             // slivers round away, neighbours close the gap
		{9, 8, 7, 6, 5, 4, 3, 2, 1}, // many blocks
		{42},
	}
	for _, values := range cases {
		const w, h = 41, 13
		grid := make([]int, w*h)
		for i, c := range Layout(values, 0, 0, w, h) {
			for y := c.Y; y < c.Y+c.H; y++ {
				for x := c.X; x < c.X+c.W; x++ {
					if x < 0 || y < 0 || x >= w || y >= h {
						t.Fatalf("%v: block %d = %+v leaves the box", values, i, c)
					}
					grid[y*w+x]++
				}
			}
		}
		for i, n := range grid {
			if n != 1 {
				t.Fatalf("%v: cell (%d,%d) covered %d times", values, i%w, i/w, n)
			}
		}
	}
}

func TestEqualValuesMakeSquareLookingBlocks(t *testing.T) {
	// 20×10 cells is square on screen, since a cell is about twice as tall as wide.
	cells := Layout([]float64{1, 1, 1, 1}, 0, 0, 20, 10)
	for i, c := range cells {
		if c.W != 10 || c.H != 5 {
			t.Errorf("block %d = %+v, want 10×5", i, c)
		}
	}
}

func TestZerosGetNoBlock(t *testing.T) {
	cells := Layout([]float64{5, 0, 5}, 3, 2, 10, 4)
	if !cells[1].Empty() {
		t.Errorf("a zero value should get no block, got %+v", cells[1])
	}
	if cells[0].X < 3 || cells[0].Y < 2 {
		t.Errorf("offsets not applied: %+v", cells[0])
	}
}
