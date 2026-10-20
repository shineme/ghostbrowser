package main

// 启动时打开指纹检测页: after a UI launch, open a fingerprint check site as
// one extra tab behind the profile's own start pages (or its restored
// session), focused. A global setting (synced with the other settings) and a
// per-profile override decide whether and which page; "仅首次启动" opens it
// once per profile. Launches from the local API, MCP, RPA and the recorder
// never open it, so automation always sees exactly the tabs it expects.

import (
	"crypto/rand"
	"encoding/hex"
	"strings"
	"sync"
	"time"
)

// LaunchOpts are per-launch flags.
type LaunchOpts struct {
	// UI: started by the user from the app window (start button, batch start,
	// restart, site shortcut). Only UI launches may open the check page.
	UI bool
}

// FpCheckSite is one entry of the check page list.
type FpCheckSite struct {
	Key  string `json:"key"`
	Name string `json:"name"`
	URL  string `json:"url"`
	Note string `json:"note"`
}

const fpCheckBuiltin = "builtin"
const fpCheckCustom = "custom"

// fpCheckSites: every external URL answered 200 when added (2026-10-20).
var fpCheckSites = []FpCheckSite{
	{"browserscan", "BrowserScan", "https://www.browserscan.net/", "综合指纹 + 机器人检测"},
	{"pixelscan", "Pixelscan", "https://pixelscan.net/", "一致性检测（IP / 时区 / 语言 / UA）"},
	{"creepjs", "CreepJS", "https://abrahamjuliot.github.io/creepjs/", "严格的篡改与谎言检测"},
	{"browserleaks", "BrowserLeaks", "https://browserleaks.com/", "分项检测：WebRTC、Canvas、字体…"},
	{"iphey", "iphey", "https://iphey.com/", "IP 与指纹可信度"},
	{"whoer", "whoer", "https://whoer.net/", "IP 匿名度、DNS、时区"},
	{fpCheckBuiltin, "内置自检", "", "本机页面：对比浏览器实际暴露的值与环境配置"},
}

// FpCheckSites lists the check pages for the settings and the profile drawer.
func (a *App) FpCheckSites() []FpCheckSite { return fpCheckSites }

func fpCheckSiteURL(key string) (string, bool) {
	for _, s := range fpCheckSites {
		if s.Key == key {
			return s.URL, true
		}
	}
	return "", false
}

// fpCheckPlan is what a launch should open: URL (an http(s) page) or Builtin
// (the local self-check page, URL filled in at launch), and whether the
// profile is marked so a "仅首次启动" check does not open again.
type fpCheckPlan struct {
	URL     string
	Builtin bool
	Once    bool
}

func (c fpCheckPlan) none() bool { return c.URL == "" && !c.Builtin }

// resolveFpCheck decides the check page for one launch.
//   - non-UI launches (local API, MCP, RPA, recorder, default browser) never open it
//   - profile mode "off" / "on" overrides the global switch; "" follows it
//   - the profile's own site ("" = global site) and frequency ("" = global)
//   - once: skipped when the profile already had its check page (FpCheckAt)
func resolveFpCheck(st Settings, p Profile, o LaunchOpts) fpCheckPlan {
	if !o.UI {
		return fpCheckPlan{}
	}
	on := st.FpCheckOn
	switch p.FpCheck {
	case "on":
		on = true
	case "off":
		on = false
	}
	if !on {
		return fpCheckPlan{}
	}
	once := st.FpCheckOnce
	switch p.FpCheckFreq {
	case "once":
		once = true
	case "every":
		once = false
	}
	if once && p.FpCheckAt != 0 {
		return fpCheckPlan{}
	}
	key, custom := st.FpCheckSite, st.FpCheckURL
	if p.FpCheckSite != "" {
		key, custom = p.FpCheckSite, p.FpCheckURL
	}
	if key == "" {
		key = "browserscan"
	}
	switch key {
	case fpCheckBuiltin:
		return fpCheckPlan{Builtin: true, Once: once}
	case fpCheckCustom:
		u := strings.TrimSpace(custom)
		if u != "" && !strings.Contains(u, "://") {
			u = "https://" + u
		}
		if !safeSiteURL(u) {
			return fpCheckPlan{}
		}
		return fpCheckPlan{URL: u, Once: once}
	}
	u, ok := fpCheckSiteURL(key)
	if !ok || u == "" {
		u, _ = fpCheckSiteURL("browserscan")
	}
	return fpCheckPlan{URL: u, Once: once}
}

// withCheckURL appends the check page after the profile's own start pages.
// A list that only holds blank tabs (none, "", about:blank) is replaced, so
// the check page lands in the first tab instead of next to an empty one.
func withCheckURL(own []string, check string) []string {
	if check == "" {
		return own
	}
	real := false
	for _, u := range own {
		if t := strings.TrimSpace(u); t != "" && t != "about:blank" {
			real = true
			break
		}
	}
	if !real {
		return []string{check}
	}
	out := append([]string(nil), own...)
	return append(out, check)
}

// ---- Firefox: the check page is opened over BiDi after start ----

var fpCheckFF = struct {
	sync.Mutex
	m map[string]string
}{m: map[string]string{}}

func fpCheckSetFirefox(id, u string) {
	fpCheckFF.Lock()
	if u == "" {
		delete(fpCheckFF.m, id)
	} else {
		fpCheckFF.m[id] = u
	}
	fpCheckFF.Unlock()
}

func fpCheckTakeFirefox(id string) string {
	fpCheckFF.Lock()
	defer fpCheckFF.Unlock()
	u := fpCheckFF.m[id]
	delete(fpCheckFF.m, id)
	return u
}

// firefoxOpenCheck opens u in a new tab behind the start pages and focuses
// it. With no start pages it reuses the first (blank) tab.
func firefoxOpenCheck(c *bidiClient, u string, reuseFirst bool) {
	time.Sleep(800 * time.Millisecond) // let the homepage tabs register first
	ctx := ""
	if reuseFirst {
		ctx, _ = c.firstContext()
	}
	if ctx == "" {
		var err error
		if ctx, err = c.NewTab(); err != nil || ctx == "" {
			return
		}
	}
	_ = c.Activate(ctx)
	_, _ = c.Call("browsingContext.navigate", map[string]any{"context": ctx, "url": u, "wait": "none"})
	_ = c.Activate(ctx)
}

func newFpToken() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
