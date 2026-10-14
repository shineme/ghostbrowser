package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// Live recorder tests: a real Chromium profile, a local site, real input
// events through CDP Input.dispatch* (as a user would), then the recorded
// workflow replayed with RunWorkflowWith on a fresh profile.
//
// GHOST_CHROME=/usr/bin/chromium go test -run 'TestRecord.*Live' -v .

const recSitePage1 = `<!doctype html><html><head><meta charset="utf-8"><title>登录</title>
<style>body{font:16px sans-serif} input,select,button{display:block;margin:8px;padding:6px;width:220px} label{display:block;margin:8px}</style></head><body>
<form action="/p2" method="get">
  <input id="user" name="user" placeholder="用户名">
  <input type="password" name="pw" placeholder="密码">
  <select id="city" name="city"><option value="bj">北京</option><option value="sh">上海</option><option value="gz">广州</option></select>
  <label><input type="checkbox" name="remember" value="1"> 记住我</label>
  <input type="file" id="avatar" onchange="document.getElementById('fn').value = this.files[0] ? this.files[0].name : ''">
  <input type="hidden" name="fn" id="fn">
  <input type="hidden" name="more" id="more" value="0">
  <div id="slot"></div>
  <button type="submit" class="btn primary">登录</button>
</form>
<script>
setTimeout(() => {
  const b = document.createElement('button'); b.type = 'button'; b.className = 'late'; b.textContent = '加载更多';
  b.onclick = () => { document.getElementById('more').value = '1'; b.textContent = '已加载'; };
  document.getElementById('slot').appendChild(b);
}, 700);
</script></body></html>`

const recSitePage2 = `<!doctype html><html><head><meta charset="utf-8"><title>搜索</title></head><body>
<p id="hi">欢迎</p>
<form action="/p3" method="get"><input id="q" name="q" style="width:200px;padding:6px"></form>
</body></html>`

const recSitePage3 = `<!doctype html><html><head><meta charset="utf-8"><title>结果</title></head><body>
<button id="open" style="padding:8px" onclick="window.open('/tab', '_blank')">打开新窗口</button>
<div id="dbl" style="margin:20px;padding:20px;background:#eee;width:200px" ondblclick="this.textContent = '双击了'">双击这里</div>
</body></html>`

const recSiteTab = `<!doctype html><html><head><meta charset="utf-8"><title>新窗口</title></head><body>
<button id="hello" style="padding:8px" onclick="fetch('/hello')">打招呼</button>
</body></html>`

type recSite struct {
	*httptest.Server
	mu   sync.Mutex
	hits []string // path?query of every page request
}

func newRecSite() *recSite {
	s := &recSite{}
	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/favicon.ico" {
			http.NotFound(w, r)
			return
		}
		s.mu.Lock()
		s.hits = append(s.hits, r.URL.RequestURI())
		s.mu.Unlock()
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		switch r.URL.Path {
		case "/":
			fmt.Fprint(w, recSitePage1)
		case "/p2":
			fmt.Fprint(w, recSitePage2)
		case "/p3":
			fmt.Fprint(w, recSitePage3)
		case "/tab":
			fmt.Fprint(w, recSiteTab)
		default:
			fmt.Fprint(w, "ok")
		}
	}))
	return s
}

func (s *recSite) hit(prefix string) (string, int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	last, n := "", 0
	for _, h := range s.hits {
		if strings.HasPrefix(h, prefix) {
			last, n = h, n+1
		}
	}
	return last, n
}

func (s *recSite) reset() {
	s.mu.Lock()
	s.hits = nil
	s.mu.Unlock()
}

// cdpUser drives a tab with real input events.
type cdpUser struct {
	t   *testing.T
	c   *cdpConn
	sid string
	tid string
}

func newCDPUser(t *testing.T, ws string) *cdpUser {
	c, err := dialCDP(ws, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { c.c.Close() })
	return &cdpUser{t: t, c: c}
}

func (u *cdpUser) pageTargets() []struct{ TargetID, URL string } {
	raw, err := u.c.call("", "Target.getTargets", nil)
	if err != nil {
		u.t.Fatal(err)
	}
	var tl struct {
		TargetInfos []struct{ TargetID, Type, URL string } `json:"targetInfos"`
	}
	_ = json.Unmarshal(raw, &tl)
	out := []struct{ TargetID, URL string }{}
	for _, ti := range tl.TargetInfos {
		if ti.Type == "page" {
			out = append(out, struct{ TargetID, URL string }{ti.TargetID, ti.URL})
		}
	}
	return out
}

