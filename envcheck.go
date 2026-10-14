package main

// 环境自检 / 一键修复: checks a profile's launch config against its current exit
// IP and its own internal consistency, and repairs the items that can be fixed
// in the profile config (with a per-profile repair log that can be rolled back).

import (
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"math"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"
)

// EnvItem is one line of the self-check.
type EnvItem struct {
	Key      string `json:"key"`      // exit | language | timezone | geo | webrtc | webdriver | fonts | seed | ua | clock
	Label    string `json:"label"`    // 中文名称
	Status   string `json:"status"`   // pass | warn | fail | unknown
	Actual   string `json:"actual"`   // what the profile has / what was read
	Expected string `json:"expected"` // what it should be
	Detail   string `json:"detail"`   // explanation / guidance
	Fixable  bool   `json:"fixable"`  // RepairEnvironment can fix it
	FixHint  string `json:"fixHint"`  // what the fix will do
}

// EnvReport is the result of CheckEnvironment.
type EnvReport struct {
	ProfileID   string      `json:"profileId"`
	ProfileName string      `json:"profileName"`
	Kernel      string      `json:"kernel"` // custom | chromium
	CheckedAt   int64       `json:"checkedAt"`
	Exit        GeoMatch    `json:"exit"`
	Items       []EnvItem   `json:"items"`
	Pass        int         `json:"pass"`
	Warn        int         `json:"warn"`
	Fail        int         `json:"fail"`
	Fixable     []string    `json:"fixable"` // keys of non-passing items that can be repaired
	Clock       ClockReport `json:"clock"`
}

// envInput is everything the report is built from, so tests can fake it.
type envInput struct {
	Profile  Profile
	Proxy    *Proxy // nil = direct connection
	Exit     GeoMatch
	Chromium bool
	Clock    ClockReport
	Now      time.Time
}

// fixableKeys: items RepairEnvironment knows how to change in the config.
var fixableKeys = map[string]bool{"language": true, "timezone": true, "geo": true, "webrtc": true, "seed": true, "fonts": true, "webdriver": true}

func profileProxy(s *Store, p *Profile) *Proxy {
	if p.InlineProxy != nil && p.InlineProxy.Host != "" && p.InlineProxy.Type != "none" {
		return p.InlineProxy
	}
	if p.ProxyID != "" {
		if pr, ok := s.GetProxy(p.ProxyID); ok && pr.Type != "none" {
			return pr
		}
	}
	return nil
}

// CheckEnvironment runs the self-check for one profile. The exit-IP probe and
// the clock check run in parallel.
func (s *Store) CheckEnvironment(id string) (EnvReport, error) {
	p, ok := s.GetProfile(id)
	if !ok {
		return EnvReport{}, fmt.Errorf("环境不存在")
	}
	px := profileProxy(s, p)
	in := envInput{Profile: *p, Proxy: px, Chromium: p.useChromium(), Now: time.Now()}
	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); in.Exit = s.liveExit(px, 60*time.Second) }()
	go func() { defer wg.Done(); in.Clock = CheckClock(nil) }()
	wg.Wait()
	return buildEnvReport(in), nil
}

func buildEnvReport(in envInput) EnvReport {
	p, fp, m := in.Profile, in.Profile.Fingerprint, in.Exit
	r := EnvReport{ProfileID: p.ID, ProfileName: p.Name, Kernel: "custom", CheckedAt: in.Now.Unix(), Exit: m, Clock: in.Clock}
	if in.Chromium {
		r.Kernel = "chromium"
	}
	add := func(it EnvItem) { r.Items = append(r.Items, it) }
	geoKnown := m.OK && m.Country != ""

	add(checkExit(in))
	add(checkLanguage(fp, m, geoKnown))
	add(checkTimezone(fp, m, in.Now))
	add(checkGeo(fp, m))
	add(checkWebRTC(fp, in.Proxy != nil))
	add(checkWebdriver(fp, in.Chromium))
	add(checkFonts(fp))
	add(checkSeed(fp))
	add(checkUA(fp))
	add(clockItem(in.Clock))

	for _, it := range r.Items {
		switch it.Status {
		case "pass":
			r.Pass++
		case "warn":
			r.Warn++
		case "fail":
			r.Fail++
		}
		if it.Fixable && it.Status != "pass" {
			r.Fixable = append(r.Fixable, it.Key)
		}
	}
	return r
}

