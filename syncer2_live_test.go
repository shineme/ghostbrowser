package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"
)

// Live tests for the 多窗口同步器 upgrade with three real Chromium instances.
// GHOST_CHROME=/usr/bin/chromium go test -run 'TestSyncer2' .

const sync2Page = `<!doctype html><html><head><title>s2</title></head><body style="margin:0">
<button id="b" style="display:block;width:200px;height:50px">btn</button>
<input id="i" style="display:block;width:400px;height:30px">
<a id="blank" href="/opened" target="_blank" style="display:block;height:30px">open</a>
<div style="height:4000px"></div>
<script>
window.clicks = 0; window.enters = 0; window.clickAt = [];
document.getElementById('b').addEventListener('click', () => { window.clicks++; window.clickAt.push(Date.now()); });
document.addEventListener('keydown', e => { if (e.key === 'Enter') window.enters++; });
</script></body></html>`

func sync2Server(t *testing.T) *httptest.Server {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, strings.Replace(sync2Page, "<title>s2</title>", "<title>"+r.URL.Path+"</title>", 1))
	}))
	t.Cleanup(srv.Close)
	return srv
}

func liveWS(id string) (string, bool) {
	if rs, ok := runReg.get(id); ok {
		return rs.WSURL, true
	}
	return "", false
}

func idName(s string) string { return s }

// tabSession attaches to a given tab.
func tabSession(t *testing.T, ws, tid string) (*cdpConn, string) {
	c, err := dialCDP(ws, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { c.c.Close() })
	sid, err := attachTarget(c, tid)
	if err != nil {
		t.Fatal(err)
	}
	return c, sid
}

