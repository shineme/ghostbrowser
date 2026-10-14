package main

import (
	"context"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"ghostbrowser/internal/undetectable"
)

func ffBase() Fingerprint {
	return Fingerprint{OS: "windows", DeviceType: "desktop", Languages: []string{"de-DE", "de", "en"}, Timezone: "Europe/Berlin",
		ScreenWidth: 1920, ScreenHeight: 1080, HardwareConc: 8, WebRTC: "proxy"}
}

func prefEq(t *testing.T, p ffPrefs, k string, want any) {
	t.Helper()
	v, ok := p.Get(k)
	if !ok {
		t.Fatalf("pref %s missing", k)
	}
	if fmt.Sprint(v) != fmt.Sprint(want) {
		t.Fatalf("pref %s = %v, want %v", k, v, want)
	}
}

func prefAbsent(t *testing.T, p ffPrefs, k string) {
	t.Helper()
	if v, ok := p.Get(k); ok {
		t.Fatalf("pref %s should be absent, got %v", k, v)
	}
}

func TestFirefoxPrefsLanguageAndTZ(t *testing.T) {
	fp := ffBase()
	p := firefoxPrefs(fp, nil, nil)
	prefEq(t, p, "intl.accept_languages", "de-DE, de, en")
	prefEq(t, p, "intl.locale.requested", "de-DE")
	prefEq(t, p, "privacy.resistFingerprinting", false)
	prefEq(t, p, "network.proxy.type", 0)
	env := firefoxEnv([]string{"PATH=/bin", "TZ=Asia/Tokyo", "XRE_PROFILE_PATH=/x"}, fp)
	if strings.Join(env, ";") != "PATH=/bin;TZ=Europe/Berlin" {
		t.Fatalf("env = %v", env)
	}
}

func TestFirefoxPrefsStrictExclusive(t *testing.T) {
	fp := ffBase()
	fp.FFStrict, fp.LanguageFromIP, fp.TimezoneFromIP = true, true, true
	n := normalizeFirefoxFP(fp)
	if n.LanguageFromIP || n.TimezoneFromIP {
		t.Fatal("strict mode must turn off language/timezone follows IP")
	}
	p := firefoxPrefs(fp, nil, nil)
	prefEq(t, p, "privacy.resistFingerprinting", true)
	prefAbsent(t, p, "intl.accept_languages")
	for _, e := range firefoxEnv(nil, fp) {
		if strings.HasPrefix(e, "TZ=") {
			t.Fatal("strict mode must not set TZ (RFP reports UTC)")
		}
	}
}

func TestFirefoxPrefsWebRTC(t *testing.T) {
	fp := ffBase()
	fp.WebRTC = "disabled"
	prefEq(t, firefoxPrefs(fp, nil, nil), "media.peerconnection.enabled", false)
	fp.WebRTC = "proxy"
	p := firefoxPrefs(fp, &ffProxy{"http", "1.2.3.4", 8080}, nil)
	prefEq(t, p, "media.peerconnection.ice.default_address_only", true)
	prefEq(t, p, "media.peerconnection.ice.no_host", true)
	prefEq(t, p, "media.peerconnection.ice.proxy_only_if_behind_proxy", true)
	fp.WebRTC = "real"
	p = firefoxPrefs(fp, nil, nil)
	prefEq(t, p, "media.peerconnection.enabled", true)
	prefAbsent(t, p, "media.peerconnection.ice.no_host")
}

func TestFirefoxPrefsPrivacyAndMedia(t *testing.T) {
	fp := ffBase()
	fp.DoNotTrack, fp.GPC, fp.FFWebGLOff, fp.SpeechVoices, fp.BlockImages, fp.WebGPU = true, true, true, "off", true, "off"
	fp.FFFontVisibility, fp.HardwareConc = 1, 4
	p := firefoxPrefs(fp, nil, []string{"example.com", " ", "https://a.test/x"})
	prefEq(t, p, "privacy.donottrackheader.enabled", true)
	prefEq(t, p, "privacy.globalprivacycontrol.enabled", true)
	prefEq(t, p, "webgl.disabled", true)
	prefEq(t, p, "dom.webgpu.enabled", false)
	prefEq(t, p, "media.webspeech.synth.enabled", false)
	prefEq(t, p, "permissions.default.image", 2)
	prefEq(t, p, "dom.maxHardwareConcurrency", 4)
	prefEq(t, p, "layout.css.font-visibility.standard", 1)
	prefEq(t, p, "layout.css.font-visibility.resistFingerprinting", 1)
	prefEq(t, p, "browser.startup.homepage", "https://example.com|https://a.test/x")
	prefEq(t, p, "browser.startup.page", 1)
	fp.FFFontVisibility = 9 // out of range = Firefox default
	prefAbsent(t, firefoxPrefs(fp, nil, nil), "layout.css.font-visibility.standard")
}

