package main

// Firefox kernel (Profile.Kernel == "firefox"). The installed Mozilla Firefox
// (or one installed by GhostBrowser into %APPDATA%/GhostBrowser/firefox) runs
// with a separate profile folder per environment (-profile <dir> -no-remote).
// Fingerprint settings are written as user.js prefs (firefox_prefs.go), the
// time zone goes in through TZ, credentialed proxies go through the same local
// relay the Chromium kernel uses, and GhostBrowser controls the browser over
// WebDriver BiDi (bidi.go): focus, cookie snapshot / restore, graceful close.
//
// RunState.WSURL stays empty for Firefox so CDP-only features (RPA, syncer,
// recorder, window arrange) skip it; RunState.BiDiURL holds the BiDi endpoint.

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"

	"ghostbrowser/internal/undetectable"
)

// ---------- detection ----------

// firefoxCandidates lists where Firefox usually lives, most specific first.
func firefoxCandidates() []string {
	var c []string
	switch runtime.GOOS {
	case "windows":
		pf, pf86, local := os.Getenv("ProgramFiles"), os.Getenv("ProgramFiles(x86)"), os.Getenv("LOCALAPPDATA")
		if w := os.Getenv("ProgramW6432"); w != "" {
			pf = w
		}
		// Installed by GhostBrowser's one-click install.
		for _, n := range []string{"esr", "release"} {
			c = append(c, filepath.Join(firefoxInstallRoot(), n, "firefox.exe"))
		}
		if p := firefoxFromRegistry(); p != "" {
			c = append(c, p)
		}
		for _, base := range []string{pf, pf86, local} {
			if base != "" {
				c = append(c,
					filepath.Join(base, "Mozilla Firefox", "firefox.exe"),
					filepath.Join(base, "Firefox Developer Edition", "firefox.exe"),
					filepath.Join(base, "Firefox Nightly", "firefox.exe"))
			}
		}
	case "darwin":
		c = []string{"/Applications/Firefox.app/Contents/MacOS/firefox", "/Applications/Firefox ESR.app/Contents/MacOS/firefox"}
	default:
		c = []string{"/usr/bin/firefox", "/usr/bin/firefox-esr", "/usr/lib/firefox/firefox", "/usr/lib/firefox-esr/firefox-esr", "/snap/bin/firefox", "/opt/firefox/firefox"}
	}
	return c
}

func findFirefox() string {
	for _, p := range firefoxCandidates() {
		if fileExists(p) {
			return p
		}
	}
	return ""
}

// firefoxKernel returns the Firefox to launch: the setting when it exists, else auto-detected.
func firefoxKernel(st Settings) string {
	k := strings.Trim(strings.TrimSpace(st.FirefoxPath), "\"")
	if k != "" && fileExists(k) {
		return k
	}
	return findFirefox()
}

var ffVerRe = regexp.MustCompile(`(?m)^Version=([0-9][0-9a-z.]*)`)
var ffVerOut = regexp.MustCompile(`Firefox\s+([0-9][0-9a-z.]*)`)

var ffVerCache sync.Map // path -> version

// firefoxVersion reads application.ini next to the executable (Windows
// firefox.exe prints nothing on --version), falling back to --version.
func firefoxVersion(path string) string {
	if path == "" {
		return ""
	}
	if v, ok := ffVerCache.Load(path); ok {
		return v.(string)
	}
	ver := ""
	real := path
	if r, err := filepath.EvalSymlinks(path); err == nil {
		real = r
	}
	for _, dir := range []string{filepath.Dir(real), filepath.Join(filepath.Dir(real), "..", "Resources")} {
		if b, err := os.ReadFile(filepath.Join(dir, "application.ini")); err == nil {
			if m := ffVerRe.FindSubmatch(b); m != nil {
				ver = string(m[1])
				break
			}
		}
	}
	if ver == "" && runtime.GOOS != "windows" {
		cmd := exec.Command(path, "--version")
		out := make(chan []byte, 1)
		go func() { b, _ := cmd.Output(); out <- b }()
		select {
		case b := <-out:
			if m := ffVerOut.FindSubmatch(b); m != nil {
				ver = string(m[1])
			}
		case <-time.After(8 * time.Second):
			if cmd.Process != nil {
				_ = cmd.Process.Kill()
			}
		}
	}
	if ver != "" {
		ffVerCache.Store(path, ver)
	}
	return ver
}

