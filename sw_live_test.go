package main

import (
	"encoding/json"
	"fmt"
	"ghostbrowser/internal/undetectable"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// GHOST_CHROME=/usr/bin/chromium go test -run TestServiceWorkerFingerprintLive .
//
// A service worker attaches to the emulator twice (browser-level auto-attach
// and the auto-attach of the page it serves). It must still activate (two
// sessions driving the worker debugger used to leave it stuck before
// activation, so pages using a service worker never finished) and run the
// page script before its own code, also after it was stopped and started
// again by a navigation (an idle worker keeps its sessions and waits for the
// debugger on them: unanswered, that navigation hung).
func TestServiceWorkerFingerprintLive(t *testing.T) {
	bin := os.Getenv("GHOST_CHROME")
	if bin == "" {
		t.Skip("set GHOST_CHROME to a Chromium binary")
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/sw.js":
			w.Header().Set("Content-Type", "text/javascript")
			fmt.Fprint(w, `const first = [navigator.hardwareConcurrency, navigator.userAgent];
self.addEventListener('install', e => self.skipWaiting());
self.addEventListener('activate', e => e.waitUntil(clients.claim()));
self.addEventListener('fetch', e => { if (e.request.url.includes('/never')) e.respondWith(fetch(e.request)); });
self.addEventListener('message', e => e.source.postMessage({first, now: [navigator.hardwareConcurrency, navigator.userAgent]}));`)
		default:
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprint(w, `<html><body><script>
window.res = (async () => {
  await navigator.serviceWorker.register('/sw.js');
  const reg = await navigator.serviceWorker.ready;
  return await new Promise(ok => { navigator.serviceWorker.onmessage = e => ok(e.data); reg.active.postMessage('q'); });
})();
</script></body></html>`)
		}
	}))
	defer srv.Close()
	fp := fpLiveBase("sw-live")
	fp.HardwareConc = 12
	port := freePort()
	args := append([]string{"--headless=new", "--no-sandbox", "--no-proxy-server", "--user-data-dir=" + liveUserDataDir(t), fmt.Sprintf("--remote-debugging-port=%d", port), "--user-agent=" + fp.UserAgent}, launchFlags(fp)...)
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
	raw, err := c.call("", "Target.createTarget", map[string]any{"url": srv.URL + "/"})
	if err != nil {
		t.Fatal(err)
	}
	var ct struct{ TargetID string }
	_ = json.Unmarshal(raw, &ct)
	r, err := c.call("", "Target.attachToTarget", map[string]any{"targetId": ct.TargetID, "flatten": true})
	if err != nil {
		t.Fatal(err)
	}
	var at struct{ SessionID string }
	_ = json.Unmarshal(r, &at)
	swCheck(t, c, at.SessionID, "first start")
	// Stop the worker; the next navigation in its scope has to start it again.
	for i := 0; i < 2; i++ {
		_, _ = c.call(at.SessionID, "ServiceWorker.enable", nil)
		if _, err := c.call(at.SessionID, "ServiceWorker.stopAllWorkers", nil); err != nil {
			t.Fatal(err)
		}
		time.Sleep(300 * time.Millisecond)
		t0 := time.Now()
		if _, err := c.call(at.SessionID, "Page.navigate", map[string]any{"url": fmt.Sprintf("%s/?n=%d", srv.URL, i)}); err != nil {
			t.Fatalf("navigation after the service worker stopped: %v (%v)", err, time.Since(t0))
		}
		if d := time.Since(t0); d > 3*time.Second {
			t.Errorf("navigation after the service worker stopped took %v", d)
		}
		swCheck(t, c, at.SessionID, fmt.Sprintf("restart %d", i+1))
	}
}

func swCheck(t *testing.T, c *cdpConn, sid, what string) {
	t.Helper()
	var got struct {
		First []any `json:"first"`
		Now   []any `json:"now"`
	}
	ok := false
	for end := time.Now().Add(10 * time.Second); time.Now().Before(end) && !ok; time.Sleep(200 * time.Millisecond) {
		r, err := c.call(sid, "Runtime.evaluate", map[string]any{"returnByValue": true, "awaitPromise": true, "timeout": 1500,
			"expression": `Promise.race([window.res ? window.res.then(v => JSON.stringify(v)) : '', new Promise(ok => setTimeout(() => ok(''), 1000))])`})
		if err != nil {
			continue
		}
		var ev struct{ Result struct{ Value string } }
		_ = json.Unmarshal(r, &ev)
		if ev.Result.Value != "" && json.Unmarshal([]byte(ev.Result.Value), &got) == nil {
			ok = true
		}
	}
	if !ok {
		t.Fatalf("%s: service worker never activated / answered", what)
	}
	for _, v := range [][]any{got.First, got.Now} {
		if len(v) != 2 || v[0] != float64(12) || !strings.Contains(fmt.Sprint(v[1]), "Windows") {
			t.Errorf("%s: service worker values %v (first %v), want hardwareConcurrency 12 and the Windows UA", what, v, got.First)
		}
	}
}

