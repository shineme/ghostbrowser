package main

import (
	"encoding/json"
	"fmt"
	"ghostbrowser/internal/undetectable"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

// Page-load benchmark: bare Chromium vs. the fingerprint emulator and each of
// its features. Not a regular test; run it explicitly:
//
//	GHOST_PERF=1 GHOST_CHROME=/usr/bin/chromium go test -run TestPerfLive -v -timeout 30m .
//
// Knobs: GHOST_PERF_VARIANTS=bare,default,... (default: all),
// GHOST_PERF_LATENCY=30 (ms per response, default 30; 0 = none),
// GHOST_PERF_LOCAL=1 serves from 127.0.0.1 / 127.0.0.2 instead of the
// mapped site-a.test / site-b.test hosts (worst case for port-scan
// protection, whose patterns match loopback hosts),
// GHOST_PERF_REAL=1 adds real sites (bare vs default only).

type perfSite struct {
	a, b       *http.Server
	aURL, bURL string
	latency    time.Duration
	mu         sync.Mutex
	marks      map[string]time.Time // "load:r" / "done:r"
	reqs       int
}

func (s *perfSite) mark(k string) {
	s.mu.Lock()
	if _, ok := s.marks[k]; !ok {
		s.marks[k] = time.Now()
	}
	s.mu.Unlock()
}

func (s *perfSite) waitMark(k string, d time.Duration) (time.Time, bool) {
	for end := time.Now().Add(d); time.Now().Before(end); time.Sleep(5 * time.Millisecond) {
		s.mu.Lock()
		t, ok := s.marks[k]
		s.mu.Unlock()
		if ok {
			return t, true
		}
	}
	return time.Time{}, false
}

const perfPNG = "\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\x00\x00\x00\x01\x00\x00\x00\x01\x08\x06\x00\x00\x00\x1f\x15\xc4\x89\x00\x00\x00\rIDATx\x9cc\xf8\x0f\x00\x00\x01\x01\x00\x05\x18\xd8N\x00\x00\x00\x00IEND\xaeB`\x82"

func (s *perfSite) handler(cross bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		s.reqs++
		s.mu.Unlock()
		if s.latency > 0 {
			time.Sleep(s.latency)
		}
		q := r.URL.Query()
		rv := q.Get("r")
		w.Header().Set("Cache-Control", "no-store")
		p := r.URL.Path
		switch {
		case p == "/mark":
			s.mark(q.Get("ev") + ":" + rv)
			w.Header().Set("Access-Control-Allow-Origin", "*")
		case p == "/p":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			fmt.Fprint(w, s.topPage(rv))
		case p == "/f":
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			var b strings.Builder
			fmt.Fprintf(&b, "<!doctype html><title>f</title><link rel=stylesheet href='/s/f.css?r=%s&i=%s'>", rv, q.Get("i"))
			for k := 0; k < 3; k++ {
				fmt.Fprintf(&b, "<script src='/s/f%d.js?r=%s&i=%s'></script><img src='/s/f%d.png?r=%s&i=%s'>", k, rv, q.Get("i"), k, rv, q.Get("i"))
			}
			if cross {
				fmt.Fprintf(&b, "<script>const w=new Worker('/w.js?r=%s');w.onmessage=()=>parent.postMessage('xw','*');</script>", rv)
			}
			b.WriteString("<p>frame</p>")
			fmt.Fprint(w, b.String())
		case p == "/w.js":
			w.Header().Set("Content-Type", "text/javascript")
			fmt.Fprintf(w, "importScripts('/s/wi1.js?r=%s','/s/wi2.js?r=%s');postMessage(navigator.userAgent);", rv, rv)
		case p == "/sw.js":
			w.Header().Set("Content-Type", "text/javascript")
			fmt.Fprint(w, "self.addEventListener('install',e=>self.skipWaiting());self.addEventListener('activate',e=>e.waitUntil(clients.claim()));self.addEventListener('fetch',e=>{if(new URL(e.request.url).pathname.startsWith('/s/sw'))e.respondWith(fetch(e.request));});")
		case strings.HasSuffix(p, ".css"):
			w.Header().Set("Content-Type", "text/css")
			fmt.Fprintf(w, ".c%s{color:#123}\n", strings.TrimSuffix(filepath.Base(p), ".css"))
		case strings.HasSuffix(p, ".js"):
			w.Header().Set("Content-Type", "text/javascript")
			// a little real work per script
			fmt.Fprint(w, "(()=>{let x=0;for(let i=0;i<20000;i++)x+=i%7;self.__x=(self.__x||0)+x;})();")
		case strings.HasSuffix(p, ".png"):
			w.Header().Set("Content-Type", "image/png")
			fmt.Fprint(w, perfPNG)
		default:
			http.NotFound(w, r)
		}
	})
}