// FirefoxInfo is what Settings and the kernel pickers show.
type FirefoxInfo struct {
	Path        string `json:"path"`        // the Firefox that will launch ("" = none)
	Auto        string `json:"auto"`        // auto-detected one
	Version     string `json:"version"`     // e.g. 128.5.0esr
	ESR         bool   `json:"esr"`         //
	Custom      bool   `json:"custom"`      // from Settings.firefoxPath
	Missing     bool   `json:"missing"`     // the configured path does not exist
	DownloadURL string `json:"downloadUrl"` // official download page
}

const firefoxDownloadPage = "https://www.mozilla.org/zh-CN/firefox/new/"

func firefoxInfo(st Settings) FirefoxInfo {
	k := firefoxKernel(st)
	in := FirefoxInfo{Path: k, Auto: findFirefox(), DownloadURL: firefoxDownloadPage}
	if p := strings.Trim(strings.TrimSpace(st.FirefoxPath), "\""); p != "" {
		in.Custom = true
		in.Missing = !fileExists(p)
	}
	in.Version = firefoxVersion(k)
	in.ESR = strings.Contains(in.Version, "esr") || strings.Contains(strings.ToLower(k), "esr")
	return in
}

// ---------- running browsers ----------

// firefoxRun is the live control connection of one running Firefox.
type firefoxRun struct {
	id     string
	dir    string
	bidi   *bidiClient
	cmd    *exec.Cmd
	relay  *undetectable.Relay
	exited chan struct{}
	device string // this device's id, written into session snapshots
	once   sync.Once
	mu     sync.Mutex // serializes snapshot vs. close
}

var ffRuns = struct {
	sync.Mutex
	m map[string]*firefoxRun
}{m: map[string]*firefoxRun{}}

func getFirefoxRun(id string) *firefoxRun {
	ffRuns.Lock()
	defer ffRuns.Unlock()
	return ffRuns.m[id]
}

// firefoxProfileDir is the Firefox profile inside the environment's folder.
func (s *Store) firefoxProfileDir(id string) string {
	return filepath.Join(s.ProfileDir(id), "firefox")
}

// waitBiDi waits until Firefox's remote agent listens on port, reading the
// actual port from WebDriverBiDiServer.json when Firefox picked another one.
func waitBiDi(port int, profileDir string, timeout time.Duration, exited <-chan struct{}) (string, error) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		select {
		case <-exited:
			return "", errors.New("exited")
		default:
		}
		if b, err := os.ReadFile(filepath.Join(profileDir, "WebDriverBiDiServer.json")); err == nil {
			var v struct {
				Host string `json:"ws_host"`
				Port int    `json:"ws_port"`
			}
			if json.Unmarshal(b, &v) == nil && v.Port > 0 && (v.Host == "127.0.0.1" || v.Host == "localhost" || v.Host == "::1" || v.Host == "") {
				port = v.Port
			}
		}
		if c, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 300*time.Millisecond); err == nil {
			c.Close()
			return fmt.Sprintf("ws://127.0.0.1:%d/session", port), nil
		}
		time.Sleep(250 * time.Millisecond)
	}
	return "", errors.New("timeout")
}

// firefoxArgs are the command-line switches for one launch.
func firefoxArgs(profileDir string, port int, fp Fingerprint, headless bool) []string {
	args := []string{"-profile", profileDir, "-no-remote", "-new-instance",
		fmt.Sprintf("--remote-debugging-port=%d", port), "--remote-allow-hosts", "127.0.0.1,localhost"}
	if headless {
		args = append(args, "-headless")
	}
	return append(args, firefoxWindowArgs(fp)...)
}

