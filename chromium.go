package main

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"ghostbrowser/internal/undetectable"

	"github.com/gorilla/websocket"
)

// Open-source kernel: Chromium, Chrome for Testing, ungoogled-chromium, Edge,
// Chrome or a Chromium you built yourself. Fingerprints are applied through a
// CDP session GhostBrowser keeps open for the life of the browser: device
// emulation (UA + UA-CH, viewport, DPR, touch, orientation, timezone, locale,
// geolocation) plus a document-start script for GPU, noise, fonts and battery.
// The fingerprint itself needs no extension. Extensions the user assigns in the
// Extension Center are passed with --load-extension; on Chrome 137+ that switch
// only works with DisableLoadExtensionCommandLineSwitch added to the single
// --disable-features flag (branded Google Chrome may still ignore it).

func findChrome() string {
	var cands []string
	switch runtime.GOOS {
	case "windows":
		pf, pf86, local := os.Getenv("ProgramFiles"), os.Getenv("ProgramFiles(x86)"), os.Getenv("LOCALAPPDATA")
		cands = []string{
			filepath.Join(local, "Chromium/Application/chrome.exe"),
			filepath.Join(pf, "Chromium/Application/chrome.exe"),
			filepath.Join(pf, "Google/Chrome/Application/chrome.exe"),
			filepath.Join(pf86, "Google/Chrome/Application/chrome.exe"),
			filepath.Join(local, "Google/Chrome/Application/chrome.exe"),
			filepath.Join(pf86, "Microsoft/Edge/Application/msedge.exe"),
			filepath.Join(pf, "Microsoft/Edge/Application/msedge.exe"),
		}
	case "darwin":
		cands = []string{
			"/Applications/Chromium.app/Contents/MacOS/Chromium",
			"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
			"/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge",
		}
	default:
		cands = []string{"/usr/bin/chromium", "/usr/bin/chromium-browser", "/usr/bin/google-chrome", "/snap/bin/chromium"}
	}
	for _, c := range cands {
		if fileExists(c) {
			return c
		}
	}
	return ""
}

// chromiumKernel returns the open-source kernel to launch, never the custom one.
func chromiumKernel(st Settings) string {
	k := strings.Trim(strings.TrimSpace(st.ChromiumPath), "\"")
	if k != "" && fileExists(k) && !strings.Contains(strings.ToLower(k), "undetectable-kit") {
		return k
	}
	return findChrome()
}

