package main

// user.js generation for the Firefox kernel. Everything Firefox can do about a
// fingerprint is a preference (or the TZ environment variable), so a profile's
// settings become one user.js written into its Firefox profile folder before
// every launch. GhostBrowser owns that file and rewrites it whole; settings a
// user changes in about:config live in prefs.js and are kept, except where
// user.js sets the same key (user.js wins at every start).
//
// Honest limits: Firefox has no switch for the WebGL vendor / renderer, canvas
// noise, screen size or the platform, and general.useragent.override only
// changes the UA string (navigator.oscpu, platform and the engine's real
// features still tell the truth), so the UI warns when the override is used.

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// ffPref is one user_pref line, kept in insertion order.
type ffPref struct {
	Key string
	Val any // string | bool | int
}

type ffPrefs []ffPref

func (p *ffPrefs) set(k string, v any) {
	for i := range *p {
		if (*p)[i].Key == k {
			(*p)[i].Val = v
			return
		}
	}
	*p = append(*p, ffPref{k, v})
}

// Get returns a pref's value (tests and the launcher).
func (p ffPrefs) Get(k string) (any, bool) {
	for _, x := range p {
		if x.Key == k {
			return x.Val, true
		}
	}
	return nil, false
}

// UserJS renders the prefs as a user.js file.
func (p ffPrefs) UserJS() string {
	var b strings.Builder
	b.WriteString("// Written by GhostBrowser before every launch. Edits here are overwritten;\n// use about:config for your own settings.\n")
	for _, x := range p {
		var v string
		switch t := x.Val.(type) {
		case string:
			j, _ := json.Marshal(t)
			v = string(j)
		case bool:
			v = strconv.FormatBool(t)
		case int:
			v = strconv.Itoa(t)
		default:
			j, _ := json.Marshal(t)
			v = string(j)
		}
		fmt.Fprintf(&b, "user_pref(%q, %s);\n", x.Key, v)
	}
	return b.String()
}

// ffProxy is the proxy Firefox talks to: the real one when it needs no
// password, otherwise the local relay (always http, 127.0.0.1).
type ffProxy struct {
	Scheme string // http | socks5
	Host   string
	Port   int
}

// firefoxProxyFor decides how Firefox reaches px. needsRelay is true when the
// proxy carries credentials: Firefox cannot take a proxy password from prefs
// (and never for SOCKS5), so the launcher starts the existing local relay and
// passes its URL through ffProxyFromURL.
func firefoxProxyFor(px *Proxy) (cfg *ffProxy, needsRelay bool) {
	if px == nil || px.Type == "" || px.Type == "none" || px.Host == "" || px.Port == 0 {
		return nil, false
	}
	if px.User != "" {
		return nil, true
	}
	scheme := "http"
	if strings.HasPrefix(px.Type, "socks") {
		scheme = "socks5"
	}
	return &ffProxy{Scheme: scheme, Host: px.Host, Port: px.Port}, false
}

// ffProxyFromURL turns a relay URL ("http://127.0.0.1:53121") into prefs input.
func ffProxyFromURL(raw string) (*ffProxy, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return nil, err
	}
	port, err := strconv.Atoi(u.Port())
	if err != nil || port <= 0 {
		return nil, fmt.Errorf("代理地址缺少端口：%s", raw)
	}
	scheme := "http"
	if strings.HasPrefix(u.Scheme, "socks") {
		scheme = "socks5"
	}
	return &ffProxy{Scheme: scheme, Host: u.Hostname(), Port: port}, nil
}

// normalizeFirefoxFP applies the Firefox-only rules: strict mode
// (privacy.resistFingerprinting) reports en-US and UTC by design, so it cannot
// be combined with "language follows IP" — strict wins.
func normalizeFirefoxFP(fp Fingerprint) Fingerprint {
	if fp.FFStrict {
		fp.LanguageFromIP = false
		fp.TimezoneFromIP = false
	}
	if fp.FFFontVisibility < 0 || fp.FFFontVisibility > 3 {
		fp.FFFontVisibility = 0
	}
	return fp
}

