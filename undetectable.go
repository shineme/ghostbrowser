package main

import (
	"encoding/json"
	"fmt"
	"ghostbrowser/internal/undetectable"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// NativeProfile carries a complete fingerprint JSON snapshot and a template
// selector. Unknown native fields survive editing and export.
type NativeProfile struct {
	ExtraArgs      []string       `json:"extraArgs,omitempty"`
	Headless       bool           `json:"headless,omitempty"`
	TemplateKey    string         `json:"templateKey"`
	SourceConfigID string         `json:"sourceConfigId"`
	Data           map[string]any `json:"data"`
	ConfigMeta     map[string]any `json:"configMeta"`
}

type NativePreset struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	ConfigID string `json:"configId"`
}

func findUndetectableKit() string {
	starts := []string{}
	if exe, err := os.Executable(); err == nil {
		starts = append(starts, filepath.Dir(exe))
	}
	if cwd, err := os.Getwd(); err == nil {
		starts = append(starts, cwd)
	}
	for _, start := range starts {
		for i := 0; i < 5; i++ {
			for _, p := range []string{filepath.Join(start, "undetectable-kit"), start} {
				if _, e := os.Stat(filepath.Join(p, "runtime", "chrome", "chrome.exe")); e == nil {
					return p
				}
			}
			parent := filepath.Dir(start)
			if parent == start {
				break
			}
			start = parent
		}
	}
	return ""
}

func kitPath(st Settings) (string, error) {
	kit := st.UndetectableKit
	if kit == "" {
		kit = findUndetectableKit()
	}
	if kit == "" {
		return "", fmt.Errorf("缺少定制 Chromium 与指纹包，请在设置页点「重新检测并安装环境」")
	}
	kit, err := filepath.Abs(kit)
	if err != nil {
		return "", err
	}
	if _, err = os.Stat(filepath.Join(kit, "fingerprints", "library.json")); err != nil {
		return "", fmt.Errorf("指纹库: %w", err)
	}
	return kit, nil
}

// launchHint explains why a launch could not proceed when the environment is
// missing or still being installed, instead of failing silently.
func launchHint(err error) error {
	st := env.snapshot()
	switch st.Phase {
	case "downloading", "verifying", "extracting", "checking":
		return fmt.Errorf("%w（环境正在自动安装：%s，可在设置页查看进度）", err, st.Message)
	case "failed":
		return fmt.Errorf("%w（自动安装失败：%s；在设置页点「重新检测并安装」重试）", err, st.Error)
	default:
		return fmt.Errorf("%w（在设置页点「重新检测并安装环境」可自动下载）", err)
	}
}

func readLibrary(st Settings) ([]struct {
	ID         string         `json:"id"`
	Name       string         `json:"name"`
	Data       map[string]any `json:"data"`
	ConfigMeta map[string]any `json:"configMeta"`
}, error) {
	kit, err := kitPath(st)
	if err != nil {
		return nil, err
	}
	b, err := os.ReadFile(filepath.Join(kit, "fingerprints", "library.json"))
	if err != nil {
		return nil, err
	}
	var lib struct {
		Profiles []struct {
			ID         string         `json:"id"`
			Name       string         `json:"name"`
			Data       map[string]any `json:"data"`
			ConfigMeta map[string]any `json:"configMeta"`
		} `json:"profiles"`
	}
	if err = json.Unmarshal(b, &lib); err != nil {
		return nil, err
	}
	return lib.Profiles, nil
}

func rawMap(m map[string]any) map[string]json.RawMessage {
	out := map[string]json.RawMessage{}
	for k, v := range m {
		b, _ := json.Marshal(v)
		out[k] = b
	}
	return out
}

func anyMap(m map[string]json.RawMessage) map[string]any {
	out := map[string]any{}
	b, _ := json.Marshal(m)
	_ = json.Unmarshal(b, &out)
	return out
}

func copyNative(n *NativeProfile) *NativeProfile {
	b, _ := json.Marshal(n)
	var out NativeProfile
	_ = json.Unmarshal(b, &out)
	return &out
}

var templates = map[string]string{"windows-intel-a": "windows-intel-a.bin", "windows-intel-a2": "windows-intel-a2.bin", "windows-intel-b": "windows-intel-b.bin"}

