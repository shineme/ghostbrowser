package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx   context.Context
	store *Store
	auto  *autoSyncer
}

func NewApp() *App { return &App{} }

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.store = NewStore()
	userLib.open(a.store.dir)
	env.emit = func(s EnvStatus) { wruntime.EventsEmit(a.ctx, "env:setup", s) }
	// The EXE ships alone: install the custom Chromium and fingerprint kit on
	// first run, then point the store at whatever ended up on disk.
	st := applyToSettings(a.store.GetSettings())
	if st.KernelPath != a.store.GetSettings().KernelPath || st.UndetectableKit != a.store.GetSettings().UndetectableKit {
		a.store.PutSettings(st)
	}
	go func() {
		env.Ensure(false)
		a.store.PutSettings(applyToSettings(a.store.GetSettings()))
	}()
	a.store.onTransfer = func(e TransferEvent) { wruntime.EventsEmit(a.ctx, "sync:progress", e) }
	globalAuto = &autoSyncer{s: a.store, busy: map[string]bool{}, emit: func(e SyncEvent) { wruntime.EventsEmit(a.ctx, "sync:event", e) }}
	runReg.onExit = func(id string) {
		wruntime.EventsEmit(a.ctx, "profile:exit", id)
		globalAuto.AfterExit(id)
	}
	runReg.beforeStop = func(id, ws string) { _ = snapshotSession(a.store.ProfileDir(id), a.store.GetSettings().DeviceID, ws) }
	a.auto = globalAuto
	api.start(a.store)
	// Server-first reconcile on start-up, off the UI thread.
	go func() { time.Sleep(800 * time.Millisecond); a.auto.Reconcile(); a.auto.WatchGlobal() }()
	go a.proxyHealthLoop()
}

// ---------- profiles ----------

func (a *App) ListProfiles() []Profile { return a.store.ListProfiles() }

func (a *App) SaveProfile(p Profile) (Profile, error) {
	if p.Fingerprint.UserAgent == "" && needsOpenKernel(p.Fingerprint) {
		p.Fingerprint = GenerateTyped(p.Fingerprint.DeviceType, p.Fingerprint.OS, a.ChromiumMajor())
	}
	chromium := p.useChromium()
	if chromium {
		if needsOpenKernel(p.Fingerprint) {
			p.Kernel = "chromium"
		}
		return a.store.PutProfile(p), nil
	}
	if a.store.GetSettings().KernelMode == "undetectable" && p.Native == nil {
		if n, fp, err := a.store.nativePreset(""); err == nil {
			p.Native = n
			if p.Fingerprint.UserAgent == "" {
				p.Fingerprint = fp
			}
		}
	}
	if p.Fingerprint.UserAgent == "" {
		p.Fingerprint = GenerateFingerprint(a.store.GetSettings().DefaultOS, a.KernelMajor())
	}
	if a.store.GetSettings().KernelMode == "undetectable" {
		if p.ID == "" {
			p.ID = newID()
			p.CreatedAt = time.Now().Unix()
		}
		if _, err := a.store.nativeSpec(&p, a.store.GetSettings()); err != nil {
			return Profile{}, err
		}
	}
	return a.store.PutProfile(p), nil
}

func (a *App) DeleteProfiles(ids []string) {
	for _, id := range ids {
		runReg.stop(id)
		a.store.DeleteProfile(id)
	}
}

// CreateProfiles creates count desktop profiles named prefix-N with fresh fingerprints.
func (a *App) CreateProfiles(prefix, osName string, count int, proxyID, group string) []Profile {
	return a.CreateTypedProfiles(prefix, "desktop", osName, count, proxyID, group)
}

