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
		if w := os.Getenv("ProgramW6432"); w != "" {
			pf = w // a 32-bit process sees "Program Files (x86)" as ProgramFiles
		}
		// Unbranded first: Chromium / ungoogled-chromium installers use the
		// Chromium folder and keep --load-extension; branded Chrome 137+
		// ignores it, Edge brands its UA-CH as Microsoft Edge.
		for _, base := range []string{local, pf, pf86} {
			if base != "" {
				cands = append(cands, filepath.Join(base, "Chromium/Application/chrome.exe"))
			}
		}
		for _, base := range []string{pf, pf86, local} {
			if base != "" {
				cands = append(cands, filepath.Join(base, "Google/Chrome/Application/chrome.exe"))
			}
		}
		if local != "" {
			cands = append(cands,
				filepath.Join(local, "Google/Chrome Beta/Application/chrome.exe"),
				filepath.Join(local, "Google/Chrome Dev/Application/chrome.exe"),
				filepath.Join(local, "Google/Chrome SxS/Application/chrome.exe"))
		}
		for _, base := range []string{pf86, pf, local} {
			if base != "" {
				cands = append(cands, filepath.Join(base, "Microsoft/Edge/Application/msedge.exe"))
			}
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
	fp.NoiseSeed = effectiveSeed(p.ID, fp.NoiseSeed, fp.RandomFingerprintOnLaunch)
	// The UA's Chrome major always follows the kernel, so UA, UA-CH and the
	// engine's real features agree. TVs keep their own fixed browser build.
	// (Re-checked against Browser.getVersion once CDP is up.)
	if err := kernelUnsupportedErr(kernel); err != nil {
		return nil, err
	}
	alignUAMajor(&fp, majorOf(kernelVersion(kernel)))

	userDir := s.ProfileDir(p.ID)
	_ = os.MkdirAll(userDir, 0o755)
	port := freePort()
	args := []string{
		"--user-data-dir=" + userDir,
		fmt.Sprintf("--remote-debugging-port=%d", port),
		"--remote-allow-origins=http://127.0.0.1",
		"--no-first-run", "--no-default-browser-check", "--disable-sync",
		"--disable-features=Translate,MediaRouter",
		"--lang=" + firstLang(fp.Languages),
		"--user-agent=" + fp.UserAgent, // workers and the very first request
	}
	args = append(args, launchFlags(fp)...)
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
	userArgs, rejectedArgs := parseStartupParams(fp.StartupParameters)
	args = mergeFlags(append(args, userArgs...))
	args = append(args, "about:blank")

	cmd := exec.Command(kernel, args...)
	logPath := filepath.Join(userDir, "launch.log")
	logf, _ := os.Create(logPath)
	if logf != nil {
		fmt.Fprintf(logf, "kernel: %s\nargs: %q\n\n", kernel, args)
		if len(rejectedArgs) > 0 {
			fmt.Fprintf(logf, "ignored start-up parameters: %q\n\n", rejectedArgs)
		}
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
	conn        *cdpConn
	fp          Fingerprint
	ua          map[string]any // Emulation.setUserAgentOverride params
	script      string
	existing    map[string]bool
	starts      []string
	startMu     sync.Mutex
	startTarget string
	net         *fpNet // port-scan protection / video blocking (nil = off)
	major       int    // kernel's Chrome major from Browser.getVersion (0 = unknown)
}

func (e *emulator) Close() {
	if e != nil && e.conn != nil {
		e.conn.c.Close()
	}
}

func startEmulation(ws string, fp Fingerprint, px *Proxy, startURLs []string) (*emulator, error) {
	e := &emulator{fp: fp, existing: map[string]bool{}, net: newFpNet(fp)}
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
	// The real kernel version is authoritative: the launch-time guess can miss
	// (wrapper scripts, renamed folders), and UA vs UA-CH must never disagree.
	alignUAMajor(&e.fp, e.major)
	fp = e.fp
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
	e.major = majorOf(full)
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
	if e.net.onEvent(e.conn, m) {
		return
	}
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
		e.net.attach(c, sid, ev.TargetInfo.TargetID, ev.TargetInfo.Type == "page")
		_, _ = c.call(sid, "Target.setAutoAttach", map[string]any{"autoAttach": true, "waitForDebuggerOnStart": true, "flatten": true})
		resume()
		if ev.TargetInfo.Type == "page" && ev.Waiting && e.major > 0 && e.major < chromiumLegacyBelow {
			go e.ensureScript(sid)
		}
		// The kernel's initial about:blank tab can register after our getTargets
		// snapshot, so a blank page attaching first counts as the start tab too.
		blank := ev.TargetInfo.URL == "" || ev.TargetInfo.URL == "about:blank"
		if ev.TargetInfo.Type == "page" && (e.existing[ev.TargetInfo.TargetID] || blank) {
			// Pages that were already open loaded before the overrides: load them again.
			// The start tab can attach twice (auto-attach racing the initial
			// target list); the navigation follows that one target until it
			// has left about:blank.
			navigated := false
			e.startMu.Lock()
			if len(e.starts) > 0 && (e.startTarget == "" || (e.startTarget == ev.TargetInfo.TargetID && blank)) {
				e.startTarget = ev.TargetInfo.TargetID
				navigated = true
			}
			e.startMu.Unlock()
			if navigated {
				_, _ = c.call(sid, "Page.navigate", map[string]any{"url": e.starts[0]})
			}
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
		"cm": fpMode(fp.CanvasMode, fp.CanvasNoise), "wm": fpMode(fp.WebGLImageMode, fp.WebGLNoise),
		"am": fpMode(fp.AudioMode, fp.AudioNoise), "rm": fpMode(fp.ClientRectsMode, fp.ClientRectNoise),
		"md": mediaMode(fp), "mic": clampInt(fp.MediaMics, 0, 8), "spk": clampInt(fp.MediaSpeakers, 0, 8), "cam": clampInt(fp.MediaCameras, 0, 8),
		"sv": fp.SpeechVoices, "gpu": fp.WebGPU, "ps": fp.PortScanProtection, "wl": whitelistJS(fp.ScanWhitelist), "bv": fp.BlockVideo,
	}
	if cfg["sv"] == "" {
		cfg["sv"] = "real"
	}
	if fp.SpeechVoices == "custom" {
		cfg["voices"] = speechVoicesFor(fp.OS, fp.Languages)
	}
	if px != nil && fp.WebRTC == "proxy" {
		cfg["pip"] = px.IP // the proxy's last checked exit IP
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

// Deterministic hash of the seed and a few integers (stable per profile).
const H = (...v) => { let h = C.seed | 0; for (const x of v) { h = Math.imul(h ^ (x | 0), 0x9E3779B1); h ^= h >>> 15; h = Math.imul(h, 0x85EBCA77); h ^= h >>> 13; } return h >>> 0; };
// Accessor override that keeps "this": fn(origGetter) runs with the instance.
const getter = (obj, prop, fn) => {
  const d = obj && Object.getOwnPropertyDescriptor(obj, prop); if (!d || !d.get) return;
  const og = d.get;
  const g = { get [prop]() { return fn.call(this, og); } };
  const f = Object.getOwnPropertyDescriptor(g, prop).get;
  hide.set(f, "function get " + prop + "() { [native code] }");
  Object.defineProperty(obj, prop, { get: f, configurable: true, enumerable: d.enumerable });
};
// Fake host objects (media devices, voices): real prototype, values from a WeakMap.
const fakes = new WeakMap();
const fakeProto = (P, props) => { for (const k of props) getter(P, k, function (og) { return fakes.has(this) ? fakes.get(this)[k] : og.call(this); }); };
const mkFake = (P, vals) => { const o = Object.create(P); fakes.set(o, vals); return o; };

// WebGL
const glc = new WeakSet();
if (self.HTMLCanvasElement) wrap(HTMLCanvasElement.prototype, "getContext", function (orig, a) {
  const t = String(a[0]);
  const gl = t === "webgl" || t === "webgl2" || t === "experimental-webgl";
  if (gl && C.wm === "off") return null;
  const r = orig.apply(this, a);
  if (gl && r) glc.add(this);
  return r;
});
if (C.wm === "off" && self.OffscreenCanvas) wrap(OffscreenCanvas.prototype, "getContext", function (orig, a) {
  const t = String(a[0]); if (t === "webgl" || t === "webgl2") return null; return orig.apply(this, a);
});
for (const P of [self.WebGLRenderingContext, self.WebGL2RenderingContext]) {
  if (!P) continue;
  wrap(P.prototype, "getParameter", function (orig, a) {
    if (a[0] === 37445 && C.gv) return C.gv;
    if (a[0] === 37446 && C.gr) return C.gr;
    return orig.apply(this, a);
  });
  if (C.wm === "noise") wrap(P.prototype, "readPixels", function (orig, a) {
    const r = orig.apply(this, a); const px = a[6];
    if (px && px.length) for (let i = H(1, 9) % 31; i < px.length; i += 31) if ((i & 3) !== 3) px[i] = px[i] ^ (H(i, 9) & 1);
    return r;
  });
}
// Canvas 2D: noise goes into a copy, keyed by pixel position, so the page's own
// canvas is untouched and every read of the same drawing returns the same image.
const C2 = self.CanvasRenderingContext2D && CanvasRenderingContext2D.prototype;
const gid = C2 && C2.getImageData, pid = C2 && C2.putImageData;
const noiseImg = (img, ox, oy) => {
  const d = img.data, w = img.width;
  for (let i = 0, p = 0; i < d.length; i += 4, p++) {
    if (d[i + 3] !== 255) continue;
    const h = H(ox + p % w, oy + (p / w | 0), 7);
    if (h % 19) continue;
    d[i + (h >>> 8) % 3] ^= 1;
  }
  return img;
};
const copyOf = (cv, noisy) => {
  const w = cv.width, h = cv.height;
  const c = document.createElement("canvas"); c.width = w; c.height = h;
  if (!noisy || !w || !h) return c;
  try {
    const x = c.getContext("2d"); x.drawImage(cv, 0, 0);
    pid.call(x, noiseImg(gid.call(x, 0, 0, w, h), 0, 0), 0, 0);
  } catch (e) { return null; }
  return c;
};
const modeOf = (cv) => glc.has(cv) ? (C.wm === "noise" ? "noise" : "real") : C.cm;
if (self.HTMLCanvasElement && gid) for (const name of ["toDataURL", "toBlob"]) {
  wrap(HTMLCanvasElement.prototype, name, function (orig, a) {
    const m = modeOf(this);
    if (m === "real") return orig.apply(this, a);
    const c = copyOf(this, m === "noise");
    return orig.apply(c || this, a);
  });
}
if (gid && C.cm !== "real") {
  for (const P of [C2, self.OffscreenCanvasRenderingContext2D && OffscreenCanvasRenderingContext2D.prototype]) {
    if (!P) continue;
    wrap(P, "getImageData", function (orig, a) {
      if (C.cm === "off") return new ImageData(Math.max(1, Math.abs(a[2] | 0)), Math.max(1, Math.abs(a[3] | 0)));
      return noiseImg(orig.apply(this, a), a[0] | 0, a[1] | 0);
    });
  }
}
// Audio
if (C.am === "off") {
  for (const k of ["AudioContext", "webkitAudioContext", "OfflineAudioContext", "webkitOfflineAudioContext"]) { try { delete self[k]; } catch (e) {} }
} else if (C.am === "noise" && self.AudioBuffer) {
  const done = new WeakSet(), gcd = AudioBuffer.prototype.getChannelData;
  wrap(AudioBuffer.prototype, "getChannelData", function (orig, a) {
    const d = orig.apply(this, a);
    if (!done.has(d)) { done.add(d); const ch = a[0] | 0; for (let i = H(ch, 3) % 50; i < d.length; i += 50) d[i] += ((H(i, ch, 3) % 2001) - 1000) * 1e-10; }
    return d;
  });
  wrap(AudioBuffer.prototype, "copyFromChannel", function (orig, a) {
    try { this.getChannelData(a[1] | 0); } catch (e) {}
    return orig.apply(this, a);
  });
}
// Client rects: a small offset per element, the same on every call.
if (C.rm === "noise") {
  const ek = (el) => {
    const s = (el.tagName || "") + "#" + (el.id || "") + "." + (typeof el.className === "string" ? el.className : "") + ":" + ((el.textContent || "").length);
    let h = 0; for (let i = 0; i < s.length; i++) h = Math.imul(h ^ s.charCodeAt(i), 16777619);
    return H(h, 5);
  };
  const jit = (r, k) => { const o = ((k % 2001) - 1000) * 1e-7, o2 = (((k >>> 11) % 2001) - 1000) * 1e-7; return new DOMRect(r.x + o, r.y + o2, r.width + o2, r.height + o); };
  wrap(Element.prototype, "getBoundingClientRect", function (orig, a) { return jit(orig.apply(this, a), ek(this)); });
  if (self.Range) wrap(Range.prototype, "getBoundingClientRect", function (orig, a) {
    const n = this.startContainer; return jit(orig.apply(this, a), H(this.startOffset, this.endOffset, (n && n.textContent || "").length));
  });
} else if (C.rm === "off") {
  wrap(Element.prototype, "getBoundingClientRect", function () { return new DOMRect(0, 0, 0, 0); });
}
// WebRTC
if (C.rtc === "disabled") {
  for (const k of ["RTCPeerConnection", "webkitRTCPeerConnection", "RTCDataChannel", "RTCSessionDescription", "RTCIceCandidate"]) { try { delete self[k]; } catch (e) {} }
} else if (C.rtc === "proxy" && C.pip && self.RTCPeerConnection) {
  const P = C.pip;
  const cand = (l) => String(l).replace(/^((?:a=)?candidate:\S+ \d+ \S+ \d+ )(\S+)/i, (m, a, ip) => a + (/\.local$/i.test(ip) ? ip : P))
    .replace(/( raddr )(\S+)/, (m, a, ip) => a + (ip === "0.0.0.0" || ip === "::" ? ip : P));
  const sdp = (s) => typeof s !== "string" ? s : s.split(/\r\n/).map(l => /^a=candidate:/i.test(l) ? cand(l) : l.replace(/^c=IN IP4 (?!0\.0\.0\.0)\S+/, "c=IN IP4 " + P)).join("\r\n");
  if (self.RTCIceCandidate) {
    const I = RTCIceCandidate.prototype;
    getter(I, "candidate", function (og) { const v = og.call(this); return v ? cand(v) : v; });
    getter(I, "address", function (og) { const v = og.call(this); return v && !/\.local$/i.test(v) ? P : v; });
    wrap(I, "toJSON", function (orig, a) { const j = orig.apply(this, a); if (j && j.candidate) j.candidate = cand(j.candidate); return j; });
  }
  if (self.RTCSessionDescription) {
    getter(RTCSessionDescription.prototype, "sdp", function (og) { return sdp(og.call(this)); });
    wrap(RTCSessionDescription.prototype, "toJSON", function (orig, a) { const j = orig.apply(this, a); if (j) j.sdp = sdp(j.sdp); return j; });
  }
  for (const n of ["createOffer", "createAnswer"]) wrap(RTCPeerConnection.prototype, n, function (orig, a) {
    return orig.apply(this, a).then(d => { if (d && typeof d.sdp === "string" && !(d instanceof RTCSessionDescription)) d.sdp = sdp(d.sdp); return d; });
  });
}
// Media devices
if (self.MediaDevices && C.md !== "real") {
  let list = [];
  if (C.md === "custom") {
    const hex = (k) => { let s = ""; for (let j = 0; j < 8; j++) s += H(k, j, 11).toString(16).padStart(8, "0"); return s; };
    const MP = self.MediaDeviceInfo && MediaDeviceInfo.prototype, IP = self.InputDeviceInfo ? InputDeviceInfo.prototype : MP;
    if (MP) {
      fakeProto(MP, ["deviceId", "kind", "label", "groupId"]);
      wrap(MP, "toJSON", function (orig, a) { return fakes.has(this) ? Object.assign({}, fakes.get(this)) : orig.apply(this, a); });
      const add = (kind, n, P) => { for (let i = 0; i < n; i++) list.push(mkFake(P, { deviceId: hex(kind.length * 100 + i), kind, label: "", groupId: hex(kind.length * 100 + i + 50) })); };
      add("audioinput", C.mic, IP); add("videoinput", C.cam, IP); add("audiooutput", C.spk, MP);
    }
  }
  wrap(MediaDevices.prototype, "enumerateDevices", function () { return Promise.resolve(list.slice()); });
}
// Speech voices
if (self.speechSynthesis && C.sv !== "real") {
  let voices = [];
  const VK = ["voiceURI", "name", "lang", "localService", "default"];
  if (C.sv === "custom" && self.SpeechSynthesisVoice) {
    const VP = SpeechSynthesisVoice.prototype;
    fakeProto(VP, VK);
    voices = (C.voices || []).map(v => mkFake(VP, v));
  } else if (C.sv === "custom") {
    // Some kernels (Chromium 120 and older Linux builds) expose no
    // SpeechSynthesisVoice interface: voices get a private look-alike prototype.
    const VP = {};
    for (const k of VK) {
      const g = { get [k]() { return fakes.has(this) ? fakes.get(this)[k] : undefined; } };
      const f = Object.getOwnPropertyDescriptor(g, k).get;
      hide.set(f, "function get " + k + "() { [native code] }");
      Object.defineProperty(VP, k, { get: f, configurable: true, enumerable: true });
    }
    Object.defineProperty(VP, Symbol.toStringTag, { value: "SpeechSynthesisVoice", configurable: true });
    voices = (C.voices || []).map(v => mkFake(VP, v));
  }
  wrap(Object.getPrototypeOf(speechSynthesis), "getVoices", function () { return voices.slice(); });
}
// WebGPU
if (C.gpu === "off") {
  try { delete Navigator.prototype.gpu; } catch (e) {}
  try { if (self.WorkerNavigator) delete WorkerNavigator.prototype.gpu; } catch (e) {}
  for (const k of Object.getOwnPropertyNames(self)) if (/^GPU[A-Z]/.test(k) || k === "GPU") { try { delete self[k]; } catch (e) {} }
}
// Port-scan protection: WebSockets to local addresses look like closed ports.
if (C.ps && self.WebSocket) {
  const localHost = (h) => { h = h.replace(/^\[|\]$/g, "").toLowerCase(); return h === "localhost" || h.endsWith(".localhost") || /^127\./.test(h) || h === "::1" || h === "0.0.0.0" || h === "::"; };
  if (!localHost(location.hostname)) {
    const WS = WebSocket, wl = new Set(C.wl || []), moved = new WeakMap();
    const W = function WebSocket(url, protocols) {
      if (!new.target) return WS(url, protocols);
      let shown = null;
      try {
        const u = new URL(url, location.href), port = u.port || (u.protocol === "wss:" || u.protocol === "https:" ? "443" : "80"), h = u.hostname.replace(/^\[|\]$/g, "").toLowerCase();
        if (localHost(u.hostname) && !wl.has(port) && !wl.has(h + ":" + port) && !wl.has(h)) { shown = u.href.replace(/^http/, "ws"); url = "ws://127.0.0.1:1/"; }
      } catch (e) {}
      const sock = arguments.length > 1 ? Reflect.construct(WS, [url, protocols], new.target) : Reflect.construct(WS, [url], new.target);
      if (shown) moved.set(sock, shown);
      return sock;
    };
    getter(WS.prototype, "url", function (og) { return moved.has(this) ? moved.get(this) : og.call(this); });
    hide.set(W, "function WebSocket() { [native code] }");
    W.prototype = WS.prototype;
    for (const k of ["CONNECTING", "OPEN", "CLOSING", "CLOSED"]) Object.defineProperty(W, k, { value: WS[k], enumerable: true });
    Object.defineProperty(WS.prototype, "constructor", { value: W, writable: true, configurable: true });
    Object.setPrototypeOf(W, Object.getPrototypeOf(WS));
    Object.defineProperty(self, "WebSocket", { value: W, writable: true, configurable: true, enumerable: false });
  }
}
// Video blocking (network side is done over CDP): videos refuse to play.
if (C.bv && self.HTMLVideoElement) {
  wrap(HTMLMediaElement.prototype, "play", function (orig, a) {
    if (this instanceof HTMLVideoElement) return Promise.reject(new DOMException("play() failed because the user didn't interact with the document first.", "NotAllowedError"));
    return orig.apply(this, a);
  });
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