// use attaches to the page whose URL contains sub ("" = first page).
func (u *cdpUser) use(sub string) {
	u.t.Helper()
	deadline := time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) {
		for _, p := range u.pageTargets() {
			if sub == "" || strings.Contains(p.URL, sub) {
				raw, err := u.c.call("", "Target.attachToTarget", map[string]any{"targetId": p.TargetID, "flatten": true})
				if err != nil {
					u.t.Fatal(err)
				}
				var at struct{ SessionID string }
				_ = json.Unmarshal(raw, &at)
				u.sid, u.tid = at.SessionID, p.TargetID
				_, _ = u.c.call("", "Target.activateTarget", map[string]any{"targetId": p.TargetID})
				return
			}
		}
		time.Sleep(150 * time.Millisecond)
	}
	u.t.Fatalf("no tab with %q: %+v", sub, u.pageTargets())
}

func (u *cdpUser) eval(expr string, out any) {
	u.t.Helper()
	raw, err := u.c.call(u.sid, "Runtime.evaluate", map[string]any{"expression": expr, "returnByValue": true, "awaitPromise": true})
	if err != nil {
		u.t.Fatalf("eval %s: %v", expr, err)
	}
	var r struct {
		Result struct {
			Value json.RawMessage `json:"value"`
		} `json:"result"`
	}
	_ = json.Unmarshal(raw, &r)
	if out != nil {
		if err := json.Unmarshal(r.Result.Value, out); err != nil {
			u.t.Fatalf("eval %s: %s", expr, raw)
		}
	}
}

