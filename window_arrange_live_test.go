package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// GHOST_CHROME=/usr/bin/chromium go test -run 'Live' -skip TestEnsureRejectsTamperedPackage .

func liveBrowser(t *testing.T, id, url string) string {
	bin := os.Getenv("GHOST_CHROME")
	if bin == "" {
		t.Skip("set GHOST_CHROME to a Chromium binary")
	}
	port := freePort()
	cmd := exec.Command(bin, "--headless=new", "--no-sandbox", "--window-size=800,600", "--user-data-dir="+t.TempDir(), fmt.Sprintf("--remote-debugging-port=%d", port), url)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = cmd.Process.Kill(); _, _ = cmd.Process.Wait() })
	ws, err := waitDebugger(port, 20*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	runReg.add(id, &RunState{ProfileID: id, WSURL: ws, DebugPort: port}, cmd)
	t.Cleanup(func() { runReg.mu.Lock(); delete(runReg.m, id); runReg.mu.Unlock() })
	return ws
}

func windowBounds(t *testing.T, ws string) Rect {
	c, err := dialCDP(ws, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.c.Close()
	_, wid, err := firstPageWindow(c)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := c.call("", "Browser.getWindowBounds", map[string]any{"windowId": wid})
	if err != nil {
		t.Fatal(err)
	}
	var b struct {
		Bounds struct{ Left, Top, Width, Height int } `json:"bounds"`
	}
	_ = json.Unmarshal(raw, &b)
	return Rect{b.Bounds.Left, b.Bounds.Top, b.Bounds.Width, b.Bounds.Height}
}

func TestArrangeLive(t *testing.T) {
	a := liveBrowser(t, "arrA", "about:blank")
	b := liveBrowser(t, "arrB", "about:blank")
	time.Sleep(500 * time.Millisecond)
	before := []Rect{windowBounds(t, a), windowBounds(t, b)}
	screen := Rect{0, 0, 1600, 900}
	res := arrangeRunning([]string{"arrA", "arrB"}, "grid", screen, func(s string) string { return s })
	if res.Arranged != 2 || len(res.Failed) != 0 {
		t.Fatalf("arrange: %+v", res)
	}
	after := []Rect{windowBounds(t, a), windowBounds(t, b)}
	t.Logf("before %v after %v cells %v", before, after, res.Cells)
	for i := range after {
		if after[i] != res.Cells[i] {
			t.Fatalf("window %d at %v, want %v", i, after[i], res.Cells[i])
		}
	}
}

const syncTestPage = `<!doctype html><html><body style="margin:0">
<div style="height:40px"></div>
<button id="b" style="display:block;width:200px;height:60px" onclick="window.clicks=(window.clicks||0)+1;document.title='c'+window.clicks">go</button>
<input id="i" style="display:block;width:300px;height:30px">
<select id="s"><option value="a">A</option><option value="z">Z</option></select>
<div style="height:3000px"></div></body></html>`

func pageSession(t *testing.T, ws string) (*cdpConn, string) {
	c, err := dialCDP(ws, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { c.c.Close() })
	_, sid, err := attachPage(c)
	if err != nil {
		t.Fatal(err)
	}
	return c, sid
}

func evalStr(t *testing.T, c *cdpConn, sid, expr string) string {
	raw, err := c.call(sid, "Runtime.evaluate", map[string]any{"expression": "String(" + expr + ")", "returnByValue": true})
	if err != nil {
		t.Fatal(err)
	}
	var r struct {
		Result struct{ Value string } `json:"result"`
	}
	_ = json.Unmarshal(raw, &r)
	return r.Result.Value
}

func waitFor(t *testing.T, what string, f func() bool) {
	for i := 0; i < 50; i++ {
		if f() {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
}

func TestSyncerLive(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, syncTestPage)
	}))
	defer srv.Close()
	m := liveBrowser(t, "synM", srv.URL+"/")
	fo := liveBrowser(t, "synF", srv.URL+"/")
	time.Sleep(800 * time.Millisecond)
	var states []SyncerState
	groupSync.emit = func(s SyncerState) { states = append(states, s) }
	defer func() { groupSync.emit = nil }()
	wsOf := func(id string) (string, bool) {
		if rs, ok := runReg.get(id); ok {
			return rs.WSURL, true
		}
		return "", false
	}
	st, err := groupSync.Start("synM", []string{"synF"}, wsOf, func(s string) string { return "名" + s })
	if err != nil {
		t.Fatal(err)
	}
	if !st.Running || len(st.Followers) != 1 {
		t.Fatalf("state %+v", st)
	}
	mc, msid := pageSession(t, m)
	fc, fsid := pageSession(t, fo)
	_ = fo

	// The page's own world must not see the binding or the capture flag.
	if v := evalStr(t, mc, msid, `Object.keys(window).filter(k => k.startsWith('__gb')).join(',')`); v != "" {
		t.Fatalf("binding leaked into page world: %q", v)
	}

	// Click the master's button → follower's click handler fires.
	click := func(c *cdpConn, sid string, x, y float64) {
		for _, ty := range []string{"mousePressed", "mouseReleased"} {
			if _, err := c.call(sid, "Input.dispatchMouseEvent", map[string]any{"type": ty, "x": x, "y": y, "button": "left", "clickCount": 1}); err != nil {
				t.Fatal(err)
			}
		}
	}
	click(mc, msid, 50, 60)
	waitFor(t, "master click", func() bool { return evalStr(t, mc, msid, "window.clicks") == "1" })
	waitFor(t, "follower click", func() bool { return evalStr(t, fc, fsid, "window.clicks") == "1" })

	// Type into the input.
	click(mc, msid, 60, 120)
	for _, ch := range "hi" {
		s := string(ch)
		_, _ = mc.call(msid, "Input.dispatchKeyEvent", map[string]any{"type": "keyDown", "key": s, "text": s})
		_, _ = mc.call(msid, "Input.dispatchKeyEvent", map[string]any{"type": "keyUp", "key": s})
	}
	waitFor(t, "follower typing", func() bool { return evalStr(t, fc, fsid, "document.getElementById('i').value") == "hi" })

	// Select change and wheel scroll.
	evalStr(t, mc, msid, `(() => { const s = document.getElementById('s'); s.value = 'z'; s.dispatchEvent(new Event('change', {bubbles: true})); return 1; })()`)
	waitFor(t, "follower select", func() bool { return evalStr(t, fc, fsid, "document.getElementById('s').value") == "z" })
	_, _ = mc.call(msid, "Input.dispatchMouseEvent", map[string]any{"type": "mouseWheel", "x": 100, "y": 300, "deltaX": 0, "deltaY": 400})
	waitFor(t, "follower scroll", func() bool { return evalStr(t, fc, fsid, "scrollY > 100") == "true" })

	// Master navigates → follower navigates.
	if _, err := mc.call(msid, "Page.navigate", map[string]any{"url": srv.URL + "/next?x=1"}); err != nil {
		t.Fatal(err)
	}
	waitFor(t, "follower navigation", func() bool { return evalStr(t, fc, fsid, "location.pathname+location.search") == "/next?x=1" })

	// Capture keeps working on the new document.
	time.Sleep(300 * time.Millisecond)
	click(mc, msid, 50, 60)
	waitFor(t, "follower click after navigation", func() bool { return evalStr(t, fc, fsid, "window.clicks") == "1" })

	st = groupSync.State()
	t.Logf("state %+v", st)
	if !st.Running || st.Events < 4 || !st.Followers[0].OK || st.Followers[0].Events < 4 {
		t.Fatalf("state %+v", st)
	}
	groupSync.Stop("用户停止")
	st = groupSync.State()
	if st.Running || st.Reason != "用户停止" {
		t.Fatalf("after stop %+v", st)
	}
	if len(states) == 0 || states[len(states)-1].Running {
		t.Fatalf("no stop emitted: %d states", len(states))
	}
	_ = strings.TrimSpace
}

func TestSyncerStopsWhenMasterClosesLive(t *testing.T) {
	m := liveBrowser(t, "synM2", "about:blank")
	_ = liveBrowser(t, "synF2", "about:blank")
	time.Sleep(500 * time.Millisecond)
	wsOf := func(id string) (string, bool) {
		if rs, ok := runReg.get(id); ok {
			return rs.WSURL, true
		}
		return "", false
	}
	if _, err := groupSync.Start("synM2", []string{"synF2"}, wsOf, func(s string) string { return s }); err != nil {
		t.Fatal(err)
	}
	c, err := dialCDP(m, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = c.call("", "Browser.close", nil)
	waitFor(t, "syncer stop", func() bool { return !groupSync.State().Running })
	t.Logf("reason: %s", groupSync.State().Reason)
}