func (s *perfSite) topPage(rv string) string {
	var b strings.Builder
	b.WriteString("<!doctype html><html><head><meta charset=utf-8><title>perf</title>")
	for i := 0; i < 30; i++ {
		fmt.Fprintf(&b, "<link rel=stylesheet href='/s/c%d.css?r=%s'>", i, rv)
	}
	for i := 0; i < 20; i++ {
		fmt.Fprintf(&b, "<script src='/s/h%d.js?r=%s'></script>", i, rv)
	}
	b.WriteString("</head><body><h1>perf</h1>")
	for i := 0; i < 70; i++ {
		fmt.Fprintf(&b, "<img width=4 height=4 src='/s/i%d.png?r=%s'>", i, rv)
	}
	for i := 0; i < 4; i++ {
		fmt.Fprintf(&b, "<iframe width=60 height=40 src='/f?r=%s&i=%d'></iframe>", rv, i)
	}
	for i := 0; i < 2; i++ {
		fmt.Fprintf(&b, "<iframe width=60 height=40 src='%s/f?r=%s&i=x%d'></iframe>", s.bURL, rv, i)
	}
	for i := 0; i < 20; i++ {
		fmt.Fprintf(&b, "<script defer src='/s/d%d.js?r=%s'></script>", i, rv)
	}
	fmt.Fprintf(&b, `<script>
const R=%q, mk=ev=>fetch('/mark?ev='+ev+'&r='+R,{cache:'no-store'});
addEventListener('load',()=>mk('load'));
const wk=new Promise(res=>{const w=new Worker('/w.js?r='+R);w.onmessage=res;});
const xw=new Promise(res=>{let n=0;addEventListener('message',e=>{if(e.data==='xw'&&++n===2)res();});});
const sw=navigator.serviceWorker?navigator.serviceWorker.register('/sw.js').then(()=>navigator.serviceWorker.ready).then(()=>fetch('/s/sw.js?r='+R)):Promise.resolve();
const ld=new Promise(res=>addEventListener('load',res));
if(location.search.includes('-warm-')||location.search.includes('--1-')){const o=indexedDB.open('perfdb',1);o.onupgradeneeded=()=>o.result.createObjectStore('s');o.onsuccess=()=>{const db=o.result;const tx=db.transaction('s','readwrite');const st=tx.objectStore('s');const v='x'.repeat(2048);for(let i=0;i<%d;i++)st.put({i,v,a:[i,i+1,{k:'z'+i}]},i);};}
window.__st={};[['wk',wk],['xw',xw],['sw',sw],['ld',ld]].forEach(([k,p])=>p.then(()=>__st[k]=1,e=>__st[k]=String(e)));
Promise.all([wk,xw,sw,ld]).then(()=>mk('done'));
</script></body></html>`, rv, perfIDBRecords())
	return b.String()
}

func newPerfSite(t *testing.T, local bool, latency time.Duration) *perfSite {
	s := &perfSite{marks: map[string]time.Time{}, latency: latency}
	la, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	bHost := "127.0.0.2"
	lb, err := net.Listen("tcp", bHost+":0")
	if err != nil {
		t.Fatal(err)
	}
	pa, pb := la.Addr().(*net.TCPAddr).Port, lb.Addr().(*net.TCPAddr).Port
	if local {
		s.aURL, s.bURL = fmt.Sprintf("http://127.0.0.1:%d", pa), fmt.Sprintf("http://127.0.0.2:%d", pb)
	} else {
		s.aURL, s.bURL = fmt.Sprintf("http://site-a.test:%d", pa), fmt.Sprintf("http://site-b.test:%d", pb)
	}
	s.a = &http.Server{Handler: s.handler(false)}
	s.b = &http.Server{Handler: s.handler(true)}
	go s.a.Serve(la)
	go s.b.Serve(lb)
	t.Cleanup(func() { s.a.Close(); s.b.Close() })
	return s
}

// perfFlags are added to every variant (bare included).
func (s *perfSite) flags() []string {
	if os.Getenv("GHOST_PERF_REAL") != "" {
		return []string{"--headless=new", "--no-sandbox"}
	}
	return []string{"--headless=new", "--no-sandbox", "--no-proxy-server",
		"--host-resolver-rules=MAP site-a.test 127.0.0.1,MAP site-b.test 127.0.0.2",
		"--unsafely-treat-insecure-origin-as-secure=" + s.aURL + "," + s.bURL}
}