func (s *Store) launchChromium(p *Profile, st Settings) (*RunState, error) {
	kernel := chromiumKernel(st)
	if kernel == "" {
		return nil, fmt.Errorf("没有找到开源内核：请在「设置 → 开源内核」选择 chrome.exe（Chromium、Chrome 或你自己编译的内核）")
	}
	var px *Proxy
	if p.InlineProxy != nil {
		px = p.InlineProxy
	} else if p.ProxyID != "" {
		if pr, ok := s.GetProxy(p.ProxyID); ok {
			px = pr
		}
	}
	fp, err := s.matchFP(p.Fingerprint, px)
	if err != nil {
		return nil, err
	}
	fp.DeviceType = normDevType(fp.DeviceType)
	// The UA's Chrome major always follows the kernel, so UA, UA-CH and the
	// engine's real features agree. TVs keep their own fixed browser build.
	if m := chromeMajorFrom(filepath.Dir(kernel)); m != "" && fp.DeviceType != "tv" {
		fp.UserAgent = chromeMajorTok.ReplaceAllString(fp.UserAgent, "Chrome/"+m+".0.0.0")
		fp.BrowserVersion = m
	}

	userDir := s.ProfileDir(p.ID)
	_ = os.MkdirAll(userDir, 0o755)
	port := freePort()
	win := windowFor(fp)
	args := []string{
		"--user-data-dir=" + userDir,
		fmt.Sprintf("--remote-debugging-port=%d", port),
		"--remote-allow-origins=http://127.0.0.1",
		"--no-first-run", "--no-default-browser-check", "--disable-sync",
		"--disable-features=Translate,MediaRouter",
		"--lang=" + firstLang(fp.Languages),
		"--user-agent=" + fp.UserAgent, // workers and the very first request
		fmt.Sprintf("--window-size=%d,%d", win.w, win.h),
	}
	if isMobileType(fp.DeviceType) || fp.DeviceType == "tablet" {
		args = append(args, "--touch-events=enabled")
	}
	if fp.WebRTC == "disabled" || (fp.WebRTC == "proxy" && px != nil) {
		args = append(args, "--force-webrtc-ip-handling-policy=disable_non_proxied_udp")
	}
	args = withExtensionArgs(args, extensionDirs(p))
	var relay *undetectable.Relay
	if px != nil && px.Type != "none" && px.Host != "" {
		pu := px.URL()
		if undetectable.NeedsRelay(pu.String()) {
			// Chromium can't take proxy credentials on the command line.
			if relay, err = undetectable.StartRelay(pu.String()); err != nil {
				return nil, fmt.Errorf("代理中转启动失败：%w", err)
			}
			args = append(args, "--proxy-server="+relay.URL())
		} else {
			scheme := px.Type
			if scheme == "https" {
				scheme = "http"
			}
			args = append(args, fmt.Sprintf("--proxy-server=%s://%s:%d", scheme, px.Host, px.Port))
		}
	}
	args = append(args, "about:blank")

	cmd := exec.Command(kernel, args...)
	logPath := filepath.Join(userDir, "launch.log")
	logf, _ := os.Create(logPath)
	if logf != nil {
		fmt.Fprintf(logf, "kernel: %s\nargs: %q\n\n", kernel, args)
		cmd.Stdout, cmd.Stderr = logf, logf
	}
	if err := cmd.Start(); err != nil {
		relay.Close()
		if logf != nil {
			logf.Close()
		}
		return nil, fmt.Errorf("无法启动内核 %s: %w", kernel, err)
	}
	rs := &RunState{ProfileID: p.ID, PID: cmd.Process.Pid, DebugPort: port, Started: time.Now().Unix()}
	exited := make(chan error, 1)
	runReg.add(p.ID, rs, cmd)
	go func() {
		err := cmd.Wait()
		runReg.remove(p.ID)
		relay.Close()
		if logf != nil {
			logf.Close()
		}
		exited <- err
	}()

	wsCh := make(chan string, 1)
	go func() { ws, _ := waitDebugger(port, 20*time.Second); wsCh <- ws }()
	var ws string
	select {
	case ws = <-wsCh:
	case err := <-exited:
		return nil, fmt.Errorf("浏览器启动后立即退出（%v）。内核：%s。若该环境已经打开，请先关闭它的窗口%s", err, kernel, logTail(logPath))
	}
	if ws == "" {
		runReg.stop(p.ID)
		return nil, fmt.Errorf("浏览器 20 秒内没有打开调试端口，已终止。内核：%s%s", kernel, logTail(logPath))
	}
	rs.WSURL = ws
	em, err := startEmulation(ws, fp, px, p.StartURLs)
	if err != nil {
		runReg.stop(p.ID)
		return nil, fmt.Errorf("指纹注入失败，已关闭浏览器以免裸奔：%w", err)
	}
	runReg.setCleanup(p.ID, func() { em.Close(); relay.Close() })

	p.LastRun = time.Now().Unix()
	s.PutProfile(*p)
	if strings.TrimSpace(p.Cookies) != "" {
		if err := importCookies(ws, p.Cookies); err == nil {
			p.Cookies = ""
			s.PutProfile(*p)
		}
	}
	s.afterStart(p.ID, ws)
	return rs, nil
}

// logTail returns the last few lines of a launch log for error messages.
func logTail(path string) string {
	b, err := os.ReadFile(path)
	if err != nil || len(b) == 0 {
		return ""
	}
	lines := strings.Split(strings.TrimSpace(string(b)), "\n")
	if len(lines) > 6 {
		lines = lines[len(lines)-6:]
	}
	return "\n" + strings.Join(lines, "\n")
}

var chromeMajorTok = regexp.MustCompile(`Chrome/\d+\.\d+\.\d+\.\d+`)

type winSize struct {
	w, h  int
	scale float64
}