func (u *cdpUser) waitFor(sel string) {
	u.t.Helper()
	for i := 0; i < 80; i++ {
		var ok bool
		u.eval(`document.readyState === 'complete' && !!document.querySelector(`+jsq(sel)+`)`, &ok)
		if ok {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	u.t.Fatalf("element %s never appeared", sel)
}

func (u *cdpUser) mouse(sel string, clicks int) {
	u.t.Helper()
	u.waitFor(sel)
	var pt []float64
	u.eval(`(() => { const r = document.querySelector(`+jsq(sel)+`).getBoundingClientRect(); return [r.left + Math.min(r.width / 2, 30), r.top + r.height / 2]; })()`, &pt)
	_, _ = u.c.call(u.sid, "Input.dispatchMouseEvent", map[string]any{"type": "mouseMoved", "x": pt[0], "y": pt[1]})
	for n := 1; n <= clicks; n++ {
		_, _ = u.c.call(u.sid, "Input.dispatchMouseEvent", map[string]any{"type": "mousePressed", "x": pt[0], "y": pt[1], "button": "left", "buttons": 1, "clickCount": n})
		time.Sleep(90 * time.Millisecond) // a human press lasts ~100 ms
		_, _ = u.c.call(u.sid, "Input.dispatchMouseEvent", map[string]any{"type": "mouseReleased", "x": pt[0], "y": pt[1], "button": "left", "clickCount": n})
		time.Sleep(60 * time.Millisecond)
	}
	time.Sleep(200 * time.Millisecond)
}

func (u *cdpUser) typeText(s string) {
	for _, r := range s {
		ch := string(r)
		_, _ = u.c.call(u.sid, "Input.dispatchKeyEvent", map[string]any{"type": "keyDown", "text": ch, "unmodifiedText": ch, "key": ch})
		_, _ = u.c.call(u.sid, "Input.dispatchKeyEvent", map[string]any{"type": "keyUp", "key": ch})
		time.Sleep(40 * time.Millisecond)
	}
	time.Sleep(150 * time.Millisecond)
}

func (u *cdpUser) key(key string, vk int, text string) {
	down := map[string]any{"type": "rawKeyDown", "key": key, "code": key, "windowsVirtualKeyCode": vk, "nativeVirtualKeyCode": vk}
	if text != "" {
		down["type"], down["text"] = "keyDown", text
	}
	_, _ = u.c.call(u.sid, "Input.dispatchKeyEvent", down)
	_, _ = u.c.call(u.sid, "Input.dispatchKeyEvent", map[string]any{"type": "keyUp", "key": key, "code": key, "windowsVirtualKeyCode": vk, "nativeVirtualKeyCode": vk})
	time.Sleep(200 * time.Millisecond)
}

func recSteps(steps []RpaStep) []string {
	out := []string{}
	for _, s := range steps {
		v := s.Type
		switch s.Type {
		case stGoto:
			v += " " + s.Params["url"]
		case stClick, stWaitFor:
			v += " " + s.Params["selector"]
			if s.Params["double"] == "true" {
				v += " ×2"
			}
		case stFill:
			v += " " + s.Params["selector"] + " = " + s.Params["text"]
		case stUpload:
			v += " " + s.Params["selector"] + " ← " + s.Params["files"]
		case stSelectTab, stCloseTab:
			v += " " + s.Params["index"]
		case stExecuteJS:
			v += " " + s.Name
		case stDelay:
			v += " " + s.Params["ms"]
		}
		out = append(out, v)
	}
	return out
}

func TestRecordReplayLive(t *testing.T) {
	bin := os.Getenv("GHOST_CHROME")
	if bin == "" {
		t.Skip("set GHOST_CHROME to a Chromium binary")
	}
	site := newRecSite()
	defer site.Close()
	a := newRpaE2EApp(t, bin)
	mk := func(name string) string {
		fp := GenerateTyped("desktop", "windows", "")
		fp.StartupParameters = "--headless=new --no-sandbox"
		return a.store.PutProfile(Profile{Name: name, Kernel: "chromium", Fingerprint: fp}).ID
	}
	recPID, playPID := mk("录制环境"), mk("回放环境")
	defer runReg.stop(recPID)
	defer runReg.stop(playPID)

	var evMu sync.Mutex
	events := 0
	st, err := a.StartRecording(recPID, RecordOptions{Overlay: true, PasswordVar: true})
	if err != nil {
		t.Fatal(err)
	}
	if !st.Active || st.ProfileName != "录制环境" {
		t.Fatalf("state: %+v", st)
	}
	if _, err := a.StartRecording(recPID, RecordOptions{}); err == nil {
		t.Fatal("second recording should be refused")
	}
	recordings.setEmit(func(RecordState) { evMu.Lock(); events++; evMu.Unlock() })
	rs, _ := runReg.get(recPID)
	u := newCDPUser(t, rs.WSURL)
	u.use("")
	time.Sleep(300 * time.Millisecond)

	// ---- page 1: typed URL, login form ----
	_, _ = u.c.call(u.sid, "Page.navigate", map[string]any{"url": site.URL + "/"}) // like typing in the address bar
	u.waitFor("#user")
	time.Sleep(300 * time.Millisecond)
	var overlay bool
	u.eval(`(() => { const h = document.documentElement.lastElementChild; return !!h && h.tagName === 'DIV' && h.shadowRoot === null && typeof window.__gbRec === 'undefined'; })()`, &overlay)
	if !overlay {
		t.Fatal("overlay host missing, or its shadow root / recorder state leaks to the page")
	}
	u.mouse("#user", 1)
	u.typeText("alice")
	u.mouse(`input[name=pw]`, 1)
	u.typeText("s3cret")
	u.key("Tab", 9, "")
	u.key("ArrowDown", 40, "") // 北京 → 上海
	u.mouse("label", 1)        // 记住我
	tmp := filepath.Join(t.TempDir(), "avatar.png")
	_ = os.WriteFile(tmp, []byte("png"), 0o644)
	var obj struct {
		Result struct{ ObjectID string } `json:"result"`
	}
	raw, _ := u.c.call(u.sid, "Runtime.evaluate", map[string]any{"expression": "document.querySelector('#avatar')"})
	_ = json.Unmarshal(raw, &obj)
	if _, err := u.c.call(u.sid, "DOM.setFileInputFiles", map[string]any{"files": []string{tmp}, "objectId": obj.Result.ObjectID}); err != nil {
		t.Fatal(err)
	}
	time.Sleep(250 * time.Millisecond)
	u.mouse(".late", 1) // appeared 700 ms after load
	u.mouse("button[type=submit]", 1)

	// ---- page 2: Enter submits the search ----
	u.waitFor("#q")
	u.mouse("#q", 1)
	u.typeText("幽灵")
	u.key("Enter", 13, "\r")

	// ---- page 3: new tab, work there, close it, double click ----
	u.waitFor("#open")
	time.Sleep(200 * time.Millisecond)
	u.mouse("#open", 1)
	main := *u
	u.use("/tab")
	u.mouse("#hello", 1)
	time.Sleep(200 * time.Millisecond)
	_, _ = u.c.call("", "Target.closeTarget", map[string]any{"targetId": u.tid})
	time.Sleep(400 * time.Millisecond)
	*u = main
	_, _ = u.c.call("", "Target.activateTarget", map[string]any{"targetId": u.tid})
	u.mouse("#dbl", 2)
	time.Sleep(300 * time.Millisecond)

	// pause: nothing is recorded meanwhile
	a.PauseRecording()
	u.mouse("#open", 1)
	time.Sleep(300 * time.Millisecond)
	if st := a.ResumeRecording(); st.Paused || !st.Active {
		t.Fatalf("resume: %+v", st)
	}
	final := a.StopRecording()
	if final.Active {
		t.Fatal("still active after stop")
	}
	var left bool
	u.eval(`!!document.documentElement.lastElementChild && document.documentElement.lastElementChild.tagName === 'DIV' && !document.documentElement.lastElementChild.id && document.documentElement.lastElementChild.attachShadow !== undefined && document.querySelectorAll('html > div').length > 0`, &left)
	if left {
		t.Fatal("overlay still on the page after stop")
	}
	evMu.Lock()
	t.Logf("%d rpa:record events", events)
	evMu.Unlock()
	f := a.FinalizeRecording(final.Steps, RecordFinalizeOptions{MergeFills: true, PasswordVar: true})
	got := recSteps(f.Steps)
	t.Logf("recorded %d steps:\n  %s", len(got), strings.Join(got, "\n  "))
	clean := func(s string) string { // drop the fallback xpath for comparison
		if i := strings.Index(s, " || "); i >= 0 {
			return s[:i]
		}
		return s
	}
	want := []string{
		"goto " + site.URL + "/",
		"fill #user = alice",
		`fill input[name="pw"] = {{password}}`,
		"fill #city = sh",
		"click text:记住我",
		"upload #avatar ← {{upload_file}}",
		"wait_for text:加载更多",
		"click text:加载更多",
		"click text:登录",
		"fill #q = 幽灵",
		"execute_js 按回车键",
		"click #open",
		"select_tab 2",
		"click #hello",
		"close_tab 2",
		"click #dbl ×2",
	}
	if len(got) != len(want) {
		t.Fatalf("got %d steps, want %d", len(got), len(want))
	}
	for i := range want {
		if clean(got[i]) != want[i] {
			t.Fatalf("step %d: got %q want %q", i+1, got[i], want[i])
		}
	}
	if f.Vars["password"] != "" || len(f.Vars) != 2 {
		t.Fatalf("vars: %v", f.Vars)
	}
	if strings.Contains(fmt.Sprint(final), "s3cret") {
		t.Fatal("the real password leaked into the recording state")
	}

	// ---- replay on a fresh profile ----
	site.reset()
	wf, err := a.SaveWorkflow(Workflow{Name: "录制回放", Steps: f.Steps, Vars: map[string]string{"password": "s3cret", "upload_file": tmp}})
	if err != nil {
		t.Fatal(err)
	}
	if issues := a.ValidateWorkflow(wf); len(issues) != 0 {
		t.Fatalf("issues: %+v", issues)
	}
	runID, err := a.RunWorkflowWith(wf.ID, []string{playPID}, RpaRunOptions{Concurrency: 1, AutoStart: true})
	if err != nil {
		t.Fatal(err)
	}
	run := waitRun(t, a, runID, 90*time.Second)
	if run.Status != "ok" {
		for _, s := range run.Targets[0].Steps {
			t.Logf("  %s %s %s", s.Type, s.Status, s.Log)
		}
		t.Fatalf("replay %s: %s", run.Status, run.Targets[0].Error)
	}
	p2, _ := site.hit("/p2?")
	q, _ := url.ParseQuery(strings.TrimPrefix(p2, "/p2?"))
	if q.Get("user") != "alice" || q.Get("pw") != "s3cret" || q.Get("city") != "sh" || q.Get("remember") != "1" || q.Get("fn") != "avatar.png" || q.Get("more") != "1" {
		t.Fatalf("login form submitted as %q", p2)
	}
	if p3, n := site.hit("/p3?"); n != 1 || p3 != "/p3?q="+url.QueryEscape("幽灵") {
		t.Fatalf("search submitted %d× as %q", n, p3)
	}
	if _, n := site.hit("/hello"); n != 1 {
		t.Fatalf("new tab button clicked %d×", n)
	}
	prs, _ := runReg.get(playPID)
	pu := newCDPUser(t, prs.WSURL)
	tabs := pu.pageTargets()
	if len(tabs) != 1 || !strings.Contains(tabs[0].URL, "/p3?") {
		t.Fatalf("replay tabs: %+v", tabs)
	}
	pu.use("/p3")
	var txt string
	pu.eval(`document.querySelector('#dbl').textContent`, &txt)
	if txt != "双击了" {
		t.Fatalf("double click not replayed: %q", txt)
	}
	t.Logf("replay ok: %s", p2)
}

const recSelectorPage = `<!doctype html><html><head><meta charset="utf-8"></head><body>
<input id="email" class="x">
<div id="react-1234567"><button class="css-1x2y3z">无名</button></div>
<button data-testid="save">保存</button>
<form><input name="q"></form>
<button aria-label="关闭">×</button>
<a href="#">查看详情</a>
<ul class="list"><li class="item"><span>甲</span></li><li class="item"><span>乙</span></li></ul>
<label><input type="radio" name="gender" value="m">男</label><label><input type="radio" name="gender" value="f">女</label>
<input placeholder="搜索商品">
<div class="card"><button class="buy">购买</button></div><div class="card"><button class="buy">购买</button></div>
<span data-role="tip">提示</span>
<svg id="ic" width="10" height="10"><circle r="4"></circle></svg>
</body></html>`

// TestRecordSelectorsLive checks the capture script's selector generator
// against the replay driver's resolver on a fixture page.
func TestRecordSelectorsLive(t *testing.T) {
	bin := os.Getenv("GHOST_CHROME")
	if bin == "" {
		t.Skip("set GHOST_CHROME to a Chromium binary")
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, recSelectorPage)
	}))
	defer srv.Close()
	a := newRpaE2EApp(t, bin)
	fp := GenerateTyped("desktop", "windows", "")
	fp.StartupParameters = "--headless=new --no-sandbox"
	pid := a.store.PutProfile(Profile{Name: "sel", Kernel: "chromium", Fingerprint: fp}).ID
	rs, err := a.StartProfile(pid)
	if err != nil {
		t.Fatal(err)
	}
	defer runReg.stop(pid)
	u := newCDPUser(t, rs.WSURL)
	u.use("")
	_, _ = u.c.call(u.sid, "Page.navigate", map[string]any{"url": srv.URL})
	u.waitFor("#ic")
	// install the capture script in an isolated world of our own session
	ctx, err := recWorldCtx(u.c, u.sid, u.tid)
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct{ find, want string }{
		{"#email", "#email"},
		{"#react-1234567 button", "text:无名"},
		{"[data-testid=save]", `[data-testid="save"]`},
		{"input[name=q]", `input[name="q"]`},
		{"button[aria-label]", `button[aria-label="关闭"]`},
		{"a", "text:查看详情"},
		{"li:nth-child(2) span", "li.item:nth-of-type(2) > span"},
		{"input[value=f]", `input[name="gender"][value="f"]`},
		{"input[placeholder]", `input[placeholder="搜索商品"]`},
		{".card:nth-of-type(3) .buy", "div.card:nth-of-type(3) > button.buy"},
		{".card:nth-of-type(2) .buy", "text:购买"},
		{"span[data-role]", `span[data-role="tip"]`},
	}
	for _, c := range cases {
		raw, err := u.c.call(u.sid, "Runtime.evaluate", map[string]any{"contextId": ctx, "returnByValue": true,
			"expression": recCaptureScript(false, false) + "; window.__gbRec.sel(document.querySelector(" + jsq(c.find) + "))"})
		if err != nil {
			t.Fatal(err)
		}
		var r struct {
			Result struct{ Value string } `json:"result"`
		}
		_ = json.Unmarshal(raw, &r)
		sel := r.Result.Value
		primary := sel
		if i := strings.Index(sel, " || "); i >= 0 {
			primary = sel[:i]
			if !strings.HasPrefix(sel[i+4:], "xpath:") {
				t.Fatalf("%s: fallback is not xpath: %q", c.find, sel)
			}
		}
		if primary != c.want {
			t.Fatalf("%s: got %q want %q", c.find, sel, c.want)
		}
		// the replay resolver finds the same element with the full selector,
		// and with the fallback alone when the primary breaks
		var same []bool
		u.eval(`(() => {`+rpaFind+` const want = document.querySelector(`+jsq(c.find)+`); const s = `+jsq(sel)+`;
			const fb = s.includes(' || ') ? __gbq(s.split(' || ')[1]) : want;
			const hit = __gbq(s); return [hit === want || (!!hit && want.contains(hit)), fb === want]; })()`, &same)
		if !same[0] || !same[1] {
			t.Fatalf("%s: selector %q resolves elsewhere (%v)", c.find, sel, same)
		}
	}
	// a broken primary falls back to the xpath
	var ok bool
	u.eval(`(() => {`+rpaFind+` return __gbq('#gone || xpath://*[@id="email"]') === document.querySelector('#email'); })()`, &ok)
	if !ok {
		t.Fatal("fallback selector not used")
	}
}
