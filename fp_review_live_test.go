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

// Second-pass review: a detection-style page computes fingerprint hashes in
// the main frame, a same-origin iframe, a script-created about:blank iframe
// and a dedicated worker, and probes for tell-tale patch artefacts.
// GHOST_CHROME=/usr/bin/chromium go test -run TestFpReview .

const fpDetectJS = `
const hs = (s) => { let v = 0; s = String(s); for (let i = 0; i < s.length; i++) v = (Math.imul(v, 31) + s.charCodeAt(i)) | 0; return (v >>> 0).toString(16); };
async function probe(W) {
  const D = W.document;
  const out = {};
  if (D) {
    const c = D.createElement('canvas'); c.width = 200; c.height = 50; const x = c.getContext('2d');
    x.fillStyle = '#f60'; x.fillRect(0, 0, 200, 50); x.fillStyle = '#069'; x.font = '16px Arial'; x.fillText('Cwm fjord 指纹 😀', 3, 30);
    out.canvas = hs(c.toDataURL());
    const g = D.createElement('canvas').getContext('webgl');
    out.gl = g ? g.getParameter(37446) : null;
    const el = D.body || D.documentElement; const b = D.createElement('div'); b.style.width = '123.4px'; b.textContent = 'x'; el.appendChild(b);
    out.rect = b.getBoundingClientRect().width; b.remove();
  } else {
    const oc = new OffscreenCanvas(200, 50); const x = oc.getContext('2d');
    x.fillStyle = '#f60'; x.fillRect(0, 0, 200, 50); x.fillStyle = '#069'; x.font = '16px Arial'; x.fillText('Cwm fjord 指纹 😀', 3, 30);
    const id = x.getImageData(0, 0, 200, 50).data; let s = 0; for (let i = 0; i < id.length; i++) s = (s * 31 + id[i]) | 0; out.ocanvas = s;
  }
  if (W.OfflineAudioContext) {
    const a = new W.OfflineAudioContext(1, 4410, 44100); const o = a.createOscillator(); o.type = 'triangle'; o.frequency.value = 1000;
    const k = a.createDynamicsCompressor(); o.connect(k); k.connect(a.destination); o.start(0);
    const buf = await a.startRendering(); const d = buf.getChannelData(0); let s = 0; for (let i = 0; i < d.length; i++) s += Math.abs(d[i]); out.audio = s.toFixed(12);
  }
  out.hc = W.navigator.hardwareConcurrency; out.mem = W.navigator.deviceMemory; out.ua = hs(W.navigator.userAgent);
  return out;
}
async function leaks() {
  const L = {};
  const nat = (f) => Function.prototype.toString.call(f);
  const P = HTMLCanvasElement.prototype, C2 = CanvasRenderingContext2D.prototype, G = WebGLRenderingContext.prototype;
  L.globals = Object.getOwnPropertyNames(window).filter(k => /^__|gb/i.test(k) && k !== '__collect').join(',');
  L.tsProto = 'prototype' in Function.prototype.toString;
  L.tsName = Function.prototype.toString.name;
  L.lenGid = C2.getImageData.length; L.lenGP = G.getParameter.length; L.lenGCD = AudioBuffer.prototype.getChannelData.length;
  L.lenTB = P.toBlob.length; L.lenGC = P.getContext.length;
  L.srcs = [P.toDataURL, P.getContext, C2.getImageData, G.getParameter, Element.prototype.getBoundingClientRect, Function.prototype.toString,
    Object.getOwnPropertyDescriptor(Navigator.prototype, 'hardwareConcurrency').get].map(nat).every(s => /\{ \[native code\] \}$/.test(s));
  L.navOwn = Object.getOwnPropertyNames(navigator).join(',');
  let st = ''; try { P.toDataURL.call({}); } catch (e) { st = String(e.stack); } L.stackLines = st.split('\n').length;
  L.wsProtoWritable = Object.getOwnPropertyDescriptor(WebSocket, 'prototype').writable; L.wsLen = WebSocket.length;
  return L;
}
window.__collect = async () => {
  const r = { main: await probe(window), leaks: await leaks() };
  const f1 = document.getElementById('f1'); await new Promise(res => f1.contentDocument && f1.contentDocument.readyState === 'complete' ? res() : f1.onload = res);
  r.iframe = await probe(f1.contentWindow);
  const f2 = document.createElement('iframe'); document.body.appendChild(f2); r.blank = await probe(f2.contentWindow);
  const src = probe.toString() + ';' + 'const hs=' + hs.toString() + ';onmessage = async () => postMessage(await probe(self));';
  const w = new Worker(URL.createObjectURL(new Blob([src], { type: 'text/javascript' })));
  r.worker = await new Promise(res => { w.onmessage = (e) => res(e.data); w.onerror = (e) => res({ err: String(e.message) }); w.postMessage(1); setTimeout(() => res({ err: 'timeout' }), 4000); });
  return r;
};`

