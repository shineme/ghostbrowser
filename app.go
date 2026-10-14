package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
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
	if err := mcpSrv.start(a.store, false); err != nil {
		println("mcp:", err.Error())
	}
	// Server-first reconcile on start-up, off the UI thread.
	go func() { time.Sleep(800 * time.Millisecond); a.auto.Reconcile(); go a.auto.WatchDeletes(); a.auto.WatchGlobal() }()
	go a.proxyHealthLoop()
}

// ---------- profiles ----------

func (a *App) ListProfiles() []Profile { return a.store.ListProfiles() }

func (a *App) SaveProfile(p Profile) (Profile, error) {
	act := "profile.edit"
	if p.ID == "" {
		act = "profile.create"
	}
	out, err := a.saveProfile(p)
	id, name := out.ID, out.Name
	if err != nil {
		id, name = p.ID, p.Name
	}
	a.store.auditNamed("ui", act, id, name, err, "")
	return out, err
}

func (a *App) saveProfile(p Profile) (Profile, error) {
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

// DeleteProfiles deletes profiles on this device only (cloud copies stay and
// are not downloaded again). See DeleteProfilesMode for cloud deletion.
func (a *App) DeleteProfiles(ids []string) {
	a.syncer().DeleteProfiles("ui", ids, "local", false)
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
		np := a.store.PutProfile(p)
		a.store.auditNamed("ui", "profile.create", np.ID, np.Name, nil, devType+"/"+osName)
		out = append(out, np)
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
	np := a.store.PutProfile(cp)
	a.store.auditNamed("ui", "profile.clone", np.ID, np.Name, nil, "源: "+p.Name)
	return np, nil
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
		if m := majorOf(kernelVersion(k)); m > 0 {
			return strconv.Itoa(m)
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
	if err := a.auto.BeforeLaunch(id); err != nil {
		a.store.auditNamed("ui", "profile.start", id, "", err, "")
		return nil, err
	}
	rs, err := a.store.Launch(id)
	a.store.auditProfile("ui", "profile.start", id, err, "")
	return rs, err
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
	a.store.auditProfile("ui", "profile.stop", id, nil, "")
	if rs, ok := runReg.get(id); ok && strings.HasPrefix(rs.WSURL, "ws") {
		if closeBrowser(rs.WSURL) == nil {
			return
		}
	}
	runReg.stop(id)
}

func (a *App) Running() []RunState { return runReg.list() }

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
	if s.Syncer == nil { // settings page loaded before the syncer dialog saved: keep its prefs
		s.Syncer = a.store.GetSettings().Syncer
	}
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
	// Numeric, not lexical: "99" must not beat "120".
	sort.Slice(vs, func(i, j int) bool { a, _ := strconv.Atoi(vs[i]); b, _ := strconv.Atoi(vs[j]); return a < b })
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

// ---- mcp/audit ----

// McpStatus reports the MCP server state plus ready-to-paste client configs.
func (a *App) McpStatus() McpStatus { return mcpSrv.status(a.store) }

// StartMcp enables the MCP HTTP server (remembered in mcp.json) on port (0 = keep current).
func (a *App) StartMcp(port int) (McpStatus, error) {
	cfg := readMcpConfig(a.store.dir)
	if port > 0 && port <= 65535 && port != cfg.Port {
		mcpSrv.stop()
		cfg.Port = port
	}
	cfg.Enabled = true
	writeMcpConfig(a.store.dir, cfg)
	err := mcpSrv.start(a.store, true)
	a.store.auditNamed("ui", "mcp.start", "", "", err, fmt.Sprintf("port %d", cfg.Port))
	return mcpSrv.status(a.store), err
}

// StopMcp stops and disables the MCP HTTP server.
func (a *App) StopMcp() McpStatus {
	cfg := readMcpConfig(a.store.dir)
	cfg.Enabled = false
	writeMcpConfig(a.store.dir, cfg)
	mcpSrv.stop()
	a.store.auditNamed("ui", "mcp.stop", "", "", nil, "")
	return mcpSrv.status(a.store)
}

func (a *App) AuditLog(f AuditFilter) AuditResult { return a.store.AuditQuery(f) }
func (a *App) ClearAuditLog()                     { a.store.ClearAudit() }

// ExportAuditLog saves the filtered log (format "csv" or "json") via a save
// dialog; returns the written path, or "" when the user cancelled.
func (a *App) ExportAuditLog(f AuditFilter, format string) (string, error) {
	if format != "json" {
		format = "csv"
	}
	f.Limit = 0
	items := a.store.AuditQuery(f).Items
	name := "ghostbrowser-audit-" + time.Now().Format("20060102-150405") + "." + format
	path, err := wruntime.SaveFileDialog(a.ctx, wruntime.SaveDialogOptions{
		Title:           "导出操作日志",
		DefaultFilename: name,
		Filters:         []wruntime.FileFilter{{DisplayName: strings.ToUpper(format) + " (*." + format + ")", Pattern: "*." + format}},
	})
	if err != nil || path == "" {
		return "", err
	}
	if err := os.WriteFile(path, auditExport(items, format), 0o644); err != nil {
		return "", err
	}
	return path, nil
}

func (a *App) ListGroups() []GroupInfo { return a.store.ListGroups() }
func (a *App) CreateGroup(name string) (GroupInfo, error) {
	g, err := a.store.CreateGroup(name)
	a.store.auditNamed("ui", "group.create", "", name, err, "")
	return g, err
}
func (a *App) ListTags() []TagInfo       { return a.store.ListTags() }
func (a *App) ListKernels() []KernelInfo { return a.store.ListKernels() }

// ---- extensions/cookies/mihomo ----

// ListExtensions returns every installed extension, newest first, with
// Profiles = number of profiles that load it.
func (a *App) ListExtensions() []Extension {
	used := map[string]int{}
	for _, p := range a.store.ListProfiles() {
		for _, id := range p.ExtensionIDs {
			used[id]++
		}
	}
	list := extReg.list()
	for i := range list {
		list[i].Profiles = used[list[i].ID]
	}
	return list
}

// AddExtensionByID downloads from the Chrome Web Store (bare ID or store URL).
func (a *App) AddExtensionByID(id string) (Extension, error) { return a.store.AddExtensionByID(id, "") }

// AddExtensionFromURL installs a .crx/.zip from an http(s) URL or a local file path.
func (a *App) AddExtensionFromURL(src string) (Extension, error) {
	return a.store.AddExtensionFromURL(src)
}

// RemoveExtension deletes the extension and unassigns it from every profile.
func (a *App) RemoveExtension(id string) error {
	for _, p := range a.store.ListProfiles() {
		keep := p.ExtensionIDs[:0:0]
		for _, x := range p.ExtensionIDs {
			if x != id {
				keep = append(keep, x)
			}
		}
		if len(keep) != len(p.ExtensionIDs) {
			p.ExtensionIDs = keep
			a.store.PutProfile(p)
		}
	}
	return extReg.remove(id)
}

// SetProfileExtensions replaces a profile's extension list (unknown IDs are
// rejected). Takes effect on the next launch.
func (a *App) SetProfileExtensions(profileID string, ids []string) error {
	p, ok := a.store.GetProfile(profileID)
	if !ok {
		return fmt.Errorf("环境不存在")
	}
	seen := map[string]bool{}
	out := []string{}
	for _, id := range ids {
		if seen[id] {
			continue
		}
		if _, ok := extReg.get(id); !ok {
			return fmt.Errorf("扩展 %s 不存在", id)
		}
		seen[id] = true
		out = append(out, id)
	}
	p.ExtensionIDs = out
	a.store.PutProfile(*p)
	return nil
}

// ExportCookies: format "json" | "netscape". Running profile = live over CDP,
// stopped = last session snapshot merged with staged cookies.
func (a *App) ExportCookies(id, format string) (string, error) {
	return a.store.ExportCookies(id, format)
}

// ImportCookies: format "json" | "netscape" | "auto".
func (a *App) ImportCookies(id, format, data string) (CookieImportResult, error) {
	return a.store.ImportCookies(id, format, data)
}

func (a *App) CookieInfo(id string) CookieInfo { return a.store.CookieInfo(id) }

func (a *App) MihomoConnect(controller, secret string) (MihomoStatus, error) {
	return MihomoConnect(controller, secret)
}

func (a *App) MihomoStatus() MihomoStatus { return GetMihomoStatus() }

func (a *App) MihomoProxies() (MihomoProxies, error) { return GetMihomoProxies() }

func (a *App) MihomoSelect(group, node string) error { return MihomoSelect(group, node) }

// MihomoTestDelay measures one node with the default URL (gstatic generate_204) and 5 s timeout.
func (a *App) MihomoTestDelay(node string) (int, error) { return MihomoTestDelay(node, "", 0) }

func (a *App) MihomoAddProxy() (Proxy, error) { return a.store.MihomoAddProxy() }

// ---- rpa ----

var (
	rpaOnce sync.Once
	rpaSvc  *rpaService
)

// rpa lazily builds the automation service (workflows.json next to data.json).
func (a *App) rpa() *rpaService {
	rpaOnce.Do(func() {
		rpaSvc = &rpaService{
			db:      newRpaStore(a.store.dir),
			dir:     a.store.dir,
			profile: a.store.GetProfile,
			session: func(id string) (string, bool) {
				rs, ok := runReg.get(id)
				if !ok {
					return "", false
				}
				return rs.WSURL, true
			},
			start: func(id string) (string, error) {
				rs, err := a.StartProfile(id)
				if err != nil {
					return "", err
				}
				return rs.WSURL, nil
			},
			stop: a.StopProfile,
			emit: func(e RpaEvent) {
				// Without a Wails context (tests, headless) EventsEmit would log.Fatal.
				if a.ctx != nil && a.ctx.Value("events") != nil {
					wruntime.EventsEmit(a.ctx, "rpa:event", e)
				}
			},
			newDriver: newCDPDriver,
			cancels:   map[string]context.CancelFunc{},
		}
	})
	return rpaSvc
}

func (a *App) ListWorkflows() []Workflow                 { return a.rpa().db.List() }
func (a *App) SaveWorkflow(w Workflow) (Workflow, error) { return a.rpa().db.Put(w) }
func (a *App) DeleteWorkflow(id string) error            { return a.rpa().db.Delete(id) }
func (a *App) ValidateWorkflow(w Workflow) []RpaIssue    { return validateWorkflow(w) }

// DuplicateWorkflow copies a workflow with fresh step IDs.
func (a *App) DuplicateWorkflow(id string) (Workflow, error) {
	w, ok := a.rpa().db.Get(id)
	if !ok {
		return Workflow{}, fmt.Errorf("流程不存在")
	}
	w.ID, w.Name, w.Steps = "", w.Name+" 副本", normaliseSteps(w.Steps, true)
	w.LastRunAt, w.LastStatus, w.LastRunID, w.RunCount = 0, "", "", 0
	return a.rpa().db.Put(w)
}

// RunWorkflow runs a workflow on the given profiles (3 in parallel, starting
// browsers that are not open) and returns the run id; progress arrives as rpa:event.
func (a *App) RunWorkflow(workflowID string, profileIDs []string) (string, error) {
	return a.rpa().Run(workflowID, profileIDs, RpaRunOptions{Concurrency: 3, AutoStart: true})
}

// RunWorkflowWith is RunWorkflow with explicit concurrency / auto-start options.
func (a *App) RunWorkflowWith(workflowID string, profileIDs []string, opt RpaRunOptions) (string, error) {
	return a.rpa().Run(workflowID, profileIDs, opt)
}

func (a *App) CancelWorkflowRun(runID string) bool     { return a.rpa().Cancel(runID) }
func (a *App) WorkflowRuns(workflowID string) []RpaRun { return a.rpa().db.Runs(workflowID) }
func (a *App) WorkflowRun(runID string) (RpaRun, error) {
	r, ok := a.rpa().db.Run(runID)
	if !ok {
		return RpaRun{}, fmt.Errorf("运行记录不存在")
	}
	return r, nil
}
func (a *App) RpaScreenshot(path string) (string, error) { return a.rpa().shotDataURL(path) }

func (a *App) ExportWorkflow(id string) (string, error) {
	w, ok := a.rpa().db.Get(id)
	if !ok {
		return "", fmt.Errorf("流程不存在")
	}
	return exportWorkflowJSON(w), nil
}

// ImportWorkflow saves a workflow from exported JSON as a new workflow.
func (a *App) ImportWorkflow(raw string) (Workflow, error) {
	w, err := parseWorkflowJSON(raw)
	if err != nil {
		return Workflow{}, err
	}
	return a.rpa().db.Put(w)
}

// SaveWorkflowFile writes the export JSON to a file the user picks ("" if cancelled).
func (a *App) SaveWorkflowFile(id string) (string, error) {
	js, err := a.ExportWorkflow(id)
	if err != nil {
		return "", err
	}
	w, _ := a.rpa().db.Get(id)
	name := regexp.MustCompile(`[\\/:*?"<>|]+`).ReplaceAllString(w.Name, "_")
	path, err := wruntime.SaveFileDialog(a.ctx, wruntime.SaveDialogOptions{Title: "导出流程", DefaultFilename: name + ".rpa.json",
		Filters: []wruntime.FileFilter{{DisplayName: "流程 JSON (*.json)", Pattern: "*.json"}}})
	if err != nil || path == "" {
		return "", err
	}
	return path, os.WriteFile(path, []byte(js), 0o644)
}

// OpenWorkflowFile lets the user pick a JSON file and returns its text ("" if cancelled).
func (a *App) OpenWorkflowFile() (string, error) {
	path, err := wruntime.OpenFileDialog(a.ctx, wruntime.OpenDialogOptions{Title: "导入流程",
		Filters: []wruntime.FileFilter{{DisplayName: "流程 JSON (*.json)", Pattern: "*.json"}}})
	if err != nil || path == "" {
		return "", err
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	if len(b) > 4<<20 {
		return "", fmt.Errorf("文件太大")
	}
	return string(b), nil
}

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

// ---- chromium compat ----

// ChromiumCompat reports an open-source kernel's version and whether it is
// supported, for the warning on the settings page. path is the (possibly
// unsaved) value of the path field; "" checks the auto-detected kernel.
func (a *App) ChromiumCompat(path string) KernelCompat {
	st := a.store.GetSettings()
	st.ChromiumPath = path
	k := chromiumKernel(st)
	if p := strings.Trim(strings.TrimSpace(path), "\""); p != "" && k != p {
		return KernelCompat{Path: p, Level: "missing", Message: "找不到这个文件，将改用自动检测到的内核"}
	}
	return kernelCompatFor(k, kernelVersion(k))
}

// ---- kernel install ----

// KernelInstallStatus returns the current one-click kernel install state.
func (a *App) KernelInstallStatus() KernelProgress { return kernelInst.snapshot() }

// KernelInstallStart downloads and installs Chromium for Windows x64 from
// source (mirror|official|custom) and makes it the open-source kernel.
// Progress arrives as "kernel:install" events.
func (a *App) KernelInstallStart(source, customURL string) (KernelProgress, error) {
	if kernelInst.emit == nil {
		kernelInst.emit = func(p KernelProgress) {
			if a.ctx != nil {
				wruntime.EventsEmit(a.ctx, "kernel:install", p)
			}
		}
	}
	st := a.store.GetSettings()
	st.KernelSource, st.KernelURL = source, strings.TrimSpace(customURL)
	a.store.PutSettings(st)
	return kernelInst.Start(kernelsDir(), source, customURL, func(path string) {
		s := a.store.GetSettings()
		s.ChromiumPath = path
		a.store.PutSettings(s)
		if a.ctx != nil {
			wruntime.EventsEmit(a.ctx, "settings:changed", s)
		}
	})
}

// KernelInstallCancel stops the running install; the partial download is kept for resume.
func (a *App) KernelInstallCancel() { kernelInst.Cancel() }

// KernelList returns kernels installed by one-click install.
func (a *App) KernelList() []InstalledKernel {
	return listKernels(kernelsDir(), chromiumKernel(a.store.GetSettings()))
}

// KernelDelete removes an installed kernel; clears chromiumPath when it pointed there.
func (a *App) KernelDelete(dir string) error {
	if err := deleteKernel(kernelsDir(), dir); err != nil {
		return err
	}
	st := a.store.GetSettings()
	if p := st.ChromiumPath; p != "" && strings.HasPrefix(strings.ToLower(filepath.Clean(p)), strings.ToLower(filepath.Clean(dir))+string(filepath.Separator)) {
		st.ChromiumPath = ""
		a.store.PutSettings(st)
	}
	return nil
}

// ---- syncer2 ----

// syncerWS returns a running profile's CDP websocket.
func syncerWS(id string) (string, bool) {
	if rs, ok := runReg.get(id); ok && strings.HasPrefix(rs.WSURL, "ws") {
		return rs.WSURL, true
	}
	return "", false
}

// GetSyncerPrefs returns the 多窗口同步器 dialog's last options and selections.
func (a *App) GetSyncerPrefs() SyncerPrefs {
	p := loadSyncerPrefs(a.store)
	if groupSync.running() == nil {
		groupSync.SetOptions(p.SyncOptions)
	}
	return p
}

// SaveSyncerPrefs stores the dialog's options and selections; the options
// also apply at once to a running syncer.
func (a *App) SaveSyncerPrefs(p SyncerPrefs) SyncerPrefs {
	a.arrangeHooks()
	p = saveSyncerPrefs(a.store, p)
	groupSync.SetOptions(p.SyncOptions)
	return p
}

// SetSyncOptions changes the per-type switches and delay range, also while running.
func (a *App) SetSyncOptions(o SyncOptions) SyncerState {
	a.arrangeHooks()
	return groupSync.SetOptions(o)
}

// SyncTabAction runs a window/tab action of the dialog on master + followers
// (open|closeCurrent|closeOthers|closeBlank|show|unify|size). Sync need not be running.
func (a *App) SyncTabAction(master string, followers []string, action, url string, newTab bool) (TabActionResult, error) {
	a.arrangeHooks()
	return syncTabAction(master, followers, action, url, newTab, syncerWS, a.profileName)
}

// NormalizeSyncURL validates an address for 批量访问网址 (adds https://).
func (a *App) NormalizeSyncURL(u string) (string, error) { return normalizeURL(u) }

// ---- rpa recorder ----

func (a *App) recEmit() {
	recordings.setEmit(func(st RecordState) {
		if a.ctx != nil && a.ctx.Value("events") != nil {
			wruntime.EventsEmit(a.ctx, "rpa:record", st)
		}
	})
}

// StartRecording opens the profile's browser (or attaches to the open one)
// and records what the user does there; steps arrive as "rpa:record" events.
func (a *App) StartRecording(profileID string, opts RecordOptions) (RecordState, error) {
	a.recEmit()
	p, ok := a.store.GetProfile(profileID)
	if !ok {
		return RecordState{}, fmt.Errorf("环境不存在或已删除")
	}
	if recordings.active() != nil {
		return RecordState{}, fmt.Errorf("已经在录制另一个环境，请先停止")
	}
	if !a.rpa().claim(profileID, "record") {
		return RecordState{}, fmt.Errorf("这个环境正在运行自动化流程，请等它结束再录制")
	}
	release := func() { a.rpa().release(profileID) }
	rs, err := a.StartProfile(profileID)
	if err != nil {
		release()
		return RecordState{}, fmt.Errorf("启动浏览器失败：%v", err)
	}
	if !strings.HasPrefix(rs.WSURL, "ws") {
		release()
		return RecordState{}, fmt.Errorf("这个环境的内核没有调试接口（CDP），无法录制")
	}
	st, err := recordings.Start(profileID, p.Name, rs.WSURL, opts, release)
	if err != nil {
		release()
	}
	return st, err
}

// StopRecording ends the recording and returns every captured step.
func (a *App) StopRecording() RecordState {
	if r := recordings.active(); r != nil {
		return r.Stop()
	}
	return recordings.State()
}

func (a *App) PauseRecording() RecordState {
	if r := recordings.active(); r != nil {
		return r.Pause(true)
	}
	return recordings.State()
}

func (a *App) ResumeRecording() RecordState {
	if r := recordings.active(); r != nil {
		return r.Pause(false)
	}
	return recordings.State()
}

// RecordPickMode makes the next click in the recorded tab add a step instead
// of acting: "extract" (提取文本), "assert" (等待元素) or "none" to cancel.
func (a *App) RecordPickMode(mode string) RecordState {
	if r := recordings.active(); r != nil {
		return r.Pick(mode)
	}
	return recordings.State()
}

// RecordingState returns the active (or last finished) recording.
func (a *App) RecordingState() RecordState { return recordings.State() }

// RecordRemoveStep deletes a captured step while recording.
func (a *App) RecordRemoveStep(stepID string) RecordState {
	if r := recordings.active(); r != nil {
		return r.edit(func(c *recCore) { c.remove(stepID) })
	}
	return recordings.State()
}

// RecordUpdateStep replaces a captured step (edited in the recording panel).
func (a *App) RecordUpdateStep(step RpaStep) RecordState {
	if r := recordings.active(); r != nil {
		return r.edit(func(c *recCore) { c.update(step) })
	}
	return recordings.State()
}

// FinalizeRecording applies the review toggles to the reviewed steps.
func (a *App) FinalizeRecording(steps []RpaStep, opts RecordFinalizeOptions) RecordFinalized {
	var secrets map[string]string
	recordings.mu.Lock()
	if r := recordings.last; r != nil {
		r.mu.Lock()
		secrets = map[string]string{}
		for k, v := range r.core.secrets {
			secrets[k] = v
		}
		r.mu.Unlock()
	}
	recordings.mu.Unlock()
	return finalizeRecording(steps, opts, secrets)
}

// ---- cloud delete ----

func (a *App) syncer() *autoSyncer {
	if a.auto != nil {
		return a.auto
	}
	return &autoSyncer{s: a.store, busy: map[string]bool{}}
}

// DeleteProfilesMode deletes profiles locally ("local") or on every device
// ("cloud"). force deletes even when another device holds the sync lease.
func (a *App) DeleteProfilesMode(ids []string, mode string, force bool) []DeleteResult {
	if mode != "cloud" {
		mode = "local"
	}
	st := a.store.GetSettings()
	if st.DeleteMode != mode {
		st.DeleteMode = mode
		a.store.PutSettings(st)
	}
	return a.syncer().DeleteProfiles("ui", ids, mode, force)
}

// CloudOnly lists cloud profiles without a local copy (including ones deleted
// here with "only local" and queued cloud deletes).
func (a *App) CloudOnly() CloudOnlyList { return a.store.CloudOnly() }

// RestoreLocalDeleted downloads a profile that was deleted only on this device.
func (a *App) RestoreLocalDeleted(id string) error { return a.store.RestoreLocalDeleted(id) }

// PendingCloudDeletes is the number of cloud deletes waiting for the server.
func (a *App) PendingCloudDeletes() int { return len(a.store.pendingDeletes()) }

// RetryCloudDeletes sends queued cloud deletes now; returns how many are left.
func (a *App) RetryCloudDeletes() int { return a.syncer().RetryPendingDeletes() }