func perfFP() Fingerprint {
	// The fingerprint a new Windows desktop profile gets (GenerateTyped is what
	// the UI's "new profile" calls). Follow-IP is switched off so launch does
	// not depend on an external IP check; it does not touch page loading.
	fp := GenerateTyped("desktop", "windows", "")
	fp.TimezoneFromIP, fp.LanguageFromIP, fp.GeoFromIP = false, false, false
	return fp
}

type perfResult struct {
	name             string
	load, done, ntab []float64
	total            float64
	err              string
}

func median(v []float64) float64 {
	if len(v) == 0 {
		return 0
	}
	c := append([]float64(nil), v...)
	sort.Float64s(c)
	return c[len(c)/2]
}

// perfMeasure drives navigations over its own CDP client (identical for every
// variant) and times them with server-side marks: Page.navigate → page load
// event, and → "done" (load + dedicated worker + both cross-origin iframe
// workers + service worker controlled fetch).
func perfMeasure(t *testing.T, s *perfSite, ws, tag string, navs int) perfResult {
	res := perfResult{name: tag}
	tag = strings.NewReplacer("+", "p", "-", "_").Replace(tag) // URL-safe mark keys
	c, err := dialCDP(ws, nil)
	if err != nil {
		res.err = err.Error()
		return res
	}
	defer c.c.Close()
	var sid string
	for i := 0; i < 50 && sid == ""; i++ {
		raw, _ := c.call("", "Target.getTargets", nil)
		var tl struct {
			TargetInfos []struct{ TargetID, Type, URL string } `json:"targetInfos"`
		}
		_ = json.Unmarshal(raw, &tl)
		for _, ti := range tl.TargetInfos {
			if ti.Type == "page" {
				r, err := c.call("", "Target.attachToTarget", map[string]any{"targetId": ti.TargetID, "flatten": true})
				if err == nil {
					var at struct{ SessionID string }
					_ = json.Unmarshal(r, &at)
					sid = at.SessionID
					break
				}
			}
		}
		if sid == "" {
			time.Sleep(100 * time.Millisecond)
		}
	}
	if sid == "" {
		res.err = "no page target"
		return res
	}
	start := time.Now()
	for i := -1; i < navs; i++ { // i == -1: warm-up (process spin-up, SW install)
		rv := fmt.Sprintf("%s-%d-%d", tag, i, time.Now().UnixNano())
		t0 := time.Now()
		if i == 0 {
			start = t0
		}
		if os.Getenv("GHOST_PERF_SWSTOP") != "" && i >= 0 {
			// stopped service worker: the navigation itself has to start it
			_, _ = c.call(sid, "ServiceWorker.enable", nil)
			_, _ = c.call(sid, "ServiceWorker.stopAllWorkers", nil)
			time.Sleep(200 * time.Millisecond)
			t0 = time.Now()
		}
		if _, err := c.call(sid, "Page.navigate", map[string]any{"url": s.aURL + "/p?r=" + rv}); err != nil {
			res.err = err.Error()
			return res
		}
		tl, ok1 := s.waitMark("load:"+rv, 20*time.Second)
		td, ok2 := s.waitMark("done:"+rv, 10*time.Second)
		if !ok1 || !ok2 {
			r, _ := c.call(sid, "Runtime.evaluate", map[string]any{"expression": "location.href+' '+document.readyState+' '+JSON.stringify(window.__st)", "returnByValue": true})
			s.mu.Lock()
			n := s.reqs
			s.mu.Unlock()
			res.err = fmt.Sprintf("nav %d timed out (load %v done %v) reqs=%d page=%s", i, ok1, ok2, n, r)
			if !ok1 || os.Getenv("GHOST_PERF_DONE_FATAL") != "" {
				return res
			}
			td = time.Now() // "done" never came (counted as the 10 s wait)
		}
		if i >= 0 {
			res.load = append(res.load, float64(tl.Sub(t0).Microseconds())/1000)
			res.done = append(res.done, float64(td.Sub(t0).Microseconds())/1000)
		}
	}
	res.total = float64(time.Since(start).Microseconds()) / 1000
	// new tabs: target creation → done (exercises the paused-new-target path)
	for i := 0; i < 3; i++ {
		rv := fmt.Sprintf("%s-tab%d-%d", tag, i, time.Now().UnixNano())
		t0 := time.Now()
		raw, err := c.call("", "Target.createTarget", map[string]any{"url": s.aURL + "/p?r=" + rv})
		if err != nil {
			res.err = err.Error()
			return res
		}
		td, ok := s.waitMark("done:"+rv, 30*time.Second)
		var ct struct{ TargetID string }
		_ = json.Unmarshal(raw, &ct)
		if !ok {
			res.err = "new tab timed out"
			return res
		}
		res.ntab = append(res.ntab, float64(td.Sub(t0).Microseconds())/1000)
		_, _ = c.call("", "Target.closeTarget", map[string]any{"targetId": ct.TargetID})
	}
	return res
}

