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
	globalAuto = &autoSyncer{s: a.store, busy: map[string]bool{}, emit: func(e SyncEvent) { wruntime.EventsEmit(a.ctx, "sync:event", e) }}
	runReg.onExit = func(id string) {
		wruntime.EventsEmit(a.ctx, "profile:exit", id)
		globalAuto.AfterExit(id)
	}
	a.auto = globalAuto
	api.start(a.store)
	// Server-first reconcile on start-up, off the UI thread.
	go func() { time.Sleep(800 * time.Millisecond); a.auto.Reconcile() }()
}

// ---------- profiles ----------

func (a *App) ListProfiles() []Profile { return a.store.ListProfiles() }

func (a *App) SaveProfile(p Profile) (Profile, error) {
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

// CreateProfiles creates count profiles named prefix-N with fresh fingerprints.
func (a *App) CreateProfiles(prefix, osName string, count int, proxyID, group string) []Profile {
	out := []Profile{}
	if count < 1 {
		count = 1
	}
	for i := 1; i <= count; i++ {
		name := prefix
		if count > 1 {
			name = prefix + "-" + itoa(i)
		}
		p := Profile{Name: name, Group: group, ProxyID: proxyID, Storage: "local",
			Fingerprint: GenerateFingerprint(osName, a.KernelMajor())}
		if a.store.GetSettings().KernelMode == "undetectable" && osName == "windows" {
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
		cp.Fingerprint = GenerateFingerprint(p.Fingerprint.OS, a.KernelMajor())
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
			r.Name, r.Tag, r.ChangeIP, r.Speed, r.SpeedAt = cur.Name, cur.Tag, cur.ChangeIP, cur.Speed, cur.SpeedAt
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

var verRe = regexp.MustCompile(`^(\d+)\.\d+\.\d+\.\d+$`)

// KernelMajor detects the Chrome major version from the version folder that
// sits next to chrome.exe on Windows; falls back to 141.
func (a *App) KernelMajor() string {
	if a.store.GetSettings().KernelMode == "undetectable" {
		return "153"
	}
	k := a.store.GetSettings().KernelPath
	if k != "" {
		if ents, err := os.ReadDir(filepath.Dir(k)); err == nil {
			vs := []string{}
			for _, e := range ents {
				if m := verRe.FindStringSubmatch(e.Name()); m != nil {
					vs = append(vs, m[1])
				}
			}
			sort.Strings(vs)
			if len(vs) > 0 {
				return vs[len(vs)-1]
			}
		}
	}
	return "141"
}

// ---------- cloud sync ----------

func (a *App) TestSync() error                 { return a.store.TestSync() }
func (a *App) PushProfile(id string) error     { return a.store.PushProfile(id) }
func (a *App) PullProfile(id string) error     { return a.store.PullProfile(id) }
func (a *App) ListCloud() ([]cloudMeta, error) { return a.store.ListCloud() }

func (a *App) DataDir() string { return appDataDir() }