func exitText(m GeoMatch) string {
	if !m.OK {
		return "未知"
	}
	parts := []string{m.IP}
	loc := strings.TrimSpace(strings.Join(nonEmpty(m.Country, m.Region, m.City), " · "))
	if loc != "" {
		parts = append(parts, loc)
	}
	return strings.Join(parts, "  ")
}

func nonEmpty(xs ...string) []string {
	out := []string{}
	for _, x := range xs {
		if strings.TrimSpace(x) != "" {
			out = append(out, x)
		}
	}
	return out
}

func checkExit(in envInput) EnvItem {
	m := in.Exit
	it := EnvItem{Key: "exit", Label: "出口 IP / 国家", Actual: exitText(m)}
	if in.Proxy != nil {
		it.Expected = "代理出口可连通"
		if in.Proxy.Country != "" {
			it.Expected = "代理出口 · " + in.Proxy.Country
		}
	} else {
		it.Expected = "本机网络（未配置代理）"
	}
	switch {
	case !m.OK:
		it.Status, it.Detail = "fail", "出口检测失败："+m.Error+"。时区/语言无法跟随 IP，请检查代理是否可用。"
	case m.Stale:
		it.Status, it.Detail = "warn", "本次检测失败（"+m.Error+"），显示的是上次成功的结果。"
	case m.Direct:
		it.Status, it.Detail = "warn", "未配置代理，网站看到的是本机真实 IP。"
	case in.Proxy != nil && in.Proxy.Country != "" && !strings.EqualFold(in.Proxy.Country, m.Country):
		it.Status, it.Detail = "warn", "出口国家与代理上次记录（"+in.Proxy.Country+"）不同，代理可能已轮换。"
	default:
		it.Status, it.Detail = "pass", strings.TrimSpace(m.ISP+" "+ipTypeCN(m.IPType))
	}
	return it
}

func ipTypeCN(t string) string {
	switch t {
	case "residential":
		return "住宅 IP"
	case "hosting":
		return "机房 IP"
	case "mobile":
		return "移动 IP"
	}
	return ""
}

func langBase(l string) string {
	l = strings.ToLower(strings.TrimSpace(l))
	if i := strings.IndexAny(l, "-_"); i > 0 {
		return l[:i]
	}
	return l
}

func checkLanguage(fp Fingerprint, m GeoMatch, known bool) EnvItem {
	it := EnvItem{Key: "language", Label: "语言", Actual: strings.Join(fp.Languages, ", "), FixHint: "改为跟随 IP，并写入出口国家的语言"}
	if it.Actual == "" {
		it.Actual = "未设置"
	}
	if known {
		it.Expected = strings.Join(m.Languages, ", ")
	} else {
		it.Expected = "与出口国家一致"
	}
	switch {
	case fp.LanguageFromIP && known:
		it.Status, it.Actual, it.Detail = "pass", "跟随 IP → "+it.Expected, "启动时按出口 IP 自动设置"
	case fp.LanguageFromIP:
		it.Status, it.Actual, it.Detail = "unknown", "跟随 IP", "出口未知，暂时无法确认启动后的语言"
	case !known:
		it.Status, it.Detail, it.Fixable = "unknown", "出口未知，无法比对", false
	case len(fp.Languages) == 0:
		it.Status, it.Detail, it.Fixable = "fail", "未设置语言，将使用系统语言", true
	case strings.EqualFold(fp.Languages[0], m.Languages[0]):
		it.Status, it.Detail = "pass", "首选语言与出口国家一致"
	case langBase(fp.Languages[0]) == langBase(m.Languages[0]):
		it.Status, it.Detail, it.Fixable = "warn", "语种一致，但地区变体不同（"+fp.Languages[0]+" / "+m.Languages[0]+"）", true
	default:
		inList := false
		for _, l := range m.Languages {
			if langBase(l) == langBase(fp.Languages[0]) {
				inList = true
			}
		}
		if inList {
			it.Status, it.Detail, it.Fixable = "warn", "首选语言在该国家常见，但不是主要语言", true
		} else {
			it.Status, it.Detail, it.Fixable = "fail", "首选语言与出口国家不符，容易被识别", true
		}
	}
	return it
}

