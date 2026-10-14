package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image/png"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"
)

// Live iframe coverage against real Chromium with site isolation, so the
// cross-origin frames are real out-of-process iframes (OOPIFs).
//
// GHOST_CHROME=/usr/bin/chromium go test -run 'IframeLive' .
//
// Two servers on different hosts: the top page is served from 127.0.0.1, the
// cross-origin child from localhost (a different site).

// ifSite is the two-host fixture. Children report what they received to
// hits (keyed by the child's name), so tests check the server-side end state.
type ifSite struct {
	top, cross *httptest.Server
	crossBase  string
	mu         sync.Mutex
	hits       map[string]string
}

const ifChild = `<!doctype html><html><head><meta charset="utf-8"><style>body{margin:4px;font:14px sans-serif}</style></head><body>
<input id="name" style="width:120px"><button id="go">提交</button><input type="file" id="f" style="width:90px">
<p id="out" style="margin:2px 0;background:#0a0;color:#fff;width:80px;height:20px">-</p>
<script>window.pageVar = 'pv-%[1]s';
document.getElementById('go').addEventListener('click', ev => {
  const v = document.getElementById('name').value;
  document.getElementById('out').textContent = 'done:' + v + ':' + ev.isTrusted;
  fetch('/hit?n=%[1]s&v=' + encodeURIComponent(v) + '&t=' + ev.isTrusted);
});
document.getElementById('f').addEventListener('change', ev => fetch('/hit?n=%[1]s-file&v=' + encodeURIComponent([...ev.target.files].map(f => f.name).join(','))));
</script></body></html>`

func newIfSite() *ifSite {
	s := &ifSite{hits: map[string]string{}}
	h := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		q := r.URL.Query()
		switch r.URL.Path {
		case "/hit", "/child":
			if r.Header.Get("Accept-Language") != "" {
				s.mu.Lock()
				s.hits["#lang"] = r.Header.Get("Accept-Language")
				s.mu.Unlock()
			}
		}
		switch r.URL.Path {
		case "/hit":
			s.mu.Lock()
			s.hits[q.Get("n")] = q.Get("v") + "|" + q.Get("t")
			s.mu.Unlock()
		case "/child":
			fmt.Fprintf(w, ifChild, q.Get("n"))
		case "/nest":
			fmt.Fprintf(w, `<!doctype html><body style="margin:6px"><p id="mid">middle</p><iframe name="inner" src="%s/child?n=nest" style="width:380px;height:90px;border:3px solid #444"></iframe></body>`, s.crossBase)
		default:
			fmt.Fprint(w, s.topPage())
		}
	}
	s.cross = httptest.NewServer(http.HandlerFunc(h))
	_, port, _ := net.SplitHostPort(s.cross.Listener.Addr().String())
	s.crossBase = "http://localhost:" + port
	s.top = httptest.NewServer(http.HandlerFunc(h))
	return s
}

func (s *ifSite) topPage() string {
	c := s.crossBase
	return `<!doctype html><html><head><meta charset="utf-8"><title>top</title><style>body{margin:8px;font:14px sans-serif} iframe{width:400px;height:80px;display:block}</style></head><body>
<p id="topmark">TOP</p><input id="topin">
<iframe id="same" src="/child?n=same"></iframe>
<iframe id="cross" title="支付" src="` + c + `/child?n=cross"></iframe>
<iframe id="nest" src="/nest" style="height:120px"></iframe>
<iframe id="sd" srcdoc="<input id=name><button id=go onclick=&quot;document.getElementById('out').textContent='done:'+document.getElementById('name').value+':'+event.isTrusted&quot;>go</button><p id=out>-</p><script>window.pageVar='pv-sd'</script>"></iframe>
<div id="later"></div>
<div style="height:40px"></div>
<div id="box" style="height:160px;overflow:auto;margin-left:60px;border:1px solid #888"><div style="height:500px"></div>
  <iframe id="deep" src="` + c + `/child?n=deep" style="border:7px solid #333;padding:5px 9px;margin-left:30px"></iframe><div style="height:300px"></div></div>
<div style="height:1500px"></div>
<script>setTimeout(() => { const f = document.createElement('iframe'); f.id = 'late'; f.src = '` + c + `/child?n=late'; document.getElementById('later').appendChild(f); }, 900);</script>
</body></html>`
}