// firefoxHeadless is for tests and servers (GHOST_FIREFOX_HEADLESS=1).
func firefoxHeadless() bool { return os.Getenv("GHOST_FIREFOX_HEADLESS") == "1" }

func (s *Store) launchFirefox(p *Profile, st Settings) (*RunState, error) {
	kernel := firefoxKernel(st)
	if kernel == "" {
		return nil, fmt.Errorf("没有找到 Firefox：请在「设置 → Firefox 内核」选择 firefox.exe，或一键安装官方 Firefox（下载页 %s）", firefoxDownloadPage)
	}
	var px *Proxy
	if p.InlineProxy != nil {
		px = p.InlineProxy
	} else if p.ProxyID != "" {
		if pr, ok := s.GetProxy(p.ProxyID); ok {
			px = pr
		}
	}
	fp, err := s.matchFP(normalizeFirefoxFP(p.Fingerprint), px)
	if err != nil {
		return nil, err
	}
	fp = normalizeFirefoxFP(fp)

	var relay *undetectable.Relay
	cfg, needsRelay := firefoxProxyFor(px)
	if needsRelay {
		if relay, err = undetectable.StartRelay(px.URL().String()); err != nil {
			return nil, fmt.Errorf("代理中转启动失败：%w", err)
		}
		if cfg, err = ffProxyFromURL(relay.URL()); err != nil {
			relay.Close()
			return nil, err
		}
	}

	dir := s.firefoxProfileDir(p.ID)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		relay.Close()
		return nil, err
	}
	_ = os.Remove(filepath.Join(dir, "WebDriverBiDiServer.json"))
	prefs := firefoxPrefs(fp, cfg, p.StartURLs)
	if err := os.WriteFile(filepath.Join(dir, "user.js"), []byte(prefs.UserJS()), 0o644); err != nil {
		relay.Close()
		return nil, fmt.Errorf("写入 user.js 失败：%w", err)
	}

	port := freePort()
	args := firefoxArgs(dir, port, fp, firefoxHeadless())
	cmd := exec.Command(kernel, args...)
	cmd.Env = firefoxEnv(os.Environ(), fp)
	logPath := filepath.Join(s.ProfileDir(p.ID), "firefox-launch.log")
	logf, _ := os.Create(logPath)
	if logf != nil {
		fmt.Fprintf(logf, "kernel: %s\nargs: %q\nTZ: %s\n\n", kernel, args, fp.Timezone)
		cmd.Stdout, cmd.Stderr = logf, logf
	}
	if err := cmd.Start(); err != nil {
		relay.Close()
		if logf != nil {
			logf.Close()
		}
		return nil, fmt.Errorf("无法启动 Firefox %s: %w", kernel, err)
	}
	run := &firefoxRun{id: p.ID, dir: s.ProfileDir(p.ID), cmd: cmd, relay: relay, exited: make(chan struct{}), device: st.DeviceID}
	rs := &RunState{ProfileID: p.ID, PID: cmd.Process.Pid, DebugPort: port, Started: time.Now().Unix(), Kernel: "firefox"}
	runReg.add(p.ID, rs, cmd)
	ffRuns.Lock()
	ffRuns.m[p.ID] = run
	ffRuns.Unlock()
	go func() {
		_ = cmd.Wait()
		close(run.exited)
		ffRuns.Lock()
		if ffRuns.m[p.ID] == run {
			delete(ffRuns.m, p.ID)
		}
		ffRuns.Unlock()
		runReg.remove(p.ID)
		run.bidi.Close()
		relay.Close()
		if logf != nil {
			logf.Close()
		}
	}()

	url, err := waitBiDi(port, dir, 30*time.Second, run.exited)
	if err != nil {
		if err.Error() == "exited" {
			return nil, fmt.Errorf("Firefox 启动后立即退出。若该环境已经打开，请先关闭它的窗口%s", logTail(logPath))
		}
		run.kill()
		return nil, fmt.Errorf("Firefox 30 秒内没有打开 WebDriver BiDi 端口，已终止%s", logTail(logPath))
	}
	var c *bidiClient
	for i := 0; i < 20; i++ { // the port opens a moment before the session endpoint accepts
		if c, err = dialBiDi(url, nil); err == nil {
			if _, err = c.NewSession(); err == nil {
				break
			}
			c.Close()
		}
		time.Sleep(250 * time.Millisecond)
	}
	if err != nil {
		run.kill()
		return nil, fmt.Errorf("无法连接 Firefox 的 WebDriver BiDi：%w", err)
	}
	run.bidi = c
	rs.BiDiURL = url
	runReg.setCleanup(p.ID, run.cleanup)

	p.LastRun = time.Now().Unix()
	s.PutProfile(*p)
	device := st.DeviceID
	firefoxRestoreSession(run.dir, device, c)
	if strings.TrimSpace(p.Cookies) != "" {
		if list := stagedCookies(p); len(list) > 0 {
			if _, err := c.SetCookies(list); err == nil {
				p.Cookies = ""
				s.PutProfile(*p)
			}
		}
	}
	// Start pages (prefs open them on a normal start; BiDi also covers a restored session).
	if urls := cleanStartURLs(p.StartURLs); len(urls) > 0 {
		if ctx, err := c.firstContext(); err == nil {
			go func() { _ = c.Navigate(ctx, urls[0]) }()
		}
	}
	go run.keepSnapshot(device)
	return rs, nil
}