func tzOffset(name string, at time.Time) (int, bool) {
	if name == "" {
		return 0, false
	}
	loc, err := time.LoadLocation(name)
	if err != nil {
		return 0, false
	}
	_, off := at.In(loc).Zone()
	return off, true
}

func fmtOffset(off int) string {
	sign := "+"
	if off < 0 {
		sign, off = "-", -off
	}
	return fmt.Sprintf("UTC%s%02d:%02d", sign, off/3600, off%3600/60)
}

func checkTimezone(fp Fingerprint, m GeoMatch, now time.Time) EnvItem {
	it := EnvItem{Key: "timezone", Label: "时区", Actual: fp.Timezone, FixHint: "改为跟随 IP，并写入出口 IP 的时区"}
	if it.Actual == "" {
		it.Actual = "未设置"
	}
	known := m.OK && m.Timezone != ""
	it.Expected = "与出口 IP 一致"
	if known {
		it.Expected = m.Timezone
		if off, ok := tzOffset(m.Timezone, now); ok {
			it.Expected += "（" + fmtOffset(off) + "）"
		}
	}
	switch {
	case fp.TimezoneFromIP && known:
		it.Status, it.Actual, it.Detail = "pass", "跟随 IP → "+m.Timezone, "启动时按出口 IP 自动设置"
	case fp.TimezoneFromIP:
		it.Status, it.Actual, it.Detail = "unknown", "跟随 IP", "出口未知，暂时无法确认启动后的时区"
	case !known:
		it.Status, it.Detail = "unknown", "出口未知，无法比对"
	case fp.Timezone == "":
		it.Status, it.Detail, it.Fixable = "fail", "未设置时区，将暴露本机时区", true
	case fp.Timezone == m.Timezone:
		it.Status, it.Detail = "pass", "与出口 IP 一致"
	default:
		a, ok1 := tzOffset(fp.Timezone, now)
		b, ok2 := tzOffset(m.Timezone, now)
		if !ok1 {
			it.Status, it.Detail, it.Fixable = "fail", "无法识别的时区名", true
		} else if ok2 && a == b {
			it.Status, it.Detail, it.Fixable = "warn", "UTC 偏移相同（"+fmtOffset(a)+"），但时区名不同，夏令时切换日可能露馅", true
		} else {
			it.Status, it.Detail, it.Fixable = "fail", "与出口 IP 时区不符（"+fmtOffset(a)+"），网站可比对 JS 时间与 IP", true
		}
	}
	return it
}

func haversineKm(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371.0
	r := math.Pi / 180
	dlat, dlon := (lat2-lat1)*r, (lon2-lon1)*r
	a := math.Sin(dlat/2)*math.Sin(dlat/2) + math.Cos(lat1*r)*math.Cos(lat2*r)*math.Sin(dlon/2)*math.Sin(dlon/2)
	return 2 * R * math.Asin(math.Sqrt(a))
}

