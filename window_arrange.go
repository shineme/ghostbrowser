package main

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"
)

// One-click window arrangement for running profiles. The layout math is pure
// (arrangeLayout) so it can be tested without a browser; applying it talks to
// each browser over its CDP endpoint (Browser.getWindowForTarget +
// Browser.setWindowBounds), which works for both kernels.

// Rect is a window or screen rectangle in screen DIPs (what Chromium's
// Browser.setWindowBounds expects).
type Rect struct {
	X int `json:"x"`
	Y int `json:"y"`
	W int `json:"w"`
	H int `json:"h"`
}

func (r Rect) right() int  { return r.X + r.W }
func (r Rect) bottom() int { return r.Y + r.H }

// ArrangeLayout is a preview of where N windows go.
type ArrangeLayout struct {
	Mode   string `json:"mode"`
	Screen Rect   `json:"screen"`
	Cols   int    `json:"cols"`
	Rows   int    `json:"rows"`
	Cells  []Rect `json:"cells"`
}

// ArrangeResult reports what ArrangeWindows did.
type ArrangeResult struct {
	ArrangeLayout
	Arranged int      `json:"arranged"`
	Total    int      `json:"total"`
	Failed   []string `json:"failed"` // "环境名：原因"
}

const (
	arrangeMinW    = 500 // Chromium refuses to make a window narrower than this
	arrangeMinH    = 300
	arrangeCellAsp = 1.25 // a browser page reads best a little wider than tall
)

// gridDims picks columns × rows for n windows on a screen: cells should be
// close to a browser-friendly aspect, wide enough for Chromium's minimum
// window width, with as few empty slots as possible.
func gridDims(n int, screen Rect) (cols, rows int) {
	if n <= 1 {
		return 1, 1
	}
	best := math.Inf(1)
	for c := 1; c <= n; c++ {
		r := (n + c - 1) / c
		if (c-1)*r >= n { // same rows with fewer columns already fits
			continue
		}
		cw, ch := float64(screen.W)/float64(c), float64(screen.H)/float64(r)
		score := math.Abs(math.Log((cw / ch) / arrangeCellAsp))
		score += 0.5 * float64(c*r-n) / float64(n)
		if cw < arrangeMinW {
			score += 2 + (arrangeMinW-cw)/arrangeMinW
		}
		if ch < arrangeMinH {
			score += 2 + (arrangeMinH-ch)/arrangeMinH
		}
		if score < best-1e-9 {
			best, cols, rows = score, c, r
		}
	}
	return cols, rows
}

// split divides total into k integer parts that add up exactly.
func split(total, k, i int) (off, size int) {
	off = total * i / k
	return off, total*(i+1)/k - off
}

// arrangeLayout computes the window rectangles for n windows.
//   - grid: tiles the whole screen with no gaps or overlap; an incomplete last
//     row is stretched so the screen stays filled.
//   - cascade: equal windows stepped diagonally, wrapping when they reach the edge.
//   - stack: every window at the same centred rectangle (handy with the syncer).
func arrangeLayout(n int, mode string, screen Rect) ArrangeLayout {
	mode = normArrangeMode(mode)
	out := ArrangeLayout{Mode: mode, Screen: screen, Cells: []Rect{}}
	if n <= 0 || screen.W <= 0 || screen.H <= 0 {
		return out
	}
	switch mode {
	case "cascade":
		w, h := clampI(screen.W*62/100, arrangeMinW, screen.W), clampI(screen.H*78/100, arrangeMinH, screen.H)
		step := 32
		for i := 0; i < n; i++ {
			x, y := (i*step)%(screen.W-w+1), (i*step)%(screen.H-h+1)
			out.Cells = append(out.Cells, Rect{screen.X + x, screen.Y + y, w, h})
		}
		out.Cols, out.Rows = 1, n
	case "stack":
		w, h := clampI(screen.W*72/100, arrangeMinW, screen.W), clampI(screen.H*86/100, arrangeMinH, screen.H)
		r := Rect{screen.X + (screen.W-w)/2, screen.Y + (screen.H-h)/2, w, h}
		for i := 0; i < n; i++ {
			out.Cells = append(out.Cells, r)
		}
		out.Cols, out.Rows = 1, 1
	default:
		cols, rows := gridDims(n, screen)
		out.Cols, out.Rows = cols, rows
		for i := 0; i < n; i++ {
			row := i / cols
			inRow := cols
			if row == rows-1 {
				inRow = n - row*cols
			}
			y, h := split(screen.H, rows, row)
			x, w := split(screen.W, inRow, i%cols)
			out.Cells = append(out.Cells, Rect{screen.X + x, screen.Y + y, w, h})
		}
	}
	return out
}

func normArrangeMode(m string) string {
	switch strings.ToLower(strings.TrimSpace(m)) {
	case "cascade", "stack":
		return strings.ToLower(strings.TrimSpace(m))
	}
	return "grid"
}

func clampI(v, lo, hi int) int {
	if v < lo {
		v = lo
	}
	if v > hi {
		v = hi
	}
	return v
}

// ---------- screen ----------

