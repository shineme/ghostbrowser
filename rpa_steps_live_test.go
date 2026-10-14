package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Live coverage for every RPA step type against real Chromium. Each subtest
// asserts the effect inside the page, not just that the step returned nil.
//
// GHOST_CHROME=/usr/bin/chromium go test -run TestRpaStepsLive

const rpaStepsPage = `<!doctype html><html><head><meta charset="utf-8"><title>steps</title>
<style>body{margin:0;font:16px sans-serif} #box{height:120px;overflow:auto;border:1px solid #000} .tall{height:5000px}</style></head><body>
<div id="mark">A</div>
<input id="name"><textarea id="ta">old</textarea><div id="ce" contenteditable="true">x</div>
<select id="sel"><option value="a">苹果</option><option value="b">香蕉</option></select>
<input type="file" id="f" multiple onchange="window.files=[...this.files].map(f=>f.name+':'+f.size)">
<button id="dbl" ondblclick="window.nDbl=(window.nDbl||0)+1" onclick="window.nClk=(window.nClk||0)+1">dbl</button>
<button id="covered" onclick="window.nCov=1">cov</button><div style="position:absolute;left:0;top:0;width:100%;height:1px"></div>
<div id="spinner">loading</div><div id="gone">bye</div><div id="later" style="display:none">ready</div>
<a id="blank" href="/b?tab=2" target="_blank">new tab</a>
<iframe id="frm" srcdoc="<p id='inframe'>inside</p>"></iframe>
<div id="box"><div style="height:2000px"></div><p id="deep">deep</p></div>
<div class="tall"></div><p id="far">far</p>
<script>
document.getElementById('box').addEventListener('scroll', () => window.boxScrolled = document.getElementById('box').scrollTop);
setTimeout(() => { document.getElementById('spinner').style.display = 'none'; document.getElementById('gone').remove(); document.getElementById('later').style.display = 'block'; }, 600);
</script></body></html>`

func rpaStepsServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		switch r.URL.Path {
		case "/b":
			// Same selector as the first page so a stale read would be caught.
			fmt.Fprint(w, `<!doctype html><title>B</title><div id="mark">B</div><div id="tab">`+r.URL.Query().Get("tab")+`</div>`)
		case "/slow":
			time.Sleep(700 * time.Millisecond)
			fmt.Fprint(w, `<!doctype html><title>slow</title><div id="mark">SLOW</div>`)
		default:
			fmt.Fprint(w, rpaStepsPage)
		}
	}))
}

