package main

// Default browser (per-profile): GhostBrowser registers once with Windows as
// a web browser (defaultbrowser_windows.go); the profile starred in the list
// (Settings.DefaultProfileID, device-local) opens every external link.
//
//   - Running profile: the link opens in a new tab and the window comes forward.
//   - Not running: it launches through the normal path (sync pull, geo match,
//     overrides) and its first tab lands on the link instead of about:blank.
//   - Links arriving while it launches queue and open as tabs afterwards.
//   - No default (or it was deleted): the main window shows a small picker.

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode"

	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"
	"golang.org/x/net/idna"
)

// ---------------------------------------------------------------- arguments

// parseOpenURLArgs extracts the links from a command line: `--open-url <u>`,
// `--open-url=<u>`, and bare http(s)/file URLs or existing file paths (what
// Explorer passes for "Open with"). Other flags are ignored. Results are
// sanitised; invalid ones are dropped.
func parseOpenURLArgs(args []string) []string {
	out := []string{}
	add := func(raw string) {
		if u, ok := sanitizeExternalURL(raw); ok {
			out = append(out, u)
		}
	}
	for i := 0; i < len(args); i++ {
		a := strings.TrimSpace(args[i])
		switch {
		case a == "--open-url":
			if i+1 < len(args) {
				add(args[i+1])
				i++
			}
		case strings.HasPrefix(a, "--open-url="):
			add(strings.TrimPrefix(a, "--open-url="))
		case a == "--after-update": // updater: next arg is a PID
			i++
		case strings.HasPrefix(a, "-"):
			// other flags (--mcp-stdio, Chromium-style switches passed by shells)
		default:
			l := strings.ToLower(a)
			if strings.HasPrefix(l, "http://") || strings.HasPrefix(l, "https://") || strings.HasPrefix(l, "file:") || looksLikePath(a) {
				add(a)
			}
		}
	}
	return out
}

var winDrivePath = regexp.MustCompile(`^[A-Za-z]:[\\/]`)

// looksLikePath: an absolute Windows/UNC/Unix path.
func looksLikePath(s string) bool {
	s = strings.Trim(s, `"`)
	return winDrivePath.MatchString(s) || strings.HasPrefix(s, `\\`) || (strings.HasPrefix(s, "/") && !strings.HasPrefix(s, "//"))
}

