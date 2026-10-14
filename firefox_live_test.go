package main

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// Live Firefox kernel test. Runs only with GHOST_FIREFOX=/usr/bin/firefox (any
// Firefox 129+). Launches headless through a password-protected HTTP proxy
// (so the local relay is used), checks language / time zone / hardware cap /
// proxy, then cookie export, import, snapshot and restore on "another device".
func TestFirefoxLive(t *testing.T) {
	exe := os.Getenv("GHOST_FIREFOX")
	if exe == "" {
		t.Skip("set GHOST_FIREFOX=/usr/bin/firefox to run")
	}
	t.Setenv("GHOST_FIREFOX_HEADLESS", "1")

	// Fake upstream proxy requiring Basic auth; serves http://ghost.test/ itself.
	var mu sync.Mutex
	var seen []string
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	wantAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte("u:p@ss"))
	go http.Serve(ln, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		seen = append(seen, r.Method+" "+r.Host+" auth="+fmt.Sprint(r.Header.Get("Proxy-Authorization") == wantAuth))
		mu.Unlock()
		if r.Header.Get("Proxy-Authorization") != wantAuth {
			w.Header().Set("Proxy-Authenticate", `Basic realm="x"`)
			w.WriteHeader(http.StatusProxyAuthRequired)
			return
		}
		if r.Method == http.MethodConnect {
			// The relay tunnels everything with CONNECT: act as the origin inside the tunnel.
			hj, _ := w.(http.Hijacker)
			conn, rw, err := hj.Hijack()
			if err != nil {
				return
			}
			defer conn.Close()
			rw.WriteString("HTTP/1.1 200 Connection Established\r\n\r\n")
			rw.Flush()
			for {
				req, err := http.ReadRequest(rw.Reader)
				if err != nil {
					return
				}
				mu.Lock()
				seen = append(seen, "tunnel "+req.Method+" "+req.Host+req.URL.Path)
				mu.Unlock()
				body := fmt.Sprintf("<!doctype html><title>ghost</title><body>AL=%s</body>", req.Header.Get("Accept-Language"))
				extra := ""
				if req.URL.Path == "/setcookie" {
					extra = "Set-Cookie: login=ok123; Path=/; Max-Age=172800\r\n"
				}
				fmt.Fprintf(rw, "HTTP/1.1 200 OK\r\nContent-Type: text/html; charset=utf-8\r\n%sContent-Length: %d\r\n\r\n%s", extra, len(body), body)
				rw.Flush()
			}
		}
		http.Error(w, "plain proxying not expected", http.StatusBadGateway)
	}))
	port := ln.Addr().(*net.TCPAddr).Port

	s := &Store{dir: t.TempDir(), Profiles: map[string]*Profile{}, Proxies: map[string]*Proxy{},
		Settings: Settings{ProfilesDir: t.TempDir(), FirefoxPath: exe, DeviceID: "devA"}}
	fp := Fingerprint{OS: "windows", DeviceType: "desktop", Languages: []string{"de-DE", "de"}, Timezone: "Asia/Tokyo",
		HardwareConc: 2, ScreenWidth: 1600, ScreenHeight: 900, WebRTC: "proxy", DoNotTrack: true, GPC: true}
	p := s.PutProfile(Profile{Name: "ff live", Kernel: "firefox", Fingerprint: fp,
		InlineProxy: &Proxy{Type: "http", Host: "127.0.0.1", Port: port, User: "u", Pass: "p@ss"}})

	t.Logf("firefox %s", firefoxVersion(exe))
	rs, err := s.Launch(p.ID)
	if err != nil {
		t.Fatalf("launch: %v", err)
	}
	defer runReg.stop(p.ID)
	if rs.Kernel != "firefox" || rs.WSURL != "" || !strings.HasPrefix(rs.BiDiURL, "ws://127.0.0.1:") {
		t.Fatalf("run state %+v", rs)
	}
	run := getFirefoxRun(p.ID)
	c := run.bidi
	ctx, err := c.firstContext()
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Navigate(ctx, "http://ghost.test/setcookie"); err != nil {
		t.Fatalf("navigate: %v", err)
	}
	v, err := c.Eval(ctx, `({lang: navigator.language, langs: navigator.languages.join(','), tz: Intl.DateTimeFormat().resolvedOptions().timeZone,
		off: new Date(2026,0,15).getTimezoneOffset(), hc: navigator.hardwareConcurrency, wd: navigator.webdriver, body: document.body.textContent,
		dnt: navigator.doNotTrack, gpc: navigator.globalPrivacyControl === true, cookie: document.cookie, ua: navigator.userAgent})`)
	if err != nil {
		t.Fatal(err)
	}
	m := v.(map[string]any)
	t.Logf("page: %v", m)
	if m["lang"] != "de-DE" || m["langs"] != "de-DE,de" {
		t.Errorf("language %v %v", m["lang"], m["langs"])
	}
	if m["tz"] != "Asia/Tokyo" || m["off"] != -540.0 {
		t.Errorf("timezone %v offset %v", m["tz"], m["off"])
	}
	if m["hc"] != 2.0 {
		t.Errorf("hardwareConcurrency %v", m["hc"])
	}
	if !strings.Contains(fmt.Sprint(m["body"]), "AL=de-DE") {
		t.Errorf("Accept-Language through proxy: %v", m["body"])
	}
	if m["dnt"] != "1" || m["gpc"] != true {
		t.Errorf("dnt %v gpc %v", m["dnt"], m["gpc"])
	}
	if !strings.Contains(fmt.Sprint(m["cookie"]), "login=ok123") {
		t.Errorf("cookie not set: %v", m["cookie"])
	}
	mu.Lock()
	ours := []string{}
	for _, l := range seen {
		if strings.Contains(l, "ghost.test") {
			ours = append(ours, l)
		}
	}
	t.Logf("proxy saw %d requests, ours: %v", len(seen), ours)
	authed := false
	for _, l := range seen {
		if strings.HasPrefix(l, "CONNECT ghost.test") && strings.HasSuffix(l, "auth=true") {
			authed = true
		}
	}
	mu.Unlock()
	if !authed {
		t.Error("request did not reach the proxy with credentials (relay)")
	}
	if err := focusFirefox(p.ID); err != nil {
		t.Errorf("focus: %v", err)
	}

	// Live cookie export / import go through BiDi.
	out, err := s.ExportCookies(p.ID, "json")
	if err != nil || !strings.Contains(out, "ok123") {
		t.Fatalf("export %v %s", err, out)
	}
	res, err := s.ImportCookies(p.ID, "json", `[{"name":"imp","value":"v1","domain":"ghost.test","path":"/","expirationDate":`+fmt.Sprint(time.Now().Add(time.Hour).Unix())+`}]`)
	if err != nil || res.Mode != "live" || res.Count != 1 {
		t.Fatalf("import %+v %v", res, err)
	}

	// Graceful stop: snapshot written, process gone.
	pid := rs.PID
	runReg.stop(p.ID)
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		if _, ok := runReg.get(p.ID); !ok {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if _, ok := runReg.get(p.ID); ok {
		t.Fatal("firefox still registered after stop")
	}
	snap, _ := readSnapshot(s.ProfileDir(p.ID))
	names := []string{}
	for _, ck := range snap {
		names = append(names, ck.Name)
	}
	if !strings.Contains(strings.Join(names, ","), "login") || !strings.Contains(strings.Join(names, ","), "imp") {
		t.Fatalf("snapshot after stop: %v", names)
	}
	t.Logf("stopped pid %d, snapshot %v", pid, names)

	// "Another device": the Firefox profile (cookies.sqlite) is gone, only the
	// synced snapshot is there. The restore must bring the login back.
	if err := os.RemoveAll(s.firefoxProfileDir(p.ID)); err != nil {
		t.Fatal(err)
	}
	s.Settings.DeviceID = "devB"
	// Second launch also switches on the Firefox-only switches.
	pp, _ := s.GetProfile(p.ID)
	pp.Fingerprint.FFWebGLOff, pp.Fingerprint.WebRTC, pp.Fingerprint.SpeechVoices, pp.Fingerprint.FFStrict = true, "disabled", "off", true
	s.PutProfile(*pp)
	if _, err := s.Launch(p.ID); err != nil {
		t.Fatalf("relaunch: %v", err)
	}
	c = getFirefoxRun(p.ID).bidi
	ctx, _ = c.firstContext()
	if err := c.Navigate(ctx, "http://ghost.test/"); err != nil {
		t.Fatal(err)
	}
	ck, _ := c.Eval(ctx, "document.cookie")
	if !strings.Contains(fmt.Sprint(ck), "login=ok123") || !strings.Contains(fmt.Sprint(ck), "imp=v1") {
		t.Fatalf("restored cookies: %v", ck)
	}
	sw, err := c.Eval(ctx, `({webgl: !!document.createElement('canvas').getContext('webgl'), rtc: typeof RTCPeerConnection, speech: typeof speechSynthesis,
		off: new Date(2026,0,15).getTimezoneOffset(), lang: navigator.language})`)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("switches: %v", sw)
	if mm := sw.(map[string]any); mm["webgl"] != false || mm["rtc"] != "undefined" || mm["speech"] != "undefined" || mm["off"] != 0.0 || mm["lang"] != "en-US" {
		t.Errorf("webgl/webrtc/speech/strict switches not applied: %v", mm)
	}
	b, _ := os.ReadFile(filepath.Join(s.ProfileDir(p.ID), sessionFile))
	if !strings.Contains(string(b), `"device":"devB"`) {
		t.Error("snapshot not re-taken as devB after restore")
	}
	userJS, _ := os.ReadFile(filepath.Join(s.firefoxProfileDir(p.ID), "user.js"))
	if strings.Contains(string(userJS), "p@ss") {
		t.Error("proxy password written to user.js")
	}
	sc := bufio.NewScanner(strings.NewReader(string(userJS)))
	n := 0
	for sc.Scan() {
		n++
	}
	t.Logf("user.js lines: %d", n)
	runReg.stop(p.ID)
}