// CreateTypedProfiles creates count profiles of one device type (desktop, phone, tablet, ...).
func (a *App) CreateTypedProfiles(prefix, devType, osName string, count int, proxyID, group string) []Profile {
	out := []Profile{}
	if count < 1 {
		count = 1
	}
	for i := 1; i <= count; i++ {
		name := prefix
		if count > 1 {
			name = prefix + "-" + itoa(i)
		}
		p := Profile{Name: name, Group: group, ProxyID: proxyID, Storage: "local"}
		p.Fingerprint = a.NewDeviceFingerprint(devType, osName)
		if needsOpenKernel(p.Fingerprint) {
			p.Kernel = "chromium"
		} else if a.store.GetSettings().KernelMode == "undetectable" {
			if n, fp, e := a.store.nativePreset(""); e == nil {
				p.Native = n
				p.Fingerprint = fp
			}
		}
		out = append(out, a.store.PutProfile(p))
	}
	return out
}

func (a *App) CloneProfile(id string) (Profile, error) {
	p, ok := a.store.GetProfile(id)
	if !ok {
		return Profile{}, errf("profile not found")
	}
	cp := *p
	cp.ID, cp.Name, cp.CloudVer, cp.SyncedAt, cp.Storage = "", p.Name+" (copy)", 0, 0, "local"
	if cp.Native != nil {
		cp.Native = copyNative(cp.Native)
	} else {
		cp.Fingerprint = a.NewDeviceFingerprint(p.Fingerprint.DeviceType, p.Fingerprint.OS)
	}
	return a.store.PutProfile(cp), nil
}

func (a *App) NewFingerprint(osName string) Fingerprint {
	if a.store.GetSettings().KernelMode == "undetectable" && osName == "windows" {
		if _, fp, e := a.store.nativePreset(""); e == nil {
			return fp
		}
	}
	return GenerateFingerprint(osName, a.KernelMajor())
}

// NewDeviceFingerprint generates a fingerprint for one device type and OS.
// Windows desktops/laptops on the custom kernel start from its template.
func (a *App) NewDeviceFingerprint(devType, osName string) Fingerprint {
	devType = normDevType(devType)
	if devType == "desktop" || devType == "laptop" {
		if osName == "" {
			osName = "windows"
		}
		if osName == "windows" && a.store.GetSettings().KernelMode == "undetectable" && devType == "desktop" {
			if _, fp, e := a.store.nativePreset(""); e == nil {
				fp.DeviceType = devType
				return fp
			}
		}
		major := a.KernelMajor()
		if osName != "windows" {
			major = a.ChromiumMajor()
		}
		return GenerateTyped(devType, osName, major)
	}
	return GenerateTyped(devType, osName, a.ChromiumMajor())
}

// DeviceTypes lists the device categories with the OSes each one allows.
func (a *App) DeviceTypes() []DeviceTypeInfo { return DeviceTypes }

// DevicePresetsByType lists presets of one device type ("" os = every OS).
func (a *App) DevicePresetsByType(devType, osName string) []DevicePreset {
	return PresetsByType(devType, osName)
}

// FingerprintCatalogFor feeds the editor's dropdowns for one OS + device type.
// UA versions follow whichever kernel that combination launches on.
func (a *App) FingerprintCatalogFor(osName, devType string) FPCatalog {
	major := a.KernelMajor()
	if needsOpenKernel(Fingerprint{OS: osName, DeviceType: devType}) {
		major = a.ChromiumMajor()
	}
	return FingerprintCatalogFor(osName, devType, major)
}

// ChromiumPath is the open-source kernel that will be used (setting or auto-detected).
func (a *App) ChromiumPath() string { return chromiumKernel(a.store.GetSettings()) }

// ChromiumMajor is the open-source kernel's Chrome major version (141 when unknown).
func (a *App) ChromiumMajor() string {
	if k := chromiumKernel(a.store.GetSettings()); k != "" {
		if m := chromeMajorFrom(filepath.Dir(k)); m != "" {
			return m
		}
	}
	return "141"
}

// PickChromium opens a file dialog for the open-source kernel's executable.
func (a *App) PickChromium() string {
	dir := filepath.Dir(chromiumKernel(a.store.GetSettings()))
	f, _ := wruntime.OpenFileDialog(a.ctx, wruntime.OpenDialogOptions{
		Title:            "选择开源内核 (chrome.exe：Chromium / Chrome / 自己编译的内核)",
		DefaultDirectory: dir,
		Filters:          []wruntime.FileFilter{{DisplayName: "浏览器程序 (*.exe)", Pattern: "*.exe"}},
	})
	return f
}