func checkGeo(fp Fingerprint, m GeoMatch) EnvItem {
	it := EnvItem{Key: "geo", Label: "地理位置", FixHint: "改为跟随 IP，并写入出口 IP 的坐标"}
	if fp.Latitude != 0 || fp.Longitude != 0 {
		it.Actual = fmt.Sprintf("%.4f, %.4f", fp.Latitude, fp.Longitude)
	} else {
		it.Actual = "未设置"
	}
	known := m.OK && (m.Lat != 0 || m.Lon != 0)
	it.Expected = "出口 IP 附近"
	if known {
		it.Expected = fmt.Sprintf("%.4f, %.4f", m.Lat, m.Lon)
		if m.City != "" {
			it.Expected += "（" + m.City + "）"
		}
	}
	switch {
	case fp.GeoFromIP && known:
		it.Status, it.Actual, it.Detail = "pass", "跟随 IP → "+it.Expected, "启动时按出口 IP 自动设置"
	case fp.GeoFromIP:
		it.Status, it.Actual, it.Detail = "unknown", "跟随 IP", "出口未知，暂时无法确认"
	case !known:
		it.Status, it.Detail = "unknown", "出口未知，无法比对"
	case fp.Latitude == 0 && fp.Longitude == 0:
		it.Status, it.Detail, it.Fixable = "warn", "未设置坐标；网站请求定位时会拿到真实位置或被拒绝", true
	default:
		d := haversineKm(fp.Latitude, fp.Longitude, m.Lat, m.Lon)
		switch {
		case d <= 300:
			it.Status, it.Detail = "pass", fmt.Sprintf("距出口约 %.0f km", d)
		case d <= 1500:
			it.Status, it.Detail, it.Fixable = "warn", fmt.Sprintf("距出口约 %.0f km", d), true
		default:
			it.Status, it.Detail, it.Fixable = "fail", fmt.Sprintf("距出口约 %.0f km，与 IP 位置明显不符", d), true
		}
	}
	return it
}

func webrtcCN(v string) string {
	switch v {
	case "proxy":
		return "替换为代理 IP"
	case "disabled":
		return "禁用 UDP（仅走代理）"
	case "real":
		return "真实"
	}
	return "未设置"
}

func checkWebRTC(fp Fingerprint, hasProxy bool) EnvItem {
	it := EnvItem{Key: "webrtc", Label: "WebRTC 防泄漏", Actual: webrtcCN(fp.WebRTC), FixHint: "改为「替换为代理 IP」"}
	if hasProxy {
		it.Expected = "替换为代理 IP / 禁用 UDP"
	} else {
		it.Expected = "任意（未使用代理）"
	}
	switch fp.WebRTC {
	case "proxy", "disabled":
		it.Status, it.Detail = "pass", "WebRTC 不会暴露本机真实 IP"
	case "real":
		if hasProxy {
			it.Status, it.Detail, it.Fixable = "fail", "使用代理时 WebRTC 为真实模式，会泄漏本机真实 IP", true
		} else {
			it.Status, it.Detail = "pass", "未使用代理，真实模式无泄漏风险"
		}
	default:
		it.Status, it.Detail, it.Fixable = "warn", "未设置 WebRTC 策略，按内核默认处理", true
	}
	return it
}

var automationFlag = regexp.MustCompile(`(?i)--(enable-automation|remote-debugging-pipe)\b\S*|--disable-blink-features=\S*AutomationControlled\S*`)

func checkWebdriver(fp Fingerprint, chromium bool) EnvItem {
	it := EnvItem{Key: "webdriver", Label: "navigator.webdriver", Expected: "false", FixHint: "从启动参数中移除自动化参数"}
	bad := automationFlag.FindAllString(fp.StartupParameters, -1)
	hasAuto := false
	for _, b := range bad {
		if strings.Contains(strings.ToLower(b), "enable-automation") {
			hasAuto = true
		}
	}
	switch {
	case hasAuto:
		it.Status, it.Actual, it.Fixable = "fail", "true（启动参数含 --enable-automation）", true
		it.Detail = "自定义启动参数会打开自动化标记，网站可直接识别"
	case chromium:
		it.Status, it.Actual, it.Detail = "pass", "false（页面脚本覆盖）", "开源内核在每个页面注入前固定为 false（按配置判断，未实时读取）"
	default:
		it.Status, it.Actual, it.Detail = "pass", "false（内核原生）", "定制内核原生隐藏自动化标记（按配置判断，未实时读取）"
	}
	return it
}