// pathToFileURL turns C:\a b\x.html into file:///C:/a%20b/x.html.
func pathToFileURL(p string) string {
	p = strings.ReplaceAll(p, `\`, "/")
	if strings.HasPrefix(p, "//") { // UNC: //server/share/x -> file://server/share/x
		u := url.URL{Scheme: "file", Host: strings.SplitN(p[2:], "/", 2)[0]}
		if parts := strings.SplitN(p[2:], "/", 2); len(parts) == 2 {
			u.Path = "/" + parts[1]
		}
		return u.String()
	}
	if !strings.HasPrefix(p, "/") {
		p = "/" + p
	}
	return (&url.URL{Scheme: "file", Path: p}).String()
}

const maxExternalURL = 8192

// sanitizeExternalURL accepts only http(s) URLs with a host and file URLs /
// absolute paths; everything else (javascript:, data:, about:, ms-settings:
// ...) is refused. Quotes and whitespace around the value are removed and
// control characters are rejected.
func sanitizeExternalURL(raw string) (string, bool) {
	s := strings.TrimSpace(raw)
	s = strings.Trim(s, `"'`)
	s = strings.TrimSpace(s)
	if s == "" || len(s) > maxExternalURL {
		return "", false
	}
	for _, r := range s {
		if unicode.IsControl(r) {
			return "", false
		}
	}
	if looksLikePath(s) {
		s = pathToFileURL(s)
	}
	u, err := url.Parse(s)
	if err != nil {
		return "", false
	}
	switch strings.ToLower(u.Scheme) {
	case "http", "https":
		h := u.Hostname()
		if h == "" || strings.ContainsAny(h, " <>\"{}|\\^`") {
			return "", false
		}
		if a, e := idna.Lookup.ToASCII(h); e == nil && a != h { // IDN host -> punycode
			if p := u.Port(); p != "" {
				u.Host = a + ":" + p
			} else {
				u.Host = a
			}
		}
		u.Scheme = strings.ToLower(u.Scheme)
	case "file":
		u.Scheme = "file"
		if u.Path == "" && u.Opaque == "" {
			return "", false
		}
		if u.Opaque != "" { // file:C:/x -> file:///C:/x
			u.Path, u.Opaque = "/"+u.Opaque, ""
		}
	default:
		return "", false
	}
	return u.String(), true
}

// ---------------------------------------------------------------- routing

type linkAction string

const (
	linkPick   linkAction = "pick"   // no usable default: ask in the main window
	linkTab    linkAction = "tab"    // running: open a new tab
	linkQueue  linkAction = "queue"  // launching: open once it is up
	linkLaunch linkAction = "launch" // stopped: launch straight to the link
)

// routeLink is the routing decision for one link.
func routeLink(profileID string, exists, running, launching bool) linkAction {
	switch {
	case profileID == "" || !exists:
		return linkPick
	case launching:
		return linkQueue
	case running:
		return linkTab
	default:
		return linkLaunch
	}
}

// linkRouter holds links until the app is ready and serialises launches per
// profile. The functions are injected so tests can fake the browser side.
type linkRouter struct {
	mu        sync.Mutex
	ready     bool
	early     []string            // arrived before start-up finished
	launching map[string][]string // profile -> links waiting for its launch
	picks     []string            // waiting for the user to choose a profile

	defaultID  func() string
	exists     func(id string) (string, bool)
	isRunning  func(id string) bool
	openTab    func(id, u string) error
	launch     func(id, u string) error
	showPicker func(u string)
	report     func(id, name, u string, launched bool, err error)
}

// Open routes one link (already sanitised) to the default profile.
func (r *linkRouter) Open(u string) {
	r.mu.Lock()
	if !r.ready {
		r.early = append(r.early, u)
		r.mu.Unlock()
		return
	}
	r.mu.Unlock()
	r.OpenIn(r.defaultID(), u)
}

// OpenIn routes a link to one profile ("" or unknown = ask).
func (r *linkRouter) OpenIn(id, u string) {
	name, ok := "", false
	if id != "" {
		name, ok = r.exists(id)
	}
	r.mu.Lock()
	_, launching := r.launching[id]
	act := routeLink(id, ok, ok && r.isRunning(id), launching)
	switch act {
	case linkPick:
		r.picks = append(r.picks, u)
		r.mu.Unlock()
		r.showPicker(u)
		return
	case linkQueue:
		r.launching[id] = append(r.launching[id], u)
		r.mu.Unlock()
		return
	case linkLaunch:
		r.launching[id] = []string{}
	}
	r.mu.Unlock()

	if act == linkTab {
		r.report(id, name, u, false, r.openTab(id, u))
		return
	}
	err := r.launch(id, u)
	r.report(id, name, u, true, err)
	r.mu.Lock()
	queued := r.launching[id]
	delete(r.launching, id)
	r.mu.Unlock()
	for _, q := range queued {
		if err != nil { // launch failed: try again for the next link
			r.OpenIn(id, q)
			continue
		}
		r.report(id, name, q, false, r.openTab(id, q))
	}
}

// SetReady flushes the links that arrived during start-up.
func (r *linkRouter) SetReady() {
	r.mu.Lock()
	r.ready = true
	early := r.early
	r.early = nil
	r.mu.Unlock()
	for _, u := range early {
		r.Open(u)
	}
}

// TakePicks returns and clears the links waiting for a profile choice.
func (r *linkRouter) TakePicks() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := r.picks
	r.picks = nil
	if out == nil {
		out = []string{}
	}
	return out
}

func (r *linkRouter) dropPick(u string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i, p := range r.picks {
		if p == u {
			r.picks = append(r.picks[:i], r.picks[i+1:]...)
			return
		}
	}
}

func newLinkRouter() *linkRouter { return &linkRouter{launching: map[string][]string{}} }

var links = newLinkRouter()

// ---------------------------------------------------------------- app wiring

// coldLinks are the links the process was started with (a link click while
// GhostBrowser was not running): start hidden and only open them.
var coldLinks []string

// DefaultBrowserStatus is the 设置 → 默认浏览器 card and the star's state.
type DefaultBrowserStatus struct {
	Supported       bool   `json:"supported"`
	Registered      bool   `json:"registered"`
	IsSystemDefault bool   `json:"isSystemDefault"`
	ProfileID       string `json:"profileId"`
	ProfileName     string `json:"profileName"`
	Error           string `json:"error,omitempty"`
}

// LinkOpened is emitted as "defaultbrowser:opened" after a link was routed.
type LinkOpened struct {
	ProfileID string `json:"profileId"`
	Name      string `json:"name"`
	URL       string `json:"url"`
	Launched  bool   `json:"launched"`
	Error     string `json:"error,omitempty"`
}

