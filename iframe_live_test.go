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
			if !strings.Contains(logs[0], "⧉ ") {
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