func TestFirefoxPrefsGeoAndUA(t *testing.T) {
	fp := ffBase()
	fp.Latitude, fp.Longitude = 52.52, 13.405
	fp.UserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:128.0) Gecko/20100101 Firefox/128.0"
	p := firefoxPrefs(fp, nil, nil)
	v, _ := p.Get("geo.provider.network.url")
	if s := v.(string); !strings.HasPrefix(s, "data:application/json,") || !strings.Contains(s, `"lat":52.52`) || !strings.Contains(s, `"lng":13.405`) {
		t.Fatalf("geo url = %v", v)
	}
	prefEq(t, p, "geo.provider.ms-windows-location", false)
	prefAbsent(t, p, "general.useragent.override") // only when asked
	fp.FFUAOverride = true
	prefEq(t, firefoxPrefs(fp, nil, nil), "general.useragent.override", fp.UserAgent)
}

func TestFirefoxPrefsProxy(t *testing.T) {
	// No credentials: direct prefs.
	cfg, relay := firefoxProxyFor(&Proxy{Type: "socks5", Host: "5.6.7.8", Port: 1080})
	if relay || cfg == nil || cfg.Scheme != "socks5" {
		t.Fatalf("socks5 cfg %+v relay %v", cfg, relay)
	}
	p := firefoxPrefs(ffBase(), cfg, nil)
	prefEq(t, p, "network.proxy.type", 1)
	prefEq(t, p, "network.proxy.socks", "5.6.7.8")
	prefEq(t, p, "network.proxy.socks_port", 1080)
	prefEq(t, p, "network.proxy.socks_remote_dns", true)
	prefAbsent(t, p, "network.proxy.http")
	// Credentials: always through the local relay (http on 127.0.0.1).
	px := &Proxy{Type: "socks5", Host: "127.0.0.1", Port: 1, User: "u", Pass: "p@ss"}
	if _, need := firefoxProxyFor(px); !need {
		t.Fatal("credentialed proxy must use the relay")
	}
	r, err := undetectable.StartRelay(px.URL().String())
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	cfg, err = ffProxyFromURL(r.URL())
	if err != nil || cfg.Scheme != "http" || cfg.Host != "127.0.0.1" {
		t.Fatalf("relay cfg %+v %v", cfg, err)
	}
	p = firefoxPrefs(ffBase(), cfg, nil)
	prefEq(t, p, "network.proxy.http", "127.0.0.1")
	prefEq(t, p, "network.proxy.ssl", "127.0.0.1")
	prefEq(t, p, "network.proxy.http_port", cfg.Port)
	prefEq(t, p, "network.proxy.ssl_port", cfg.Port)
	js := p.UserJS()
	if strings.Contains(js, "p@ss") {
		t.Fatal("password leaked into user.js")
	}
	if !strings.Contains(js, `user_pref("network.proxy.http", "127.0.0.1");`) || !strings.Contains(js, fmt.Sprintf(`user_pref("network.proxy.http_port", %d);`, cfg.Port)) {
		t.Fatalf("user.js:\n%s", js)
	}
	if _, err := ffProxyFromURL("http://127.0.0.1"); err == nil {
		t.Fatal("missing port must fail")
	}
}

func TestFirefoxUserJSEscaping(t *testing.T) {
	var p ffPrefs
	p.set("a.b", `x"y\z`)
	p.set("c", 3)
	p.set("d", true)
	p.set("c", 4) // overwrite keeps order
	js := p.UserJS()
	if !strings.Contains(js, `user_pref("a.b", "x\"y\\z");`) || !strings.Contains(js, `user_pref("c", 4);`) || strings.Count(js, `"c"`) != 1 || !strings.Contains(js, `user_pref("d", true);`) {
		t.Fatalf("user.js:\n%s", js)
	}
}

func TestFirefoxArgsAndUseKernel(t *testing.T) {
	fp := ffBase()
	fp.WindowMode, fp.WindowWidth, fp.WindowHeight = "custom", 1280, 720
	a := strings.Join(firefoxArgs("/p", 9222, fp, true), " ")
	for _, want := range []string{"-profile /p", "-no-remote", "--remote-debugging-port=9222", "-headless", "-width 1280 -height 720"} {
		if !strings.Contains(a, want) {
			t.Fatalf("args %q missing %q", a, want)
		}
	}
	p := Profile{Kernel: "firefox", Fingerprint: Fingerprint{OS: "macos"}}
	if !p.useFirefox() || p.useChromium() {
		t.Fatal("firefox kernel must not be routed to chromium")
	}
}

