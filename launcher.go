package main

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
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

// Launch starts the browser kernel for a profile and returns the run state.
func (s *Store) Launch(id string) (*RunState, error) {
	s.launchMu.Lock()
	defer s.launchMu.Unlock()
	if rs, ok := runReg.get(id); ok {
		return rs, nil
	}
	p, ok := s.GetProfile(id)
	if !ok {
		return nil, fmt.Errorf("profile not found")
	}
	return s.launchUndetectable(p, s.GetSettings())
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
