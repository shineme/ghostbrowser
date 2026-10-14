package main

import "testing"

func overlap(a, b Rect) bool {
	return a.X < b.right() && b.X < a.right() && a.Y < b.bottom() && b.Y < a.bottom()
}

func TestArrangeGridTilesScreen(t *testing.T) {
	screens := []Rect{{0, 0, 1920, 1040}, {0, 0, 1366, 728}, {-1280, 40, 2560, 1400}, {0, 0, 3840, 2120}}
	for _, sc := range screens {
		for n := 1; n <= 9; n++ {
			lay := arrangeLayout(n, "grid", sc)
			if len(lay.Cells) != n {
				t.Fatalf("n=%d got %d cells", n, len(lay.Cells))
			}
			area := 0
			for i, c := range lay.Cells {
				if c.W <= 0 || c.H <= 0 || c.X < sc.X || c.Y < sc.Y || c.right() > sc.right() || c.bottom() > sc.bottom() {
					t.Fatalf("screen %v n=%d cell %d %v outside", sc, n, i, c)
				}
				area += c.W * c.H
				for j := i + 1; j < n; j++ {
					if overlap(c, lay.Cells[j]) {
						t.Fatalf("screen %v n=%d cells %d %v and %d %v overlap", sc, n, i, c, j, lay.Cells[j])
					}
				}
			}
			if area != sc.W*sc.H {
				t.Fatalf("screen %v n=%d covers %d of %d", sc, n, area, sc.W*sc.H)
			}
			if lay.Cols*lay.Rows < n {
				t.Fatalf("n=%d dims %dx%d", n, lay.Cols, lay.Rows)
			}
		}
	}
}

func TestArrangeCascadeStackOnScreen(t *testing.T) {
	for _, sc := range []Rect{{0, 0, 1920, 1040}, {100, 30, 1366, 728}} {
		for _, mode := range []string{"cascade", "stack"} {
			for n := 1; n <= 30; n++ {
				lay := arrangeLayout(n, mode, sc)
				if lay.Mode != mode || len(lay.Cells) != n {
					t.Fatalf("%s n=%d: %+v", mode, n, lay)
				}
				for i, c := range lay.Cells {
					if c.X < sc.X || c.Y < sc.Y || c.right() > sc.right() || c.bottom() > sc.bottom() || c.W < arrangeMinW || c.H < arrangeMinH {
						t.Fatalf("%s screen %v n=%d cell %d %v off screen", mode, sc, n, i, c)
					}
				}
			}
		}
	}
	if arrangeLayout(3, "bogus", Rect{0, 0, 1920, 1040}).Mode != "grid" {
		t.Fatal("unknown mode should fall back to grid")
	}
	if len(arrangeLayout(0, "grid", Rect{0, 0, 1920, 1040}).Cells) != 0 {
		t.Fatal("n=0 should have no cells")
	}
}