// fontRange: plausible installed-font counts per OS for a desktop profile.
func fontRange(os string) (int, int, []string, bool) {
	switch os {
	case "windows":
		return 25, 120, winFonts, true
	case "macos":
		return 25, 150, macFonts, true
	case "linux":
		return 6, 80, linuxFonts, true
	}
	return 0, 0, nil, false
}

// foreignFonts: tell-tale fonts of the other desktop systems.
var foreignFonts = map[string][]string{
	"windows": {"PingFang SC", "Helvetica Neue", "Menlo", "Monaco", "SF Pro", "Lucida Grande", "DejaVu Sans", "Ubuntu"},
	"macos":   {"Segoe UI", "Microsoft YaHei", "Calibri", "Consolas", "SimSun", "DejaVu Sans", "Ubuntu"},
	"linux":   {"Segoe UI", "Microsoft YaHei", "Calibri", "PingFang SC", "Helvetica Neue", "SF Pro"},
}

func checkFonts(fp Fingerprint) EnvItem {
	it := EnvItem{Key: "fonts", Label: "字体数量", Actual: fmt.Sprintf("%d 个", len(fp.Fonts))}
	lo, hi, pool, desktop := fontRange(fp.OS)
	if !desktop || (fp.DeviceType != "" && fp.DeviceType != "desktop" && fp.DeviceType != "laptop") {
		it.Expected, it.Status, it.Detail = "按设备类型", "pass", "移动/电视设备不检测字体列表"
		return it
	}
	it.Expected = fmt.Sprintf("%d–%d 个，且属于 %s 系统", lo, hi, osCN(fp.OS))
	it.FixHint = "按 " + osCN(fp.OS) + " 字体库重新生成字体列表"
	var foreign []string
	have := map[string]bool{}
	for _, f := range fp.Fonts {
		have[f] = true
	}
	for _, f := range foreignFonts[fp.OS] {
		if have[f] {
			foreign = append(foreign, f)
		}
	}
	n := len(fp.Fonts)
	switch {
	case n == 0:
		it.Status, it.Detail, it.Fixable = "warn", "未设置字体列表，网站会看到本机真实字体", pool != nil
	case len(foreign) > 0:
		it.Status, it.Detail, it.Fixable = "fail", "含其他系统的字体："+strings.Join(foreign, "、"), true
	case n < lo || n > hi:
		it.Status, it.Detail, it.Fixable = "warn", "字体数量不像真实设备", true
	default:
		it.Status, it.Detail = "pass", "数量合理"
	}
	return it
}

func osCN(os string) string {
	switch os {
	case "windows":
		return "Windows"
	case "macos":
		return "macOS"
	case "linux":
		return "Linux"
	case "android":
		return "Android"
	case "ios":
		return "iOS"
	}
	return os
}

func checkSeed(fp Fingerprint) EnvItem {
	it := EnvItem{Key: "seed", Label: "Canvas / WebGL 噪声种子", Expected: "已开启噪声且种子非 0", FixHint: "生成新的随机噪声种子"}
	cm, wm := fpMode(fp.CanvasMode, fp.CanvasNoise), fpMode(fp.WebGLImageMode, fp.WebGLNoise)
	modes := fmt.Sprintf("Canvas %s · WebGL %s", modeCN(cm), modeCN(wm))
	anyNoise := cm == "noise" || wm == "noise"
	switch {
	case fp.RandomFingerprintOnLaunch:
		it.Status, it.Actual, it.Detail = "pass", modes+" · 每次启动随机", "每次启动使用新的随机种子"
	case !anyNoise:
		it.Status, it.Actual, it.Detail = "warn", modes, "未开启噪声，Canvas/WebGL 指纹为真实值，多个环境可能相同（修复不会改变噪声模式，请在编辑里开启）"
	case fp.NoiseSeed == 0:
		it.Status, it.Actual, it.Fixable = "warn", modes+" · 种子 0", true
		it.Detail = "种子为 0（仍会按环境 ID 派生），建议生成独立种子"
	default:
		it.Status, it.Actual, it.Detail = "pass", fmt.Sprintf("%s · 种子 %d", modes, fp.NoiseSeed), "已配置独立噪声种子"
	}
	return it
}