// keepSnapshot refreshes the plain cookie snapshot every 15 s while Firefox runs.
func (r *firefoxRun) keepSnapshot(device string) {
	select {
	case <-time.After(5 * time.Second):
	case <-r.exited:
		return
	}
	for {
		r.mu.Lock()
		if r.bidi != nil {
			_ = firefoxSnapshotSession(r.dir, device, r.bidi)
		}
		r.mu.Unlock()
		select {
		case <-time.After(15 * time.Second):
		case <-r.exited:
			return
		}
	}
}

func (r *firefoxRun) kill() {
	if r.cmd != nil && r.cmd.Process != nil {
		_ = r.cmd.Process.Kill()
	}
}

// cleanup is the runReg cleanup: on a stop it snapshots cookies, closes
// Firefox through BiDi and kills it if it has not exited after 8 s; after an
// exit it only releases the relay and the socket.
func (r *firefoxRun) cleanup() {
	r.once.Do(func() {
		select {
		case <-r.exited:
		default:
			r.mu.Lock()
			if r.bidi != nil {
				_ = firefoxSnapshotSession(r.dir, r.device, r.bidi)
				_ = r.bidi.CloseBrowser()
			}
			r.mu.Unlock()
			select {
			case <-r.exited:
			case <-time.After(8 * time.Second):
				r.kill()
			}
		}
		r.bidi.Close()
		r.relay.Close()
	})
}

// focusFirefox brings the environment's window to the front.
func focusFirefox(id string) error {
	r := getFirefoxRun(id)
	if r == nil || r.bidi == nil {
		return errors.New("Firefox 没有在运行")
	}
	ctx, err := r.bidi.firstContext()
	if err != nil {
		return err
	}
	return r.bidi.Activate(ctx)
}

// ---------- session snapshot (same file and format as sessionsync.go) ----------

// sessionCookieStore is what a session snapshot needs from a running browser,
// so the Chromium (CDP) and Firefox (BiDi) kernels share ghost-session.json.
type sessionCookieStore interface {
	GetCookies() ([]Cookie, error)
	SetCookies([]Cookie) (int, error)
}

var _ sessionCookieStore = (*bidiClient)(nil)

// snapCookie is a Cookie in the CDP shape sessionsync.go / cookies.go read.
type snapCookie struct {
	Name     string  `json:"name"`
	Value    string  `json:"value"`
	Domain   string  `json:"domain"`
	Path     string  `json:"path"`
	Expires  float64 `json:"expires"`
	HTTPOnly bool    `json:"httpOnly"`
	Secure   bool    `json:"secure"`
	Session  bool    `json:"session"`
	SameSite string  `json:"sameSite,omitempty"`
}

