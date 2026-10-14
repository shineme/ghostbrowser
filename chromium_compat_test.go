package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestCompareVersions(t *testing.T) {
	for _, c := range []struct {
		a, b string
		want int
	}{{"99.0.1.2", "120.0.0.0", -1}, {"120.0.6099.224", "120.0.6099.109", 1}, {"155.0.8059.39", "155.0.8059.39", 0}} {
		if got := compareVersions(c.a, c.b); got != c.want {
			t.Errorf("compareVersions(%s,%s)=%d want %d", c.a, c.b, got, c.want)
		}
	}
}

func TestChromeVersionFromFolder(t *testing.T) {
	dir := t.TempDir()
	for _, d := range []string{"99.0.4844.51", "120.0.6099.109", "Locales"} {
		_ = os.Mkdir(filepath.Join(dir, d), 0o755)
	}
	_ = os.WriteFile(filepath.Join(dir, "chrome.exe"), []byte("x"), 0o644)
	if got := chromeFullFrom(dir); got != "120.0.6099.109" {
		t.Errorf("chromeFullFrom = %q", got)
	}
	if got := chromeMajorFrom(dir); got != "120" {
		t.Errorf("chromeMajorFrom = %q (lexical sort would pick 99)", got)
	}
	// Chrome for Testing / ungoogled portable: <version>.manifest next to chrome.exe.
	dir2 := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir2, "131.0.6778.85.manifest"), []byte("x"), 0o644)
	if got := chromeFullFrom(dir2); got != "131.0.6778.85" {
		t.Errorf("manifest version = %q", got)
	}
}

func TestPEProductVersion(t *testing.T) {
	// Fake chrome.exe: VS_VERSIONINFO-style UTF-16LE "ProductVersion\0" + padding + value.
	blob := append([]byte("MZ\x90\x00junk"), utf16le("FileVersion")...)
	blob = append(blob, 0, 0, 0, 0)
	blob = append(blob, utf16le("ProductVersion")...)
	blob = append(blob, 0, 0, 0, 0)
	blob = append(blob, utf16le("131.0.6778.85")...)
	blob = append(blob, 0, 0)
	exe := filepath.Join(t.TempDir(), "chrome.exe")
	_ = os.WriteFile(exe, blob, 0o644)
	if got := peProductVersion(exe); got != "131.0.6778.85" {
		t.Fatalf("peProductVersion = %q", got)
	}
	if got := kernelVersion(exe); got != "131.0.6778.85" {
		t.Fatalf("kernelVersion(exe) = %q", got)
	}
}

func TestKernelVersionFromBinary(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("chrome.exe --version opens a window; folder detection only")
	}
	bin := filepath.Join(t.TempDir(), "chrome")
	_ = os.WriteFile(bin, []byte("#!/bin/sh\necho 'Chromium 118.0.5993.70 built on Debian'\n"), 0o755)
	if got := kernelVersion(bin); got != "118.0.5993.70" {
		t.Fatalf("kernelVersion = %q", got)
	}
	if err := kernelUnsupportedErr(bin); err != nil {
		t.Errorf("118 should be allowed: %v", err)
	}
	old := filepath.Join(t.TempDir(), "chrome")
	_ = os.WriteFile(old, []byte("#!/bin/sh\necho 'Google Chrome 104.0.5112.79'\n"), 0o755)
	if err := kernelUnsupportedErr(old); err == nil || !strings.Contains(err.Error(), "不受支持") {
		t.Errorf("104 should be refused in Chinese, got %v", err)
	}
}

func TestKernelCompatLevels(t *testing.T) {
	for _, c := range []struct{ path, full, level string }{
		{"", "", "missing"},
		{"/x/chrome", "", "unknown"},
		{"/x/chrome", "104.0.5112.79", "unsupported"},
		{"/x/chrome", "115.0.5790.170", "warn"},
		{"/x/chrome", "155.0.8059.39", "ok"},
		{`C:\Program Files (x86)\Microsoft\Edge\Application\msedge.exe`, "141.0.3537.57", "warn"},
	} {
		k := kernelCompatFor(c.path, c.full)
		if k.Level != c.level || (c.level != "ok" && k.Message == "") {
			t.Errorf("%s %s: %+v want %s", c.path, c.full, k, c.level)
		}
	}
}

func TestAlignUAMajor(t *testing.T) {
	fp := GenerateTyped("phone", "android", "141")
	alignUAMajor(&fp, 120)
	if !strings.Contains(fp.UserAgent, "Chrome/120.0.0.0") || fp.BrowserVersion != "120" {
		t.Errorf("ua %q bv %q", fp.UserAgent, fp.BrowserVersion)
	}
	tv := GenerateTyped("tv", "tizen", "141")
	ua := tv.UserAgent
	alignUAMajor(&tv, 120)
	if tv.UserAgent != ua {
		t.Errorf("tv UA changed: %q", tv.UserAgent)
	}
}