func TestFirefoxVersionFromApplicationIni(t *testing.T) {
	dir := t.TempDir()
	exe := filepath.Join(dir, "firefox.exe")
	os.WriteFile(exe, []byte("x"), 0o755)
	os.WriteFile(filepath.Join(dir, "application.ini"), []byte("[App]\nVendor=Mozilla\nName=Firefox\nVersion=128.5.0esr\nBuildID=1\n"), 0o644)
	if v := firefoxVersion(exe); v != "128.5.0esr" {
		t.Fatalf("version %q", v)
	}
	in := firefoxInfo(Settings{FirefoxPath: exe})
	if in.Path != exe || !in.ESR || !in.Custom || in.Missing {
		t.Fatalf("info %+v", in)
	}
	if in := firefoxInfo(Settings{FirefoxPath: filepath.Join(dir, "nope.exe")}); !in.Missing {
		t.Fatal("missing path not flagged")
	}
}

// The installer follows Mozilla's bouncer, refuses non-Mozilla / non-HTTPS
// hosts and rejects a file whose SHA-512 differs from SHA512SUMS.
func TestFirefoxInstallVerify(t *testing.T) {
	body := []byte("fake firefox installer")
	sum := sha512.Sum512(body)
	good := hex.EncodeToString(sum[:])
	sums := good
	mux := http.NewServeMux()
	var srvURL string
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("product") != "" {
			http.Redirect(w, r, srvURL+"/pub/firefox/releases/128.5.0esr/win64/zh-CN/Firefox%20Setup%20128.5.0esr.exe", http.StatusFound)
			return
		}
		http.NotFound(w, r)
	})
	mux.HandleFunc("/pub/firefox/releases/128.5.0esr/win64/zh-CN/", func(w http.ResponseWriter, r *http.Request) { w.Write(body) })
	mux.HandleFunc("/sums/128.5.0esr/SHA512SUMS", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "%s  win64/en-US/Firefox Setup 128.5.0esr.exe\n%s  win64/zh-CN/Firefox Setup 128.5.0esr.exe\n", strings.Repeat("0", 128), sums)
	})
	srv := httptest.NewTLSServer(mux)
	defer srv.Close()
	srvURL = srv.URL
	u, _ := url.Parse(srv.URL)
	oldBase, oldSums, oldHosts, oldCl, oldRun := firefoxDownloadBase, firefoxSumsBase, firefoxAllowedHosts, newKernelClient, firefoxRunInstaller
	defer func() {
		firefoxDownloadBase, firefoxSumsBase, firefoxAllowedHosts, newKernelClient, firefoxRunInstaller = oldBase, oldSums, oldHosts, oldCl, oldRun
	}()
	firefoxDownloadBase, firefoxSumsBase = srv.URL+"/", srv.URL+"/sums/"
	firefoxAllowedHosts = []string{u.Hostname()}
	newKernelClient = srv.Client
	t.Setenv("APPDATA", t.TempDir())
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	firefoxRunInstaller = func(ctx context.Context, setup, dir string) error {
		os.MkdirAll(dir, 0o755)
		return os.WriteFile(filepath.Join(dir, "firefox.exe"), []byte("exe"), 0o755)
	}
	wait := func() KernelProgress {
		for i := 0; i < 200; i++ {
			if s := ffInst.snapshot(); !s.Running {
				return s
			}
			time.Sleep(25 * time.Millisecond)
		}
		t.Fatal("install did not finish")
		return KernelProgress{}
	}
	var got string
	if _, err := firefoxInstallStart("esr", func(p string) { got = p }); err != nil {
		t.Fatal(err)
	}
	st := wait()
	if st.Phase != "done" || st.Version != "128.5.0esr" || !strings.HasSuffix(got, filepath.Join("esr", "firefox.exe")) {
		t.Fatalf("install: %+v got %q", st, got)
	}
	// Tampered file.
	body = []byte("tampered")
	if _, err := firefoxInstallStart("esr", nil); err != nil {
		t.Fatal(err)
	}
	if st := wait(); st.Phase != "error" || !strings.Contains(st.Error, "SHA512") {
		t.Fatalf("tampered: %+v", st)
	}
	// Host outside the allow list.
	firefoxAllowedHosts = []string{"download.mozilla.org"}
	if _, err := firefoxInstallStart("esr", nil); err != nil {
		t.Fatal(err)
	}
	if st := wait(); st.Phase != "error" {
		t.Fatalf("foreign host accepted: %+v", st)
	}
}
