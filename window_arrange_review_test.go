package main

import "testing"

// Review: layout invariants over many counts and odd screens.
func TestArrangeReviewInvariants(t *testing.T) {
	screens := []Rect{{0, 0, 1920, 1040}, {-1920, 0, 1920, 1080}, {0, 0, 1366, 728}, {0, 0, 480, 270}, {100, 50, 3440, 1400}}
	for _, sc := range screens {
		for n := 1; n <= 40; n++ {
			g := arrangeLayout(n, "grid", sc)
			if len(g.Cells) != n || g.Cols*g.Rows < n {
				t.Fatalf("grid n=%d %v: %d cells %dx%d", n, sc, len(g.Cells), g.Cols, g.Rows)
			}
			area := 0
			for i, c := range g.Cells {
				if c.W <= 0 || c.H <= 0 || c.X < sc.X || c.Y < sc.Y || c.right() > sc.right() || c.bottom() > sc.bottom() {
					t.Fatalf("grid n=%d %v cell %d = %v", n, sc, i, c)
				}
				area += c.W * c.H
				for j := 0; j < i; j++ {
					o := g.Cells[j]
					if c.X < o.right() && o.X < c.right() && c.Y < o.bottom() && o.Y < c.bottom() {
						t.Fatalf("grid n=%d overlap %v %v", n, c, o)
					}
				}
			}
			if area != sc.W*sc.H {
				t.Fatalf("grid n=%d %v leaves gaps: %d of %d", n, sc, area, sc.W*sc.H)
			}
			for _, mode := range []string{"cascade", "stack"} {
				l := arrangeLayout(n, mode, sc)
				for i, c := range l.Cells {
					if c.W <= 0 || c.H <= 0 || c.X < sc.X || c.Y < sc.Y || c.right() > sc.right() || c.bottom() > sc.bottom() {
						t.Fatalf("%s n=%d %v cell %d = %v", mode, n, sc, i, c)
					}
				}
			}
		}
	}
	if got := normArrangeMode(" Cascade "); got != "cascade" {
		t.Fatalf("mode %q", got)
	}
}