type fpRun struct {
	t    *testing.T
	c    *cdpConn
	sid  string
	stop func()
}

func startFpReview(t *testing.T, fp Fingerprint, page string) *fpRun {
	t.Helper()
	bin := os.Getenv("GHOST_CHROME")
	if bin == "" {
		t.Skip("set GHOST_CHROME to a Chromium binary")
	}
	port := freePort()
	args := append([]string{"--headless=new", "--no-sandbox", "--user-data-dir=" + liveUserDataDir(t), fmt.Sprintf("--remote-debugging-port=%d", port), "--no-proxy-server", "--host-resolver-rules=MAP shop.example.test 127.0.0.1"}, launchFlags(fp)...)
	cmd := exec.Command(bin, append(mergeFlags(args), "about:blank")...)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	ws, err := waitDebugger(port, 20*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	em, err := startEmulation(ws, fp, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	c, err := dialCDP(ws, nil)
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(300 * time.Millisecond)
	if _, err := c.call("", "Target.createTarget", map[string]any{"url": page}); err != nil {
		t.Fatal(err)
	}
	r := &fpRun{t: t, c: c, stop: func() { c.c.Close(); em.Close(); cmd.Process.Kill(); cmd.Wait() }}
	for i := 0; i < 50 && r.sid == ""; i++ {
		time.Sleep(200 * time.Millisecond)
		raw, _ := c.call("", "Target.getTargets", nil)
		var tl struct {
			TargetInfos []struct{ TargetID, Type, URL string } `json:"targetInfos"`
		}
		_ = json.Unmarshal(raw, &tl)
		for _, ti := range tl.TargetInfos {
			if ti.Type == "page" && strings.HasPrefix(ti.URL, page) {
				rr, _ := c.call("", "Target.attachToTarget", map[string]any{"targetId": ti.TargetID, "flatten": true})
				var at struct{ SessionID string }
				_ = json.Unmarshal(rr, &at)
				r.sid = at.SessionID
			}
		}
	}
	if r.sid == "" {
		r.stop()
		t.Fatal("page did not open")
	}
	return r
}

func (r *fpRun) collect() map[string]map[string]any {
	r.t.Helper()
	for i := 0; i < 30; i++ {
		raw, err := r.c.call(r.sid, "Runtime.evaluate", map[string]any{"expression": "typeof window.__collect", "returnByValue": true})
		if err == nil && strings.Contains(string(raw), "function") {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}
	raw, err := r.c.call(r.sid, "Runtime.evaluate", map[string]any{"returnByValue": true, "awaitPromise": true, "expression": "window.__collect().then(JSON.stringify)"})
	if err != nil {
		r.t.Fatal(err)
	}
	var ev struct{ Result struct{ Value string } }
	_ = json.Unmarshal(raw, &ev)
	out := map[string]map[string]any{}
	if err := json.Unmarshal([]byte(ev.Result.Value), &out); err != nil {
		dbg, _ := r.c.call(r.sid, "Runtime.evaluate", map[string]any{"expression": "location.href + ' ' + document.documentElement.outerHTML.slice(0, 300)", "returnByValue": true})
		r.t.Fatalf("collect: %s\n%s", raw, dbg)
	}
	return out
}

func (r *fpRun) reload() {
	_, _ = r.c.call(r.sid, "Page.reload", nil)
	time.Sleep(1200 * time.Millisecond)
}

func fpReviewServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if req.URL.Path == "/frame" {
			fmt.Fprint(w, "<!doctype html><body>frame</body>")
			return
		}
		fmt.Fprint(w, "<!doctype html><body><iframe id=f1 src='/frame'></iframe><script>"+fpDetectJS+"</script></body>")
	}))
}