func perfStartBare(t *testing.T, bin string, args []string) (ws string, stop func()) {
	cmd := exec.Command(bin, append(mergeFlags(args), "about:blank")...)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	stop = func() { _ = cmd.Process.Kill(); _, _ = cmd.Process.Wait() }
	port := 0
	for _, a := range args {
		if v, ok := strings.CutPrefix(a, "--remote-debugging-port="); ok {
			port, _ = strconv.Atoi(v)
		}
	}
	ws, err := waitDebugger(port, 20*time.Second)
	if err != nil {
		stop()
		t.Fatal(err)
	}
	return ws, stop
}

type perfVariant struct {
	name   string
	setup  func(fp *Fingerprint) // fingerprint toggles
	knobs  func()                // emuPerf toggles
	extra  func(t *testing.T, ws string) func()
	bare   bool
	real   bool // full launchChromium path (session store, restore, ...)
	und    bool // Undetectable (定制 kernel) path: runtime.go flags + Overrider
	undCDP bool // ... with the Overrider attached
	navs   int
}

func perfVariants() []perfVariant {
	snapLoop := func(every time.Duration) func(t *testing.T, ws string) func() {
		return func(t *testing.T, ws string) func() {
			dir := t.TempDir()
			stop := make(chan struct{})
			go func() {
				for n := 1; ; n++ {
					select {
					case <-stop:
						return
					case <-time.After(every):
					}
					t0 := time.Now()
					_ = snapshotSessionTick(dir, "dev", ws, n)
					perfLog("snapshot tick %d (idb=%v) took %v", n, n%sessionIDBEvery == 0, time.Since(t0))
				}
			}()
			return func() { close(stop) }
		}
	}
	// realSnapLoop is afterStart's cadence (sessionsync.go): 5 s, then every
	// 15 s, IndexedDB on every 4th tick.
	realSnapLoop := func(t *testing.T, ws string) func() {
		dir := t.TempDir()
		stop := make(chan struct{})
		go func() {
			wait := 5 * time.Second
			for n := 1; ; n++ {
				select {
				case <-stop:
					return
				case <-time.After(wait):
				}
				wait = 15 * time.Second
				t0 := time.Now()
				_ = snapshotSessionTick(dir, "dev", ws, n)
				perfLog("snapshot tick %d (idb=%v) took %v", n, n%sessionIDBEvery == 0, time.Since(t0))
			}
		}()
		return func() { close(stop) }
	}
	return []perfVariant{
		{name: "bare", bare: true},
		{name: "default", real: true},
		{name: "emu"},
		{name: "emu+portscan", setup: func(fp *Fingerprint) { fp.PortScanProtection = true }},
		{name: "emu+blockvideo", setup: func(fp *Fingerprint) { fp.BlockVideo = true }},
		{name: "emu+mediaalias", setup: func(fp *Fingerprint) { fp.MediaIDAlias = true }},
		{name: "emu+snap1s", extra: snapLoop(time.Second)},
		{name: "emu+framereg", extra: perfFrameReg},
		{name: "emu-nowait", knobs: func() { emuPerf.noWait = true }},
		{name: "emu-noscript", knobs: func() { emuPerf.noScript = true }},
		{name: "emu-noiframe", knobs: func() { emuPerf.noChild = true }},
		{name: "emu-noworker", knobs: func() { emuPerf.noWorker = true }},
		{name: "emu-v081sw", knobs: func() { emuPerf.legacySW = true }}, // v0.8.1 service worker handling
		// Undetectable path (internal/undetectable/runtime.go Spec.Start) on
		// stock Chromium: same command line, same Overrider, same afterStart
		// session store loop; only the kernel's native patches are missing.
		{name: "und-flags", und: true},
		{name: "und", und: true, undCDP: true},
		{name: "und+snap1s", und: true, undCDP: true, extra: snapLoop(time.Second)},
		{name: "und-full", und: true, undCDP: true, extra: realSnapLoop, navs: 16},
		{name: "bare-long", bare: true, navs: 16},
	}
}

