package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

// GHOST_CHROME=/usr/bin/chromium go test -run TestChromiumEmulationLive
func TestChromiumEmulationLive(t *testing.T) {
	bin := os.Getenv("GHOST_CHROME")
	if bin == "" {
		t.Skip("set GHOST_CHROME to a Chromium binary")
	}
	var gotUA, gotCH string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			gotUA, gotCH = r.Header.Get("User-Agent"), r.Header.Get("Sec-CH-UA-Mobile")
		}
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, "<html><head><meta name='viewport' content='width=device-width'></head><body>hi<iframe src='/f'></iframe></body></html>")
	}))
	defer srv.Close()

	fp := GenerateTyped("phone", "android", "141")
	fp.Timezone, fp.Languages = "Asia/Tokyo", []string{"ja-JP", "ja"}
	dir := t.TempDir()
	port := freePort()
	cmd := exec.Command(bin, "--headless=new", "--no-sandbox", "--user-data-dir="+dir, fmt.Sprintf("--remote-debugging-port=%d", port), "--user-agent="+fp.UserAgent, "about:blank")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer cmd.Process.Kill()
	ws, err := waitDebugger(port, 20*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	em, err := startEmulation(ws, fp, nil, []string{srv.URL + "/"})
	if err != nil {
		t.Fatal(err)
	}
	defer em.Close()
	c, err := dialCDP(ws, nil)
	if err != nil {
		t.Fatal(err)
	}
	var tl struct {
		TargetInfos []struct{ TargetID, Type, URL string } `json:"targetInfos"`
	}
	// The start page can take a few seconds to commit on a cold profile; poll.
	for deadline := time.Now().Add(15 * time.Second); time.Now().Before(deadline); time.Sleep(300 * time.Millisecond) {
		raw, _ := c.call("", "Target.getTargets", nil)
		_ = json.Unmarshal(raw, &tl)
		found := false
		for _, ti := range tl.TargetInfos {
			if ti.Type == "page" && strings.HasPrefix(ti.URL, srv.URL) {
				found = true
			}
		}
		if found {
			time.Sleep(500 * time.Millisecond)
			break
		}
	}
	sid := ""
	for _, ti := range tl.TargetInfos {
		if ti.Type == "page" && strings.HasPrefix(ti.URL, srv.URL) {
			r, _ := c.call("", "Target.attachToTarget", map[string]any{"targetId": ti.TargetID, "flatten": true})
			var at struct{ SessionID string }
			_ = json.Unmarshal(r, &at)
			sid = at.SessionID
		}
	}
	if sid == "" {
		t.Fatalf("start page not open: %+v", tl.TargetInfos)
	}
	r, err := c.call(sid, "Runtime.evaluate", map[string]any{"returnByValue": true, "awaitPromise": true, "expression": `(async () => {
		const gl = document.createElement('canvas').getContext('webgl');
		const ext = gl && gl.getExtension('WEBGL_debug_renderer_info');
		const he = await navigator.userAgentData.getHighEntropyValues(['model','platformVersion']);
		const fr = document.querySelector('iframe').contentWindow.navigator;
		return JSON.stringify({ua: navigator.userAgent, plat: navigator.platform, mobile: navigator.userAgentData.mobile, chPlat: navigator.userAgentData.platform,
		  model: he.model, pver: he.platformVersion, touch: navigator.maxTouchPoints, sw: screen.width, dpr: devicePixelRatio, iw: innerWidth,
		  coarse: matchMedia('(pointer: coarse)').matches, tz: Intl.DateTimeFormat().resolvedOptions().timeZone, lang: navigator.language,
		  gpu: ext ? gl.getParameter(ext.UNMASKED_RENDERER_WEBGL) : '', hc: navigator.hardwareConcurrency, frameUA: fr.userAgent, frameMem: fr.deviceMemory})
	})()`})
	if err != nil {
		t.Fatal(err)
	}
	var ev struct{ Result struct{ Value string } }
	_ = json.Unmarshal(r, &ev)
	var got map[string]any
	if err := json.Unmarshal([]byte(ev.Result.Value), &got); err != nil {
		t.Fatalf("eval: %s", r)
	}
	t.Logf("fp %s %dx%d@%g %s | got %v | header UA=%q mobile=%q", fp.DeviceName, fp.ScreenWidth, fp.ScreenHeight, fp.PixelRatio, fp.WebGLRenderer, got, gotUA, gotCH)
	check := func(k string, want any) {
		if fmt.Sprint(got[k]) != fmt.Sprint(want) {
			t.Errorf("%s = %v, want %v", k, got[k], want)
		}
	}
	check("mobile", true)
	check("chPlat", "Android")
	check("model", fp.DeviceModel)
	check("plat", fp.Platform)
	check("touch", fp.MaxTouchPoints)
	check("sw", fp.ScreenWidth)
	check("dpr", fp.PixelRatio)
	check("iw", fp.ScreenWidth)
	check("coarse", true)
	check("tz", "Asia/Tokyo")
	check("lang", "ja-JP")
	check("gpu", fp.WebGLRenderer)
	check("hc", fp.HardwareConc)
	check("frameMem", fp.DeviceMemory)
	if !strings.Contains(fmt.Sprint(got["ua"]), "Mobile Safari") || got["frameUA"] != got["ua"] {
		t.Errorf("ua %v / frame %v", got["ua"], got["frameUA"])
	}
	if gotCH != "?1" || !strings.Contains(gotUA, "Android") {
		t.Errorf("request headers UA=%q mobile=%q", gotUA, gotCH)
	}
}