func (s *ifSite) hit(n string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.hits[n]
}

func (s *ifSite) waitHit(t *testing.T, n, want string) {
	t.Helper()
	for i := 0; i < 50; i++ {
		if s.hit(n) == want {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Errorf("server hit %s = %q, want %q", n, s.hit(n), want)
}

func (s *ifSite) lang() string { return s.hit("#lang") }

func (s *ifSite) close() { s.top.Close(); s.cross.Close() }

// startIsolatedChrome launches headless Chromium with strict site isolation.
func startIsolatedChrome(t *testing.T, extra ...string) (ws string, kill func()) {
	t.Helper()
	bin := os.Getenv("GHOST_CHROME")
	if bin == "" {
		t.Skip("set GHOST_CHROME to a Chromium binary")
	}
	ud, err := os.MkdirTemp("", "iframe-live-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { time.Sleep(300 * time.Millisecond); _ = os.RemoveAll(ud) })
	port := freePort()
	args := append([]string{"--headless=new", "--no-sandbox", "--site-per-process", "--window-size=1000,700", "--user-data-dir=" + ud, fmt.Sprintf("--remote-debugging-port=%d", port)}, extra...)
	cmd := exec.Command(bin, append(args, "about:blank")...)
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

// iframeTargets lists the URLs of the browser's "iframe" targets (OOPIFs).
func iframeTargets(t *testing.T, ws string) []string {
	c, err := dialCDP(ws, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.c.Close()
	raw, _ := c.call("", "Target.getTargets", nil)
	var tl struct {
		TargetInfos []struct{ Type, URL string } `json:"targetInfos"`
	}
	_ = json.Unmarshal(raw, &tl)
	out := []string{}
	for _, ti := range tl.TargetInfos {
		if ti.Type == "iframe" {
			out = append(out, ti.URL)
		}
	}
	return out
}

func ifStep(typ string, kv ...string) RpaStep {
	s := RpaStep{ID: newID(), Type: typ, Params: map[string]string{}}
	for i := 0; i+1 < len(kv); i += 2 {
		s.Params[kv[i]] = kv[i+1]
	}
	return s
}

func runIfSteps(t *testing.T, d RpaDriver, steps ...RpaStep) (map[string]string, []string, [][]byte) {
	t.Helper()
	return runIfStepsV(t, d, nil, steps...)
}

func runIfStepsV(t *testing.T, d RpaDriver, wv map[string]string, steps ...RpaStep) (map[string]string, []string, [][]byte) {
	t.Helper()
	var logs []string
	var shots [][]byte
	vars, err := runRpa(context.Background(), Workflow{Name: "t", Steps: steps, Vars: wv}, d, func(e RpaStepEvent) {
		if e.Status != "running" {
			logs = append(logs, e.Type+" "+e.Status+": "+e.Log)
		}
	}, func(i int, b []byte) (string, error) {
		shots = append(shots, b)
		return fmt.Sprintf("shot%d.png", i), nil
	})
	if err != nil {
		t.Fatalf("run: %v\n%s", err, strings.Join(logs, "\n"))
	}
	return vars, logs, shots
}

func TestIframeRunnerLive(t *testing.T) {
	ws, kill := startIsolatedChrome(t)
	defer kill()
	site := newIfSite()
	defer site.close()
	d, err := newCDPDriver(ws)
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	runIfSteps(t, d, ifStep("goto", "url", site.top.URL+"/"))

	oop := iframeTargets(t, ws)
	if len(oop) < 2 || !strings.Contains(strings.Join(oop, " "), site.crossBase) {
		t.Fatalf("expected OOPIF iframe targets for %s, got %v", site.crossBase, oop)
	}
	t.Logf("OOPIF targets: %v", oop)

	cases := []struct{ name, frame string }{
		{"same", "iframe#same"},
		{"cross", "iframe#cross"},
		{"nest", "iframe#nest >> iframe[name=inner]"},
		{"late", "#later > iframe"},
		{"deep", "#deep"},
		{"sd", "iframe#sd"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			v := "v-" + c.name + "-中文"
			vars, logs, _ := runIfSteps(t, d,
				ifStep("fill", "frame", c.frame, "selector", "#name", "text", v),
				ifStep("click", "frame", c.frame, "selector", "#go"),
				ifStep("wait_for", "frame", c.frame, "selector", "text:done:", "state", "visible"),
				ifStep("extract_text", "frame", c.frame, "selector", "#out", "var", "out"),
				ifStep("execute_js", "frame", c.frame, "code", "return window.pageVar + '|' + location.href.length", "saveAs", "pv"),
			)
			if want := "done:" + v + ":true"; vars["out"] != want {
				t.Errorf("out = %q, want %q (trusted real click)\n%s", vars["out"], want, strings.Join(logs, "\n"))
			}
			if !strings.HasPrefix(vars["pv"], "pv-"+c.name+"|") {
				t.Errorf("execute_js page world = %q", vars["pv"])
			}
			if !strings.Contains(logs[0], "框架 ") {
				t.Errorf("log lacks frame prefix: %s", logs[0])
			}
			if c.name != "sd" {
				site.waitHit(t, c.name, v+"|true")
			}
		})
	}

	t.Run("switch_frame", func(t *testing.T) {
		vars, logs, _ := runIfStepsV(t, d, map[string]string{"who": "sw", "fid": "cross"},
			ifStep("switch_frame", "frame", "iframe#nest"),
			ifStep("extract_text", "selector", "#mid", "var", "mid"),
			ifStep("switch_frame", "frame", "iframe[name=inner]"),
			ifStep("fill", "selector", "#name", "text", "{{who}}"),
			ifStep("click", "selector", "#go"),
			ifStep("wait_for", "selector", "text:done:sw"),
			ifStep("switch_frame", "frame", "parent"),
			ifStep("extract_text", "selector", "#mid", "var", "mid2"),
			ifStep("switch_frame", "frame", "top"),
			ifStep("extract_text", "selector", "#topmark", "var", "top"),
			// A step-level frame overrides the context; variables work in it.
			ifStep("extract_text", "frame", "iframe#{{fid}}", "selector", "#out", "var", "crossOut"),
		)
		_ = logs
		if vars["mid"] != "middle" || vars["mid2"] != "middle" || vars["top"] != "TOP" {
			t.Errorf("vars %v", vars)
		}
		site.waitHit(t, "nest", "sw|true")
		if !strings.HasPrefix(vars["crossOut"], "done:") {
			t.Errorf("crossOut %q", vars["crossOut"])
		}
	})

	t.Run("upload_scroll_shot", func(t *testing.T) {
		f, _ := os.CreateTemp("", "up-*.txt")
		f.WriteString("hello")
		f.Close()
		defer os.Remove(f.Name())
		_, logs, shots := runIfSteps(t, d,
			ifStep("upload", "frame", "#deep", "selector", "#f", "files", f.Name()),
			ifStep("scroll", "frame", "#deep", "mode", "el", "selector", "#out"),
			ifStep("screenshot", "frame", "#deep", "selector", "#out"),
		)
		base := f.Name()[strings.LastIndex(f.Name(), "/")+1:]
		site.waitHit(t, "deep-file", base+"|")
		if len(shots) != 1 {
			t.Fatalf("shots %d %v", len(shots), logs)
		}
		img, err := png.Decode(bytes.NewReader(shots[0]))
		if err != nil {
			t.Fatal(err)
		}
		b := img.Bounds()
		green, all := 0, 0
		for y := b.Min.Y; y < b.Max.Y; y++ {
			for x := b.Min.X; x < b.Max.X; x++ {
				r, g, bl, _ := img.At(x, y).RGBA()
				all++
				if r>>8 < 60 && g>>8 > 120 && bl>>8 < 60 {
					green++
				}
			}
		}
		t.Logf("element shot %dx%d, %d/%d green", b.Dx(), b.Dy(), green, all)
		if b.Dx() < 70 || b.Dx() > 90 || b.Dy() < 15 || b.Dy() > 25 || green*2 < all {
			t.Errorf("element screenshot is not the green #out box: %dx%d green %d/%d", b.Dx(), b.Dy(), green, all)
		}
	})

	t.Run("missing_frame", func(t *testing.T) {
		_, err := runRpa(context.Background(), Workflow{Name: "t", Steps: []RpaStep{ifStep("click", "frame", "iframe#nope", "selector", "#go")}}, d, nil, nil)
		if err == nil || !strings.Contains(err.Error(), "找不到框架") {
			t.Errorf("err = %v", err)
		}
		vars, _, _ := runIfSteps(t, d, func() RpaStep {
			s := ifStep("condition", "kind", "exists", "frame", "iframe#nope", "selector", "#go")
			s.Else = []RpaStep{ifStep("execute_js", "code", "return 'no'", "saveAs", "r")}
			return s
		}())
		if vars["r"] != "no" {
			t.Errorf("condition on missing frame: %v", vars)
		}
	})
}

// TestIframeFingerprintLive: the fingerprint overrides reach out-of-process
// iframes (the emulator auto-attaches recursively), so navigator/UA/timezone
// read inside a cross-origin iframe match the top frame.
func TestIframeFingerprintLive(t *testing.T) {
	fp := GenerateTyped("desktop", "windows", "141")
	fp.Timezone, fp.Languages = "America/Sao_Paulo", []string{"pt-BR", "pt"}
	ws, kill := startIsolatedChrome(t, "--user-agent="+fp.UserAgent)
	defer kill()
	site := newIfSite()
	defer site.close()
	var starts []string
	if os.Getenv("IFX_START") != "" {
		starts = []string{site.top.URL + "/"}
	}
	em, err := startEmulation(ws, fp, nil, starts)
	if err != nil {
		t.Fatal(err)
	}
	defer em.Close()
	d, err := newCDPDriver(ws)
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	if starts == nil {
		runIfSteps(t, d, ifStep("goto", "url", site.top.URL+"/"))
	} else {
		time.Sleep(3 * time.Second)
	}
	runIfSteps(t, d, ifStep("wait_for", "frame", "#later > iframe", "selector", "#go"))
	probe := `return JSON.stringify({ua: navigator.userAgent, plat: navigator.platform, hc: navigator.hardwareConcurrency, mem: navigator.deviceMemory,
	  lang: navigator.language, langs: navigator.languages, tz: Intl.DateTimeFormat().resolvedOptions().timeZone, off: new Date(2026, 0, 1).getTimezoneOffset(),
	  sw: screen.width, sh: screen.height, cd: screen.colorDepth, dpr: devicePixelRatio, wd: navigator.webdriver,
	  ch: navigator.userAgentData ? navigator.userAgentData.platform + '/' + navigator.userAgentData.brands.map(b => b.brand + b.version).join(',') : '',
	  canvas: (() => { const c = document.createElement('canvas'); c.width = 60; c.height = 20; const x = c.getContext('2d'); x.fillStyle = '#f60'; x.fillRect(1, 1, 40, 12); x.fillText('gb', 4, 12); return c.toDataURL().length; })()})`
	frames := []string{"", "iframe#same", "iframe#cross", "iframe#nest >> iframe[name=inner]", "#later > iframe"}
	var top map[string]any
	for _, f := range frames {
		vars, _, _ := runIfSteps(t, d, ifStep("execute_js", "frame", f, "code", probe, "saveAs", "fp"))
		var got map[string]any
		if err := json.Unmarshal([]byte(vars["fp"]), &got); err != nil {
			t.Fatalf("%s: %v %q", f, err, vars["fp"])
		}
		if f == "" {
			top = got
			t.Logf("top: %v", got)
			if got["tz"] != "America/Sao_Paulo" || got["plat"] != fp.Platform || got["ua"] != em.fp.UserAgent || fmt.Sprint(got["langs"]) != "[pt-BR pt]" {
				t.Errorf("top frame not emulated: %v", got)
			}
			continue
		}
		for k, v := range top {
			if fmt.Sprint(got[k]) != fmt.Sprint(v) {
				t.Errorf("frame %s: %s = %v, top has %v", f, k, got[k], v)
			}
		}
	}
	if h := site.lang(); !strings.HasPrefix(h, "pt-BR,pt;q=") {
		t.Errorf("Accept-Language header %q", h)
	}
	if oop := iframeTargets(t, ws); len(oop) < 3 {
		t.Errorf("expected OOPIFs, got %v", oop)
	}
}

// ifUser acts like a person on a page: real mouse clicks (translated into
// frames by the RPA driver's frame resolver) and real key presses.
type ifUser struct {
	t *testing.T
	d *cdpDriver
}

func newIfUser(t *testing.T, ws string) *ifUser {
	dr, err := newCDPDriver(ws)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(dr.Close)
	return &ifUser{t: t, d: dr.(*cdpDriver)}
}

func (u *ifUser) click(frame, sel string) {
	u.t.Helper()
	if err := u.d.UseFrame(parseFramePath(frame), 10*time.Second); err != nil {
		u.t.Fatal(err)
	}
	if err := u.d.Click(sel, false); err != nil {
		u.t.Fatalf("click %s in %q: %v", sel, frame, err)
	}
	time.Sleep(250 * time.Millisecond)
}

func (u *ifUser) typeText(s string) {
	for _, r := range s {
		ch := string(r)
		_, _ = u.d.call("Input.dispatchKeyEvent", map[string]any{"type": "keyDown", "text": ch, "unmodifiedText": ch, "key": ch})
		_, _ = u.d.call("Input.dispatchKeyEvent", map[string]any{"type": "keyUp", "key": ch})
		time.Sleep(30 * time.Millisecond)
	}
	time.Sleep(150 * time.Millisecond)
}

func TestIframeRecordReplayLive(t *testing.T) {
	bin := os.Getenv("GHOST_CHROME")
	if bin == "" {
		t.Skip("set GHOST_CHROME to a Chromium binary")
	}
	site := newIfSite()
	defer site.close()
	a := newRpaE2EApp(t, bin)
	mk := func(name string) string {
		fp := GenerateTyped("desktop", "windows", "")
		fp.StartupParameters = "--headless=new --no-sandbox --site-per-process"
		return a.store.PutProfile(Profile{Name: name, Kernel: "chromium", Fingerprint: fp}).ID
	}
	recPID, playPID := mk("录制环境"), mk("回放环境")
	defer runReg.stop(recPID)
	defer runReg.stop(playPID)
	if _, err := a.StartRecording(recPID, RecordOptions{Overlay: true, StartURL: site.top.URL + "/"}); err != nil {
		t.Fatal(err)
	}
	rs, _ := runReg.get(recPID)
	u := newIfUser(t, rs.WSURL)
	if err := u.d.WaitFor("#topin", "visible", 10*time.Second); err != nil {
		t.Fatal(err)
	}
	time.Sleep(1500 * time.Millisecond) // let the late iframe appear and frames get their scripts
	if oop := iframeTargets(t, rs.WSURL); len(oop) < 3 {
		t.Fatalf("no OOPIFs in the recorded browser: %v", oop)
	}
	u.click("", "#topin")
	u.typeText("top")
	frames := []struct{ name, path string }{
		{"same", "iframe#same"},
		{"cross", "iframe#cross"},
		{"nest", `iframe#nest >> iframe[name="inner"]`},
		{"late", "iframe#late"},
		{"deep", "iframe#deep"},
		{"sd", "iframe#sd"},
	}
	for _, f := range frames {
		u.click(f.path, "#name")
		u.typeText("r-" + f.name)
		u.click(f.path, "#go")
	}
	// Pick mode inside a cross-origin frame: the click adds 提取文本 and
	// does not reach the page.
	before := site.hit("cross")
	a.RecordPickMode("extract")
	time.Sleep(400 * time.Millisecond)
	u.click("iframe#cross", "#go")
	time.Sleep(300 * time.Millisecond)
	if site.hit("cross") != before {
		t.Errorf("a click in pick mode reached the iframe page")
	}
	final := a.StopRecording()
	f := a.FinalizeRecording(final.Steps, RecordFinalizeOptions{MergeFills: true})
	got := []string{}
	for _, s := range f.Steps {
		line := s.Type + " " + strings.SplitN(s.Params["selector"], " || ", 2)[0]
		if s.Params["frame"] != "" {
			line = "[" + s.Params["frame"] + "] " + line
		}
		if s.Type == stFill {
			line += " = " + s.Params["text"]
		}
		if s.Type == stGoto {
			line += s.Params["url"]
		}
		got = append(got, line)
	}
	t.Logf("recorded:\n  %s", strings.Join(got, "\n  "))
	want := []string{"goto " + site.top.URL + "/", "fill #topin = top"}
	for _, fr := range frames {
		want = append(want, "["+fr.path+"] fill #name = r-"+fr.name, "["+fr.path+"] click #go")
	}
	want = append(want, "[iframe#cross] extract_text #go")
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("recorded steps differ; want:\n  %s", strings.Join(want, "\n  "))
	}
	for _, fr := range frames[:5] {
		if h := site.hit(fr.name); h != "r-"+fr.name+"|true" {
			t.Errorf("recording session: %s hit %q", fr.name, h)
		}
	}

	// ---- replay on a fresh profile ----
	site.mu.Lock()
	site.hits = map[string]string{}
	site.mu.Unlock()
	wf, err := a.SaveWorkflow(Workflow{Name: "框架录制回放", Steps: f.Steps})
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
	for _, fr := range frames[:5] {
		site.waitHit(t, fr.name, "r-"+fr.name+"|true")
	}
	if v := run.Targets[0].Vars["text1"]; !strings.Contains(v, "提交") {
		t.Errorf("extracted text in replay = %q", v)
	}
}

// TestIframeSyncerLive: clicks and typing inside the master's iframes
// (cross-origin OOPIFs included) are mirrored into the same iframes of two
// followers.
func TestIframeSyncerLive(t *testing.T) {
	site := newIfSite()
	defer site.close()
	m := liveBrowser(t, "ifM", site.top.URL+"/")
	f1 := liveBrowser(t, "ifF1", site.top.URL+"/")
	f2 := liveBrowser(t, "ifF2", site.top.URL+"/")
	time.Sleep(1800 * time.Millisecond) // pages + the late iframe
	if oop := iframeTargets(t, m); len(oop) < 3 {
		t.Fatalf("no OOPIFs in the master: %v", oop)
	}
	groupSync.SetOptions(defaultSyncOptions())
	if _, err := groupSync.Start("ifM", []string{"ifF1", "ifF2"}, liveWS, idName); err != nil {
		t.Fatal(err)
	}
	defer groupSync.Stop("test end")
	time.Sleep(600 * time.Millisecond)
	u := newIfUser(t, m)
	followers := []*cdpDriver{}
	for _, ws := range []string{f1, f2} {
		dr, err := newCDPDriver(ws)
		if err != nil {
			t.Fatal(err)
		}
		defer dr.Close()
		followers = append(followers, dr.(*cdpDriver))
	}
	read := func(d *cdpDriver, frame string) string {
		if err := d.UseFrame(parseFramePath(frame), 3*time.Second); err != nil {
			return "ERR " + err.Error()
		}
		v, _ := d.Text("#out", "")
		return v
	}
	for _, fr := range []struct{ name, path string }{
		{"cross", "iframe#cross"},
		{"same", "iframe#same"},
		{"nest", "iframe#nest >> iframe[name=inner]"},
		{"late", "iframe#late"},
		{"deep", "iframe#deep"},
	} {
		v := "sync" + fr.name
		u.click(fr.path, "#name")
		u.typeText(v)
		u.click(fr.path, "#go")
		if got := read(u.d, fr.path); got != "done:"+v+":true" {
			t.Fatalf("master %s: %q", fr.name, got)
		}
		for i, d := range followers {
			want := "done:" + v + ":true"
			var got string
			for k := 0; k < 40; k++ {
				if got = read(d, fr.path); got == want {
					break
				}
				time.Sleep(150 * time.Millisecond)
			}
			if got != want {
				t.Errorf("follower %d, %s: #out = %q, want %q", i+1, fr.name, got, want)
			}
		}
	}
	st := groupSync.State()
	t.Logf("syncer: %d events, followers %+v", st.Events, st.Followers)
}

// TestIframeSelectorGenLive checks cdpFrameSelFn (the recorder's iframe
// selector generator) on a fixture: preference order and uniqueness.
func TestIframeSelectorGenLive(t *testing.T) {
	ws, kill := startLiveChrome(t)
	defer kill()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprint(w, `<!doctype html><body>
<iframe id="pay"></iframe>
<iframe id="frame-123456789" name="card"></iframe>
<iframe title="验证码"></iframe>
<iframe src="/widgets/chat/index.html?sess=8f7a6b5c4d3e2f1a"></iframe>
<div class="ads"><iframe src="/ad.html"></iframe><iframe src="/ad.html"></iframe></div>
<iframe srcdoc="<b>x</b>"></iframe>
</body>`)
	}))
	defer srv.Close()
	d, err := newCDPDriver(ws)
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	cd := d.(*cdpDriver)
	if err := cd.Goto(srv.URL+"/", 10*time.Second); err != nil {
		t.Fatal(err)
	}
	raw, err := cd.call("Runtime.evaluate", map[string]any{"expression": "Array.from(document.querySelectorAll('iframe'))"})
	if err != nil {
		t.Fatal(err)
	}
	var arr struct {
		Result struct{ ObjectID string } `json:"result"`
	}
	_ = json.Unmarshal(raw, &arr)
	want := []string{"iframe#pay", `iframe[name="card"]`, `iframe[title="验证码"]`, `iframe[src*="/widgets/chat/index.html"]`, "div > iframe:nth-of-type(1)", "div > iframe:nth-of-type(2)", "iframe:nth-of-type(5)"}
	for i := range want {
		r, err := cd.call("Runtime.callFunctionOn", map[string]any{"objectId": arr.Result.ObjectID, "functionDeclaration": "function(i, f) { const el = this[i]; const sel = eval('(' + f + ')').call(el); const n = document.querySelectorAll(sel); return JSON.stringify([sel, n.length === 1 && n[0] === el]); }",
			"arguments": []any{map[string]any{"value": i}, map[string]any{"value": cdpFrameSelFn}}, "returnByValue": true})
		if err != nil {
			t.Fatal(err)
		}
		var v string
		if err := decodeEvalResult(r, &v); err != nil {
			t.Fatal(err)
		}
		var got []any
		_ = json.Unmarshal([]byte(v), &got)
		t.Logf("iframe %d → %v", i, got)
		if got[1] != true {
			t.Errorf("iframe %d: selector %v not unique", i, got[0])
		}
		if got[0] != want[i] {
			t.Errorf("iframe %d: %v, want %s", i, got[0], want[i])
		}
	}
}