func writeSessionSnapshot(dir, device string, list []Cookie) error {
	raws := make([]json.RawMessage, 0, len(list))
	for _, c := range list {
		sc := snapCookie{Name: c.Name, Value: c.Value, Domain: c.Domain, Path: c.Path, Expires: c.Expires, HTTPOnly: c.HTTPOnly, Secure: c.Secure, SameSite: c.SameSite}
		if c.Expires <= 0 {
			sc.Expires, sc.Session = -1, true
		}
		b, _ := json.Marshal(sc)
		raws = append(raws, b)
	}
	defer sessionLock(dir)()
	// A schema-2 snapshot (Chromium session store: LocalStorage / IndexedDB /
	// partitioned cookies) is updated in place, never downgraded to schema 1;
	// a snapshot from a newer client is left alone.
	prev, perr := readSessionFile(dir)
	if perr == errSessionSchema {
		return perr
	}
	if perr == nil && prev.Schema >= 2 {
		for _, raw := range prev.Cookies { // BiDi cannot see Chromium's partitioned cookies: keep them
			var c map[string]any
			if json.Unmarshal(raw, &c) == nil && c["partitionKey"] != nil {
				raws = append(raws, raw)
			}
		}
		prev.Device, prev.SavedAt, prev.Cookies = device, time.Now().Unix(), raws
		return writeSessionFile(dir, prev)
	}
	b, _ := json.Marshal(sessionSnap{Device: device, SavedAt: time.Now().Unix(), Cookies: raws})
	tmp := filepath.Join(dir, sessionFile+".tmp")
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, filepath.Join(dir, sessionFile))
}

// firefoxSnapshotSession saves the running browser's cookies to ghost-session.json.
func firefoxSnapshotSession(dir, device string, st sessionCookieStore) error {
	list, err := st.GetCookies()
	if err != nil {
		return err
	}
	return writeSessionSnapshot(dir, device, list)
}

// firefoxRestoreSession writes the snapshot back when another device took it
// (a synced profile opened here), then re-snapshots as this device.
func firefoxRestoreSession(dir, device string, st sessionCookieStore) int {
	b, err := os.ReadFile(filepath.Join(dir, sessionFile))
	if err != nil {
		return 0
	}
	snap, err := decodeSession(b) // schema 1 "cookies" or schema 2 "cookies2"
	if err != nil || snap.Device == device || len(snap.Cookies) == 0 {
		return 0
	}
	plain := make([]json.RawMessage, 0, len(snap.Cookies))
	for _, raw := range snap.Cookies { // Chromium partitioned cookies have no BiDi equivalent here
		var c map[string]any
		if json.Unmarshal(raw, &c) == nil && c["partitionKey"] == nil {
			plain = append(plain, raw)
		}
	}
	jb, _ := json.Marshal(plain)
	all, _ := cdpCookies(jb)
	now := float64(time.Now().Unix())
	list := all[:0]
	for _, c := range all { // drop expired cookies
		if c.Expires == 0 || c.Expires > now {
			list = append(list, c)
		}
	}
	n, _ := st.SetCookies(list)
	_ = firefoxSnapshotSession(dir, device, st)
	return n
}

// firefoxLiveCookies / firefoxSetLiveCookies serve cookie import / export for a running Firefox.
func firefoxLiveCookies(id string) ([]Cookie, bool, error) {
	r := getFirefoxRun(id)
	if r == nil || r.bidi == nil {
		return nil, false, nil
	}
	l, err := r.bidi.GetCookies()
	return l, true, err
}

func firefoxSetLiveCookies(id string, list []Cookie) (int, bool, error) {
	r := getFirefoxRun(id)
	if r == nil || r.bidi == nil {
		return 0, false, nil
	}
	n, err := r.bidi.SetCookies(list)
	return n, true, err
}