func startLiveChrome(t *testing.T) (ws string, kill func()) {
	t.Helper()
	bin := os.Getenv("GHOST_CHROME")
	if bin == "" {
		t.Skip("set GHOST_CHROME to a Chromium binary")
	}
	// Not t.TempDir: Chromium helpers may still write to the profile after
	// the kill, which makes its strict cleanup fail the test.
	ud, err := os.MkdirTemp("", "rpa-live-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { time.Sleep(300 * time.Millisecond); _ = os.RemoveAll(ud) })
	port := freePort()
	cmd := exec.Command(bin, "--headless=new", "--no-sandbox", "--window-size=1000,700", "--user-data-dir="+ud, fmt.Sprintf("--remote-debugging-port=%d", port), "about:blank")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	kill = func() { _ = cmd.Process.Kill(); _ = cmd.Wait() }
	ws, err = waitDebugger(port, 20*time.Second)
	if err != nil {
		kill()
		t.Fatal(err)
	}
	return ws, kill
}

func TestRpaStepsLive(t *testing.T) {
	ws, kill := startLiveChrome(t)
	defer kill()
	srv := rpaStepsServer()
	defer srv.Close()
	d, err := newCDPDriver(ws)
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	js := func(code string) string {
		t.Helper()
		v, err := d.Eval(code, nil)
		if err != nil {
			t.Fatalf("eval %s: %v", code, err)
		}
		return v
	}
	exec := func(t *testing.T, steps ...RpaStep) (map[string]string, [][]byte) {
		t.Helper()
		var shots [][]byte
		var failed []string
		vars, err := runRpa(context.Background(), Workflow{Name: "t", Steps: steps}, d, func(e RpaStepEvent) {
			if e.Status == "failed" {
				failed = append(failed, e.Type+": "+e.Log)
			}
		}, func(_ int, b []byte) (string, error) { shots = append(shots, b); return fmt.Sprintf("s%d.png", len(shots)), nil })
		if err != nil {
			t.Fatalf("run: %v %v", err, failed)
		}
		return vars, shots
	}
	home := srv.URL + "/"

	t.Run("goto", func(t *testing.T) {
		exec(t, st(stGoto, "url", home))
		// Navigating to a page that has the same selector must not read the old document.
		vars, _ := exec(t, st(stGoto, "url", srv.URL+"/slow"), st(stExtractText, "selector", "#mark", "var", "m"))
		if vars["m"] != "SLOW" {
			t.Fatalf("goto returned before the new page loaded: #mark = %q", vars["m"])
		}
		exec(t, st(stGoto, "url", home))
		if js("document.title") != "steps" {
			t.Fatal("not on home")
		}
	})
	t.Run("fill", func(t *testing.T) {
		exec(t, st(stFill, "selector", "#name", "text", "张三"), st(stFill, "selector", "#ta", "text", " 追加", "clear", "false"),
			st(stFill, "selector", "#ce", "text", "富文本"), st(stFill, "selector", "#sel", "text", "香蕉"))
		got := js("['name','ta','ce','sel'].map(i => { const e = document.getElementById(i); return e.isContentEditable ? e.innerText : e.value; }).join('|')")
		if got != "张三|old 追加|富文本|b" {
			t.Fatalf("fill = %q", got)
		}
	})
	t.Run("click_double", func(t *testing.T) {
		exec(t, st(stClick, "selector", "#dbl", "double", "true"))
		if got := js("(window.nDbl||0) + ':' + (window.nClk||0)"); got != "1:2" {
			t.Fatalf("dblclick:click = %q, want 1:2", got)
		}
		exec(t, st(stClick, "selector", "xpath://button[@id='covered']"))
		if js("String(window.nCov)") != "1" {
			t.Fatal("xpath click did not fire")
		}
	})
	t.Run("upload", func(t *testing.T) {
		dir := t.TempDir()
		a, b := filepath.Join(dir, "主图.txt"), filepath.Join(dir, "b.bin")
		_ = os.WriteFile(a, []byte("hello"), 0o644)
		_ = os.WriteFile(b, make([]byte, 1234), 0o644)
		exec(t, st(stUpload, "selector", "#f", "files", a+"\n"+b))
		if got := js("JSON.stringify(window.files)"); got != `["主图.txt:5","b.bin:1234"]` {
			t.Fatalf("files = %s", got)
		}
		_, _, err := run(t, d, st(stUpload, "selector", "#f", "files", filepath.Join(dir, "missing.txt")))
		if err == nil || !strings.Contains(err.Error(), "文件不存在") {
			t.Fatalf("missing file err = %v", err)
		}
	})
	t.Run("wait_for", func(t *testing.T) {
		exec(t, st(stGoto, "url", home))
		t0 := time.Now()
		exec(t, st(stWaitFor, "selector", "#spinner", "state", "hidden", "timeout", "5000"),
			st(stWaitFor, "selector", "#gone", "state", "hidden", "timeout", "5000"),
			st(stWaitFor, "selector", "#later", "timeout", "5000"))
		if el := time.Since(t0); el < 400*time.Millisecond {
			t.Fatalf("wait_for returned too early (%v)", el)
		}
		if js("String(!!document.getElementById('gone'))") != "false" || js("getComputedStyle(later).display") != "block" {
			t.Fatal("page state not reached")
		}
		exec(t, st(stWaitFor, "selector", "#name", "state", "present", "timeout", "1000"))
		_, _, err := run(t, d, st(stWaitFor, "selector", "#nope", "timeout", "600"))
		if err == nil || !strings.Contains(err.Error(), "超时") {
			t.Fatalf("timeout err = %v", err)
		}
		_, _, err = run(t, d, st(stWaitFor, "selector", "##bad[", "timeout", "600"))
		if err == nil || !strings.Contains(err.Error(), "选择器写法有误") {
			t.Fatalf("bad selector err = %v", err)
		}
	})
	t.Run("delay", func(t *testing.T) {
		t0 := time.Now()
		exec(t, st(stDelay, "ms", "300", "maxMs", "400"))
		if el := time.Since(t0); el < 300*time.Millisecond || el > 1500*time.Millisecond {
			t.Fatalf("delay took %v", el)
		}
	})
	t.Run("scroll", func(t *testing.T) {
		exec(t, st(stScroll, "mode", "by", "y", "800"))
		if y := js("String(Math.round(scrollY))"); y != "800" {
			t.Fatalf("scroll by: scrollY = %s", y)
		}
		exec(t, st(stScroll, "mode", "to", "x", "0", "y", "100"))
		if y := js("String(Math.round(scrollY))"); y != "100" {
			t.Fatalf("scroll to: scrollY = %s", y)
		}
		exec(t, st(stScroll, "mode", "el", "selector", "#far"))
		if v := js("const r = far.getBoundingClientRect(); String(r.top >= 0 && r.bottom <= innerHeight && scrollY > 4000)"); v != "true" {
			t.Fatalf("scroll to #far: in view = %s scrollY=%s", v, js("String(scrollY)"))
		}
		// element inside a scrollable container
		exec(t, st(stScroll, "mode", "el", "selector", "#deep"))
		if v := js("String(box.scrollTop > 1500)"); v != "true" {
			t.Fatalf("container scrollTop = %s", js("String(box.scrollTop)"))
		}
		// A leftover selector must not override "by" mode (the editor keeps hidden fields).
		exec(t, st(stScroll, "mode", "to", "y", "0"), st(stScroll, "mode", "by", "y", "300", "selector", "#far"))
		if y := js("String(Math.round(scrollY))"); y != "300" {
			t.Fatalf("mode by with stale selector: scrollY = %s", y)
		}
	})
	t.Run("extract_text", func(t *testing.T) {
		vars, _ := exec(t, st(stFill, "selector", "#name", "text", "张三"), st(stExtractText, "selector", "#name", "attr", "value", "var", "v"),
			st(stExtractText, "selector", "text:new tab", "attr", "href", "var", "h"),
			st(stExtractText, "selector", "#mark", "attr", "html", "var", "html"))
		if vars["v"] != "张三" || vars["h"] != "/b?tab=2" || vars["html"] != "A" {
			t.Fatalf("vars = %v", vars)
		}
	})
	t.Run("execute_js", func(t *testing.T) {
		vars, _ := exec(t, st(stExecuteJS, "code", "await new Promise(r=>setTimeout(r,50)); return {n: 1}", "saveAs", "o"),
			st(stExecuteJS, "code", "1+1", "saveAs", "two"),
			// an expression whose only return is inside a nested function
			st(stExecuteJS, "code", "['a','b'].map(x => { return x.toUpperCase() }).join('')", "saveAs", "up"),
			st(stExecuteJS, "code", "if (vars.two === '2') { return 'yes' }\nreturn 'no'", "saveAs", "body"))
		if vars["o"] != `{"n":1}` || vars["two"] != "2" || vars["up"] != "AB" || vars["body"] != "yes" {
			t.Fatalf("vars = %v", vars)
		}
		_, _, err := run(t, d, st(stExecuteJS, "code", "throw new Error('坏了')"))
		if err == nil || !strings.Contains(err.Error(), "坏了") {
			t.Fatalf("js error = %v", err)
		}
	})
	t.Run("screenshot", func(t *testing.T) {
		_, shots := exec(t, st(stScreenshot), st(stScreenshot, "fullPage", "true"))
		if len(shots) != 2 {
			t.Fatalf("shots = %d", len(shots))
		}
		vp, err := png.DecodeConfig(bytes.NewReader(shots[0]))
		if err != nil {
			t.Fatal(err)
		}
		full, err := png.DecodeConfig(bytes.NewReader(shots[1]))
		if err != nil {
			t.Fatal(err)
		}
		t.Logf("viewport %dx%d, full page %dx%d", vp.Width, vp.Height, full.Width, full.Height)
		if full.Height < 5000 || full.Height <= vp.Height {
			t.Fatalf("full page png height %d (viewport %d)", full.Height, vp.Height)
		}
	})
	t.Run("tabs", func(t *testing.T) {
		exec(t, st(stGoto, "url", home), st(stClick, "selector", "#blank"))
		time.Sleep(500 * time.Millisecond)
		vars, _ := exec(t, st(stSelectTab, "index", "2"), st(stWaitFor, "selector", "#tab", "timeout", "5000"),
			st(stExtractText, "selector", "#tab", "var", "tab"), st(stCloseTab), st(stExecuteJS, "code", "document.title", "saveAs", "back"))
		if vars["tab"] != "2" || vars["back"] != "steps" {
			t.Fatalf("vars = %v", vars)
		}
		if n := len(d.(*cdpDriver).tabs()); n != 1 {
			t.Fatalf("tabs left = %d", n)
		}
	})
	t.Run("condition_loop_break", func(t *testing.T) {
		vars, _ := exec(t,
			withElse(withKids(st(stCondition, "kind", "exists", "selector", "#nope"), st(stExecuteJS, "code", "'then'", "saveAs", "br")),
				st(stExecuteJS, "code", "'else'", "saveAs", "br")),
			st(stExecuteJS, "code", "window.n = 0; return ''"),
			withKids(st(stLoop, "mode", "while", "kind", "js", "code", "return window.n < 100", "max", "50", "indexVar", "i"),
				st(stExecuteJS, "code", "window.n++; return String(window.n)", "saveAs", "n"),
				withKids(st(stCondition, "kind", "equals", "left", "{{i}}", "right", "4"), st(stBreakLoop))),
			withKids(st(stLoop, "count", "3", "indexVar", "k"), st(stFill, "selector", "#name", "text", "第{{k}}轮")),
		)
		if vars["br"] != "else" || vars["n"] != "4" || js("document.getElementById('name').value") != "第3轮" {
			t.Fatalf("vars = %v name=%q", vars, js("document.getElementById('name').value"))
		}
	})
	t.Run("iframe_not_reached", func(t *testing.T) {
		// Documented limitation: selectors act on the top document only.
		ok, err := d.Exists("#inframe")
		if err != nil || ok {
			t.Fatalf("iframe element visible to top-level selector? ok=%v err=%v", ok, err)
		}
	})
}

func withElse(s RpaStep, kids ...RpaStep) RpaStep { s.Else = kids; return s }

// Cancelling a run must interrupt a long wait at once, and a browser that
// dies mid-run must fail the step quickly with a Chinese message.
func TestRpaInterruptLive(t *testing.T) {
	ws, kill := startLiveChrome(t)
	defer kill()
	srv := rpaStepsServer()
	defer srv.Close()

	d, err := newCDPDriver(ws)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(600*time.Millisecond, cancel)
	t0 := time.Now()
	_, err = runRpa(ctx, Workflow{Name: "c", Steps: []RpaStep{st(stGoto, "url", srv.URL), st(stWaitFor, "selector", "#never", "timeout", "30000")}}, d, nil, nil)
	if !errors.Is(err, context.Canceled) || time.Since(t0) > 3*time.Second {
		t.Fatalf("cancel: err=%v after %v", err, time.Since(t0))
	}
	d.Close()

	// The user closes the tab being driven while a step waits on it.
	raw, err := dialCDP(ws, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = raw.call("", "Target.createTarget", map[string]any{"url": srv.URL + "/b"})
	time.Sleep(300 * time.Millisecond)
	d3, err := newCDPDriver(ws)
	if err != nil {
		t.Fatal(err)
	}
	victim := d3.(*cdpDriver).target
	time.AfterFunc(500*time.Millisecond, func() { _, _ = raw.call("", "Target.closeTarget", map[string]any{"targetId": victim}) })
	t0 = time.Now()
	_, err = runRpa(context.Background(), Workflow{Name: "tab", Steps: []RpaStep{st(stWaitFor, "selector", "#never", "timeout", "30000")}}, d3, nil, nil)
	if !errors.Is(err, errTabGone) || time.Since(t0) > 5*time.Second {
		t.Fatalf("tab closed mid-wait: err=%v after %v", err, time.Since(t0))
	}
	t.Logf("tab closed mid-run: %v after %v", err, time.Since(t0).Round(time.Millisecond))
	d3.Close()
	raw.c.Close()

	d2, err := newCDPDriver(ws)
	if err != nil {
		t.Fatal(err)
	}
	defer d2.Close()
	// like the user closing the window (Browser.close), not a SIGKILL whose
	// orphaned helpers can keep the socket open for a while
	time.AfterFunc(600*time.Millisecond, func() { _ = closeBrowser(ws) })
	t0 = time.Now()
	var last RpaStepEvent
	_, err = runRpa(context.Background(), Workflow{Name: "k", Steps: []RpaStep{st(stGoto, "url", srv.URL), st(stWaitFor, "selector", "#never", "timeout", "30000")}}, d2,
		func(e RpaStepEvent) { last = e; t.Logf("%v %s %s %s", time.Since(t0).Round(time.Millisecond), e.Type, e.Status, e.Log) }, nil)
	if err == nil || time.Since(t0) > 4*time.Second {
		t.Fatalf("browser closed: err=%v after %v", err, time.Since(t0))
	}
	if !strings.Contains(last.Log, "浏览器已关闭") {
		t.Fatalf("browser closed log = %q", last.Log)
	}
	t.Logf("closed mid-run: %v", err)
}