func (a *App) GPUOptions(osName string) []map[string]string {
	if a.store.GetSettings().KernelMode == "undetectable" {
		if _, fp, e := a.store.nativePreset(""); e == nil {
			return []map[string]string{{"vendor": fp.WebGLVendor, "renderer": fp.WebGLRenderer}}
		}
	}
	return GPUOptions(osName)
}
func (a *App) ScreenOptions(osName string) []string { return ScreenOptions(osName) }

// FingerprintCatalog feeds the editor's linked dropdowns for one OS.
func (a *App) FingerprintCatalog(osName string) FPCatalog {
	return FingerprintCatalog(osName, a.KernelMajor())
}

// ---------- user fingerprint library ----------

// LibraryAdd saves one custom value (a UA, a GPU "vendor\x1frenderer", a font, a screen "WxH@dpr", ...).
func (a *App) LibraryAdd(osName, field, value string) error { return userLib.Add(osName, field, value) }
func (a *App) LibraryRemove(osName, field, value string) error {
	return userLib.Remove(osName, field, value)
}
func (a *App) SaveDevicePreset(p DevicePreset) (DevicePreset, error) { return userLib.SavePreset(p) }
func (a *App) DeleteDevicePreset(id string) error                    { return userLib.DeletePreset(id) }

// DevicePresets returns GhostBrowser's own device library for one OS ("" = all).
func (a *App) DevicePresets(osName string) []DevicePreset { return DevicePresetsFor(osName) }

// ---------- run ----------

func (a *App) StartProfile(id string) (*RunState, error) {
	if _, ok := runReg.get(id); ok {
		rs, _ := runReg.get(id)
		return rs, nil
	}
	a.auto.BeforeLaunch(id)
	return a.store.Launch(id)
}

// SyncNow runs the same server-first comparison as start-up.
func (a *App) SyncNow() SyncEvent { return a.auto.Reconcile() }

// PendingSyncPrompt returns the open "cloud is newer" prompt, if any (the UI
// asks on load in case it missed the event).
func (a *App) PendingSyncPrompt() *SyncPrompt { return a.auto.Pending() }

// SyncHold stops the prompt's countdown while the user is choosing.
func (a *App) SyncHold() { a.auto.Hold() }

// SyncDecide answers the prompt: listed profiles are pulled from the cloud,
// the others keep the local copy (which is then uploaded over the cloud one).
// Transfers lists uploads/downloads in progress.
func (a *App) Transfers() []TransferEvent { return ActiveTransfers() }

func (a *App) SyncDecide(pull []string) bool { return a.auto.Decide(pull) }

func (a *App) StopProfile(id string) {
	if rs, ok := runReg.get(id); ok && strings.HasPrefix(rs.WSURL, "ws") {
		if closeBrowser(rs.WSURL) == nil {
			return
		}
	}
	runReg.stop(id)
}

func (a *App) Running() []RunState { return runReg.list() }

func (a *App) ExportCookies(id string) (string, error) {
	rs, ok := runReg.get(id)
	if !ok || !strings.HasPrefix(rs.WSURL, "ws") {
		return "", errf("start the profile first to export its cookies")
	}
	return exportCookies(rs.WSURL)
}

// ---------- proxies ----------

func (a *App) ListProxies() []Proxy                  { return a.store.ListProxies() }
func (a *App) SaveProxy(p Proxy) Proxy               { return a.store.PutProxy(p) }
func (a *App) DeleteProxy(id string)                 { a.store.DeleteProxy(id) }
func (a *App) ParseProxy(line string) (Proxy, error) { return ParseProxy(line) }

func (a *App) ImportProxies(text string) []Proxy {
	out := []Proxy{}
	for _, pv := range PreviewProxyText(text, "http", a.store.ListProxies()) {
		if pv.OK && !pv.Dup {
			out = append(out, a.store.PutProxy(pv.Proxy))
		}
	}
	return out
}