// GHOST_CHROME=/usr/bin/chromium go test -run TestOverriderServiceWorkerRestartLive .
//
// The Undetectable kernel's CDP layer (internal/undetectable Overrider) on
// stock Chromium: a navigation that has to start a stopped service worker
// must not hang.
func TestOverriderServiceWorkerRestartLive(t *testing.T) {
	bin := os.Getenv("GHOST_CHROME")
	if bin == "" {
		t.Skip("set GHOST_CHROME to a Chromium binary")
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/sw.js" {
			w.Header().Set("Content-Type", "text/javascript")
			fmt.Fprint(w, `self.addEventListener('install', e => self.skipWaiting()); self.addEventListener('activate', e => e.waitUntil(clients.claim()));
self.addEventListener('fetch', e => { if (e.request.url.includes('/never')) e.respondWith(fetch(e.request)); });`)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<script>window.st='reg';navigator.serviceWorker.register('/sw.js').then(()=>navigator.serviceWorker.ready).then(()=>window.st='ready')</script>`)
	}))
	defer srv.Close()
	port := freePort()
	cmd := exec.Command(bin, "--headless=new", "--no-sandbox", "--no-proxy-server", "--user-data-dir="+liveUserDataDir(t), fmt.Sprintf("--remote-debugging-port=%d", port), "about:blank")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { cmd.Process.Kill(); cmd.Wait() }()
	ws, err := waitDebugger(port, 20*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	cfg := &undetectable.Config{Override: &undetectable.Expected{UA: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/150.0.0.0 Safari/537.36", Platform: "Win32", UADPlatform: "Windows"}}
	stopOv, err := undetectable.NewOverrider(cfg).Start(port)
	if err != nil {
		t.Fatal(err)
	}
	defer stopOv()
	c, err := dialCDP(ws, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.c.Close()
	raw, _ := c.call("", "Target.createTarget", map[string]any{"url": srv.URL + "/"})
	var ct struct{ TargetID string }
	_ = json.Unmarshal(raw, &ct)
	r, err := c.call("", "Target.attachToTarget", map[string]any{"targetId": ct.TargetID, "flatten": true})
	if err != nil {
		t.Fatal(err)
	}
	var at struct{ SessionID string }
	_ = json.Unmarshal(r, &at)
	state := func() string {
		r, _ := c.call(at.SessionID, "Runtime.evaluate", map[string]any{"expression": "window.st", "returnByValue": true})
		var ev struct{ Result struct{ Value string } }
		_ = json.Unmarshal(r, &ev)
		return ev.Result.Value
	}
	for end := time.Now().Add(10 * time.Second); time.Now().Before(end) && state() != "ready"; time.Sleep(200 * time.Millisecond) {
	}
	if s := state(); s != "ready" {
		t.Fatalf("service worker state %q", s)
	}
	for i := 0; i < 2; i++ {
		_, _ = c.call(at.SessionID, "ServiceWorker.enable", nil)
		_, _ = c.call(at.SessionID, "ServiceWorker.stopAllWorkers", nil)
		time.Sleep(300 * time.Millisecond)
		t0 := time.Now()
		if _, err := c.call(at.SessionID, "Page.navigate", map[string]any{"url": fmt.Sprintf("%s/?n=%d", srv.URL, i)}); err != nil {
			t.Fatalf("navigation after the service worker stopped: %v (%v)", err, time.Since(t0))
		}
		if d := time.Since(t0); d > 3*time.Second {
			t.Errorf("navigation after the service worker stopped took %v", d)
		}
		for end := time.Now().Add(5 * time.Second); time.Now().Before(end) && state() != "ready"; time.Sleep(100 * time.Millisecond) {
		}
		if s := state(); s != "ready" {
			t.Errorf("restart %d: service worker state %q", i+1, s)
		}
	}
}
