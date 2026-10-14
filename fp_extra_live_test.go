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

// liveUserDataDir is a temp profile dir removed with retries: Chromium's helper
// processes (crashpad, utility) can still be writing into it for a moment after
// the browser process was killed, which made t.TempDir's cleanup flaky
// ("directory not empty").
func liveUserDataDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "gb-fp-live-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		var err error
		for i := 0; i < 50; i++ {
			if err = os.RemoveAll(dir); err == nil {
				return
			}
			time.Sleep(100 * time.Millisecond)
		}
		t.Logf("leaving temp dir %s: %v", dir, err)
	})
	return dir
}

// fpLive launches headless Chromium with the fingerprint applied, opens a test
// page and returns the JSON value of expr evaluated in it.
func fpLive(t *testing.T, fp Fingerprint, expr string) map[string]any {
	t.Helper()
	return fpLiveAt(t, fp, expr, "")
}

// fpLiveAt serves the test page under host (mapped to 127.0.0.1) when set.
func fpLiveAt(t *testing.T, fp Fingerprint, expr, host string) map[string]any {
	t.Helper()
	bin := os.Getenv("GHOST_CHROME")
	if bin == "" {
		t.Skip("set GHOST_CHROME to a Chromium binary")
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, "<html><body><div id='box' style='width:123.4px;height:50px'>hello</div></body></html>")
	}))
	defer srv.Close()
	port := freePort()
	page := srv.URL + "/"
	var extra []string
	if host != "" {
		page = strings.Replace(page, "127.0.0.1", host, 1)
		extra = append(extra, "--host-resolver-rules=MAP "+host+" 127.0.0.1")
	}
	args := append([]string{"--headless=new", "--no-sandbox", "--user-data-dir=" + liveUserDataDir(t), fmt.Sprintf("--remote-debugging-port=%d", port)}, launchFlags(fp)...)
	args = append(args, extra...)
	cmd := exec.Command(bin, append(mergeFlags(args), "about:blank")...)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { cmd.Process.Kill(); cmd.Wait() }()
	ws, err := waitDebugger(port, 20*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	em, err := startEmulation(ws, fp, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer em.Close()
	c, err := dialCDP(ws, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.c.Close()
	time.Sleep(300 * time.Millisecond)
	if _, err := c.call("", "Target.createTarget", map[string]any{"url": page}); err != nil {
		t.Fatal(err)
	}
	sid := ""
	for i := 0; i < 50 && sid == ""; i++ {
		time.Sleep(200 * time.Millisecond)
		raw, _ := c.call("", "Target.getTargets", nil)
		var tl struct {
			TargetInfos []struct{ TargetID, Type, URL string } `json:"targetInfos"`
		}
		_ = json.Unmarshal(raw, &tl)
		for _, ti := range tl.TargetInfos {
			if ti.Type == "page" && strings.HasPrefix(ti.URL, page) {
				r, _ := c.call("", "Target.attachToTarget", map[string]any{"targetId": ti.TargetID, "flatten": true})
				var at struct{ SessionID string }
				_ = json.Unmarshal(r, &at)
				sid = at.SessionID
			}
		}
	}
	if sid == "" {
		t.Fatal("test page did not open")
	}
	time.Sleep(500 * time.Millisecond)
	r, err := c.call(sid, "Runtime.evaluate", map[string]any{"returnByValue": true, "awaitPromise": true, "expression": "(async () => JSON.stringify(await (async () => {" + expr + "})()))()"})
	if err != nil {
		t.Fatal(err)
	}
	var ev struct {
		Result           struct{ Value string }
		ExceptionDetails any
	}
	_ = json.Unmarshal(r, &ev)
	var got map[string]any
	if err := json.Unmarshal([]byte(ev.Result.Value), &got); err != nil {
		t.Fatalf("eval: %s", r)
	}
	return got
}

const canvasExpr = `
const draw = () => { const c = document.createElement('canvas'); c.width = 220; c.height = 60; const x = c.getContext('2d');
  x.fillStyle = '#f60'; x.fillRect(0, 0, 220, 60); x.fillStyle = '#069'; x.font = '18px Arial'; x.fillText('GhostBrowser 指纹 😀', 4, 30); return c; };
const h = (s) => { let v = 0; for (let i = 0; i < s.length; i++) v = (Math.imul(v, 31) + s.charCodeAt(i)) | 0; return String(v >>> 0); };
const c = draw(); const a = c.toDataURL(), b = c.toDataURL(), d = draw().toDataURL();
const id = c.getContext('2d').getImageData(0, 0, 220, 60).data; let sum = 0; for (let i = 0; i < id.length; i++) sum += id[i];
return { h: h(a), same: a === b, fresh: a === d, sum };`

func fpLiveBase(id string) Fingerprint {
	fp := GenerateFingerprint("windows", "141")
	fp.NoiseSeed = effectiveSeed(id, fp.NoiseSeed, false)
	fp.CanvasMode = "noise"
	return fp
}

func TestFpExtraCanvasLive(t *testing.T) {
	base := fpLiveBase("A")
	seedA := base.NoiseSeed
	a := fpLive(t, base, canvasExpr)
	a2 := fpLive(t, base, canvasExpr)
	bfp := base
	bfp.NoiseSeed = effectiveSeed("B", 7, false)
	b := fpLive(t, bfp, canvasExpr)
	real := base
	real.CanvasMode = "real"
	r := fpLive(t, real, canvasExpr)
	off := base
	off.CanvasMode = "off"
	o := fpLive(t, off, canvasExpr)
	t.Logf("seedA=%d A=%v A2=%v B=%v real=%v off=%v", seedA, a, a2, b, r, o)
	if a["h"] != a2["h"] {
		t.Error("same profile not stable across launches")
	}
	if a["h"] == b["h"] {
		t.Error("two profiles share a canvas hash")
	}
	if a["h"] == r["h"] || b["h"] == r["h"] {
		t.Error("noise did not change the canvas")
	}
	if a["same"] != true || a["fresh"] != true {
		t.Error("toDataURL not stable within a page")
	}
	if a["sum"] == r["sum"] {
		t.Error("getImageData not noised")
	}
	if o["h"] == r["h"] || o["sum"] != float64(0) {
		t.Errorf("canvas off should read back blank: %v", o)
	}
}

func TestFpExtraAPIsLive(t *testing.T) {
	fp := fpLiveBase("C")
	fp.WebRTC = "disabled"
	fp.HardwareConc, fp.DeviceMemory = 12, 4
	fp.WebGPU = "off"
	fp.MediaDevicesMode, fp.MediaMics, fp.MediaSpeakers, fp.MediaCameras = "custom", 2, 1, 1
	fp.SpeechVoices = "custom"
	fp.AudioMode = "off"
	fp.WebGLImageMode = "off"
	fp.ClientRectsMode = "noise"
	fp.DoNotTrack = true
	got := fpLive(t, fp, `
const devs = await navigator.mediaDevices.enumerateDevices();
const el = document.getElementById('box'); const r1 = el.getBoundingClientRect().width, r2 = el.getBoundingClientRect().width;
return { rtc: typeof RTCPeerConnection, hc: navigator.hardwareConcurrency, mem: navigator.deviceMemory, gpu: 'gpu' in navigator,
  media: devs.length, mics: devs.filter(d => d.kind === 'audioinput').length, cams: devs.filter(d => d.kind === 'videoinput').length,
  idlen: devs[0] && devs[0].deviceId.length, isInfo: devs[0] instanceof MediaDeviceInfo,
  voices: speechSynthesis.getVoices().length, voice0: (speechSynthesis.getVoices()[0] || {}).name,
  audio: typeof AudioContext, gl: document.createElement('canvas').getContext('webgl') === null,
  rectStable: r1 === r2, rectJitter: r1 !== 123.4, dnt: navigator.doNotTrack,
  ts: Function.prototype.toString.call(navigator.mediaDevices.enumerateDevices) };`)
	t.Logf("got %v", got)
	want := map[string]any{"rtc": "undefined", "hc": 12.0, "mem": 4.0, "gpu": false, "media": 4.0, "mics": 2.0, "cams": 1.0, "idlen": 64.0, "isInfo": true,
		"audio": "undefined", "gl": true, "rectStable": true, "rectJitter": true, "dnt": "1", "ts": "function enumerateDevices() { [native code] }"}
	for k, v := range want {
		if fmt.Sprint(got[k]) != fmt.Sprint(v) {
			t.Errorf("%s = %v, want %v", k, got[k], v)
		}
	}
	if n, _ := got["voices"].(float64); n < 3 || !strings.HasPrefix(fmt.Sprint(got["voice0"]), "Microsoft") {
		t.Errorf("voices %v %v", got["voices"], got["voice0"])
	}
}

func TestFpExtraPortScanLive(t *testing.T) {
	// The test page itself is on 127.0.0.1, so local requests must still work there.
	fp := fpLiveBase("D")
	fp.PortScanProtection = true
	got := fpLive(t, fp, `const r = await fetch(location.href).then(r => r.status).catch(e => 'blocked'); return { st: r };`)
	if fmt.Sprint(got["st"]) != "200" {
		t.Errorf("local page fetching itself: %v", got)
	}
}

func TestFpExtraPortScanBlockLive(t *testing.T) {
	// A page on a public-looking host must not reach services on 127.0.0.1,
	// over fetch or WebSocket, unless the port is whitelisted.
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		fmt.Fprint(w, "open")
	}))
	defer target.Close()
	_, port, _ := urlHostPort(target.URL)
	expr := `const f = await fetch('` + target.URL + `/').then(r => r.text()).catch(e => 'blocked');
const ws = await new Promise(res => { const s = new WebSocket('ws://127.0.0.1:` + port + `/'); s.onerror = () => res('err:' + s.url); s.onopen = () => res('open'); setTimeout(() => res('timeout'), 3000); });
return { f, ws };`
	fp := fpLiveBase("E")
	fp.PortScanProtection = true
	got := fpLiveAt(t, fp, expr, "shop.example.test")
	t.Logf("protected: %v", got)
	if got["f"] != "blocked" || got["ws"] != "err:ws://127.0.0.1:"+port+"/" {
		t.Errorf("local service reachable from a remote page: %v", got)
	}
	fp.ScanWhitelist = port
	got = fpLiveAt(t, fp, expr, "shop.example.test")
	t.Logf("whitelisted: %v", got)
	if got["f"] != "open" {
		t.Errorf("whitelisted port blocked: %v", got)
	}
	fp.PortScanProtection = false
	got = fpLiveAt(t, fp, expr, "shop.example.test")
	if got["f"] != "open" {
		t.Errorf("protection off but blocked: %v", got)
	}
}