// PreviewProxies parses pasted text and marks duplicates, without saving.
func (a *App) PreviewProxies(text, defType string) []ProxyPreview {
	return PreviewProxyText(text, defType, a.store.ListProxies())
}

// ImportProxyList saves parsed proxies; tag labels the batch, and when it is
// set the names become "tag #n".
func (a *App) ImportProxyList(list []Proxy, tag string) []Proxy {
	tag = strings.TrimSpace(tag)
	out := []Proxy{}
	for i, p := range list {
		p.ID, p.Status = "", "unknown"
		if tag != "" {
			p.Tag = tag
			if p.Name == fmt.Sprintf("%s:%d", p.Host, p.Port) {
				p.Name = fmt.Sprintf("%s #%d", tag, i+1)
			}
		}
		out = append(out, a.store.PutProxy(p))
	}
	return out
}

func (a *App) emitProxy(p Proxy) {
	if a.ctx != nil {
		wruntime.EventsEmit(a.ctx, "proxy:checked", p)
	}
}

func (a *App) CheckProxy(id string) Proxy {
	p, ok := a.store.GetProxy(id)
	if !ok {
		return Proxy{}
	}
	r := a.store.PutProxy(CheckProxy(*p))
	a.emitProxy(r)
	return r
}

func (a *App) CheckProxyDraft(p Proxy) Proxy { return CheckProxy(p) }

// CheckProxies checks the given proxies (all when ids is empty), 16 at a time,
// emitting "proxy:checked" as each one finishes.
func (a *App) CheckProxies(ids []string) []Proxy {
	runPool(a.pickProxies(ids), 16, func(p Proxy) {
		r := CheckProxy(p)
		if cur, ok := a.store.GetProxy(p.ID); ok { // keep edits made meanwhile
			r.Name, r.Tag, r.ChangeIP, r.Speed, r.SpeedAt, r.RotateOnLaunch = cur.Name, cur.Tag, cur.ChangeIP, cur.Speed, cur.SpeedAt, cur.RotateOnLaunch
		}
		a.emitProxy(a.store.PutProxy(r))
	})
	return a.store.ListProxies()
}

func (a *App) CheckAllProxies() []Proxy { return a.CheckProxies(nil) }

// SpeedTestProxies measures download speed through each proxy, 4 at a time.
func (a *App) SpeedTestProxies(ids []string) []Proxy {
	runPool(a.pickProxies(ids), 4, func(p Proxy) {
		r := SpeedTestProxy(p)
		if cur, ok := a.store.GetProxy(p.ID); ok {
			c := *cur
			c.Speed, c.SpeedAt = r.Speed, r.SpeedAt
			if r.Speed == 0 {
				c.Error = r.Error
			}
			a.emitProxy(a.store.PutProxy(c))
		}
	})
	return a.store.ListProxies()
}

func (a *App) DeleteProxies(ids []string) {
	for _, id := range ids {
		a.store.DeleteProxy(id)
	}
}

func (a *App) pickProxies(ids []string) []Proxy {
	all := a.store.ListProxies()
	if len(ids) == 0 {
		return all
	}
	want := map[string]bool{}
	for _, id := range ids {
		want[id] = true
	}
	out := []Proxy{}
	for _, p := range all {
		if want[p.ID] {
			out = append(out, p)
		}
	}
	return out
}

// MatchGeo checks the exit IP of a saved proxy (proxyID), an inline proxy, or
// the direct connection (both empty) and returns the timezone, languages and
// coordinates a profile should use there.
func (a *App) MatchGeo(proxyID string, inline *Proxy) GeoMatch {
	var px *Proxy
	if inline != nil && inline.Host != "" {
		px = inline
	} else if proxyID != "" {
		if p, ok := a.store.GetProxy(proxyID); ok {
			px = p
		} else {
			return GeoMatch{Error: "代理不存在"}
		}
	}
	m := a.store.liveExit(px, 5*time.Minute)
	if px != nil && px.ID != "" {
		if p, ok := a.store.GetProxy(px.ID); ok {
			a.emitProxy(*p)
		}
	}
	return m
}