func tabsOf(t *testing.T, ws string) []pageTab {
	c, err := dialCDP(ws, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.c.Close()
	tabs, err := pageTabs(c)
	if err != nil {
		t.Fatal(err)
	}
	return tabs
}

func pathsOf(tabs []pageTab) string {
	var p []string
	for _, t := range tabs {
		u := t.URL
		if i := strings.Index(u, "://"); i >= 0 && strings.HasPrefix(u, "http") {
			u = u[i+3:]
			if j := strings.Index(u, "/"); j >= 0 {
				u = u[j:]
			}
		}
		p = append(p, u)
	}
	// Target.getTargets order is not the tab strip order: compare as sets.
	sort.Strings(p)
	return strings.Join(p, ",")
}

func tabByPath(t *testing.T, tabs []pageTab, p string) string {
	for _, x := range tabs {
		if pathsOf([]pageTab{x}) == p {
			return x.TargetID
		}
	}
	t.Fatalf("no tab %s in %s", p, pathsOf(tabs))
	return ""
}

// activePath returns the path of the browser's visible tab.
func activePath(t *testing.T, ws string) string {
	c, err := dialCDP(ws, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.c.Close()
	tabs, _ := pageTabs(c)
	act := activeOf(c, tabs)
	for _, x := range tabs {
		if x.TargetID == act {
			return pathsOf([]pageTab{x})
		}
	}
	return ""
}

func winState(t *testing.T, ws string) (Rect, string) {
	b, err := openBrowserTabs(ws)
	if err != nil {
		t.Fatal(err)
	}
	defer b.close()
	_, r, st, err := b.window(b.tabs[0].TargetID)
	if err != nil {
		t.Fatal(err)
	}
	return r, st
}

func waitLong(t *testing.T, what string, f func() bool) {
	t.Helper()
	for i := 0; i < 100; i++ {
		if f() {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
}

func click(t *testing.T, c *cdpConn, sid string, x, y float64) {
	for _, ty := range []string{"mousePressed", "mouseReleased"} {
		if _, err := c.call(sid, "Input.dispatchMouseEvent", map[string]any{"type": ty, "x": x, "y": y, "button": "left", "clickCount": 1}); err != nil {
			t.Fatal(err)
		}
	}
}

func typeText(c *cdpConn, sid, s string) {
	for _, ch := range s {
		k := string(ch)
		_, _ = c.call(sid, "Input.dispatchKeyEvent", map[string]any{"type": "keyDown", "key": k, "text": k, "windowsVirtualKeyCode": int(strings.ToUpper(k)[0])})
		_, _ = c.call(sid, "Input.dispatchKeyEvent", map[string]any{"type": "keyUp", "key": k, "windowsVirtualKeyCode": int(strings.ToUpper(k)[0])})
	}
}

func pressEnter(c *cdpConn, sid string) {
	_, _ = c.call(sid, "Input.dispatchKeyEvent", map[string]any{"type": "keyDown", "key": "Enter", "code": "Enter", "text": "\r", "windowsVirtualKeyCode": 13})
	_, _ = c.call(sid, "Input.dispatchKeyEvent", map[string]any{"type": "keyUp", "key": "Enter", "code": "Enter", "windowsVirtualKeyCode": 13})
}

func three(t *testing.T, prefix, url string) (string, string, string) {
	m := liveBrowser(t, prefix+"M", url)
	f1 := liveBrowser(t, prefix+"F1", url)
	f2 := liveBrowser(t, prefix+"F2", url)
	time.Sleep(900 * time.Millisecond)
	return m, f1, f2
}

func TestSyncer2TogglesLive(t *testing.T) {
	srv := sync2Server(t)
	m, f1, f2 := three(t, "tg", srv.URL+"/")
	groupSync.SetOptions(defaultSyncOptions())
	defer groupSync.SetOptions(defaultSyncOptions())
	if _, err := groupSync.Start("tgM", []string{"tgF1", "tgF2"}, liveWS, idName); err != nil {
		t.Fatal(err)
	}
	defer groupSync.Stop("test end")
	mc, ms := pageSession(t, m)
	c1, s1 := pageSession(t, f1)
	c2, s2 := pageSession(t, f2)
	both := func(what, expr, want string) {
		t.Helper()
		waitFor(t, what+" f1", func() bool { return evalStr(t, c1, s1, expr) == want })
		waitFor(t, what+" f2", func() bool { return evalStr(t, c2, s2, expr) == want })
	}
	stays := func(what, expr, want string) {
		t.Helper()
		for _, fc := range []struct {
			c *cdpConn
			s string
		}{{c1, s1}, {c2, s2}} {
			if v := evalStr(t, fc.c, fc.s, expr); v != want {
				t.Fatalf("%s: %s = %q, want %q (event should have been suppressed)", what, expr, v, want)
			}
		}
	}
	set := func(mut func(*SyncOptions)) {
		o := groupSync.Options()
		mut(&o)
		st := groupSync.SetOptions(o)
		if st.Options != o.norm() || !st.Running {
			t.Fatalf("options not applied: %+v", st)
		}
	}
	val := "document.getElementById('i').value"

	// baseline: all on
	click(t, mc, ms, 50, 25)
	both("click on", "window.clicks", "1")
	click(t, mc, ms, 60, 65) // focus input
	typeText(mc, ms, "ab")
	both("input on", val, "ab")

	// 同步点击 off: the click is not replayed (typing still is, as a barrier)
	set(func(o *SyncOptions) { o.Click = false })
	click(t, mc, ms, 50, 25)
	waitFor(t, "master click", func() bool { return evalStr(t, mc, ms, "window.clicks") == "2" })
	click(t, mc, ms, 60, 65) // refocus the master's input (followers never left it)
	typeText(mc, ms, "c")
	both("barrier after click-off", val, "abc")
	stays("click off", "window.clicks", "1")

	// 同步输入 off: text not replayed, Enter (同步按键) still is
	set(func(o *SyncOptions) { o.Click = true; o.Input = false })
	typeText(mc, ms, "zz")
	waitFor(t, "master typed", func() bool { return evalStr(t, mc, ms, val) == "abczz" })
	pressEnter(mc, ms)
	both("barrier after input-off", "window.enters", "1")
	stays("input off", val, "abc")

	// 同步按键 off: Enter not replayed, text is
	set(func(o *SyncOptions) { o.Input = true; o.Key = false })
	pressEnter(mc, ms)
	waitFor(t, "master enter", func() bool { return evalStr(t, mc, ms, "window.enters") == "2" })
	typeText(mc, ms, "d")
	both("barrier after key-off", val, "abcd")
	stays("key off", "window.enters", "1")

	// 同步滚动 off: wheel not replayed, click is
	set(func(o *SyncOptions) { o.Key = true; o.Scroll = false })
	_, _ = mc.call(ms, "Input.dispatchMouseEvent", map[string]any{"type": "mouseWheel", "x": 100, "y": 300, "deltaX": 0, "deltaY": 600})
	waitFor(t, "master scrolled", func() bool { return evalStr(t, mc, ms, "scrollY > 100") == "true" })
	time.Sleep(200 * time.Millisecond)
	_, _ = mc.call(ms, "Runtime.evaluate", map[string]any{"expression": "scrollTo(0,0)"})
	time.Sleep(150 * time.Millisecond)
	click(t, mc, ms, 50, 25)
	both("barrier after scroll-off", "window.clicks", "2")
	stays("scroll off", "scrollY", "0")

	// back on while running: the wheel is replayed again
	set(func(o *SyncOptions) { o.Scroll = true })
	_, _ = mc.call(ms, "Input.dispatchMouseEvent", map[string]any{"type": "mouseWheel", "x": 100, "y": 300, "deltaX": 0, "deltaY": 600})
	both("scroll on again", "scrollY > 100", "true")
	t.Logf("state %+v", groupSync.State())
}

func TestSyncer2DelayLive(t *testing.T) {
	srv := sync2Server(t)
	m, f1, f2 := three(t, "dl", srv.URL+"/")
	const lo, hi = 250, 600
	groupSync.SetOptions(SyncOptions{Click: true, Input: true, Key: true, Scroll: true, Tabs: true, DelayMin: lo, DelayMax: hi})
	defer groupSync.SetOptions(defaultSyncOptions())
	if _, err := groupSync.Start("dlM", []string{"dlF1", "dlF2"}, liveWS, idName); err != nil {
		t.Fatal(err)
	}
	defer groupSync.Stop("test end")
	mc, ms := pageSession(t, m)
	c1, s1 := pageSession(t, f1)
	c2, s2 := pageSession(t, f2)
	const n = 6
	for i := 0; i < n; i++ {
		click(t, mc, ms, 50, 25)
		time.Sleep(900 * time.Millisecond) // > max delay: no ordering carry-over
	}
	waitFor(t, "f1 clicks", func() bool { return evalStr(t, c1, s1, "window.clicks") == strconv.Itoa(n) })
	waitFor(t, "f2 clicks", func() bool { return evalStr(t, c2, s2, "window.clicks") == strconv.Itoa(n) })
	times := func(c *cdpConn, s string) []int64 {
		var v []int64
		_ = json.Unmarshal([]byte(evalStr(t, c, s, "JSON.stringify(window.clickAt)")), &v)
		return v
	}
	mt, t1, t2 := times(mc, ms), times(c1, s1), times(c2, s2)
	t.Logf("master %v\nf1 %v\nf2 %v", mt, t1, t2)
	differ := 0
	var lat []int64
	for i := 0; i < n; i++ {
		for _, ft := range [][]int64{t1, t2} {
			d := ft[i] - mt[i]
			lat = append(lat, d)
			// capture is polled every 50 ms, replay adds a few ms of CDP work
			if d < lo-10 || d > hi+250 {
				t.Errorf("click %d latency %d ms outside %d..%d(+tolerance)", i, d, lo, hi)
			}
		}
		if a, b := t1[i]-t2[i], int64(15); a > b || a < -b {
			differ++
		}
	}
	sort.Slice(lat, func(i, j int) bool { return lat[i] < lat[j] })
	t.Logf("latencies sorted %v; followers differed on %d/%d clicks", lat, differ, n)
	if differ < 2 {
		t.Errorf("followers are not jittered independently (differed on %d/%d)", differ, n)
	}
	if lat[len(lat)-1]-lat[0] < 60 {
		t.Errorf("delays are not spread over the range: %v", lat)
	}
	for _, f := range groupSync.State().Followers {
		if f.LatencyMs < lo-10 {
			t.Errorf("reported latency %d < min", f.LatencyMs)
		}
	}
}

func TestSyncer2TabsLive(t *testing.T) {
	srv := sync2Server(t)
	m, f1, f2 := three(t, "tb", srv.URL+"/one")
	groupSync.SetOptions(defaultSyncOptions())
	if _, err := groupSync.Start("tbM", []string{"tbF1", "tbF2"}, liveWS, idName); err != nil {
		t.Fatal(err)
	}
	defer groupSync.Stop("test end")
	mcon, err := dialCDP(m, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer mcon.c.Close()
	fws := []string{f1, f2}
	val := "document.getElementById('i').value"
	// value of the input in the follower tab whose path is p
	valIn := func(ws, p string) string {
		for _, x := range tabsOf(t, ws) {
			if pathsOf([]pageTab{x}) == p {
				c, s := tabSession(t, ws, x.TargetID)
				return evalStr(t, c, s, val)
			}
		}
		return "<no tab " + p + ">"
	}

	// 1. new tab in the master (user-opened) -> followers open it too
	raw, err := mcon.call("", "Target.createTarget", map[string]any{"url": srv.URL + "/two"})
	if err != nil {
		t.Fatal(err)
	}
	var nt struct {
		TargetID string `json:"targetId"`
	}
	_ = json.Unmarshal(raw, &nt)
	for i, ws := range fws {
		waitLong(t, fmt.Sprintf("f%d new tab", i+1), func() bool { return pathsOf(tabsOf(t, ws)) == "/one,/two" })
		waitLong(t, fmt.Sprintf("f%d switched to new tab", i+1), func() bool { return activePath(t, ws) == "/two" })
	}
	waitLong(t, "mapping", func() bool {
		st := groupSync.State()
		return st.Tabs == 2 && st.Followers[0].Tabs == 2 && st.Followers[1].Tabs == 2
	})
	// input in the master's new tab lands in the followers' mapped tab
	c2, s2 := tabSession(t, m, nt.TargetID)
	time.Sleep(300 * time.Millisecond)
	click(t, c2, s2, 60, 65)
	typeText(c2, s2, "two")
	for i, ws := range fws {
		waitLong(t, fmt.Sprintf("f%d typed in /two", i+1), func() bool { return valIn(ws, "/two") == "two" })
		if v := valIn(ws, "/one"); v != "" {
			t.Fatalf("f%d: input leaked into /one: %q", i+1, v)
		}
	}

	// 2. switch back to the first tab in the master -> followers switch
	first := tabByPath(t, tabsOf(t, m), "/one")
	if _, err := mcon.call("", "Target.activateTarget", map[string]any{"targetId": first}); err != nil {
		t.Fatal(err)
	}
	for i, ws := range fws {
		waitLong(t, fmt.Sprintf("f%d switched to /one", i+1), func() bool { return activePath(t, ws) == "/one" })
	}
	c1, s1 := tabSession(t, m, first)
	click(t, c1, s1, 60, 65)
	typeText(c1, s1, "one")
	for i, ws := range fws {
		waitLong(t, fmt.Sprintf("f%d typed in /one", i+1), func() bool { return valIn(ws, "/one") == "one" })
		if v := valIn(ws, "/two"); v != "two" {
			t.Fatalf("f%d: /two changed: %q", i+1, v)
		}
	}

	// 3. close the master's second tab -> followers close theirs
	if _, err := mcon.call("", "Target.closeTarget", map[string]any{"targetId": nt.TargetID}); err != nil {
		t.Fatal(err)
	}
	for i, ws := range fws {
		waitLong(t, fmt.Sprintf("f%d closed /two", i+1), func() bool { return pathsOf(tabsOf(t, ws)) == "/one" })
	}

	// 4. a link with target=_blank: followers' own replayed click opens their
	// tab, which is adopted (no duplicate tab)
	click(t, c1, s1, 20, 95)
	waitLong(t, "master opened tab", func() bool { return len(tabsOf(t, m)) == 2 })
	for i, ws := range fws {
		waitLong(t, fmt.Sprintf("f%d opened /opened", i+1), func() bool { return pathsOf(tabsOf(t, ws)) == "/one,/opened" })
	}
	time.Sleep(1800 * time.Millisecond) // past the adoption window: still no extra tab
	for i, ws := range fws {
		if p := pathsOf(tabsOf(t, ws)); p != "/one,/opened" {
			t.Fatalf("f%d tabs after opener tab: %s", i+1, p)
		}
	}
	var op string
	for _, x := range tabsOf(t, m) {
		if x.TargetID != first {
			op = x.TargetID
		}
	}
	co, so := tabSession(t, m, op)
	time.Sleep(300 * time.Millisecond)
	click(t, co, so, 60, 65)
	typeText(co, so, "op")
	for i, ws := range fws {
		waitLong(t, fmt.Sprintf("f%d typed in /opened", i+1), func() bool { return valIn(ws, "/opened") == "op" })
	}

	// 5. a tab action while syncing is not mirrored twice
	res, err := syncTabAction("tbM", []string{"tbF1", "tbF2"}, "open", srv.URL+"/batch", true, liveWS, idName)
	if err != nil || res.Done != 3 {
		t.Fatalf("batch open while syncing: %+v %v", res, err)
	}
	time.Sleep(1500 * time.Millisecond)
	for i, ws := range append([]string{m}, fws...) {
		if p := pathsOf(tabsOf(t, ws)); p != "/batch,/one,/opened" {
			t.Fatalf("browser %d after batch open while syncing: %s", i, p)
		}
	}
	waitLong(t, "remapped", func() bool {
		st := groupSync.State()
		return st.Running && st.Tabs == 3 && st.Followers[0].Tabs == 3 && st.Followers[1].Tabs == 3
	})
	// after the re-map, input in the master's /batch tab lands in /batch
	cb, sb := tabSession(t, m, tabByPath(t, tabsOf(t, m), "/batch"))
	_, _ = mcon.call("", "Target.activateTarget", map[string]any{"targetId": tabByPath(t, tabsOf(t, m), "/batch")})
	time.Sleep(400 * time.Millisecond)
	click(t, cb, sb, 60, 65)
	typeText(cb, sb, "bt")
	for i, ws := range fws {
		waitLong(t, fmt.Sprintf("f%d typed in /batch", i+1), func() bool { return valIn(ws, "/batch") == "bt" })
		if v := valIn(ws, "/opened"); v != "op" {
			t.Fatalf("f%d: /opened changed: %q", i+1, v)
		}
	}
	t.Logf("state %+v", groupSync.State())
}

func TestSyncer2ActionsLive(t *testing.T) {
	srv := sync2Server(t)
	m, f1, f2 := three(t, "ac", srv.URL+"/start")
	all := []string{m, f1, f2}
	fol := []string{"acF1", "acF2"}
	run := func(action, url string, newTab bool) TabActionResult {
		t.Helper()
		res, err := syncTabAction("acM", fol, action, url, newTab, liveWS, idName)
		if err != nil || len(res.Failed) > 0 {
			t.Fatalf("%s: %+v %v", action, res, err)
		}
		time.Sleep(400 * time.Millisecond)
		return res
	}
	if groupSync.running() != nil {
		t.Fatal("syncer should not be running for this test")
	}

	// 批量访问网址: current tab, URL without scheme is normalised
	host := strings.TrimPrefix(srv.URL, "http://")
	if _, err := syncTabAction("acM", fol, "open", "exa mple", false, liveWS, idName); err == nil {
		t.Fatal("bad URL accepted")
	}
	res := run("open", "http://"+host+"/a", false)
	if res.Done != 3 || res.Total != 3 {
		t.Fatalf("open: %+v", res)
	}
	for _, ws := range all {
		waitFor(t, "current tab /a", func() bool { return pathsOf(tabsOf(t, ws)) == "/a" })
	}
	// new tab
	run("open", srv.URL+"/b", true)
	for _, ws := range all {
		waitFor(t, "new tab /b", func() bool { return pathsOf(tabsOf(t, ws)) == "/a,/b" })
	}

	// 统一标签页: master /a,/b,/c with /b active; f1 has extra tabs, f2 fewer
	mb, _ := openBrowserTabs(m)
	_, _ = mb.newTab(srv.URL + "/c")
	time.Sleep(300 * time.Millisecond)
	tabs, _ := pageTabs(mb.c)
	_ = mb.activate(tabByPath(t, tabs, "/b"))
	mb.close()
	b1, _ := openBrowserTabs(f1)
	_, _ = b1.newTab(srv.URL + "/x")
	_, _ = b1.newTab(srv.URL + "/y")
	b1.close()
	b2, _ := openBrowserTabs(f2)
	_ = b2.closeTab(tabByPath(t, b2.tabs, "/b"))
	b2.close()
	time.Sleep(500 * time.Millisecond)
	if p := pathsOf(tabsOf(t, m)); p != "/a,/b,/c" || activePath(t, m) != "/b" {
		t.Fatalf("master setup: %s active %s", p, activePath(t, m))
	}
	run("unify", "", false)
	for i, ws := range []string{f1, f2} {
		waitLong(t, fmt.Sprintf("f%d unified", i+1), func() bool { return pathsOf(tabsOf(t, ws)) == "/a,/b,/c" })
		waitFor(t, fmt.Sprintf("f%d active /b", i+1), func() bool { return activePath(t, ws) == "/b" })
	}

	// 关闭空白标签: two blank tabs in every window
	for _, ws := range all {
		b, _ := openBrowserTabs(ws)
		_, _ = b.newTab("about:blank")
		_, _ = b.newTab("about:blank")
		b.close()
	}
	time.Sleep(300 * time.Millisecond)
	run("closeBlank", "", false)
	for _, ws := range all {
		waitFor(t, "blank closed", func() bool { return pathsOf(tabsOf(t, ws)) == "/a,/b,/c" })
	}

	// 关闭当前标签: activate /c everywhere then close the current tab
	for _, ws := range all {
		b, _ := openBrowserTabs(ws)
		_ = b.activate(tabByPath(t, b.tabs, "/c"))
		b.close()
	}
	time.Sleep(300 * time.Millisecond)
	run("closeCurrent", "", false)
	for _, ws := range all {
		waitFor(t, "current closed", func() bool { return pathsOf(tabsOf(t, ws)) == "/a,/b" })
	}

	// 关闭其他标签: keep only the active tab
	for _, ws := range all {
		b, _ := openBrowserTabs(ws)
		_ = b.activate(tabByPath(t, b.tabs, "/a"))
		b.close()
	}
	time.Sleep(300 * time.Millisecond)
	run("closeOthers", "", false)
	for _, ws := range all {
		waitFor(t, "others closed", func() bool { return pathsOf(tabsOf(t, ws)) == "/a" })
	}
	// closing the current (last) tab blanks it instead of closing the window
	run("closeCurrent", "", false)
	for _, ws := range all {
		waitFor(t, "last tab blanked", func() bool { return pathsOf(tabsOf(t, ws)) == "about:blank" })
	}

	// 统一大小: followers take the master's size and keep their position
	mbw, _ := openBrowserTabs(m)
	wid, _, _, _ := mbw.window(mbw.tabs[0].TargetID)
	_, _ = mbw.c.call("", "Browser.setWindowBounds", map[string]any{"windowId": wid, "bounds": map[string]any{"left": 0, "top": 0, "width": 1000, "height": 720}})
	mbw.close()
	for i, ws := range []string{f1, f2} {
		b, _ := openBrowserTabs(ws)
		w, _, _, _ := b.window(b.tabs[0].TargetID)
		_, _ = b.c.call("", "Browser.setWindowBounds", map[string]any{"windowId": w, "bounds": map[string]any{"left": 100 + 200*i, "top": 50 + 30*i, "width": 640, "height": 480}})
		b.close()
	}
	before := []Rect{}
	for _, ws := range []string{f1, f2} {
		r, _ := winState(t, ws)
		before = append(before, r)
	}
	run("size", "", false)
	for i, ws := range []string{f1, f2} {
		r, _ := winState(t, ws)
		t.Logf("f%d before %v after %v", i+1, before[i], r)
		if r.W != 1000 || r.H != 720 || r.X != before[i].X || r.Y != before[i].Y {
			t.Fatalf("f%d after size %v (before %v)", i+1, r, before[i])
		}
	}

	// 显示窗口: minimised windows are restored
	for _, ws := range all {
		b, _ := openBrowserTabs(ws)
		w, _, _, _ := b.window(b.tabs[0].TargetID)
		_, _ = b.c.call("", "Browser.setWindowBounds", map[string]any{"windowId": w, "bounds": map[string]any{"windowState": "minimized"}})
		b.close()
	}
	time.Sleep(200 * time.Millisecond)
	if _, st := winState(t, f1); st != "minimized" {
		t.Fatalf("setup: f1 window %s", st)
	}
	run("show", "", false)
	for i, ws := range all {
		if _, st := winState(t, ws); st != "normal" {
			t.Fatalf("browser %d window %s after show", i, st)
		}
	}

	// a stopped profile is reported, the rest still runs
	res, err := syncTabAction("acM", []string{"acF1", "nope"}, "show", "", false, liveWS, idName)
	if err != nil || res.Done != 2 || len(res.Failed) != 1 || !strings.Contains(res.Failed[0], "没有在运行") {
		t.Fatalf("partial: %+v %v", res, err)
	}
}