// perfFrameReg mimics a recorder / syncer master session: a second client on
// the tab auto-attaching (paused) iframes and guarding the UA on each.
func perfFrameReg(t *testing.T, ws string) func() {
	var c *cdpConn
	c, err := dialCDP(ws, func(m cdpMsg) {
		if m.Method != "Target.attachedToTarget" {
			return
		}
		var ev struct {
			SessionID  string                `json:"sessionId"`
			TargetInfo struct{ Type string } `json:"targetInfo"`
			Waiting    bool                  `json:"waitingForDebugger"`
		}
		_ = json.Unmarshal(m.Params, &ev)
		if ev.TargetInfo.Type == "iframe" {
			guardUA(c, ws, ev.SessionID)
			_, _ = c.call(ev.SessionID, "Page.enable", nil)
			autoAttachIframes(c, ev.SessionID, true)
		}
		if ev.Waiting {
			_, _ = c.call(ev.SessionID, "Runtime.runIfWaitingForDebugger", nil)
		}
	})
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := c.call("", "Target.getTargets", nil)
	var tl struct {
		TargetInfos []struct{ TargetID, Type string } `json:"targetInfos"`
	}
	_ = json.Unmarshal(raw, &tl)
	for _, ti := range tl.TargetInfos {
		if ti.Type == "page" {
			r, _ := c.call("", "Target.attachToTarget", map[string]any{"targetId": ti.TargetID, "flatten": true})
			var at struct{ SessionID string }
			_ = json.Unmarshal(r, &at)
			guardUA(c, ws, at.SessionID)
			_, _ = c.call(at.SessionID, "Page.enable", nil)
			autoAttachIframes(c, at.SessionID, true)
			break
		}
	}
	return func() { c.c.Close() }
}