func (a *App) linksStartup() {
	links.defaultID = func() string { return a.store.GetSettings().DefaultProfileID }
	links.exists = func(id string) (string, bool) {
		p, ok := a.store.GetProfile(id)
		if !ok || p.Trashed {
			return "", false
		}
		return p.Name, true
	}
	links.isRunning = func(id string) bool { _, ok := runReg.get(id); return ok }
	links.openTab = func(id, u string) error { return openTabIn(id, u) }
	links.launch = a.launchToURL
	links.showPicker = func(u string) {
		a.showMainWindow()
		wruntime.EventsEmit(a.ctx, "defaultbrowser:pick", u)
	}
	links.report = func(id, name, u string, launched bool, err error) {
		ev := LinkOpened{ProfileID: id, Name: name, URL: u, Launched: launched}
		if err != nil {
			ev.Error = err.Error()
		}
		a.store.auditNamed("link", "defaultbrowser.open", id, name, err, u)
		if launched && err == nil {
			wruntime.EventsEmit(a.ctx, "profile:started", id)
		}
		wruntime.EventsEmit(a.ctx, "defaultbrowser:opened", ev)
		if err != nil { // the user clicked a link and nothing appeared: say why
			a.showMainWindow()
		}
	}
	if len(coldLinks) > 0 && !trayAvailable() {
		wruntime.WindowShow(a.ctx) // no tray: started hidden, park it on the taskbar
		wruntime.WindowMinimise(a.ctx)
	}
	go func() {
		if r := ensureDefaultBrowserRegistration(); r != "" {
			println("defaultbrowser:", r)
		}
	}()
	for _, u := range coldLinks {
		links.Open(u)
	}
	coldLinks = nil
	go links.SetReady()
}

func (a *App) showMainWindow() {
	if a.ctx == nil {
		return
	}
	wruntime.WindowShow(a.ctx)
	wruntime.WindowUnminimise(a.ctx)
}

// onSecondInstance is Wails' OnSecondInstanceLaunch: a link clicked while
// GhostBrowser runs, or the user starting the EXE again.
func (a *App) onSecondInstance(args []string) {
	urls := parseOpenURLArgs(args)
	if len(urls) == 0 {
		a.showMainWindow()
		return
	}
	for _, u := range urls {
		go links.Open(u)
	}
}

// launchToURL starts a stopped profile through the normal launch path (share
// guard, sync pull, geo match, fingerprint overrides) with u as its only start
// page: the launchers navigate the initial blank tab once overrides are in.
func (a *App) launchToURL(id, u string) error {
	if err := a.store.shareLaunchGuard(id); err != nil {
		return err
	}
	if err := a.auto.BeforeLaunch(id); err != nil {
		return err
	}
	launchCfg.setURL(id, u)
	_, err := a.store.Launch(id)
	launchCfg.takeURL(id) // never leak into a later launch
	return err
}

// openTabIn opens u in a new tab of a running profile and brings it forward.
func openTabIn(id, u string) error {
	rs, ok := runReg.get(id)
	if !ok {
		return fmt.Errorf("环境没有在运行")
	}
	if rs.Kernel == "firefox" {
		r := getFirefoxRun(id)
		if r == nil || r.bidi == nil {
			return fmt.Errorf("Firefox 没有在运行")
		}
		ctx, err := r.bidi.NewTab()
		if err != nil {
			return err
		}
		if err = r.bidi.Navigate(ctx, u); err != nil {
			return err
		}
		return r.bidi.Activate(ctx)
	}
	return cdpOpenTab(rs.WSURL, u)
}

// cdpOpenTab: Target.createTarget on the browser connection (the emulator's
// auto-attach patches the new page before it runs), then activate it and
// restore its window if minimised.
func cdpOpenTab(ws, u string) error {
	if !strings.HasPrefix(ws, "ws") {
		return fmt.Errorf("环境的调试连接不可用")
	}
	raw, err := cdpCall(ws, "Target.createTarget", map[string]any{"url": u})
	if err != nil {
		return fmt.Errorf("无法在运行中的环境打开链接：%v", err)
	}
	var t struct {
		TargetID string `json:"targetId"`
	}
	_ = json.Unmarshal(raw, &t)
	if t.TargetID == "" {
		return nil
	}
	if w, e := cdpCall(ws, "Browser.getWindowForTarget", map[string]any{"targetId": t.TargetID}); e == nil {
		var win struct {
			WindowID int `json:"windowId"`
		}
		if json.Unmarshal(w, &win) == nil && win.WindowID != 0 {
			_, _ = cdpCall(ws, "Browser.setWindowBounds", map[string]any{"windowId": win.WindowID, "bounds": map[string]any{"windowState": "normal"}})
		}
	}
	_, _ = cdpCall(ws, "Target.activateTarget", map[string]any{"targetId": t.TargetID})
	return nil
}

