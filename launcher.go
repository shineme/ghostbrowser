package main

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

func findChrome() string {
	var cands []string
	switch runtime.GOOS {
	case "windows":
		pf := os.Getenv("ProgramFiles")
		pf86 := os.Getenv("ProgramFiles(x86)")
		local := os.Getenv("LOCALAPPDATA")
		cands = []string{
			filepath.Join(pf, "Google/Chrome/Application/chrome.exe"),
			filepath.Join(pf86, "Google/Chrome/Application/chrome.exe"),
			filepath.Join(local, "Google/Chrome/Application/chrome.exe"),
			filepath.Join(pf, "Chromium/Application/chrome.exe"),
			filepath.Join(pf86, "Microsoft/Edge/Application/msedge.exe"),
		}
	case "darwin":
		cands = []string{
			"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
			"/Applications/Chromium.app/Contents/MacOS/Chromium",
			"/Applications/Microsoft Edge.app/Contents/MacOS/Microsoft Edge",
		}
	default:
		cands = []string{"/usr/bin/google-chrome", "/usr/bin/chromium", "/usr/bin/chromium-browser", "/snap/bin/chromium"}
	}
	for _, c := range cands {
		if st, err := os.Stat(c); err == nil && !st.IsDir() {
			return c
		}
	}
	return ""
}

func defaultKernelPath(st Settings) string {
	if st.KernelMode == "undetectable" {
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
	}
	return findChrome()
}

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
	st := s.GetSettings()
	if st.KernelMode == "undetectable" {
		return s.launchUndetectable(p, st)
	}
	kernel := st.KernelPath
	if kernel == "" {
		kernel = findChrome()
	}
	if kernel == "" {
		return nil, fmt.Errorf("no browser kernel found — set the kernel path in Settings")
	}

	var px *Proxy
	if p.InlineProxy != nil {
		px = p.InlineProxy
	} else if p.ProxyID != "" {
		if pr, ok := s.GetProxy(p.ProxyID); ok {
			px = pr
		}
	}
	fp := resolveFP(p.Fingerprint, px)

	userDir := s.ProfileDir(id)
	_ = os.MkdirAll(userDir, 0o755)

	// write fingerprint.json next to the profile (used by custom kernel + extension)
	fpBytes, _ := json.MarshalIndent(fp, "", "  ")
	_ = os.WriteFile(filepath.Join(userDir, "fingerprint.json"), fpBytes, 0o644)

	port := freePort()
	args := []string{
		"--user-data-dir=" + userDir,
		fmt.Sprintf("--remote-debugging-port=%d", port),
		"--no-first-run", "--no-default-browser-check",
		"--disable-sync", "--disable-features=Translate",
		"--lang=" + firstLang(fp.Languages),
		fmt.Sprintf("--window-size=%d,%d", fp.ScreenWidth*8/10, fp.ScreenHeight*8/10),
	}

	switch st.KernelMode {
	case "custom":
		// Patched Chromium: everything is applied natively from the config file.
		args = append(args,
			"--fingerprint-config="+filepath.Join(userDir, "fingerprint.json"),
			"--fingerprint-seed="+fmt.Sprint(fp.NoiseSeed),
		)
		extDir := filepath.Join(userDir, "_fp_ext")
		if err := writeFPExtension(extDir, fp, px, false); err == nil {
			args = append(args, "--load-extension="+extDir)
		}
	default:
		// Stock Chrome: apply what flags + a generated extension can do.
		args = append(args, "--user-agent="+fp.UserAgent)
		if fp.WebRTC == "disabled" {
			args = append(args, "--force-webrtc-ip-handling-policy=disable_non_proxied_udp")
		}
		extDir := filepath.Join(userDir, "_fp_ext")
		if err := writeFPExtension(extDir, fp, px, true); err == nil {
			args = append(args, "--load-extension="+extDir, "--disable-extensions-except="+extDir)
		}
	}

	if px != nil && px.Type != "none" && px.Host != "" {
		scheme := px.Type
		if scheme == "https" {
			scheme = "http"
		}
		args = append(args, fmt.Sprintf("--proxy-server=%s://%s:%d", scheme, px.Host, px.Port))
	}
	for _, u := range p.StartURLs {
		if strings.TrimSpace(u) != "" {
			args = append(args, u)
		}
	}

	cmd := exec.Command(kernel, args...)
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	rs := &RunState{ProfileID: id, PID: cmd.Process.Pid, DebugPort: port,
		WSURL: fmt.Sprintf("http://127.0.0.1:%d/json/version", port), Started: time.Now().Unix()}

	p.LastRun = time.Now().Unix()
	s.PutProfile(*p)

	runReg.add(id, rs, cmd)
	go func() { _ = cmd.Wait(); runReg.remove(id) }()

	if ws, err := waitDebugger(port, 15*time.Second); err == nil {
		rs.WSURL = ws
		if strings.TrimSpace(p.Cookies) != "" {
			if err := importCookies(ws, p.Cookies); err == nil {
				p.Cookies = ""
				s.PutProfile(*p)
			}
		}
	}
	return rs, nil
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

func firstLang(l []string) string {
	if len(l) > 0 {
		return l[0]
	}
	return "en-US"
}