// screenHook lets the app supply the primary screen from the Wails runtime
// when the OS-specific work area is not available.
var screenHook func() (Rect, bool)

// arrangeScreen is the area windows are laid out in: the primary monitor's
// work area (taskbar excluded) in DIPs.
func arrangeScreen() Rect {
	if r, ok := osWorkArea(); ok && r.W > 0 && r.H > 0 {
		return r
	}
	if screenHook != nil {
		if r, ok := screenHook(); ok && r.W > 0 && r.H > 0 {
			return r
		}
	}
	return Rect{0, 0, 1920, 1040}
}

// ---------- CDP ----------

// firstPageWindow finds the browser window of a profile's first page tab.
func firstPageWindow(c *cdpConn) (targetID string, windowID int, err error) {
	raw, err := c.call("", "Target.getTargets", nil)
	if err != nil {
		return "", 0, err
	}
	var tl struct {
		TargetInfos []struct {
			TargetID string `json:"targetId"`
			Type     string `json:"type"`
			URL      string `json:"url"`
		} `json:"targetInfos"`
	}
	_ = json.Unmarshal(raw, &tl)
	for _, t := range tl.TargetInfos {
		if t.Type != "page" || strings.HasPrefix(t.URL, "devtools://") {
			continue
		}
		r, e := c.call("", "Browser.getWindowForTarget", map[string]any{"targetId": t.TargetID})
		if e != nil {
			err = e
			continue
		}
		var w struct {
			WindowID int `json:"windowId"`
		}
		_ = json.Unmarshal(r, &w)
		if w.WindowID != 0 {
			return t.TargetID, w.WindowID, nil
		}
	}
	if err == nil {
		err = fmt.Errorf("浏览器没有打开的窗口")
	}
	return "", 0, err
}

// placeWindow restores a window and moves it to r. It returns the bounds the
// browser actually applied (Chromium clamps to its minimum size).
func placeWindow(ws string, r Rect, activate bool) (Rect, error) {
	c, err := dialCDP(ws, nil)
	if err != nil {
		return Rect{}, err
	}
	defer c.c.Close()
	tid, wid, err := firstPageWindow(c)
	if err != nil {
		return Rect{}, err
	}
	// Bounds can't be set on a maximised, minimised or fullscreen window.
	_, _ = c.call("", "Browser.setWindowBounds", map[string]any{"windowId": wid, "bounds": map[string]any{"windowState": "normal"}})
	if _, err := c.call("", "Browser.setWindowBounds", map[string]any{"windowId": wid, "bounds": map[string]any{
		"left": r.X, "top": r.Y, "width": r.W, "height": r.H}}); err != nil {
		return Rect{}, err
	}
	if activate {
		_, _ = c.call("", "Target.activateTarget", map[string]any{"targetId": tid})
	}
	raw, err := c.call("", "Browser.getWindowBounds", map[string]any{"windowId": wid})
	if err != nil {
		return r, nil
	}
	var b struct {
		Bounds struct {
			Left, Top, Width, Height int
		} `json:"bounds"`
	}
	_ = json.Unmarshal(raw, &b)
	return Rect{b.Bounds.Left, b.Bounds.Top, b.Bounds.Width, b.Bounds.Height}, nil
}

// arrangeRunning lays out the given running profiles (all running ones when
// ids is empty). Windows are placed in parallel; for "stack" and "cascade"
// they are then raised in order so the first one ends up on top.
func arrangeRunning(ids []string, mode string, screen Rect, name func(string) string) ArrangeResult {
	type run struct{ id, ws string }
	var runs []run
	seen := map[string]bool{}
	if len(ids) == 0 {
		for _, s := range runReg.list() {
			ids = append(ids, s.ProfileID)
		}
	}
	for _, id := range ids {
		if seen[id] {
			continue
		}
		seen[id] = true
		if rs, ok := runReg.get(id); ok && strings.HasPrefix(rs.WSURL, "ws") {
			runs = append(runs, run{id, rs.WSURL})
		}
	}
	lay := arrangeLayout(len(runs), mode, screen)
	res := ArrangeResult{ArrangeLayout: lay, Total: len(runs), Failed: []string{}}
	var mu sync.Mutex
	var wg sync.WaitGroup
	for i, r := range runs {
		wg.Add(1)
		go func(i int, r run) {
			defer wg.Done()
			_, err := placeWindow(r.ws, lay.Cells[i], false)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				res.Failed = append(res.Failed, name(r.id)+"："+err.Error())
				return
			}
			res.Arranged++
		}(i, r)
	}
	wg.Wait()
	// Raise windows so the z-order matches the layout: in a cascade the last
	// window is on top (every title bar stays visible), in a stack the first.
	if lay.Mode != "grid" {
		for k := range runs {
			i := k
			if lay.Mode == "stack" {
				i = len(runs) - 1 - k
			}
			if c, err := dialCDP(runs[i].ws, nil); err == nil {
				if tid, _, e := firstPageWindow(c); e == nil {
					_, _ = c.call("", "Target.activateTarget", map[string]any{"targetId": tid})
				}
				c.c.Close()
				time.Sleep(40 * time.Millisecond)
			}
		}
	}
	return res
}