// ---------------------------------------------------------------- bindings

func (a *App) DefaultBrowserStatus() DefaultBrowserStatus {
	st := DefaultBrowserStatus{Supported: defaultBrowserSupported()}
	id := a.store.GetSettings().DefaultProfileID
	if p, ok := a.store.GetProfile(id); ok && id != "" {
		st.ProfileID, st.ProfileName = id, p.Name
	}
	if st.Supported {
		reg, def, err := defaultBrowserState()
		st.Registered, st.IsSystemDefault = reg, def
		if err != nil {
			st.Error = err.Error()
		}
	}
	return st
}

func (a *App) emitDefaultBrowser() DefaultBrowserStatus {
	st := a.DefaultBrowserStatus()
	if a.ctx != nil {
		wruntime.EventsEmit(a.ctx, "defaultbrowser:changed", st)
	}
	return st
}

// SetDefaultProfile stars a profile as the one that opens external links
// ("" clears it).
func (a *App) SetDefaultProfile(id string) (DefaultBrowserStatus, error) {
	name := ""
	if id != "" {
		p, ok := a.store.GetProfile(id)
		if !ok {
			return a.DefaultBrowserStatus(), fmt.Errorf("环境不存在")
		}
		name = p.Name
	}
	s := a.store.GetSettings()
	prev := s.DefaultProfileID
	s.DefaultProfileID = id
	a.store.PutSettings(s)
	if id != "" {
		a.store.auditNamed("ui", "defaultbrowser.set", id, name, nil, "")
	} else if prev != "" {
		a.store.auditNamed("ui", "defaultbrowser.clear", prev, "", nil, "")
	}
	return a.emitDefaultBrowser(), nil
}

// RegisterDefaultBrowser registers GhostBrowser with Windows (HKCU, no admin)
// and opens the system Default Apps page: Windows 10/11 only lets the user
// make the final choice there.
func (a *App) RegisterDefaultBrowser() (DefaultBrowserStatus, error) {
	if !defaultBrowserSupported() {
		return a.DefaultBrowserStatus(), fmt.Errorf("只有 Windows 版可以设为系统默认浏览器")
	}
	exe, err := os.Executable()
	if err != nil {
		return a.DefaultBrowserStatus(), err
	}
	if p, e := filepath.EvalSymlinks(exe); e == nil {
		exe = p
	}
	err = registerDefaultBrowser(exe)
	a.store.auditNamed("ui", "defaultbrowser.register", "", "", err, exe)
	if err != nil {
		return a.emitDefaultBrowser(), fmt.Errorf("注册失败：%v", err)
	}
	time.Sleep(150 * time.Millisecond)
	_ = openDefaultAppsSettings()
	return a.emitDefaultBrowser(), nil
}

func (a *App) UnregisterDefaultBrowser() (DefaultBrowserStatus, error) {
	if !defaultBrowserSupported() {
		return a.DefaultBrowserStatus(), fmt.Errorf("只有 Windows 版支持")
	}
	err := unregisterDefaultBrowser()
	a.store.auditNamed("ui", "defaultbrowser.unregister", "", "", err, "")
	if err != nil {
		err = fmt.Errorf("取消注册失败：%v", err)
	}
	return a.emitDefaultBrowser(), err
}

func (a *App) OpenDefaultAppsSettings() error {
	if !defaultBrowserSupported() {
		return fmt.Errorf("只有 Windows 版支持")
	}
	return openDefaultAppsSettings()
}

// OpenURLIn opens a link from the picker dialog in the chosen profile.
func (a *App) OpenURLIn(profileID, rawURL string) error {
	u, ok := sanitizeExternalURL(rawURL)
	if !ok {
		return fmt.Errorf("链接无效：只支持 http / https / file")
	}
	if _, ok := a.store.GetProfile(profileID); !ok {
		return fmt.Errorf("环境不存在")
	}
	links.dropPick(u)
	go links.OpenIn(profileID, u)
	return nil
}

// PendingLinks returns links still waiting for a profile choice (the UI asks
// on load in case it missed "defaultbrowser:pick" during start-up).
func (a *App) PendingLinks() []string { return links.TakePicks() }

// DismissLink drops a link from the picker without opening it.
func (a *App) DismissLink(rawURL string) { links.dropPick(rawURL) }

// keepDeviceLocal copies the per-device settings that are never synced or
// edited through the settings page (default browser profile) from cur to s.
func keepDeviceLocal(s *Settings, cur Settings) {
	s.DefaultProfileID = cur.DefaultProfileID
}