func nativeToFingerprint(n *NativeProfile) Fingerprint {
	c, _ := (&undetectable.Spec{ProfileID: "preview", Name: "Preview", Data: rawMap(n.Data), ConfigMeta: rawMap(n.ConfigMeta)}).Config()
	if c == nil {
		return Fingerprint{}
	}
	e := c.Expected()
	return Fingerprint{OS: "windows", OSVersion: e.OSVersion, BrowserVersion: e.FullVersion, UserAgent: e.UA, Platform: e.Platform, ScreenWidth: int(e.Width), ScreenHeight: int(e.Height), PixelRatio: e.DPR, ColorDepth: int(e.ColorDepth), HardwareConc: int(e.Cores), DeviceMemory: int(e.MemoryGB), WebGLVendor: e.WebGLVendor, WebGLRenderer: e.WebGLRender, Languages: e.Languages, Timezone: e.Timezone, WebRTC: "real"}
}

func (s *Store) nativePreset(sourceID string) (*NativeProfile, Fingerprint, error) {
	lib, err := readLibrary(s.GetSettings())
	if err != nil {
		return nil, Fingerprint{}, err
	}
	for _, p := range lib {
		if sourceID == "" || p.ID == sourceID {
			n := &NativeProfile{TemplateKey: "windows-intel-a", SourceConfigID: fmt.Sprint(p.Data["configID"]), Data: p.Data, ConfigMeta: p.ConfigMeta}
			return n, nativeToFingerprint(n), nil
		}
	}
	return nil, Fingerprint{}, fmt.Errorf("指纹快照不存在: %s", sourceID)
}

func (a *App) UseUndetectableKit() (Settings, error) {
	st := a.store.GetSettings()
	if envReady() {
		out := a.SaveSettings(applyToSettings(st))
		return out, nil
	}
	kit, err := kitPath(st)
	if err != nil {
		return st, err
	}
	st.UndetectableKit = kit
	st.KernelMode = "undetectable"
	st.KernelPath = filepath.Join(kit, "runtime", "chrome", "chrome.exe")
	return a.SaveSettings(st), nil
}

func (a *App) NativePresets() ([]NativePreset, error) {
	lib, err := readLibrary(a.store.GetSettings())
	if err != nil {
		return nil, err
	}
	out := []NativePreset{}
	for _, p := range lib {
		out = append(out, NativePreset{p.ID, p.Name, fmt.Sprint(p.Data["configID"])})
	}
	return out, nil
}

func (a *App) CreateNativeProfile(sourceID, name string) (Profile, error) {
	n, fp, err := a.store.nativePreset(sourceID)
	if err != nil {
		return Profile{}, err
	}
	if name == "" {
		name = "Native Profile"
	}
	p := Profile{Name: name, Storage: "local", Fingerprint: fp, Native: n, StartURLs: []string{"about:blank"}}
	return a.store.PutProfile(p), nil
}

var uaChromeVersion = regexp.MustCompile(`Chrome/(\d+\.\d+\.\d+\.\d+)`)