// windowFor sizes the browser window: desktops get 80% of the screen, emulated
// devices get their viewport (scaled down to fit a 1080p monitor) plus browser chrome.
func windowFor(fp Fingerprint) winSize {
	if !emulatedViewport(fp.DeviceType) {
		return winSize{fp.ScreenWidth * 8 / 10, fp.ScreenHeight * 8 / 10, 1}
	}
	const maxW, maxH, chromeW, chromeH = 1280, 900, 16, 120
	sc := math.Min(1, math.Min(float64(maxW-chromeW)/float64(fp.ScreenWidth), float64(maxH-chromeH)/float64(fp.ScreenHeight)))
	w := int(float64(fp.ScreenWidth)*sc) + chromeW
	if w < 520 { // Chromium's minimum window width
		w = 520
	}
	return winSize{w, int(float64(fp.ScreenHeight)*sc) + chromeH, sc}
}

// ---------- CDP session ----------

type cdpMsg struct {
	ID      int64           `json:"id,omitempty"`
	Method  string          `json:"method,omitempty"`
	Params  json.RawMessage `json:"params,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Session string          `json:"sessionId,omitempty"`
	Error   *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

type cdpConn struct {
	c       *websocket.Conn
	wmu     sync.Mutex
	next    int64
	mu      sync.Mutex
	pending map[int64]chan cdpMsg
	onEvent func(cdpMsg)
	done    chan struct{}
}

func dialCDP(ws string, onEvent func(cdpMsg)) (*cdpConn, error) {
	d := websocket.Dialer{HandshakeTimeout: 10 * time.Second}
	c, _, err := d.Dial(ws, nil)
	if err != nil {
		return nil, err
	}
	c.SetReadLimit(64 << 20)
	cc := &cdpConn{c: c, pending: map[int64]chan cdpMsg{}, onEvent: onEvent, done: make(chan struct{})}
	go cc.read()
	return cc, nil
}

func (c *cdpConn) read() {
	defer close(c.done)
	for {
		var m cdpMsg
		if err := c.c.ReadJSON(&m); err != nil {
			c.mu.Lock()
			for id, ch := range c.pending {
				close(ch)
				delete(c.pending, id)
			}
			c.mu.Unlock()
			return
		}
		if m.ID != 0 {
			c.mu.Lock()
			ch := c.pending[m.ID]
			delete(c.pending, m.ID)
			c.mu.Unlock()
			if ch != nil {
				ch <- m
			}
		} else if m.Method != "" && c.onEvent != nil {
			go c.onEvent(m)
		}
	}
}

func (c *cdpConn) call(session, method string, params any) (json.RawMessage, error) {
	id := atomic.AddInt64(&c.next, 1)
	ch := make(chan cdpMsg, 1)
	c.mu.Lock()
	c.pending[id] = ch
	c.mu.Unlock()
	msg := map[string]any{"id": id, "method": method, "params": params}
	if params == nil {
		msg["params"] = map[string]any{}
	}
	if session != "" {
		msg["sessionId"] = session
	}
	c.wmu.Lock()
	err := c.c.WriteJSON(msg)
	c.wmu.Unlock()
	if err != nil {
		return nil, err
	}
	select {
	case m, ok := <-ch:
		if !ok {
			return nil, fmt.Errorf("browser closed")
		}
		if m.Error != nil {
			return nil, fmt.Errorf("%s: %s", method, m.Error.Message)
		}
		return m.Result, nil
	case <-time.After(15 * time.Second):
		c.mu.Lock()
		delete(c.pending, id)
		c.mu.Unlock()
		return nil, fmt.Errorf("%s: timeout", method)
	}
}

// ---------- device emulation ----------

type emulator struct {
	conn     *cdpConn
	fp       Fingerprint
	ua       map[string]any // Emulation.setUserAgentOverride params
	script   string
	existing map[string]bool
	starts   []string
	first    sync.Once
}

func (e *emulator) Close() {
	if e != nil && e.conn != nil {
		e.conn.c.Close()
	}
}

func startEmulation(ws string, fp Fingerprint, px *Proxy, startURLs []string) (*emulator, error) {
	e := &emulator{fp: fp, existing: map[string]bool{}}
	for _, u := range startURLs {
		if u = strings.TrimSpace(u); u != "" {
			e.starts = append(e.starts, u)
		}
	}
	conn, err := dialCDP(ws, e.onEvent)
	if err != nil {
		return nil, err
	}
	e.conn = conn
	brands, full := e.kernelBrands()
	e.ua = uaOverride(fp, brands, full)
	e.script = pageScript(fp, px)

	raw, err := conn.call("", "Target.getTargets", nil)
	if err != nil {
		conn.c.Close()
		return nil, err
	}
	var tl struct {
		TargetInfos []struct {
			TargetID string `json:"targetId"`
			Type     string `json:"type"`
		} `json:"targetInfos"`
	}
	_ = json.Unmarshal(raw, &tl)
	for _, t := range tl.TargetInfos {
		if t.Type == "page" {
			e.existing[t.TargetID] = true
		}
	}
	if _, err = conn.call("", "Target.setAutoAttach", map[string]any{"autoAttach": true, "waitForDebuggerOnStart": true, "flatten": true}); err != nil {
		conn.c.Close()
		return nil, err
	}
	// Extra start pages open as new targets: auto-attach pauses them until emulated.
	go func() {
		time.Sleep(600 * time.Millisecond)
		for i, u := range e.starts {
			if i > 0 {
				_, _ = conn.call("", "Target.createTarget", map[string]any{"url": u})
			}
		}
	}()
	return e, nil
}

// kernelBrands reads the kernel's real UA-CH brand list (Chrome vs Chromium vs
// Edge, and its GREASE entry), so the override never invents a brand set.
func (e *emulator) kernelBrands() ([]map[string]string, []map[string]string) {
	raw, err := e.conn.call("", "Browser.getVersion", nil)
	full := ""
	if err == nil {
		var v struct {
			Product string `json:"product"`
		}
		_ = json.Unmarshal(raw, &v)
		if _, after, ok := strings.Cut(v.Product, "/"); ok {
			full = after
		}
	}
	major := strings.Split(full, ".")[0]
	fallback := []map[string]string{{"brand": "Chromium", "version": major}, {"brand": "Not_A Brand", "version": "24"}}
	fallbackFull := []map[string]string{{"brand": "Chromium", "version": full}, {"brand": "Not_A Brand", "version": "24.0.0.0"}}
	raw, err = e.conn.call("", "Target.getTargets", nil)
	if err != nil {
		return fallback, fallbackFull
	}
	var tl struct {
		TargetInfos []struct {
			TargetID, Type, URL string
		} `json:"targetInfos"`
	}
	_ = json.Unmarshal(raw, &tl)
	for _, t := range tl.TargetInfos {
		if t.Type != "page" {
			continue
		}
		r, err := e.conn.call("", "Target.attachToTarget", map[string]any{"targetId": t.TargetID, "flatten": true})
		if err != nil {
			continue
		}
		var at struct {
			SessionID string `json:"sessionId"`
		}
		_ = json.Unmarshal(r, &at)
		r, err = e.conn.call(at.SessionID, "Runtime.evaluate", map[string]any{
			"expression":    `navigator.userAgentData ? navigator.userAgentData.getHighEntropyValues(["fullVersionList"]).then(v => JSON.stringify({b: navigator.userAgentData.brands, f: v.fullVersionList})) : ""`,
			"awaitPromise":  true,
			"returnByValue": true,
		})
		_, _ = e.conn.call("", "Target.detachFromTarget", map[string]any{"sessionId": at.SessionID})
		if err != nil {
			break
		}
		var ev struct {
			Result struct {
				Value string `json:"value"`
			} `json:"result"`
		}
		_ = json.Unmarshal(r, &ev)
		var bv struct {
			B []map[string]string `json:"b"`
			F []map[string]string `json:"f"`
		}
		if json.Unmarshal([]byte(ev.Result.Value), &bv) == nil && len(bv.B) > 0 && len(bv.F) > 0 {
			return bv.B, bv.F
		}
		break
	}
	return fallback, fallbackFull
}

// uaOverride builds Emulation.setUserAgentOverride params with UA-CH metadata
// that agrees with the fingerprint (platform, version, model, mobile).
func uaOverride(fp Fingerprint, brands, fullList []map[string]string) map[string]any {
	platform, pver, arch, bitness := "Windows", "15.0.0", "x86", "64"
	switch fp.OS {
	case "windows":
		if fp.OSVersion == "10" {
			pver = "10.0.0"
		}
	case "macos":
		platform, pver = "macOS", dotted(fp.OSVersion)
		if strings.Contains(fp.WebGLRenderer, "Apple M") {
			arch = "arm"
		}
	case "linux":
		platform, pver = "Linux", "6.8.0"
	case "android":
		platform, pver, arch, bitness = "Android", dotted(fp.OSVersion), "", ""
	case "chromeos":
		platform, pver = "Chrome OS", fp.OSVersion
		if strings.Contains(fp.Platform, "aarch64") || strings.Contains(fp.UserAgent, "aarch64") {
			arch = "arm"
		}
	case "tizen", "webos":
		platform, pver, arch, bitness = "Linux", "", "arm", "32"
	}
	// Keep the brand versions on the kernel's own numbers; the UA's major follows them.
	full := ""
	for _, b := range fullList {
		if !strings.Contains(b["brand"], "Not") {
			full = b["version"]
		}
	}
	meta := map[string]any{
		"brands": brands, "fullVersionList": fullList, "fullVersion": full,
		"platform": platform, "platformVersion": pver, "architecture": arch, "bitness": bitness,
		"model": fp.DeviceModel, "mobile": isMobileType(fp.DeviceType), "wow64": false,
	}
	return map[string]any{
		"userAgent":         fp.UserAgent,
		"acceptLanguage":    acceptLanguage(fp.Languages),
		"platform":          fp.Platform,
		"userAgentMetadata": meta,
	}
}

func dotted(v string) string {
	parts := strings.Split(v, ".")
	for len(parts) < 3 {
		parts = append(parts, "0")
	}
	return strings.Join(parts[:3], ".")
}

func acceptLanguage(langs []string) string {
	out := []string{}
	for i, l := range langs {
		if i == 0 {
			out = append(out, l)
		} else {
			q := 1 - float64(i)*0.1
			if q < 0.1 {
				q = 0.1
			}
			out = append(out, fmt.Sprintf("%s;q=%.1f", l, q))
		}
	}
	return strings.Join(out, ",")
}

func (e *emulator) onEvent(m cdpMsg) {
	if m.Method != "Target.attachedToTarget" {
		return
	}
	var ev struct {
		SessionID  string `json:"sessionId"`
		TargetInfo struct {
			TargetID string `json:"targetId"`
			Type     string `json:"type"`
			URL      string `json:"url"`
		} `json:"targetInfo"`
		Waiting bool `json:"waitingForDebugger"`
	}
	if json.Unmarshal(m.Params, &ev) != nil {
		return
	}
	sid, c := ev.SessionID, e.conn
	resume := func() {
		if ev.Waiting {
			_, _ = c.call(sid, "Runtime.runIfWaitingForDebugger", nil)
		}
	}
	switch ev.TargetInfo.Type {
	case "page", "iframe":
		e.emulate(sid, ev.TargetInfo.Type == "page")
		_, _ = c.call(sid, "Target.setAutoAttach", map[string]any{"autoAttach": true, "waitForDebuggerOnStart": true, "flatten": true})
		resume()
		if ev.TargetInfo.Type == "page" && e.existing[ev.TargetInfo.TargetID] {
			// Pages that were already open loaded before the overrides: load them again.
			navigated := false
			e.first.Do(func() {
				if len(e.starts) > 0 {
					_, _ = c.call(sid, "Page.navigate", map[string]any{"url": e.starts[0]})
					navigated = true
				}
			})
			if !navigated && strings.HasPrefix(ev.TargetInfo.URL, "http") {
				_, _ = c.call(sid, "Page.reload", map[string]any{"ignoreCache": false})
			}
		}
	case "worker", "shared_worker", "service_worker":
		_, _ = c.call(sid, "Network.setUserAgentOverride", e.ua)
		resume()
	default:
		resume()
	}
}

func (e *emulator) emulate(sid string, top bool) {
	c, fp := e.conn, e.fp
	_, _ = c.call(sid, "Page.enable", nil)
	_, _ = c.call(sid, "Page.addScriptToEvaluateOnNewDocument", map[string]any{"source": e.script, "runImmediately": true})
	_, _ = c.call(sid, "Emulation.setUserAgentOverride", e.ua)
	_, _ = c.call(sid, "Network.setUserAgentOverride", e.ua)
	if fp.Timezone != "" {
		_, _ = c.call(sid, "Emulation.setTimezoneOverride", map[string]any{"timezoneId": fp.Timezone})
	}
	if l := firstLang(fp.Languages); l != "" {
		_, _ = c.call(sid, "Emulation.setLocaleOverride", map[string]any{"locale": l})
	}
	if fp.Latitude != 0 || fp.Longitude != 0 {
		_, _ = c.call(sid, "Emulation.setGeolocationOverride", map[string]any{"latitude": fp.Latitude, "longitude": fp.Longitude, "accuracy": 30 + float64(fp.NoiseSeed%50)})
	}
	if fp.HardwareConc > 0 {
		_, _ = c.call(sid, "Emulation.setHardwareConcurrencyOverride", map[string]any{"hardwareConcurrency": fp.HardwareConc})
	}
	if fp.MaxTouchPoints > 0 {
		_, _ = c.call(sid, "Emulation.setTouchEmulationEnabled", map[string]any{"enabled": true, "maxTouchPoints": fp.MaxTouchPoints})
		if isMobileType(fp.DeviceType) || fp.DeviceType == "tablet" {
			_, _ = c.call(sid, "Emulation.setEmitTouchEventsForMouse", map[string]any{"enabled": true, "configuration": "mobile"})
		}
	}
	if top && emulatedViewport(fp.DeviceType) {
		orient := map[string]any{"type": "portraitPrimary", "angle": 0}
		if fp.ScreenWidth > fp.ScreenHeight {
			orient = map[string]any{"type": "landscapePrimary", "angle": 0}
		}
		if fp.DeviceType == "phone" || fp.DeviceType == "foldable" || fp.DeviceType == "tablet" {
			if fp.ScreenWidth > fp.ScreenHeight {
				orient["angle"] = 90
			}
		}
		_, _ = c.call(sid, "Emulation.setDeviceMetricsOverride", map[string]any{
			"width": fp.ScreenWidth, "height": fp.ScreenHeight, "deviceScaleFactor": fp.PixelRatio,
			"mobile": fp.DeviceType != "tv", "screenWidth": fp.ScreenWidth, "screenHeight": fp.ScreenHeight,
			"scale": windowFor(fp).scale, "screenOrientation": orient,
		})
	}
}

// pageScript is evaluated in every frame before the page's own scripts. It
// covers what CDP emulation does not: GPU strings, memory, screen (desktops),
// noise, fonts, WebRTC, media devices and battery.
func pageScript(fp Fingerprint, px *Proxy) string {
	cfg := map[string]any{
		"t": normDevType(fp.DeviceType), "os": fp.OS, "vp": emulatedViewport(normDevType(fp.DeviceType)),
		"sw": fp.ScreenWidth, "sh": fp.ScreenHeight, "dpr": fp.PixelRatio, "cd": fp.ColorDepth,
		"hc": fp.HardwareConc, "mem": fp.DeviceMemory, "gv": fp.WebGLVendor, "gr": fp.WebGLRenderer,
		"cn": fp.CanvasNoise, "an": fp.AudioNoise, "wn": fp.WebGLNoise, "rn": fp.ClientRectNoise,
		"seed": fp.NoiseSeed, "rtc": fp.WebRTC, "fonts": fp.Fonts, "dnt": fp.DoNotTrack, "media": fp.HideMedia,
	}
	b, _ := json.Marshal(cfg)
	return strings.Replace(pageJS, "__CFG__", string(b), 1)
}

const pageJS = `(() => {
if (window.__gb) return; Object.defineProperty(window, "__gb", { value: 1 });
const C = __CFG__;
let s = (C.seed % 2147483647) || 1;
const rnd = () => (s = s * 16807 % 2147483647) / 2147483647;
const hide = new WeakMap();
const nts = Function.prototype.toString;
const pts = function toString() { return hide.has(this) ? hide.get(this) : nts.call(this); };
hide.set(pts, "function toString() { [native code] }");
Function.prototype.toString = pts;
const def = (obj, prop, getter) => {
  const d = Object.getOwnPropertyDescriptor(obj, prop) || {};
  const g = { get [prop]() { return getter(); } };
  const fn = Object.getOwnPropertyDescriptor(g, prop).get;
  hide.set(fn, "function get " + prop + "() { [native code] }");
  Object.defineProperty(obj, prop, { get: fn, configurable: true, enumerable: d.enumerable !== false });
};
const wrap = (obj, name, impl) => {
  const orig = obj && obj[name]; if (!orig) return;
  const w = { [name](...a) { return impl.call(this, orig, a); } }[name];
  hide.set(w, "function " + name + "() { [native code] }");
  obj[name] = w;
};
const N = Navigator.prototype;
if (C.hc) def(N, "hardwareConcurrency", () => C.hc);
if (C.mem) def(N, "deviceMemory", () => C.mem);
def(N, "webdriver", () => false);
if (C.dnt) def(N, "doNotTrack", () => "1");

// Desktop screen (phones, tablets and TVs get theirs from device emulation)
if (!C.vp) {
  const S = Screen.prototype, bar = C.os === "windows" ? 48 : C.os === "macos" ? 25 : 0;
  def(S, "width", () => C.sw); def(S, "height", () => C.sh);
  def(S, "availWidth", () => C.sw); def(S, "availHeight", () => C.sh - bar);
  def(S, "colorDepth", () => C.cd); def(S, "pixelDepth", () => C.cd);
  def(window, "devicePixelRatio", () => C.dpr);
}

// WebGL
for (const P of [self.WebGLRenderingContext, self.WebGL2RenderingContext]) {
  if (!P) continue;
  wrap(P.prototype, "getParameter", function (orig, a) {
    if (a[0] === 37445 && C.gv) return C.gv;
    if (a[0] === 37446 && C.gr) return C.gr;
    return orig.apply(this, a);
  });
  if (C.wn) wrap(P.prototype, "readPixels", function (orig, a) {
    const r = orig.apply(this, a); const px = a[6];
    if (px && px.length) for (let i = 0; i < px.length; i += 97) px[i] = px[i] ^ 1;
    return r;
  });
}
if (C.cn) {
  const noisify = (cv) => {
    try {
      const ctx = cv.getContext("2d"); if (!ctx || !cv.width || !cv.height) return;
      const img = ctx.getImageData(0, 0, cv.width, cv.height);
      for (let i = (C.seed % 53) * 4; i < img.data.length; i += 4 * 53) img.data[i] = img.data[i] ^ 1;
      ctx.putImageData(img, 0, 0);
    } catch (e) {}
  };
  wrap(HTMLCanvasElement.prototype, "toDataURL", function (orig, a) { noisify(this); return orig.apply(this, a); });
  wrap(HTMLCanvasElement.prototype, "toBlob", function (orig, a) { noisify(this); return orig.apply(this, a); });
}
if (C.an && self.AudioBuffer) {
  const done = new WeakSet();
  wrap(AudioBuffer.prototype, "getChannelData", function (orig, a) {
    const d = orig.apply(this, a);
    if (!done.has(d)) { done.add(d); for (let i = 0; i < d.length; i += 100) d[i] += (rnd() - .5) * 1e-7; }
    return d;
  });
}
if (C.rn) {
  const off = (rnd() - .5) * 0.0002;
  wrap(Element.prototype, "getBoundingClientRect", function (orig, a) {
    const r = orig.apply(this, a);
    return new DOMRect(r.x + off, r.y + off, r.width + off, r.height + off);
  });
}
if (C.rtc === "disabled") {
  for (const k of ["RTCPeerConnection", "webkitRTCPeerConnection", "RTCDataChannel"]) { try { delete self[k]; } catch (e) {} }
}
if (C.media && self.MediaDevices) {
  wrap(MediaDevices.prototype, "enumerateDevices", function () { return Promise.resolve([]); });
}
if (C.fonts && C.fonts.length && self.FontFaceSet) {
  const set = new Set(C.fonts.map(f => f.toLowerCase()));
  wrap(FontFaceSet.prototype, "check", function (orig, a) {
    const m = /(?:\d+px\s+)?["']?([^"',]+)/.exec(a[0] || ""); const fam = m ? m[1].trim().toLowerCase() : "";
    if (["serif", "sans-serif", "monospace", "system-ui"].includes(fam)) return orig.apply(this, a);
    return set.has(fam);
  });
}
// Battery: desktops and TVs are on mains power; portable devices discharge.
if (N.getBattery) {
  const mains = C.t === "desktop" || C.t === "tv";
  const level = mains ? 1 : Math.round((0.35 + rnd() * 0.6) * 100) / 100;
  wrap(N, "getBattery", function (orig, a) {
    return orig.apply(this, a).then(b => {
      try {
        def(b, "charging", () => mains); def(b, "level", () => level);
        def(b, "chargingTime", () => mains ? 0 : Infinity);
        def(b, "dischargingTime", () => mains ? Infinity : Math.round(7200 + level * 20000));
      } catch (e) {}
      return b;
    });
  });
}
})();`
