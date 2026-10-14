package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// Second-pass review tests for the group-control syncer with three real
// Chromium instances (1 master + 2 followers).
// GHOST_CHROME=/usr/bin/chromium go test -run TestSyncerReview .

const syncReviewPage = `<!doctype html><html><head><title>p</title></head><body style="margin:0">
<div style="height:40px"></div>
<input id="i" style="display:block;width:400px;height:30px">
<a id="lnk" href="/page2" style="display:block;height:30px">next</a>
<div id="ctx" style="width:200px;height:40px;background:#ccc">ctx</div>
<div style="height:3000px"></div>
<script>
window.ctxn = 0; window.keys = '';
document.getElementById('ctx').addEventListener('contextmenu', e => { e.preventDefault(); window.ctxn++; });
document.addEventListener('keydown', e => { if (document.activeElement === document.body) window.keys += e.key; });
</script></body></html>`

func TestSyncerReviewLive(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if r.URL.Path == "/page2" {
			fmt.Fprint(w, `<!doctype html><title>two</title><body><input id="j"></body>`)
			return
		}
		fmt.Fprint(w, syncReviewPage)
	}))
	defer srv.Close()
	m := liveBrowser(t, "rvM", srv.URL+"/")
	f1 := liveBrowser(t, "rvF1", srv.URL+"/")
	f2 := liveBrowser(t, "rvF2", srv.URL+"/")
	time.Sleep(800 * time.Millisecond)
	wsOf := func(id string) (string, bool) {
		if rs, ok := runReg.get(id); ok {
			return rs.WSURL, true
		}
		return "", false
	}
	name := func(s string) string { return s }
	if _, err := groupSync.Start("rvM", []string{"rvF1", "rvF2"}, wsOf, name); err != nil {
		t.Fatal(err)
	}
	defer groupSync.Stop("test end")
	mc, msid := pageSession(t, m)
	c1, s1 := pageSession(t, f1)
	c2, s2 := pageSession(t, f2)
	both := func(what, expr, want string) {
		t.Helper()
		waitFor(t, what+" f1", func() bool { return evalStr(t, c1, s1, expr) == want })
		waitFor(t, what+" f2", func() bool { return evalStr(t, c2, s2, expr) == want })
	}
	mouse := func(button string, x, y float64) {
		for _, ty := range []string{"mousePressed", "mouseReleased"} {
			if _, err := mc.call(msid, "Input.dispatchMouseEvent", map[string]any{"type": ty, "x": x, "y": y, "button": button, "clickCount": 1}); err != nil {
				t.Fatal(err)
			}
		}
	}
	key := func(k, code string, kc, mods int, text string) {
		ty := "rawKeyDown"
		p := map[string]any{"key": k, "code": code, "windowsVirtualKeyCode": kc, "modifiers": mods}
		if text != "" {
			ty = "keyDown"
			p["text"] = text
		}
		p["type"] = ty
		_, _ = mc.call(msid, "Input.dispatchKeyEvent", p)
		delete(p, "text")
		p["type"] = "keyUp"
		_, _ = mc.call(msid, "Input.dispatchKeyEvent", p)
	}

	// 1. Fast typing: 40 characters with no delay between keys.
	mouse("left", 100, 55)
	time.Sleep(200 * time.Millisecond)
	fast := "The quick brown fox jumps over 13 dogs!?"
	for _, ch := range fast {
		s := string(ch)
		key(s, "", int(strings.ToUpper(s)[0]), 0, s)
	}
	want := fast
	waitFor(t, "master typed", func() bool { return evalStr(t, mc, msid, "document.getElementById('i').value") == want })
	both("fast typing", "document.getElementById('i').value", want)

	// 2. IME composition (Chinese): compositionstart/update via imeSetComposition, commit via insertText.
	for _, step := range []string{"n", "ni", "nih", "niha", "nihao"} {
		_, _ = mc.call(msid, "Input.imeSetComposition", map[string]any{"text": step, "selectionStart": len(step), "selectionEnd": len(step)})
	}
	_, _ = mc.call(msid, "Input.insertText", map[string]any{"text": "你好"})
	want += "你好"
	waitFor(t, "master IME", func() bool { return evalStr(t, mc, msid, "document.getElementById('i').value") == want })
	both("IME", "document.getElementById('i').value", want)

	// 3. Backspace + Ctrl+A then a replacement character.
	key("Backspace", "Backspace", 8, 0, "")
	want = strings.TrimSuffix(want, "好")
	both("backspace", "document.getElementById('i').value", want)
	key("a", "KeyA", 65, 2, "")
	key("x", "KeyX", 88, 0, "x")
	both("ctrl+a replace", "document.getElementById('i').value", "x")

	// 4. Right-click: exactly one contextmenu on followers.
	mouse("right", 50, 135)
	waitFor(t, "master ctx", func() bool { return evalStr(t, mc, msid, "window.ctxn") == "1" })
	time.Sleep(800 * time.Millisecond)
	t.Logf("ctx counts f1=%s f2=%s", evalStr(t, c1, s1, "window.ctxn"), evalStr(t, c2, s2, "window.ctxn"))
	both("contextmenu once", "window.ctxn", "1")

	// 5. Page keyboard shortcuts (focus on body): followers' keydown listeners must fire.
	_, _ = mc.call(msid, "Runtime.evaluate", map[string]any{"expression": "document.activeElement.blur()"})
	_, _ = c1.call(s1, "Runtime.evaluate", map[string]any{"expression": "document.activeElement.blur()"})
	_, _ = c2.call(s2, "Runtime.evaluate", map[string]any{"expression": "document.activeElement.blur()"})
	time.Sleep(100 * time.Millisecond)
	key("j", "KeyJ", 74, 0, "j")
	key("k", "KeyK", 75, 0, "k")
	waitFor(t, "master shortcuts", func() bool { return evalStr(t, mc, msid, "window.keys") == "jk" })
	both("page shortcuts", "window.keys", "jk")

	// 6. Link click navigation: followers end on /page2 exactly once (no double load).
	mouse("left", 20, 85)
	both("link navigation", "location.pathname", "/page2")
	time.Sleep(1200 * time.Millisecond)
	both("nav history", "history.length", evalStr(t, mc, msid, "history.length"))

	// 7. Typing right before navigation on the new page.
	time.Sleep(300 * time.Millisecond)
	mouse("left", 30, 15)
	time.Sleep(150 * time.Millisecond)
	for _, ch := range "ok" {
		key(string(ch), "", int(strings.ToUpper(string(ch))[0]), 0, string(ch))
	}
	both("typing after nav", "document.getElementById('j').value", "ok")

	// 8. A follower closes mid-sync: the syncer keeps running for the other.
	cx, err := dialCDP(f2, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = cx.call("", "Browser.close", nil)
	waitFor(t, "f2 dropped", func() bool {
		st := groupSync.State()
		for _, f := range st.Followers {
			if f.ID == "rvF2" {
				return !f.OK
			}
		}
		return false
	})
	if !groupSync.State().Running {
		t.Fatal("syncer stopped when one of two followers closed")
	}
	key("!", "", 49, 8, "!")
	waitFor(t, "f1 still synced", func() bool { return evalStr(t, c1, s1, "document.getElementById('j').value") == "ok!" })

	// 9. Stop then restart with the remaining follower.
	groupSync.Stop("用户停止")
	if groupSync.State().Running {
		t.Fatal("still running after stop")
	}
	key("?", "", 191, 8, "?")
	time.Sleep(500 * time.Millisecond)
	if v := evalStr(t, c1, s1, "document.getElementById('j').value"); v != "ok!" {
		t.Fatalf("follower changed after stop: %q", v)
	}
	if _, err := groupSync.Start("rvM", []string{"rvF1", "rvF2"}, wsOf, name); err != nil {
		t.Fatal(err)
	}
	st := groupSync.State()
	t.Logf("restart state %+v", st)
	if !st.Running || len(st.Followers) != 1 || !strings.Contains(st.Reason, "rvF2") {
		t.Fatalf("restart state %+v", st)
	}
	time.Sleep(300 * time.Millisecond)
	key("z", "KeyZ", 90, 0, "z")
	time.Sleep(1500 * time.Millisecond)
	t.Logf("after restart: master=%q f1=%q state=%+v", evalStr(t, mc, msid, "document.getElementById('j').value"), evalStr(t, c1, s1, "document.getElementById('j').value+'|'+document.activeElement.id"), groupSync.State())
	waitFor(t, "synced after restart", func() bool { return evalStr(t, c1, s1, "document.getElementById('j').value") == "ok!z" })
}