func (s *Store) nativeSpec(p *Profile, st Settings) (*undetectable.Spec, error) {
	// The kit may have been installed after these paths were saved, so prefer
	// whatever is actually on disk now.
	if !fileExists(st.KernelPath) || !fileExists(filepath.Join(st.UndetectableKit, "fingerprints", "library.json")) {
		st = applyToSettings(st)
	}
	kit, err := kitPath(st)
	if err != nil {
		return nil, err
	}
	n := p.Native
	if n == nil {
		n, _, err = s.nativePreset("")
		if err != nil {
			return nil, err
		}
	}
	n = copyNative(n)
	file, ok := templates[n.TemplateKey]
	if !ok {
		return nil, fmt.Errorf("未知模板 %q", n.TemplateKey)
	}
	px, err := s.launchProxy(p) // core nodes: local mihomo listener
	if err != nil {
		return nil, err
	}
	fp, err := s.matchFP(p.Fingerprint, px)
	if err != nil {
		return nil, err
	}
	if fp.OS != "windows" {
		return nil, fmt.Errorf("当前本地原生模板只验证过 Windows，请使用 Windows 配置")
	}
	s.noteFpExpect(p, fp, "")
	version := uaChromeVersion.FindStringSubmatch(fp.UserAgent)
	if len(version) < 2 {
		return nil, fmt.Errorf("User-Agent 必须包含完整 Chrome 版本号，如 153.0.8010.36")
	}
	full := version[1]
	n.Data["UA"], n.Data["broVers"], n.Data["osVers"] = fp.UserAgent, full, fp.OSVersion
	n.Data["CPUCores"], n.Data["memory"] = fp.HardwareConc, fp.DeviceMemory
	n.Data["resConfig"], n.Data["webGL"] = fmt.Sprintf("%dx%d", fp.ScreenWidth, fp.ScreenHeight), fp.WebGLRenderer
	n.Data["langs"], n.Data["langHeader"], n.Data["tzId"] = strings.Join(fp.Languages, "#"), strings.Join(fp.Languages, ","), fp.Timezone
	ai := n.ConfigMeta["additional_info"]
	if str, ok := ai.(string); ok {
		var decoded map[string]any
		if json.Unmarshal([]byte(str), &decoded) != nil {
			return nil, fmt.Errorf("additional_info 必须是 JSON 对象或合法 JSON 字符串")
		}
		ai = decoded
	}
	info, ok := ai.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("additional_info 缺失")
	}
	uad, ok := info["userAgentDataObj"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("userAgentDataObj 缺失")
	}
	uad["uaFullVersion"] = full
	major := strings.Split(full, ".")[0]
	for _, key := range []string{"brands", "fullVersionList"} {
		if list, ok := uad[key].([]any); ok {
			for _, item := range list {
				if b, ok := item.(map[string]any); ok && b["brand"] != "Not_A Brand" {
					v := major
					if key == "fullVersionList" {
						v = full
					}
					b["version"] = v
				}
			}
		}
	}
	n.ConfigMeta["additional_info"] = info
	root, err := filepath.Abs(st.ProfilesDir)
	if err != nil {
		return nil, err
	}
	chrome := st.KernelPath
	if chrome == "" {
		chrome = filepath.Join(kit, "runtime", "chrome", "chrome.exe")
	}
	spec := &undetectable.Spec{SchemaVersion: 1, ProfileID: p.ID, Name: p.Name, ChromePath: chrome, ProfilesRoot: root, TemplatePath: filepath.Join(kit, "fingerprints", "templates", file), Data: rawMap(n.Data), ConfigMeta: rawMap(n.ConfigMeta), Runtime: undetectable.RuntimeValues{Platform: fp.Platform, PixelRatio: fp.PixelRatio, ColorDepth: int64(fp.ColorDepth), WebGLVendor: fp.WebGLVendor}, StartURLs: p.StartURLs, ExtraArgs: n.ExtraArgs, Headless: n.Headless}
	if px != nil && px.Type != "none" && px.Host != "" {
		// Credentials are kept in the URL; Spec.Start relays them through a
		// local 127.0.0.1 proxy because Chromium can't take them on the command line.
		pu := px.URL()
		if px.User == "" && px.Type == "https" {
			pu.Scheme = "https"
		}
		spec.Proxy = pu.String()
	}
	if fp.WebRTC == "disabled" {
		spec.ExtraArgs = append(spec.ExtraArgs, "--force-webrtc-ip-handling-policy=disable_non_proxied_udp")
	}
	if px != nil && px.IP != "" && fp.WebRTC == "proxy" {
		b, _ := json.Marshal(px.IP)
		spec.Data["publicRTC"] = b
	}
	return spec, spec.Validate()
}

func (s *Store) launchUndetectable(p *Profile, st Settings) (*RunState, error) {
	spec, err := s.nativeSpec(p, st)
	if err != nil {
		if !envReady() {
			return nil, launchHint(err)
		}
		return nil, err
	}
	// Extension Center folders: added at launch only, never stored in the spec export.
	spec.ExtraArgs = withExtensionArgs(append([]string(nil), spec.ExtraArgs...), extensionDirs(p))
	session, err := spec.Start(0)
	if err != nil {
		return nil, err
	}
	rs := &RunState{ProfileID: p.ID, PID: session.Cmd.Process.Pid, DebugPort: session.Port, WSURL: session.WSURL, Started: time.Now().Unix()}
	runReg.add(p.ID, rs, session.Cmd)
	runReg.setCleanup(p.ID, session.Close)
	go func() { <-session.Done; runReg.remove(p.ID) }()
	p.LastRun = time.Now().Unix()
	s.PutProfile(*p)
	if strings.TrimSpace(p.Cookies) != "" {
		if err = importCookies(rs.WSURL, p.Cookies); err != nil {
			runReg.stop(p.ID)
			return nil, fmt.Errorf("cookies: %w", err)
		}
		p.Cookies = ""
		s.PutProfile(*p)
	}
	s.afterStart(p.ID, rs.WSURL)
	return rs, nil
}

