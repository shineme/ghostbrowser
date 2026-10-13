package main

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	wruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx   context.Context
	store *Store
}

func NewApp() *App { return &App{} }

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	a.store = NewStore()
	runReg.onExit = func(id string) { wruntime.EventsEmit(a.ctx, "profile:exit", id) }
	api.start(a.store)
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

// DevicePresets returns GhostBrowser's own device library for one OS ("" = all).
func (a *App) DevicePresets(osName string) []DevicePreset { return DevicePresetsFor(osName) }

// ---------- run ----------

func (a *App) StartProfile(id string) (*RunState, error) {
	if _, ok := runReg.get(id); ok {
		rs, _ := runReg.get(id)
		return rs, nil
	}
	return a.store.Launch(id)
}

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
	for _, line := range strings.Split(text, "\n") {
		if p, err := ParseProxy(line); err == nil {
			out = append(out, a.store.PutProxy(p))
		}
	}
	return out
}

func (a *App) CheckProxy(id string) Proxy {
	p, ok := a.store.GetProxy(id)
	if !ok {
		return Proxy{}
	}
	return a.store.PutProxy(CheckProxy(*p))
}

func (a *App) CheckProxyDraft(p Proxy) Proxy { return CheckProxy(p) }

func (a *App) CheckAllProxies() []Proxy {
	list := a.store.ListProxies()
	var wg sync.WaitGroup
	sem := make(chan struct{}, 8)
	for _, p := range list {
		wg.Add(1)
		go func(p Proxy) {
			defer wg.Done()
			sem <- struct{}{}
			a.store.PutProxy(CheckProxy(p))
			<-sem
		}(p)
	}
	wg.Wait()
	return a.store.ListProxies()
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