func TestPerfLive(t *testing.T) {
	bin := os.Getenv("GHOST_CHROME")
	if os.Getenv("GHOST_PERF") == "" || bin == "" {
		t.Skip("set GHOST_PERF=1 and GHOST_CHROME")
	}
	lat := 30
	if v := os.Getenv("GHOST_PERF_LATENCY"); v != "" {
		lat, _ = strconv.Atoi(v)
	}
	local := os.Getenv("GHOST_PERF_LOCAL") != ""
	realSites := os.Getenv("GHOST_PERF_REAL") != ""
	want := map[string]bool{}
	for _, v := range strings.Split(os.Getenv("GHOST_PERF_VARIANTS"), ",") {
		if v = strings.TrimSpace(v); v != "" {
			want[v] = true
		}
	}
	navs := 5
	site := newPerfSite(t, local, time.Duration(lat)*time.Millisecond)
	reps := 1
	if v, err := strconv.Atoi(os.Getenv("GHOST_PERF_REPS")); err == nil && v > 0 {
		reps = v
	}
	var plan []perfVariant
	for rep := 0; rep < reps; rep++ { // interleaved, so drift hits every variant alike
		for _, v := range perfVariants() {
			if len(want) == 0 || want[v.name] {
				plan = append(plan, v)
			}
		}
	}
	agg := map[string]*perfResult{}
	var order []string
	for _, v := range plan {
		emuPerf = struct{ noWait, noScript, noChild, noWorker, legacySW bool }{}
		if v.knobs != nil {
			v.knobs()
		}
		fp := perfFP()
		if v.setup != nil {
			v.setup(&fp)
		}
		var ws string
		var stop func()
		switch {
		case v.real:
			dir := t.TempDir()
			t.Setenv("XDG_CONFIG_HOME", dir)
			t.Setenv("HOME", dir)
			st := NewStore()
			set := st.GetSettings()
			set.ChromiumPath = bin
			fp.StartupParameters = strings.Join(func() []string {
				out := []string{}
				for _, f := range site.flags() {
					out = append(out, `"`+f+`"`)
				}
				return out
			}(), "\n")
			p := Profile{ID: "perf" + newID(), Name: "perf", Kernel: "chromium", Fingerprint: fp}
			st.PutProfile(p)
			rs, err := st.launchChromium(&p, set)
			if err != nil {
				t.Fatalf("%s: %v", v.name, err)
			}
			ws = rs.WSURL
			id := p.ID
			stop = func() { runReg.stop(id) }
		case v.und:
			ud := t.TempDir()
			port := freePort()
			// internal/undetectable/runtime.go Spec.Start command line
			args := []string{"--profile-id=ghost_perf", "--user-data-dir=" + ud, fmt.Sprintf("--remote-debugging-port=%d", port), "--remote-debugging-address=127.0.0.1",
				"--no-first-run", "--no-default-browser-check", "--enable-chrome-browser-cloud-management", "--enable-privacy-sandbox-ads-apis", "--lang=" + firstLang(fp.Languages)}
			args = append(args, site.flags()...)
			if os.Getenv("GHOST_PERF_NOCBCM") != "" {
				args = slicesDelete(args, "--enable-chrome-browser-cloud-management", "--enable-privacy-sandbox-ads-apis")
			}
			var kill func()
			ws, kill = perfStartBare(t, bin, args)
			stop = kill
			if v.undCDP {
				cfg := &undetectable.Config{Override: &undetectable.Expected{UA: fp.UserAgent, Platform: "Win32", UADPlatform: "Windows",
					Languages: fp.Languages, LangHeader: strings.Join(fp.Languages, ",")}}
				stopOv, err := undetectable.NewOverrider(cfg).Start(port)
				if err != nil {
					kill()
					t.Fatalf("%s: %v", v.name, err)
				}
				stop = func() { stopOv(); kill() }
			}
		default:
			ud := t.TempDir()
			port := freePort()
			args := append(chromiumBaseArgs(ud, port, fp, nil, &Profile{}), site.flags()...)
			var kill func()
			ws, kill = perfStartBare(t, bin, args)
			stop = kill
			if !v.bare {
				em, err := startEmulation(ws, fp, nil, nil)
				if err != nil {
					kill()
					t.Fatalf("%s: %v", v.name, err)
				}
				stop = func() { em.Close(); kill() }
			}
		}
		var undo func()
		if v.extra != nil {
			undo = v.extra(t, ws)
		}
		time.Sleep(800 * time.Millisecond)
		n := navs
		if v.navs > 0 {
			n = v.navs
		}
		var r perfResult
		if realSites {
			r = perfMeasureReal(t, ws, v.name)
		} else {
			r = perfMeasure(t, site, ws, v.name, n)
		}
		if undo != nil {
			undo()
		}
		stop()
		time.Sleep(500 * time.Millisecond)
		if a := agg[v.name]; a == nil {
			cp := r
			agg[v.name] = &cp
			order = append(order, v.name)
		} else {
			a.load, a.done, a.ntab = append(a.load, r.load...), append(a.done, r.done...), append(a.ntab, r.ntab...)
			a.total += r.total
			if r.err != "" {
				a.err = r.err
			}
		}
		t.Logf("%-16s load med %7.1f  done med %7.1f  total(5) %8.1f  newtab med %7.1f  load=%v done=%v tab=%v %s",
			r.name, median(r.load), median(r.done), r.total, median(r.ntab), r.load, r.done, r.ntab, r.err)
	}
	emuPerf = struct{ noWait, noScript, noChild, noWorker, legacySW bool }{}
	var b strings.Builder
	fmt.Fprintf(&b, "\nlatency=%dms local=%v real=%v reps=%d idb=%d\n| variant | n | load median ms | done median ms | navs total ms (sum over reps) | new tab median ms |\n|---|---|---|---|---|---|\n", lat, local, realSites, reps, perfIDBRecords())
	for _, name := range order {
		r := *agg[name]
		e := ""
		if r.err != "" {
			e = " (" + r.err + ")"
		}
		fmt.Fprintf(&b, "| %s%s | %d | %.0f | %.0f | %.0f | %.0f |\n", r.name, e, len(r.load), median(r.load), median(r.done), r.total, median(r.ntab))
	}
	t.Log(b.String())
	if out := os.Getenv("GHOST_PERF_OUT"); out != "" {
		f, _ := os.OpenFile(out, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if f != nil {
			f.WriteString(b.String())
			f.Close()
		}
	}
}

func perfIDBRecords() int {
	if v, err := strconv.Atoi(os.Getenv("GHOST_PERF_IDB")); err == nil {
		return v
	}
	return 2000 // ~4 MB in IndexedDB, written once by the warm-up navigation
}

func perfLog(f string, a ...any) {
	fmt.Fprintf(os.Stderr, "PERFLOG %s "+f+"\n", append([]any{time.Now().Format("15:04:05.000")}, a...)...)
}

func slicesDelete(in []string, drop ...string) []string {
	out := in[:0:0]
	for _, a := range in {
		keep := true
		for _, d := range drop {
			if a == d {
				keep = false
			}
		}
		if keep {
			out = append(out, a)
		}
	}
	return out
}

var perfRealURLs = []string{"https://github.com/golang/go", "https://github.com/golang/go/releases", "https://www.wikipedia.org/"}

// perfMeasureReal loads real sites (1 warm-up + 3 each) and reads the
// navigation timing (navigationStart → loadEventEnd) from the page.
func perfMeasureReal(t *testing.T, ws, tag string) perfResult {
	res := perfResult{name: tag}
	c, err := dialCDP(ws, nil)
	if err != nil {
		res.err = err.Error()
		return res
	}
	defer c.c.Close()
	raw, _ := c.call("", "Target.createTarget", map[string]any{"url": "about:blank"})
	var ct struct{ TargetID string }
	_ = json.Unmarshal(raw, &ct)
	r, err := c.call("", "Target.attachToTarget", map[string]any{"targetId": ct.TargetID, "flatten": true})
	if err != nil {
		res.err = err.Error()
		return res
	}
	var at struct{ SessionID string }
	_ = json.Unmarshal(r, &at)
	sid := at.SessionID
	start := time.Now()
	for _, u := range perfRealURLs {
		for i := -1; i < 3; i++ {
			_, _ = c.call(sid, "Page.navigate", map[string]any{"url": u})
			var ms float64
			for end := time.Now().Add(30 * time.Second); time.Now().Before(end); time.Sleep(50 * time.Millisecond) {
				r, err := c.call(sid, "Runtime.evaluate", map[string]any{"returnByValue": true, "expression": `(()=>{const n=performance.getEntriesByType('navigation')[0];return n&&n.loadEventEnd>0&&location.href!=='about:blank'?n.loadEventEnd:0})()`})
				if err != nil {
					continue
				}
				var ev struct{ Result struct{ Value float64 } }
				_ = json.Unmarshal(r, &ev)
				if ev.Result.Value > 0 {
					ms = ev.Result.Value
					break
				}
			}
			if i >= 0 {
				res.load = append(res.load, ms)
			}
			perfLog("%s %s #%d load %.0f ms", tag, u, i, ms)
			_, _ = c.call(sid, "Page.navigate", map[string]any{"url": "about:blank"})
			time.Sleep(300 * time.Millisecond)
		}
	}
	res.total = float64(time.Since(start).Milliseconds())
	return res
}

// TestPerfSnapshotWorldsLive counts the execution contexts and the JS heap of
// a long-lived tab after N session store ticks (each tick attaches and asks
// for an isolated world in every tab).
//
//	GHOST_PERF=1 GHOST_CHROME=/usr/bin/chromium go test -run TestPerfSnapshotWorldsLive -v .
func TestPerfSnapshotWorldsLive(t *testing.T) {
	bin := os.Getenv("GHOST_CHROME")
	if os.Getenv("GHOST_PERF") == "" || bin == "" {
		t.Skip("set GHOST_PERF=1 and GHOST_CHROME")
	}
	site := newPerfSite(t, false, 0)
	ticks := 40
	if v, err := strconv.Atoi(os.Getenv("GHOST_PERF_TICKS")); err == nil {
		ticks = v
	}
	port := freePort()
	ws, stop := perfStartBare(t, bin, append([]string{"--user-data-dir=" + liveUserDataDir(t), fmt.Sprintf("--remote-debugging-port=%d", port)}, site.flags()...))
	defer stop()
	c, err := dialCDP(ws, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.c.Close()
	raw, _ := c.call("", "Target.createTarget", map[string]any{"url": site.aURL + "/p?r=worlds"})
	var ct struct{ TargetID string }
	_ = json.Unmarshal(raw, &ct)
	time.Sleep(3 * time.Second)
	measure := func() (int, float64) {
		var mu sync.Mutex
		n := 0
		mc, err := dialCDP(ws, func(m cdpMsg) {
			if m.Method == "Runtime.executionContextCreated" {
				mu.Lock()
				n++
				mu.Unlock()
			}
		})
		if err != nil {
			t.Fatal(err)
		}
		defer mc.c.Close()
		r, _ := mc.call("", "Target.attachToTarget", map[string]any{"targetId": ct.TargetID, "flatten": true})
		var at struct{ SessionID string }
		_ = json.Unmarshal(r, &at)
		_, _ = mc.call(at.SessionID, "Runtime.enable", nil)
		_, _ = mc.call(at.SessionID, "HeapProfiler.collectGarbage", nil)
		r, _ = mc.call(at.SessionID, "Runtime.getHeapUsage", nil)
		var hu struct{ UsedSize, TotalSize float64 }
		_ = json.Unmarshal(r, &hu)
		time.Sleep(300 * time.Millisecond)
		mu.Lock()
		defer mu.Unlock()
		return n, hu.UsedSize / 1e6
	}
	n0, h0 := measure()
	dir := t.TempDir()
	t0 := time.Now()
	for i := 1; i <= ticks; i++ {
		_ = snapshotSessionTick(dir, "dev", ws, i)
	}
	el := time.Since(t0)
	n1, h1 := measure()
	t.Logf("before: %d contexts, heap %.1f MB; after %d ticks (%v): %d contexts, heap %.1f MB", n0, h0, ticks, el, n1, h1)
	if out := os.Getenv("GHOST_PERF_OUT"); out != "" {
		f, _ := os.OpenFile(out, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if f != nil {
			fmt.Fprintf(f, "\nsnapshot worlds: before %d contexts / heap %.1f MB; after %d ticks (%v): %d contexts / heap %.1f MB\n", n0, h0, ticks, el.Round(time.Millisecond), n1, h1)
			f.Close()
		}
	}
}

// TestPerfSnapshotJankLive measures how long the session store's IndexedDB
// capture blocks a tab's main thread (worst event-loop lag seen by the page)
// and how long a page load overlapping a capture takes.
//
//	GHOST_PERF=1 GHOST_CHROME=/usr/bin/chromium go test -run TestPerfSnapshotJankLive -v .
func TestPerfSnapshotJankLive(t *testing.T) {
	bin := os.Getenv("GHOST_CHROME")
	if os.Getenv("GHOST_PERF") == "" || bin == "" {
		t.Skip("set GHOST_PERF=1 and GHOST_CHROME")
	}
	site := newPerfSite(t, false, 0)
	port := freePort()
	ws, stop := perfStartBare(t, bin, append([]string{"--user-data-dir=" + liveUserDataDir(t), fmt.Sprintf("--remote-debugging-port=%d", port)}, site.flags()...))
	defer stop()
	c, err := dialCDP(ws, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.c.Close()
	raw, _ := c.call("", "Target.createTarget", map[string]any{"url": site.aURL + "/p?r=jank--1-x"}) // "--1-": writes the IndexedDB data
	var ct struct{ TargetID string }
	_ = json.Unmarshal(raw, &ct)
	time.Sleep(4 * time.Second)
	r, _ := c.call("", "Target.attachToTarget", map[string]any{"targetId": ct.TargetID, "flatten": true})
	var at struct{ SessionID string }
	_ = json.Unmarshal(r, &at)
	eval := func(expr string) string {
		r, _ := c.call(at.SessionID, "Runtime.evaluate", map[string]any{"expression": expr, "returnByValue": true})
		var ev struct{ Result struct{ Value any } }
		_ = json.Unmarshal(r, &ev)
		return fmt.Sprint(ev.Result.Value)
	}
	eval(`window.__lag=0;(()=>{let last=performance.now();setInterval(()=>{const n=performance.now();window.__lag=Math.max(window.__lag,n-last-10);last=n;},10);})();1`)
	time.Sleep(500 * time.Millisecond)
	base := eval(`Math.round(window.__lag)`)
	dir := t.TempDir()
	var lines []string
	for i := 1; i <= 3; i++ {
		eval(`window.__lag=0`)
		t0 := time.Now()
		_ = snapshotSessionTick(dir, "dev", ws, 4*i) // an IndexedDB tick
		el := time.Since(t0)
		time.Sleep(200 * time.Millisecond)
		lines = append(lines, fmt.Sprintf("idb tick %d: %v, worst main-thread lag %s ms", i, el.Round(time.Millisecond), eval(`Math.round(window.__lag)`)))
	}
	// a capture overlapping page loads in a second tab
	var navs []float64
	for i := 0; i < 3; i++ {
		done := make(chan struct{})
		go func() { _ = snapshotSessionTick(dir, "dev", ws, 4); close(done) }()
		time.Sleep(20 * time.Millisecond)
		rv := fmt.Sprintf("jankload%d-%d", i, time.Now().UnixNano())
		t0 := time.Now()
		raw, _ := c.call("", "Target.createTarget", map[string]any{"url": site.aURL + "/p?r=" + rv})
		var nt struct{ TargetID string }
		_ = json.Unmarshal(raw, &nt)
		if tl, ok := site.waitMark("load:"+rv, 20*time.Second); ok {
			navs = append(navs, float64(tl.Sub(t0).Milliseconds()))
		}
		<-done
		_, _ = c.call("", "Target.closeTarget", map[string]any{"targetId": nt.TargetID})
	}
	var quiet []float64
	for i := 0; i < 3; i++ {
		rv := fmt.Sprintf("quietload%d-%d", i, time.Now().UnixNano())
		t0 := time.Now()
		raw, _ := c.call("", "Target.createTarget", map[string]any{"url": site.aURL + "/p?r=" + rv})
		var nt struct{ TargetID string }
		_ = json.Unmarshal(raw, &nt)
		if tl, ok := site.waitMark("load:"+rv, 20*time.Second); ok {
			quiet = append(quiet, float64(tl.Sub(t0).Milliseconds()))
		}
		_, _ = c.call("", "Target.closeTarget", map[string]any{"targetId": nt.TargetID})
	}
	lines = append(lines, fmt.Sprintf("baseline lag %s ms; new-tab load with an IndexedDB capture running %v ms, without %v ms", base, navs, quiet))
	for _, l := range lines {
		t.Log(l)
	}
	if out := os.Getenv("GHOST_PERF_OUT"); out != "" {
		f, _ := os.OpenFile(out, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if f != nil {
			fmt.Fprintf(f, "\nsnapshot jank (idb=%d records):\n%s\n", perfIDBRecords(), strings.Join(lines, "\n"))
			f.Close()
		}
	}
}