func (a *App) ExportNativeJSON(id string) (string, error) {
	p, ok := a.store.GetProfile(id)
	if !ok {
		return "", fmt.Errorf("profile not found")
	}
	spec, err := a.store.nativeSpec(p, a.store.GetSettings())
	if err != nil {
		return "", err
	}
	b, err := json.MarshalIndent(spec, "", "  ")
	return string(withDocSchema(b)), err // portable document schema (docschema.go)
}

func (a *App) ImportNativeJSON(raw string) (Profile, error) {
	if len(raw) > 10<<20 {
		return Profile{}, fmt.Errorf("JSON 超过 10 MiB")
	}
	if err := checkDocSchema([]byte(raw), "环境 JSON"); err != nil {
		return Profile{}, err
	}
	var spec undetectable.Spec
	if err := json.Unmarshal([]byte(strings.TrimPrefix(raw, "\ufeff")), &spec); err != nil {
		return Profile{}, err
	}
	if spec.SchemaVersion > 1 {
		return Profile{}, fmt.Errorf("这个环境 JSON 来自更新版本（schemaVersion %d，当前最高支持 1），请先升级 GhostBrowser 再导入", spec.SchemaVersion)
	}
	if spec.SchemaVersion != 1 {
		return Profile{}, fmt.Errorf("schemaVersion 必须是 1")
	}
	// Runtime paths come from GhostBrowser settings, never the imported JSON.
	key := "windows-intel-a"
	for k, v := range templates {
		if filepath.Base(spec.TemplatePath) == v {
			key = k
		}
	}
	n := &NativeProfile{TemplateKey: key, Data: anyMap(spec.Data), ConfigMeta: anyMap(spec.ConfigMeta), ExtraArgs: spec.ExtraArgs, Headless: spec.Headless}
	n.SourceConfigID = fmt.Sprint(n.Data["configID"])
	fp := nativeToFingerprint(n)
	if spec.Runtime.Platform != "" {
		fp.Platform = spec.Runtime.Platform
	}
	if spec.Runtime.PixelRatio > 0 {
		fp.PixelRatio = spec.Runtime.PixelRatio
	}
	if spec.Runtime.ColorDepth > 0 {
		fp.ColorDepth = int(spec.Runtime.ColorDepth)
	}
	if spec.Runtime.WebGLVendor != "" {
		fp.WebGLVendor = spec.Runtime.WebGLVendor
	}
	p := Profile{ID: newID(), Name: spec.Name, CreatedAt: time.Now().Unix(), Storage: "local", Fingerprint: fp, Native: n, StartURLs: spec.StartURLs}
	if p.Name == "" {
		p.Name = "JSON Profile"
	}
	if spec.Proxy != "" {
		px, err := ParseProxy(spec.Proxy)
		if err != nil {
			return Profile{}, err
		}
		p.InlineProxy = &px
	}
	if _, err := a.store.nativeSpec(&p, a.store.GetSettings()); err != nil {
		return Profile{}, err
	}
	return a.store.PutProfile(p), nil
}

func (a *App) NativeSnapshot(id string) (string, error) {
	rs, ok := runReg.get(id)
	if !ok {
		return "", fmt.Errorf("请先启动环境")
	}
	p, ok := a.store.GetProfile(id)
	if !ok {
		return "", fmt.Errorf("profile not found")
	}
	spec, err := a.store.nativeSpec(p, a.store.GetSettings())
	if err != nil {
		return "", err
	}
	fp, err := undetectable.ProbeNewTab(rs.DebugPort)
	if err != nil {
		return "", err
	}
	c, err := spec.Config()
	if err != nil {
		return "", err
	}
	b, err := json.MarshalIndent(map[string]any{"live": fp, "mismatches": undetectable.CompareFull(c, fp), "checkedAt": time.Now().Format(time.RFC3339)}, "", "  ")
	return string(b), err
}