// ---------- settings ----------

func (a *App) GetSettings() Settings { return a.store.GetSettings() }

func (a *App) SaveSettings(s Settings) Settings {
	out := a.store.PutSettings(s)
	api.restart(a.store)
	return out
}

// DefaultKernelPath returns the bundled custom Chromium when it is available,
// falling back to a detected system browser for stock mode.
func (a *App) DefaultKernelPath() string {
	return defaultKernelPath(a.store.GetSettings())
}

func (a *App) PickKernel() string {
	st := a.store.GetSettings()
	defaultDir := filepath.Dir(st.KernelPath)
	if st.KernelPath == "" {
		defaultDir = filepath.Dir(a.DefaultKernelPath())
	}
	if info, err := os.Stat(defaultDir); err != nil || !info.IsDir() {
		defaultDir = filepath.Dir(a.DefaultKernelPath())
	}
	f, _ := wruntime.OpenFileDialog(a.ctx, wruntime.OpenDialogOptions{
		Title:            "选择浏览器程序 (chrome.exe / msedge.exe)",
		DefaultDirectory: defaultDir,
		Filters:          []wruntime.FileFilter{{DisplayName: "浏览器程序 (*.exe)", Pattern: "*.exe"}},
	})
	return f
}

func (a *App) PickFolder() string {
	f, _ := wruntime.OpenDirectoryDialog(a.ctx, wruntime.OpenDialogOptions{Title: "Profiles folder"})
	return f
}

// ---------- environment ----------

// EnvStatus reports the last check-and-install state for the settings page.
func (a *App) EnvStatus() EnvStatus { return env.snapshot() }

// EnvRecheck downloads anything missing right away and rebinds the settings to
// the installed kit. force reinstalls both packages even when they look present.
func (a *App) EnvRecheck(force bool) EnvStatus {
	go func() {
		env.Ensure(force)
		a.store.PutSettings(applyToSettings(a.store.GetSettings()))
		wruntime.EventsEmit(a.ctx, "env:settings", a.store.GetSettings())
	}()
	return env.snapshot()
}

// EnvCancel stops an in-flight download.
func (a *App) EnvCancel() { env.Cancel() }

var verRe = regexp.MustCompile(`^(\d+)\.\d+\.\d+\.\d+`)

// chromeMajorFrom returns the kernel's Chrome major version. A normal Chrome
// install keeps versions in sibling folders; the delivered package is flat and
// only carries <version>.manifest, so both are checked.
func chromeMajorFrom(dir string) string {
	vs := []string{}
	if ents, err := os.ReadDir(dir); err == nil {
		for _, e := range ents {
			name := e.Name()
			if m := verRe.FindStringSubmatch(name); m != nil {
				if e.IsDir() {
					vs = append(vs, m[1])
					continue
				}
				if strings.HasSuffix(name, ".manifest") {
					vs = append(vs, m[1])
				}
			}
		}
	}
	sort.Strings(vs)
	if len(vs) > 0 {
		return vs[len(vs)-1]
	}
	return ""
}

// KernelMajor detects the Chrome major version from the version folder that
// sits next to chrome.exe; falls back to 153 for the bundled kernel and 141
// otherwise.
func (a *App) KernelMajor() string {
	st := a.store.GetSettings()
	if k := st.KernelPath; k != "" {
		if m := chromeMajorFrom(filepath.Dir(k)); m != "" {
			return m
		}
	}
	if m := chromeMajorFrom(filepath.Dir(defaultKernelPath(st))); m != "" {
		return m
	}
	if st.KernelMode == "undetectable" {
		return "153"
	}
	return "141"
}

// ---------- cloud sync ----------

func (a *App) TestSync() error                 { return a.store.TestSync() }
func (a *App) PushProfile(id string) error     { return a.store.PushProfile(id) }
func (a *App) PullProfile(id string) error     { return a.store.PullProfile(id) }
func (a *App) ListCloud() ([]cloudMeta, error) { return a.store.ListCloud() }