// firefoxPrefs builds the user.js prefs for a (geo-resolved) fingerprint.
func firefoxPrefs(fp Fingerprint, px *ffProxy, startURLs []string) ffPrefs {
	fp = normalizeFirefoxFP(fp)
	var p ffPrefs
	// Quiet start: no welcome pages, default-browser nag, telemetry or updates.
	for _, kv := range []ffPref{
		{"browser.shell.checkDefaultBrowser", false},
		{"browser.aboutwelcome.enabled", false},
		{"browser.startup.homepage_override.mstone", "ignore"},
		{"startup.homepage_welcome_url", ""},
		{"startup.homepage_welcome_url.additional", ""},
		{"browser.tabs.warnOnClose", false},
		{"browser.warnOnQuit", false},
		{"browser.sessionstore.resume_from_crash", false},
		{"datareporting.policy.dataSubmissionEnabled", false},
		{"datareporting.healthreport.uploadEnabled", false},
		{"toolkit.telemetry.enabled", false},
		{"toolkit.telemetry.reportingpolicy.firstRun", false},
		{"app.update.auto", false},
		{"app.update.checkInstallTime", false},
		{"app.normandy.enabled", false},
		{"browser.discovery.enabled", false},
		{"extensions.getAddons.cache.enabled", false},
		{"remote.prefs.recommended", false}, // keep our prefs; BiDi's recommended set changes behaviour
		{"toolkit.telemetry.unified", false},
		{"toolkit.telemetry.archive.enabled", false},
		{"browser.ping-centre.telemetry", false},
		{"network.captive-portal-service.enabled", false},
		{"network.connectivity-service.enabled", false},
		{"browser.newtabpage.activity-stream.feeds.section.topstories", false},
		{"browser.newtabpage.activity-stream.showSponsored", false},
		{"browser.newtabpage.activity-stream.showSponsoredTopSites", false},
		{"browser.topsites.contile.enabled", false},
		{"browser.urlbar.quicksuggest.enabled", false},
	} {
		p.set(kv.Key, kv.Val)
	}

	// Proxy. 0 = direct (never the system proxy, so nothing leaks around it).
	if px != nil {
		p.set("network.proxy.type", 1)
		p.set("network.proxy.no_proxies_on", "")
		p.set("network.proxy.failover_direct", false)
		p.set("network.proxy.socks_remote_dns", true)
		p.set("network.trr.mode", 5) // no DNS-over-HTTPS around the proxy
		if px.Scheme == "socks5" {
			p.set("network.proxy.socks", px.Host)
			p.set("network.proxy.socks_port", px.Port)
			p.set("network.proxy.socks_version", 5)
		} else {
			p.set("network.proxy.http", px.Host)
			p.set("network.proxy.http_port", px.Port)
			p.set("network.proxy.ssl", px.Host)
			p.set("network.proxy.ssl_port", px.Port)
			p.set("network.proxy.share_proxy_settings", false)
		}
	} else {
		p.set("network.proxy.type", 0)
	}

	// Strict mode: Firefox's own anti-fingerprinting (UTC, en-US, rounded
	// screen and timers, letterboxing off so the window keeps its size).
	p.set("privacy.resistFingerprinting", fp.FFStrict)
	if fp.FFStrict {
		p.set("privacy.resistFingerprinting.letterboxing", false)
		p.set("privacy.spoof_english", 2)
	} else if len(fp.Languages) > 0 {
		langs := make([]string, 0, len(fp.Languages))
		for _, l := range fp.Languages {
			if l = strings.TrimSpace(l); l != "" {
				langs = append(langs, l)
			}
		}
		if len(langs) > 0 {
			p.set("intl.accept_languages", strings.Join(langs, ", "))
			p.set("intl.locale.requested", langs[0])
			p.set("intl.regional_prefs.use_os_locales", false)
			p.set("javascript.use_us_english_locale", false)
		}
	}

	// WebRTC.
	switch fp.WebRTC {
	case "disabled":
		p.set("media.peerconnection.enabled", false)
	case "proxy", "":
		p.set("media.peerconnection.enabled", true)
		p.set("media.peerconnection.ice.default_address_only", true)
		p.set("media.peerconnection.ice.no_host", true)
		if px != nil {
			p.set("media.peerconnection.ice.proxy_only_if_behind_proxy", true)
		}
	default: // real
		p.set("media.peerconnection.enabled", true)
	}

	// DNT / GPC.
	p.set("privacy.donottrackheader.enabled", fp.DoNotTrack)
	p.set("privacy.globalprivacycontrol.enabled", fp.GPC)
	p.set("privacy.globalprivacycontrol.functionality.enabled", fp.GPC)

	// Geolocation: a fixed position served from a data: URL instead of the
	// network location provider; the OS providers are switched off so the real
	// location never wins. The site still has to ask permission.
	p.set("geo.enabled", true)
	if fp.Latitude != 0 || fp.Longitude != 0 {
		loc, _ := json.Marshal(map[string]any{"location": map[string]float64{"lat": fp.Latitude, "lng": fp.Longitude}, "accuracy": 30})
		p.set("geo.provider.network.url", "data:application/json,"+string(loc))
		p.set("geo.provider.ms-windows-location", false)
		p.set("geo.provider.use_corelocation", false)
		p.set("geo.provider.use_gpsd", false)
		p.set("geo.provider.use_geoclue", false)
		p.set("geo.provider.network.scan", false)
	}

	// WebGL / WebGPU.
	p.set("webgl.disabled", fp.FFWebGLOff)
	if fp.WebGPU == "off" {
		p.set("dom.webgpu.enabled", false)
	}

	// Fonts visible to pages: 1 = base system fonts, 2 = + language packs, 3 = all.
	if v := fp.FFFontVisibility; v > 0 {
		for _, k := range []string{"standard", "private", "trackingprotection", "resistFingerprinting"} {
			p.set("layout.css.font-visibility."+k, v)
		}
	}

	// Speech voices.
	if fp.SpeechVoices == "off" {
		p.set("media.webspeech.synth.enabled", false)
	}
	// navigator.hardwareConcurrency is capped (it can only go down).
	if fp.HardwareConc > 0 {
		p.set("dom.maxHardwareConcurrency", fp.HardwareConc)
	}
	// Media.
	if fp.BlockImages {
		p.set("permissions.default.image", 2)
	} else {
		p.set("permissions.default.image", 1)
	}
	if fp.BlockSound {
		p.set("media.volume_scale", "0.0")
	}
	// UA string only. Inconsistent with the engine when it is not a matching Firefox UA.
	if fp.FFUAOverride && strings.TrimSpace(fp.UserAgent) != "" {
		p.set("general.useragent.override", strings.TrimSpace(fp.UserAgent))
	}

	// Startup page(s).
	urls := cleanStartURLs(startURLs)
	if len(urls) > 0 {
		p.set("browser.startup.page", 1)
		p.set("browser.startup.homepage", strings.Join(urls, "|"))
	} else {
		p.set("browser.startup.page", 0)
		p.set("browser.startup.homepage", "about:blank")
	}
	return p
}