// TestKernelCompatLive checks, on whatever kernel GHOST_CHROME points at, that
// the UA's major, navigator.userAgentData brands and the request header all
// follow the kernel's real version even when the fingerprint says otherwise,
// and that a tab opened later with a URL still gets the page script.
// GHOST_CHROME=/usr/bin/chromium go test -run TestKernelCompatLive
func TestKernelCompatLive(t *testing.T) {
	bin := os.Getenv("GHOST_CHROME")
	if bin == "" {
		t.Skip("set GHOST_CHROME to a Chromium binary")
	}
	var hdrUA string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			hdrUA = r.Header.Get("User-Agent")
		}
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, "<html><body>compat</body></html>")
	}))
	defer srv.Close()
	fp := GenerateTyped("desktop", "windows", "99") // deliberately wrong major
	fp.SpeechVoices = "custom"
	// The command-line UA (first request of a tab opened with a URL) comes
	// from launch-time detection; the CDP session must fix the rest itself.
	flagFP := fp
	alignUAMajor(&flagFP, majorOf(kernelVersion(bin)))
	port := freePort()
	cmd := exec.Command(bin, "--headless=new", "--no-sandbox", "--user-data-dir="+liveUserDataDir(t), fmt.Sprintf("--remote-debugging-port=%d", port), "--user-agent="+flagFP.UserAgent, "about:blank")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { cmd.Process.Kill(); cmd.Wait() }()
	ws, err := waitDebugger(port, 20*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	em, err := startEmulation(ws, fp, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer em.Close()
	if em.major < chromiumMinMajor {
		t.Fatalf("kernel major %d below supported minimum", em.major)
	}
	c, err := dialCDP(ws, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.c.Close()
	time.Sleep(300 * time.Millisecond)
	if _, err := c.call("", "Target.createTarget", map[string]any{"url": srv.URL + "/"}); err != nil {
		t.Fatal(err)
	}
	sid := ""
	for i := 0; i < 50 && sid == ""; i++ {
		time.Sleep(200 * time.Millisecond)
		raw, _ := c.call("", "Target.getTargets", nil)
		var tl struct {
			TargetInfos []struct{ TargetID, Type, URL string } `json:"targetInfos"`
		}
		_ = json.Unmarshal(raw, &tl)
		for _, ti := range tl.TargetInfos {
			if ti.Type == "page" && strings.HasPrefix(ti.URL, srv.URL) {
				r, _ := c.call("", "Target.attachToTarget", map[string]any{"targetId": ti.TargetID, "flatten": true})
				var at struct{ SessionID string }
				_ = json.Unmarshal(r, &at)
				sid = at.SessionID
			}
		}
	}
	if sid == "" {
		t.Fatal("page did not open")
	}
	var got map[string]any
	for i := 0; i < 20; i++ { // legacy kernels reload once; wait for the script
		time.Sleep(250 * time.Millisecond)
		r, _ := c.call(sid, "Runtime.evaluate", map[string]any{"returnByValue": true, "expression": `JSON.stringify({gb: window.__gb === 1, ua: navigator.userAgent,
		  brands: navigator.userAgentData ? navigator.userAgentData.brands.map(b => b.brand + "/" + b.version).join(",") : "",
		  voices: speechSynthesis.getVoices().length, voice0: String((speechSynthesis.getVoices()[0] || {}).name)})`})
		var ev struct{ Result struct{ Value string } }
		_ = json.Unmarshal(r, &ev)
		got = nil
		_ = json.Unmarshal([]byte(ev.Result.Value), &got)
		if got["gb"] == true {
			break
		}
	}
	want := fmt.Sprintf("Chrome/%d.0.0.0", em.major)
	t.Logf("kernel %d | %v | header %q", em.major, got, hdrUA)
	if got["gb"] != true {
		t.Errorf("page script missing in a tab opened with a URL")
	}
	if ua := fmt.Sprint(got["ua"]); !strings.Contains(ua, want) {
		t.Errorf("navigator.userAgent %q, want %s", ua, want)
	}
	if !strings.Contains(hdrUA, want) {
		t.Errorf("request header UA %q, want %s", hdrUA, want)
	}
	if b := fmt.Sprint(got["brands"]); !strings.Contains(b, fmt.Sprintf("/%d", em.major)) || strings.Contains(b, "/99") {
		t.Errorf("brands %q do not follow kernel %d", b, em.major)
	}
	if n, _ := got["voices"].(float64); n < 3 || !strings.HasPrefix(fmt.Sprint(got["voice0"]), "Microsoft") {
		t.Errorf("custom voices %v %v", got["voices"], got["voice0"])
	}
}
