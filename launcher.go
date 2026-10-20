package main

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"
)

func defaultKernelPath(st Settings) string {
	if st.UndetectableKit != "" {
		candidate := filepath.Join(st.UndetectableKit, "runtime", "chrome", "chrome.exe")
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate
		}
	}
	if kit := findUndetectableKit(); kit != "" {
		candidate := filepath.Join(kit, "runtime", "chrome", "chrome.exe")
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate
		}
	}
	return ""
}

// freePort returns an unused 127.0.0.1 TCP port for the CDP endpoint.
func freePort() int {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

// resolveFP folds proxy geo into the fingerprint when the "from IP" toggles are on.
func resolveFP(fp Fingerprint, px *Proxy) Fingerprint {
	if px != nil && px.Status == "ok" {
		if fp.TimezoneFromIP && px.Timezone != "" {
			fp.Timezone = px.Timezone
		}
		if fp.LanguageFromIP && px.Country != "" {
			fp.Languages = languageForCountry(px.Country)
		}
		if fp.GeoFromIP && (px.Lat != 0 || px.Lon != 0) {
			fp.Latitude, fp.Longitude = px.Lat, px.Lon
		}
	}
	return fp
}

// Launch starts the browser kernel for a profile (automation callers: local
// API, MCP, default browser) and returns the run state.
func (s *Store) Launch(id string) (*RunState, error) { return s.LaunchWith(id, LaunchOpts{}) }

// LaunchWith starts a profile with per-launch flags (LaunchOpts.UI: the
// fingerprint check page may open as an extra tab).
func (s *Store) LaunchWith(id string, o LaunchOpts) (*RunState, error) {
	s.launchMu.Lock()
	defer s.launchMu.Unlock()
	if rs, ok := runReg.get(id); ok {
		return rs, nil
	}
	p, ok := s.GetProfile(id)
	if !ok {
		return nil, fmt.Errorf("profile not found")
	}
	s.rotateBeforeLaunch(p)
	orig, site := p.StartURLs, launchCfg.takeURL(id)
	if site != "" { // 网站快捷入口: open this site instead of the start pages (this launch only)
		p.StartURLs = []string{site}
	}
	check := resolveFpCheck(s.GetSettings(), *p, o)
	checkURL := check.URL
	if check.Builtin {
		if u, e := fpCheckBuiltinURL(p.ID); e == nil {
			checkURL = u
		}
	}
	if checkURL != "" { // 指纹检测页: one extra tab after the start pages (this launch only)
		if p.useFirefox() {
			fpCheckSetFirefox(p.ID, checkURL)
		} else {
			p.StartURLs = withCheckURL(p.StartURLs, checkURL)
		}
	}
	var rs *RunState
	var err error
	if p.useFirefox() {
		rs, err = s.launchFirefox(p, s.GetSettings())
	} else if p.useChromium() {
		rs, err = s.launchChromium(p, s.GetSettings())
	} else {
		rs, err = s.launchUndetectable(p, s.GetSettings())
	}
	fpCheckSetFirefox(id, "")
	if site != "" || checkURL != "" { // the launchers save the profile (LastRun); keep its own start pages
		if cur, ok := s.GetProfile(id); ok {
			cur.StartURLs = orig
			if err == nil && checkURL != "" && check.Once {
				cur.FpCheckAt = time.Now().Unix()
			}
			s.PutProfile(*cur)
		}
	}
	if err == nil {
		launchCfg.launched(s, id) // 待重启: remember the launch config
	} else if _, running := runReg.get(id); !running {
		coreMgr.Release("profile:" + id)
	}
	return rs, err
}

func firstLang(l []string) string {
	if len(l) > 0 {
		return l[0]
	}
	return "en-US"
}

func (s *Store) GetProxy(id string) (*Proxy, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.Proxies[id]
	if !ok {
		return nil, false
	}
	cp := *p
	return &cp, true
}