func cleanStartURLs(in []string) []string {
	out := []string{}
	for _, u := range in {
		if u = strings.TrimSpace(u); u != "" {
			if !strings.Contains(u, "://") && !strings.HasPrefix(u, "about:") {
				u = "https://" + u
			}
			out = append(out, u)
		}
	}
	return out
}

// firefoxEnv is the environment for the Firefox process: TZ sets the time
// zone (strict mode reports UTC on its own).
func firefoxEnv(base []string, fp Fingerprint) []string {
	fp = normalizeFirefoxFP(fp)
	out := make([]string, 0, len(base)+2)
	for _, e := range base {
		k := strings.ToUpper(strings.SplitN(e, "=", 2)[0])
		// Never inherit another Firefox's profile selection or our TZ.
		if k == "TZ" || k == "XRE_PROFILE_PATH" || k == "XRE_PROFILE_LOCAL_PATH" || k == "XRE_RESTARTED_BY_PROFILE_MANAGER" || k == "MOZ_NO_REMOTE" {
			continue
		}
		out = append(out, e)
	}
	if !fp.FFStrict && fp.Timezone != "" {
		out = append(out, "TZ="+fp.Timezone)
	}
	return out
}

// firefoxWindowArgs sizes the first window (viewport) from the profile's window settings.
func firefoxWindowArgs(fp Fingerprint) []string {
	w, h := fp.WindowWidth, fp.WindowHeight
	if fp.WindowMode != "custom" || w <= 0 || h <= 0 {
		ws := windowFor(fp)
		w, h = ws.w, ws.h
	}
	if w <= 0 || h <= 0 {
		return nil
	}
	return []string{"-width", strconv.Itoa(w), "-height", strconv.Itoa(h)}
}