func modeCN(m string) string {
	switch m {
	case "noise":
		return "噪声"
	case "off":
		return "关闭"
	}
	return "真实"
}


// uaOS returns the OS a User-Agent string claims.
func uaOS(ua string) string {
	switch {
	case strings.Contains(ua, "Android"):
		return "android"
	case strings.Contains(ua, "iPhone") || strings.Contains(ua, "iPad"):
		return "ios"
	case strings.Contains(ua, "CrOS"):
		return "chromeos"
	case strings.Contains(ua, "Windows"):
		return "windows"
	case strings.Contains(ua, "Macintosh"):
		return "macos"
	case strings.Contains(ua, "Tizen"):
		return "tizen"
	case strings.Contains(ua, "Web0S") || strings.Contains(ua, "WebOS"):
		return "webos"
	case strings.Contains(ua, "Linux") || strings.Contains(ua, "X11"):
		return "linux"
	}
	return ""
}

func platformOK(os, platform string) bool {
	p := strings.ToLower(platform)
	switch os {
	case "windows":
		return p == "win32"
	case "macos":
		return p == "macintel"
	case "linux", "chromeos", "tizen", "webos", "android":
		return strings.HasPrefix(p, "linux")
	case "ios":
		return p == "iphone" || p == "ipad"
	}
	return true
}

func checkUA(fp Fingerprint) EnvItem {
	it := EnvItem{Key: "ua", Label: "UA / 平台一致性", Actual: fp.UserAgent}
	claimed := uaOS(fp.UserAgent)
	it.Expected = osCN(fp.OS) + " · navigator.platform " + fp.Platform
	var probs []string
	if fp.UserAgent == "" {
		probs = append(probs, "未设置 User-Agent")
	} else {
		if claimed != "" && claimed != fp.OS {
			probs = append(probs, "UA 声称 "+osCN(claimed)+"，但系统为 "+osCN(fp.OS))
		}
		if !platformOK(fp.OS, fp.Platform) {
			probs = append(probs, "navigator.platform「"+fp.Platform+"」与 "+osCN(fp.OS)+" 不符")
		}
		if mm := uaMajorRe.FindStringSubmatch(fp.UserAgent); len(mm) == 2 && fp.BrowserVersion != "" {
			bv := strings.SplitN(fp.BrowserVersion, ".", 2)[0]
			if bv != mm[1] {
				probs = append(probs, "UA 中 Chrome "+mm[1]+" 与浏览器版本 "+bv+" 不一致")
			}
		}
	}
	r := strings.ToLower(fp.WebGLRenderer)
	if fp.OS == "windows" && (strings.Contains(r, "metal") || strings.Contains(r, "apple m")) {
		probs = append(probs, "WebGL 渲染器为 Apple GPU，与 Windows 不符")
	}
	if fp.OS == "macos" && strings.Contains(r, "direct3d") {
		probs = append(probs, "WebGL 渲染器为 Direct3D，与 macOS 不符")
	}
	if len(probs) == 0 {
		it.Status, it.Detail = "pass", "UA、平台、WebGL 渲染器相互一致"
	} else {
		it.Status, it.Detail = "fail", strings.Join(probs, "；")+"。无法自动修复，请在编辑里重新生成指纹。"
	}
	return it
}

// ---------- system clock ----------

// ClockReport compares this computer's clock to an HTTPS server's Date header.
type ClockReport struct {
	Status    string `json:"status"`    // pass | fail | unknown (尚未确认)
	SkewSecs  int64  `json:"skewSecs"`  // local - server; positive = local clock is ahead
	Source    string `json:"source"`    // reference URL
	LocalTime int64  `json:"localTime"` // unix seconds when checked
	Server    int64  `json:"serverTime"`
	Error     string `json:"error"`
	Detail    string `json:"detail"`
}

