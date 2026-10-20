package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"
)

// GHOST_CHROME=/usr/bin/chromium go test -run TestFpCheckLive
// A UI launch opens the check page as one extra tab after the profile's own
// start pages (opened last = focused); "仅首次启动" opens it once; launches
// from the local API / RPA path never open it; the built-in page serves the
// launched profile's values.
func TestFpCheckLive(t *testing.T) {
	bin := os.Getenv("GHOST_CHROME")
	if bin == "" {
		t.Skip("set GHOST_CHROME to a Chromium binary")
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(w, "<title>%s</title><body>%s</body>", r.URL.Path, r.URL.Path)
	}))
	defer srv.Close()
	root, err := os.MkdirTemp("", "fpcheck-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { time.Sleep(500 * time.Millisecond); _ = os.RemoveAll(root) })
	s := &Store{dir: root, Profiles: map[string]*Profile{}, Proxies: map[string]*Proxy{}, Settings: Settings{ChromiumPath: bin, ProfilesDir: filepath.Join(root, "profiles"), DefaultOS: "windows", DeviceID: "fpcheck",
		FpCheckOn: true, FpCheckSite: "custom", FpCheckURL: srv.URL + "/check", FpCheckOnce: true}}
	a := &App{store: s, auto: &autoSyncer{s: s, busy: map[string]bool{}}}
	fp := GenerateTyped("desktop", "windows", "")
	fp.StartupParameters = "--headless=new --no-sandbox"
	fp.TimezoneFromIP, fp.LanguageFromIP, fp.GeoFromIP = false, false, false
	id := s.PutProfile(Profile{Name: "检测页", Kernel: "chromium", Fingerprint: fp, StartURLs: []string{srv.URL + "/own1", srv.URL + "/own2"}}).ID
	defer runReg.stop(id)

	type page struct {
		URL    string
		Origin float64
	}
	pages := func() []page {
		rs, ok := runReg.get(id)
		if !ok {
			return nil
		}
		raw, err := cdpCall(rs.WSURL, "Target.getTargets", nil)
		if err != nil {
			return nil
		}
		var tl struct {
			TargetInfos []struct{ Type, URL, TargetID string } `json:"targetInfos"`
		}
		_ = json.Unmarshal(raw, &tl)
		out := []page{}
		for _, ti := range tl.TargetInfos {
			if ti.Type == "page" {
				out = append(out, page{URL: strings.TrimPrefix(ti.URL, srv.URL)})
			}
		}
		sort.Slice(out, func(i, j int) bool { return out[i].URL < out[j].URL })
		return out
	}
	urls := func(l []page) string {
		s := []string{}
		for _, p := range l {
			s = append(s, p.URL)
		}
		return strings.Join(s, ",")
	}
	wait := func(want string) string {
		got := ""
		for deadline := time.Now().Add(20 * time.Second); time.Now().Before(deadline); time.Sleep(250 * time.Millisecond) {
			if got = urls(pages()); got == want {
				break
			}
		}
		return got
	}
	stop := func() {
		a.StopProfile(id)
		if !waitStopped(id, 15*time.Second) {
			t.Fatal("profile did not stop")
		}
		time.Sleep(500 * time.Millisecond)
	}
	// timeOrigin per page: the check page must be the last one opened (the
	// foreground tab of Target.createTarget).
	lastOpened := func() string {
		rs, _ := runReg.get(id)
		raw, _ := cdpCall(rs.WSURL, "Target.getTargets", nil)
		var tl struct {
			TargetInfos []struct{ Type, URL, TargetID string } `json:"targetInfos"`
		}
		_ = json.Unmarshal(raw, &tl)
		best, bestT := "", 0.0
		for _, ti := range tl.TargetInfos {
			if ti.Type != "page" {
				continue
			}
			var v float64
			if err := evalInTarget(rs.WSURL, ti.TargetID, "performance.timeOrigin", &v); err == nil && v > bestT {
				best, bestT = strings.TrimPrefix(ti.URL, srv.URL), v
			}
		}
		return best
	}

	// 1. first UI launch: own pages + check page, check opened last
	if _, err := a.StartProfile(id); err != nil {
		t.Fatal(err)
	}
	if got := wait("/check,/own1,/own2"); got != "/check,/own1,/own2" {
		t.Fatalf("first UI launch tabs: %s", got)
	}
	if l := lastOpened(); l != "/check" {
		t.Errorf("check page should be the last-opened tab, got %q", l)
	}
	p, _ := s.GetProfile(id)
	if len(p.StartURLs) != 2 || p.FpCheckAt == 0 {
		t.Fatalf("start pages must be kept and the once marker set: %v at=%d", p.StartURLs, p.FpCheckAt)
	}
	stop()

	// 2. 仅首次启动: second UI launch has no check page
	if _, err := a.StartProfile(id); err != nil {
		t.Fatal(err)
	}
	time.Sleep(3 * time.Second)
	if got := wait("/own1,/own2"); got != "/own1,/own2" {
		t.Fatalf("second launch (once) tabs: %s", got)
	}
	stop()

	// 3. every launch, but started from the local API path: no check page
	st := s.GetSettings()
	st.FpCheckOnce = false
	s.PutSettings(st)
	if _, err := s.Launch(id); err != nil {
		t.Fatal(err)
	}
	time.Sleep(3 * time.Second)
	if got := wait("/own1,/own2"); got != "/own1,/own2" {
		t.Fatalf("API launch must not open the check page: %s", got)
	}
	stop()
	// RPA path (startProfile without UI)
	if _, err := a.startProfile(id, LaunchOpts{}); err != nil {
		t.Fatal(err)
	}
	time.Sleep(3 * time.Second)
	if got := wait("/own1,/own2"); got != "/own1,/own2" {
		t.Fatalf("RPA launch must not open the check page: %s", got)
	}
	stop()

	// 4. built-in self check, profile without start pages: one tab with the page
	p, _ = s.GetProfile(id)
	p.StartURLs = nil
	p.FpCheckSite = "builtin"
	s.PutProfile(*p)
	if _, err := a.StartProfile(id); err != nil {
		t.Fatal(err)
	}
	var title string
	for deadline := time.Now().Add(20 * time.Second); time.Now().Before(deadline); time.Sleep(300 * time.Millisecond) {
		l := pages()
		if len(l) == 1 && strings.Contains(l[0].URL, "/fpcheck/") {
			rs, _ := runReg.get(id)
			raw, _ := cdpCall(rs.WSURL, "Target.getTargets", nil)
			var tl struct {
				TargetInfos []struct{ Type, Title string } `json:"targetInfos"`
			}
			_ = json.Unmarshal(raw, &tl)
			for _, ti := range tl.TargetInfos {
				if ti.Type == "page" {
					title = ti.Title
				}
			}
			if strings.Contains(title, "项不一致") {
				break
			}
		}
	}
	if !strings.Contains(title, "内置自检") || !strings.Contains(title, "项不一致") {
		t.Fatalf("built-in page did not render: tabs=%v title=%q", pages(), title)
	}
	t.Logf("built-in page title: %s", title)
	rs, _ := runReg.get(id)
	raw, _ := cdpCall(rs.WSURL, "Target.getTargets", nil)
	var tl struct {
		TargetInfos []struct{ Type, TargetID string } `json:"targetInfos"`
	}
	_ = json.Unmarshal(raw, &tl)
	var rows [][]string
	for _, ti := range tl.TargetInfos {
		if ti.Type == "page" {
			var js string
			if err := evalInTarget(rs.WSURL, ti.TargetID, `JSON.stringify([...document.querySelectorAll('#rows tr')].map(tr => [...tr.children].map(td => td.textContent)))`, &js); err != nil {
				t.Fatal(err)
			}
			_ = json.Unmarshal([]byte(js), &rows)
		}
	}
	status := map[string]string{}
	for _, r := range rows {
		t.Logf("  %v", r)
		if len(r) == 4 {
			status[r[0]] = r[3]
		}
	}
	for _, k := range []string{"User-Agent", "navigator.platform", "语言 languages", "时区", "CPU 核心数", "屏幕分辨率", "navigator.webdriver"} {
		if st := status[k]; st != "一致" && !strings.HasPrefix(st, "≈") {
			t.Errorf("built-in page row %s: %q", k, st)
		}
	}
	stop()
}

// evalInTarget evaluates expr in one page over its own DevTools socket.
func evalInTarget(browserWS, targetID, expr string, out any) error {
	i := strings.Index(browserWS, "/devtools/")
	if i < 0 {
		return fmt.Errorf("bad ws %s", browserWS)
	}
	raw, err := cdpCall(browserWS[:i]+"/devtools/page/"+targetID, "Runtime.evaluate", map[string]any{"expression": expr, "returnByValue": true})
	if err != nil {
		return err
	}
	var r struct {
		Result struct {
			Value json.RawMessage `json:"value"`
		} `json:"result"`
	}
	if err := json.Unmarshal(raw, &r); err != nil {
		return err
	}
	return json.Unmarshal(r.Result.Value, out)
}