func (a *App) DataDir() string { return appDataDir() }

// ResetAPIKey issues a new local API key; scripts using the old one stop working.
func (a *App) ResetAPIKey() Settings {
	st := a.store.GetSettings()
	st.APIKey = newAPIKey()
	return a.SaveSettings(st)
}

// ---- sync: server info, history, log

func (a *App) SyncServerInfo() (serverInfo, error)            { return a.store.ServerInfo(true) }
func (a *App) CloudHistory(id string) ([]CloudVersion, error) { return a.store.CloudHistory(id) }
func (a *App) RestoreVersion(id string, version int64) error {
	if _, ok := runReg.get(id); ok {
		return fmt.Errorf("请先关闭这个环境的浏览器")
	}
	return a.store.RestoreVersion(id, version)
}
func (a *App) SyncLog() []SyncLogEntry { return a.store.readSyncLog() }
func (a *App) ClearSyncLog() {
	syncLogMu.Lock()
	_ = os.Remove(a.store.syncLogFile())
	syncLogMu.Unlock()
}

// RetrySync repeats one failed operation from the sync log.
func (a *App) RetrySync(id, action string) error {
	switch action {
	case "push":
		return a.store.PushProfile(id)
	case "pull":
		return a.store.PullProfile(id)
	case "push-global":
		return a.store.PushGlobal()
	case "pull-global":
		return a.store.PullGlobal()
	}
	return fmt.Errorf("这条记录不能重试")
}

func (a *App) SyncBusy() []string { return a.auto.Busy() }

// ---- arrange/syncer ----

// arrangeHooks wires the Wails screen API (fallback when the OS work area is
// unknown) and the syncer's event emitter; done lazily so startup stays untouched.
func (a *App) arrangeHooks() {
	if screenHook == nil && a.ctx != nil {
		ctx := a.ctx
		screenHook = func() (Rect, bool) {
			scr, err := wruntime.ScreenGetAll(ctx)
			if err != nil {
				return Rect{}, false
			}
			for _, s := range scr {
				if s.IsPrimary && s.Size.Width > 0 {
					// Wails reports the full screen; leave room for a taskbar.
					return Rect{0, 0, s.Size.Width, s.Size.Height - 48}, true
				}
			}
			return Rect{}, false
		}
	}
	groupSync.mu.Lock()
	if groupSync.emit == nil && a.ctx != nil {
		groupSync.emit = func(s SyncerState) { wruntime.EventsEmit(a.ctx, "syncer:event", s) }
	}
	groupSync.mu.Unlock()
}

// ArrangeWindows lays out the browser windows of running profiles (all
// running ones when ids is empty). mode: grid | cascade | stack.
func (a *App) ArrangeWindows(ids []string, mode string) (ArrangeResult, error) {
	a.arrangeHooks()
	res := arrangeRunning(ids, mode, arrangeScreen(), a.profileName)
	if res.Total == 0 {
		return res, fmt.Errorf("没有正在运行的环境可以排列")
	}
	return res, nil
}

// ArrangePreview returns where n windows would go, for the UI preview.
func (a *App) ArrangePreview(n int, mode string) ArrangeLayout {
	a.arrangeHooks()
	return arrangeLayout(n, mode, arrangeScreen())
}

// StartSync replays the master profile's actions in the followers' browsers.
func (a *App) StartSync(master string, followers []string) (SyncerState, error) {
	a.arrangeHooks()
	wsOf := func(id string) (string, bool) {
		if rs, ok := runReg.get(id); ok && strings.HasPrefix(rs.WSURL, "ws") {
			return rs.WSURL, true
		}
		return "", false
	}
	return groupSync.Start(master, followers, wsOf, a.profileName)
}

// StopSync stops group control.
func (a *App) StopSync() SyncerState {
	a.arrangeHooks()
	groupSync.Stop("已手动停止")
	return groupSync.State()
}

// SyncerState returns the current group-control status.
func (a *App) SyncerState() SyncerState { return groupSync.State() }