const clockLimit = 120 // seconds (2 minutes)

var clockSources = []string{"https://www.cloudflare.com/", "https://www.google.com/generate_204", "https://www.apple.com/"}

// clockProbe fetches a reference time; returns server time, the local time at
// the midpoint of the request and the source.
type clockProbe func() (server, local time.Time, source string, err error)

func httpDateProbe() (time.Time, time.Time, string, error) {
	cl := &http.Client{Timeout: 6 * time.Second}
	var lastErr error
	for _, u := range clockSources {
		req, _ := http.NewRequest("HEAD", u, nil)
		t0 := time.Now()
		resp, err := cl.Do(req)
		t1 := time.Now()
		if err != nil {
			lastErr = err
			continue
		}
		resp.Body.Close()
		d := resp.Header.Get("Date")
		st, err := parseHTTPDate(d)
		if err != nil {
			lastErr = fmt.Errorf("%s 没有有效的 Date 头", u)
			continue
		}
		return st, t0.Add(t1.Sub(t0) / 2), u, nil
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("没有可用的参考服务器")
	}
	return time.Time{}, time.Time{}, "", lastErr
}

// CheckClock checks the system clock. A nil probe uses the HTTPS Date header.
// Never changes the system time.
func CheckClock(probe clockProbe) ClockReport {
	if probe == nil {
		probe = httpDateProbe
	}
	srv, local, src, err := probe()
	return classifyClock(srv, local, src, err)
}

func classifyClock(srv, local time.Time, src string, err error) ClockReport {
	r := ClockReport{LocalTime: local.Unix(), Source: src}
	if err != nil || srv.IsZero() {
		r.Status, r.LocalTime = "unknown", time.Now().Unix()
		if err != nil {
			r.Error = err.Error()
		}
		r.Detail = "尚未确认：无法连接参考服务器，稍后重试"
		return r
	}
	r.Server = srv.Unix()
	// Date has 1 s resolution; round the skew to whole seconds.
	r.SkewSecs = int64(math.Round(local.Sub(srv).Seconds()))
	abs := r.SkewSecs
	if abs < 0 {
		abs = -abs
	}
	if abs > clockLimit {
		r.Status = "fail"
		dir := "快"
		if r.SkewSecs < 0 {
			dir = "慢"
		}
		r.Detail = fmt.Sprintf("系统时钟比标准时间%s %s，超过 2 分钟。请在系统设置里开启「自动设置时间」并立即同步（本程序不会修改系统时间）。", dir, fmtSkew(abs))
	} else {
		r.Status = "pass"
		r.Detail = fmt.Sprintf("偏差 %s，在 2 分钟以内", fmtSkew(abs))
	}
	return r
}

func fmtSkew(s int64) string {
	if s < 60 {
		return fmt.Sprintf("%d 秒", s)
	}
	if s < 3600 {
		return fmt.Sprintf("%d 分 %d 秒", s/60, s%60)
	}
	return fmt.Sprintf("%d 小时 %d 分", s/3600, s%3600/60)
}

func clockItem(c ClockReport) EnvItem {
	it := EnvItem{Key: "clock", Label: "系统时钟", Expected: "偏差 ≤ 2 分钟", Detail: c.Detail}
	switch c.Status {
	case "pass", "fail":
		it.Status = c.Status
		it.Actual = fmt.Sprintf("%+d 秒", c.SkewSecs)
	default:
		it.Status, it.Actual = "unknown", "尚未确认"
	}
	if c.Status == "fail" {
		it.FixHint = "系统级设置，只能手动修复"
	}
	return it
}

func randomSeed() int64 {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return int64(binary.LittleEndian.Uint64(b[:])%2147483646) + 1
}

func parseHTTPDate(d string) (time.Time, error) { return http.ParseTime(d) }