func TestFpReviewConsistencyLive(t *testing.T) {
	srv := fpReviewServer()
	defer srv.Close()
	page := strings.Replace(srv.URL, "127.0.0.1", "shop.example.test", 1) + "/"
	mk := func(id string) Fingerprint {
		fp := fpLiveBase(id)
		fp.WebGLImageMode, fp.AudioMode, fp.ClientRectsMode = "noise", "noise", "noise"
		fp.HardwareConc, fp.DeviceMemory = 6, 4
		fp.PortScanProtection = true
		return fp
	}
	a := startFpReview(t, mk("A"), page)
	ra := a.collect()
	a.reload()
	ra2 := a.collect()
	a.stop()
	b := startFpReview(t, mk("B"), page)
	rb := b.collect()
	b.stop()
	realFp := mk("A")
	realFp.CanvasMode, realFp.AudioMode, realFp.ClientRectsMode = "real", "real", "real"
	rr := startFpReview(t, realFp, page)
	rreal := rr.collect()
	rr.stop()
	j := func(v any) string { s, _ := json.Marshal(v); return string(s) }
	t.Logf("A: %s", j(ra))
	t.Logf("A reload: %s", j(ra2))
	t.Logf("B: %s", j(rb))
	t.Logf("real: %s", j(rreal))
	for _, k := range []string{"canvas", "audio", "rect", "gl", "hc", "ua"} {
		if fmt.Sprint(ra["main"][k]) != fmt.Sprint(ra2["main"][k]) {
			t.Errorf("%s changed across reload: %v vs %v", k, ra["main"][k], ra2["main"][k])
		}
		for _, fr := range []string{"iframe", "blank"} {
			if fmt.Sprint(ra["main"][k]) != fmt.Sprint(ra[fr][k]) {
				t.Errorf("%s differs main vs %s: %v vs %v", k, fr, ra["main"][k], ra[fr][k])
			}
		}
	}
	for _, k := range []string{"canvas", "audio", "rect"} {
		if fmt.Sprint(ra["main"][k]) == fmt.Sprint(rb["main"][k]) {
			t.Errorf("%s same for profiles A and B: %v", k, ra["main"][k])
		}
		if fmt.Sprint(ra["main"][k]) == fmt.Sprint(rreal["main"][k]) {
			t.Errorf("%s noise had no effect: %v", k, ra["main"][k])
		}
	}
	for _, k := range []string{"hc", "mem", "ua"} {
		if fmt.Sprint(ra["main"][k]) != fmt.Sprint(ra["worker"][k]) {
			t.Errorf("worker %s = %v, main %v", k, ra["worker"][k], ra["main"][k])
		}
	}
	if fmt.Sprint(ra["worker"]["ocanvas"]) == fmt.Sprint(rreal["worker"]["ocanvas"]) {
		t.Errorf("worker OffscreenCanvas not noised: %v", ra["worker"]["ocanvas"])
	}
	L := ra["leaks"]
	rl := rreal["leaks"]
	_ = rl
	want := map[string]string{"globals": "", "tsProto": "false", "tsName": "toString", "lenGid": "4", "lenGP": "1", "lenGCD": "1", "lenTB": "1", "lenGC": "1",
		"srcs": "true", "navOwn": "", "wsProtoWritable": "false", "wsLen": "1"}
	for k, v := range want {
		if fmt.Sprint(L[k]) != v {
			t.Errorf("leak %s = %v, native %s", k, L[k], v)
		}
	}
}
